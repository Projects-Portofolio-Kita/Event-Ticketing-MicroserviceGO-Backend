package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/client"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakeStore is an in-memory stand-in for ReservationRepository.
type fakeStore struct {
	reservations map[uuid.UUID]*model.Reservation
	conflicts    []uuid.UUID
	seatStatus   []repository.SeatStatus
	ticketStatus []repository.TicketStatus
	// tierReserved is the "fresh count under the advisory lock" that the real
	// repository would compute; the quota verify closure reads it.
	tierReserved map[uuid.UUID]int
	createErr    error
	updateErr    error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		reservations: map[uuid.UUID]*model.Reservation{},
		tierReserved: map[uuid.UUID]int{},
	}
}

func (f *fakeStore) Create(reservation *model.Reservation) error {
	if f.createErr != nil {
		return f.createErr
	}
	// Mirror the BeforeCreate hook of the real repository.
	if reservation.ID == uuid.Nil {
		reservation.ID = uuid.New()
	}
	f.reservations[reservation.ID] = reservation
	return nil
}

// CreateTier mirrors the real repository: run the quota verify closure with
// the reserved counts, then persist only when it passes.
func (f *fakeStore) CreateTier(reservation *model.Reservation, verify func(reserved map[uuid.UUID]int) error) error {
	if f.createErr != nil {
		return f.createErr
	}
	if err := verify(f.tierReserved); err != nil {
		return err
	}
	return f.Create(reservation)
}

func (f *fakeStore) FindByID(id uuid.UUID) (*model.Reservation, error) {
	reservation, ok := f.reservations[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return reservation, nil
}

func (f *fakeStore) FindByUser(userID uuid.UUID, limit int) ([]model.Reservation, error) {
	result := []model.Reservation{}
	for _, reservation := range f.reservations {
		if reservation.UserID == userID {
			result = append(result, *reservation)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (f *fakeStore) Update(reservation *model.Reservation) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.reservations[reservation.ID] = reservation
	return nil
}

func (f *fakeStore) FindConflictingSeats(seatIDs []uuid.UUID) ([]uuid.UUID, error) {
	return f.conflicts, nil
}

func (f *fakeStore) FindSeatStatusByEvent(uuid.UUID) ([]repository.SeatStatus, error) {
	return f.seatStatus, nil
}

func (f *fakeStore) FindTicketStatusByEvent(uuid.UUID) ([]repository.TicketStatus, error) {
	return f.ticketStatus, nil
}

// fakeCatalogClient stands in for catalog-service's tier stock feed.
type fakeCatalogClient struct {
	tiers []client.TierQuota
	err   error
}

func (f *fakeCatalogClient) GetTierQuotas(string) ([]client.TierQuota, error) {
	return f.tiers, f.err
}

// fakeNotifier records what the service tried to tell the user.
type fakeNotifier struct {
	sent []notify.Message
}

func (f *fakeNotifier) Send(msg notify.Message) { f.sent = append(f.sent, msg) }

func newTestService(store *fakeStore) (*ReservationService, *fakeNotifier) {
	return newTierTestService(store, nil)
}

// newTierTestService wires a catalog stock feed, needed by TIER bookings.
func newTierTestService(store *fakeStore, catalog client.CatalogClient) (*ReservationService, *fakeNotifier) {
	notifier := &fakeNotifier{}
	return NewReservationService(store, catalog, notifier), notifier
}

// owner is the principal whose token opened the session.
func owner() authz.Caller {
	return authz.Caller{UserID: uuid.NewString()}
}

func otherUser() authz.Caller {
	return authz.Caller{UserID: uuid.NewString()}
}

func internalCall() authz.Caller {
	return authz.Caller{UserID: uuid.NewString(), Internal: true}
}

func createRequest(t *testing.T, caller authz.Caller) *dto.CreateReservationRequest {
	t.Helper()

	seats := make([]dto.SeatItemRequest, 0, 2)
	for i := 0; i < 2; i++ {
		seats = append(seats, dto.SeatItemRequest{
			SeatID:  uuid.NewString(),
			Section: "VIP",
			Row:     "A",
			Number:  "01",
			Price:   100,
		})
	}

	return &dto.CreateReservationRequest{
		UserID:  caller.UserID,
		EventID: uuid.NewString(),
		Seats:   seats,
	}
}

func TestCreateReservationTakesIdentityFromTheCaller(t *testing.T) {
	store := newFakeStore()
	service, notifier := newTestService(store)
	caller := owner()

	// A body claiming to be somebody else must not matter: the token wins.
	req := createRequest(t, caller)
	req.UserID = uuid.NewString()

	reservation, err := service.CreateReservation(req, caller)
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}

	if reservation.UserID.String() != caller.UserID {
		t.Errorf("owner = %s, want the token subject %s", reservation.UserID, caller.UserID)
	}
	if reservation.Status != model.ReservationLocked {
		t.Errorf("status = %s, want LOCKED", reservation.Status)
	}
	if !strings.HasPrefix(reservation.BookingRef, "TICK-") {
		t.Errorf("bookingRef = %q, want a TICK- reference", reservation.BookingRef)
	}
	if len(reservation.Seats) != 2 {
		t.Errorf("seats = %d, want 2", len(reservation.Seats))
	}
	if reservation.TotalAmount != 200 {
		t.Errorf("totalAmount = %v, want 200", reservation.TotalAmount)
	}

	if len(notifier.sent) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.sent))
	}
	if notifier.sent[0].Kind != notify.KindReservationLocked {
		t.Errorf("kind = %q, want %q", notifier.sent[0].Kind, notify.KindReservationLocked)
	}
	if notifier.sent[0].UserID != caller.UserID {
		t.Errorf("notified user = %q, want %q", notifier.sent[0].UserID, caller.UserID)
	}
}

func TestCreateReservationRejectsBadInput(t *testing.T) {
	caller := owner()

	for name, mutate := range map[string]func(req *dto.CreateReservationRequest){
		"no seats": func(req *dto.CreateReservationRequest) {
			req.Seats = nil
		},
		"bad seat id": func(req *dto.CreateReservationRequest) {
			req.Seats[0].SeatID = "not-a-uuid"
		},
		"bad event id": func(req *dto.CreateReservationRequest) {
			req.EventID = "not-a-uuid"
		},
	} {
		t.Run(name, func(t *testing.T) {
			service, _ := newTestService(newFakeStore())
			req := createRequest(t, caller)
			mutate(req)

			if _, err := service.CreateReservation(req, caller); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestCreateReservationFailsWhenSeatsAreTaken(t *testing.T) {
	store := newFakeStore()
	store.conflicts = []uuid.UUID{uuid.New()}
	service, notifier := newTestService(store)
	caller := owner()

	_, err := service.CreateReservation(createRequest(t, caller), caller)
	if !errors.Is(err, ErrSeatsUnavailable) {
		t.Fatalf("err = %v, want ErrSeatsUnavailable", err)
	}
	if len(notifier.sent) != 0 {
		t.Errorf("a failed booking must not notify, got %d messages", len(notifier.sent))
	}
}

func TestCreateReservationRequiresASubject(t *testing.T) {
	service, _ := newTestService(newFakeStore())

	_, err := service.CreateReservation(createRequest(t, authz.Caller{}), authz.Caller{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}

func TestGetReservationEnforcesOwnership(t *testing.T) {
	store := newFakeStore()
	service, _ := newTestService(store)

	stored := &model.Reservation{
		UserID:     uuid.MustParse(owner().UserID),
		EventID:    uuid.New(),
		BookingRef: "TICK-1-0001",
		Status:     model.ReservationLocked,
	}
	if err := store.Create(stored); err != nil {
		t.Fatalf("seed: %v", err)
	}
	id := stored.ID.String()

	t.Run("owner", func(t *testing.T) {
		got, err := service.GetReservation(id, authz.Caller{UserID: stored.UserID.String()})
		if err != nil {
			t.Fatalf("GetReservation: %v", err)
		}
		if got.ID != stored.ID {
			t.Errorf("id = %s, want %s", got.ID, stored.ID)
		}
	})

	t.Run("stranger", func(t *testing.T) {
		if _, err := service.GetReservation(id, otherUser()); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("empty subject", func(t *testing.T) {
		if _, err := service.GetReservation(id, authz.Caller{}); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("internal service", func(t *testing.T) {
		if _, err := service.GetReservation(id, internalCall()); err != nil {
			t.Errorf("GetReservation: %v", err)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		if _, err := service.GetReservation(uuid.NewString(), owner()); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("err = %v, want gorm.ErrRecordNotFound", err)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		if _, err := service.GetReservation("nope", owner()); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}

func TestConfirmReservationLifecycle(t *testing.T) {
	newLocked := func(t *testing.T) (*fakeStore, *ReservationService, *fakeNotifier, *model.Reservation, authz.Caller) {
		store := newFakeStore()
		service, notifier := newTestService(store)
		caller := owner()

		stored := &model.Reservation{
			UserID:    uuid.MustParse(caller.UserID),
			EventID:   uuid.New(),
			Status:    model.ReservationLocked,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		if err := store.Create(stored); err != nil {
			t.Fatalf("seed: %v", err)
		}
		return store, service, notifier, stored, caller
	}

	t.Run("owner confirms", func(t *testing.T) {
		store, service, notifier, stored, caller := newLocked(t)

		got, err := service.ConfirmReservation(stored.ID.String(), caller)
		if err != nil {
			t.Fatalf("ConfirmReservation: %v", err)
		}
		if got.Status != model.ReservationConfirmed {
			t.Errorf("status = %s, want CONFIRMED", got.Status)
		}
		if store.reservations[stored.ID].Status != model.ReservationConfirmed {
			t.Error("the change was not persisted")
		}
		if len(notifier.sent) != 1 || notifier.sent[0].Kind != notify.KindBookingConfirmed {
			t.Errorf("notifications = %+v, want one booking-confirmed message", notifier.sent)
		}
	})

	t.Run("stranger is refused", func(t *testing.T) {
		_, service, _, stored, _ := newLocked(t)

		if _, err := service.ConfirmReservation(stored.ID.String(), otherUser()); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("internal service confirms", func(t *testing.T) {
		_, service, _, stored, _ := newLocked(t)

		got, err := service.ConfirmReservation(stored.ID.String(), internalCall())
		if err != nil {
			t.Fatalf("ConfirmReservation: %v", err)
		}
		if got.Status != model.ReservationConfirmed {
			t.Errorf("status = %s, want CONFIRMED", got.Status)
		}
	})

	t.Run("only LOCKED can be confirmed", func(t *testing.T) {
		_, service, _, stored, caller := newLocked(t)
		stored.Status = model.ReservationConfirmed

		if _, err := service.ConfirmReservation(stored.ID.String(), caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("expired lock cannot be confirmed", func(t *testing.T) {
		store, service, _, stored, caller := newLocked(t)
		stored.ExpiresAt = time.Now().Add(-time.Minute)

		if _, err := service.ConfirmReservation(stored.ID.String(), caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
		if store.reservations[stored.ID].Status != model.ReservationExpired {
			t.Errorf("status = %s, want the lock to be marked EXPIRED", store.reservations[stored.ID].Status)
		}
	})
}

func TestCancelReservationLifecycle(t *testing.T) {
	seed := func(t *testing.T, status model.ReservationStatus) (*fakeStore, *ReservationService, *fakeNotifier, *model.Reservation, authz.Caller) {
		store := newFakeStore()
		service, notifier := newTestService(store)
		caller := owner()

		stored := &model.Reservation{
			UserID:  uuid.MustParse(caller.UserID),
			EventID: uuid.New(),
			Status:  status,
		}
		if err := store.Create(stored); err != nil {
			t.Fatalf("seed: %v", err)
		}
		return store, service, notifier, stored, caller
	}

	t.Run("owner cancels a lock", func(t *testing.T) {
		store, service, notifier, stored, caller := seed(t, model.ReservationLocked)

		got, err := service.CancelReservation(stored.ID.String(), caller)
		if err != nil {
			t.Fatalf("CancelReservation: %v", err)
		}
		if got.Status != model.ReservationCancelled {
			t.Errorf("status = %s, want CANCELLED", got.Status)
		}
		if store.reservations[stored.ID].Status != model.ReservationCancelled {
			t.Error("the change was not persisted")
		}
		if len(notifier.sent) != 1 || notifier.sent[0].Kind != notify.KindBookingCancelled {
			t.Errorf("notifications = %+v, want one booking-cancelled message", notifier.sent)
		}
	})

	t.Run("confirmed tickets are untouchable", func(t *testing.T) {
		_, service, _, stored, caller := seed(t, model.ReservationConfirmed)

		if _, err := service.CancelReservation(stored.ID.String(), caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("stranger is refused", func(t *testing.T) {
		_, service, _, stored, _ := seed(t, model.ReservationLocked)

		if _, err := service.CancelReservation(stored.ID.String(), otherUser()); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})
}

func TestGetSeatStatusValidatesEventID(t *testing.T) {
	service, _ := newTestService(newFakeStore())

	if _, err := service.GetSeatStatus("nope"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}

	store := newFakeStore()
	store.seatStatus = []repository.SeatStatus{{SeatID: uuid.New(), Status: "LOCKED"}}
	service, _ = newTestService(store)

	got, err := service.GetSeatStatus(uuid.NewString())
	if err != nil {
		t.Fatalf("GetSeatStatus: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("statuses = %d, want 1", len(got))
	}
}

// --- quota-based (TIER) reservations ---------------------------------------

// tierCatalog is the stock feed: tier ids map to catalog's price + quota.
func tierCatalog(vvip, vip, regular client.TierQuota) *fakeCatalogClient {
	return &fakeCatalogClient{tiers: []client.TierQuota{vvip, vip, regular}}
}

func standardTiers() (vvip, vip, regular client.TierQuota) {
	vvip = client.TierQuota{CategoryID: uuid.NewString(), Name: "VVIP", Price: 1500000, Quota: 50}
	vip = client.TierQuota{CategoryID: uuid.NewString(), Name: "VIP", Price: 750000, Quota: 150}
	regular = client.TierQuota{CategoryID: uuid.NewString(), Name: "Regular", Price: 350000, Quota: 500}
	return vvip, vip, regular
}

// tierRequest mirrors what the desktop sends: tier id + quantity only.
func tierRequest(eventID string, items ...dto.TicketItemRequest) *dto.CreateReservationRequest {
	return &dto.CreateReservationRequest{EventID: eventID, Items: items}
}

func TestCreateTierReservationTakesPricesFromCatalog(t *testing.T) {
	vvip, vip, _ := standardTiers()
	store := newFakeStore()
	service, notifier := newTierTestService(store, tierCatalog(vvip, vip, client.TierQuota{}))
	caller := owner()

	// The client claims the tickets cost 1 each — the catalog decides.
	req := tierRequest(uuid.NewString(),
		dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 2},
		dto.TicketItemRequest{CategoryID: vvip.CategoryID, Quantity: 1},
	)

	reservation, err := service.CreateReservation(req, caller)
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}

	if reservation.Mode != model.ReservationModeTier {
		t.Errorf("mode = %q, want TIER", reservation.Mode)
	}
	if reservation.UserID.String() != caller.UserID {
		t.Errorf("owner = %s, want the token subject %s", reservation.UserID, caller.UserID)
	}
	// 2 x 750000 + 1 x 1500000 = 3.000.000
	if reservation.TotalAmount != 3000000 {
		t.Errorf("totalAmount = %v, want 3000000 (server prices)", reservation.TotalAmount)
	}
	if len(reservation.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(reservation.Items))
	}
	if reservation.Items[0].Quantity != 2 || reservation.Items[0].UnitPrice != 750000 {
		t.Errorf("first item = %+v, want 2 x 750000", reservation.Items[0])
	}
	if len(reservation.Seats) != 0 {
		t.Errorf("seats = %d, want 0 for a TIER reservation", len(reservation.Seats))
	}

	if len(notifier.sent) != 1 || notifier.sent[0].Kind != notify.KindReservationLocked {
		t.Errorf("notifications = %+v, want one locked message", notifier.sent)
	}
}

func TestCreateTierReservationMergesDuplicateLines(t *testing.T) {
	vvip, vip, regular := standardTiers()
	store := newFakeStore()
	service, _ := newTierTestService(store, tierCatalog(vvip, vip, regular))

	req := tierRequest(uuid.NewString(),
		dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 2},
		dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 3},
	)

	reservation, err := service.CreateReservation(req, owner())
	if err != nil {
		t.Fatalf("CreateReservation: %v", err)
	}
	if len(reservation.Items) != 1 {
		t.Fatalf("items = %d, want 1 merged line", len(reservation.Items))
	}
	if reservation.Items[0].Quantity != 5 {
		t.Errorf("quantity = %d, want 5", reservation.Items[0].Quantity)
	}
	if reservation.TotalAmount != 5*750000 {
		t.Errorf("totalAmount = %v, want %v", reservation.TotalAmount, 5*750000)
	}
}

func TestCreateTierReservationRejectsBadInput(t *testing.T) {
	_, vip, _ := standardTiers()
	eventID := uuid.NewString()

	for name, mutate := range map[string]func(req *dto.CreateReservationRequest){
		"no items": func(req *dto.CreateReservationRequest) {
			req.Items = nil
		},
		"bad category id": func(req *dto.CreateReservationRequest) {
			req.Items[0].CategoryID = "not-a-uuid"
		},
		"zero quantity": func(req *dto.CreateReservationRequest) {
			req.Items[0].Quantity = 0
		},
		"basket over the cap": func(req *dto.CreateReservationRequest) {
			req.Items = append(req.Items, dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 10})
			req.Items = append(req.Items, dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 10})
		},
		"unknown mode": func(req *dto.CreateReservationRequest) {
			req.Mode = "RAFFLE"
		},
		"tier mode without items": func(req *dto.CreateReservationRequest) {
			req.Mode = "TIER"
			req.Items = nil
		},
		"seated mode without seats": func(req *dto.CreateReservationRequest) {
			req.Mode = "SEATED"
			req.Seats = nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			service, notifier := newTierTestService(newFakeStore(), tierCatalog(vip, client.TierQuota{}, client.TierQuota{}))
			req := tierRequest(eventID, dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 1})
			mutate(req)

			if _, err := service.CreateReservation(req, owner()); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
			if len(notifier.sent) != 0 {
				t.Errorf("a rejected booking must not notify, got %d", len(notifier.sent))
			}
		})
	}
}

func TestCreateTierReservationFailsWhenQuotaIsGone(t *testing.T) {
	vvip, vip, _ := standardTiers()
	store := newFakeStore()
	// 150 of the 150 VIP tickets are already locked/confirmed.
	store.tierReserved[uuid.MustParse(vip.CategoryID)] = vip.Quota

	service, notifier := newTierTestService(store, tierCatalog(vvip, vip, client.TierQuota{}))

	req := tierRequest(uuid.NewString(), dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 1})

	_, err := service.CreateReservation(req, owner())
	if !errors.Is(err, ErrTicketsUnavailable) {
		t.Fatalf("err = %v, want ErrTicketsUnavailable", err)
	}
	if len(store.reservations) != 0 {
		t.Error("a rejected booking must not be persisted")
	}
	if len(notifier.sent) != 0 {
		t.Errorf("a failed booking must not notify, got %d messages", len(notifier.sent))
	}
}

func TestCreateTierReservationRejectsTierNotSoldHere(t *testing.T) {
	_, vip, _ := standardTiers()
	service, _ := newTierTestService(newFakeStore(), tierCatalog(vip, client.TierQuota{}, client.TierQuota{}))

	// A tier the event does not sell (e.g. it is a seated event).
	unknown := uuid.NewString()
	req := tierRequest(uuid.NewString(), dto.TicketItemRequest{CategoryID: unknown, Quantity: 1})

	if _, err := service.CreateReservation(req, owner()); !errors.Is(err, ErrTicketsUnavailable) {
		t.Errorf("err = %v, want ErrTicketsUnavailable", err)
	}
}

func TestCreateTierReservationPropagatesCatalogFailures(t *testing.T) {
	_, vip, _ := standardTiers()

	t.Run("unknown event", func(t *testing.T) {
		service, _ := newTierTestService(newFakeStore(), &fakeCatalogClient{err: client.ErrEventNotFound})
		req := tierRequest(uuid.NewString(), dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 1})

		if _, err := service.CreateReservation(req, owner()); !errors.Is(err, ErrEventNotFound) {
			t.Errorf("err = %v, want ErrEventNotFound", err)
		}
	})

	t.Run("catalog down", func(t *testing.T) {
		boom := errors.New("catalog-service unreachable")
		service, _ := newTierTestService(newFakeStore(), &fakeCatalogClient{err: boom})
		req := tierRequest(uuid.NewString(), dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 1})

		if _, err := service.CreateReservation(req, owner()); !errors.Is(err, boom) {
			t.Errorf("err = %v, want the catalog error", err)
		}
	})

	t.Run("store error", func(t *testing.T) {
		boom := errors.New("db down")
		store := newFakeStore()
		store.createErr = boom
		service, _ := newTierTestService(store, tierCatalog(vip, client.TierQuota{}, client.TierQuota{}))
		req := tierRequest(uuid.NewString(), dto.TicketItemRequest{CategoryID: vip.CategoryID, Quantity: 1})

		if _, err := service.CreateReservation(req, owner()); !errors.Is(err, boom) {
			t.Errorf("err = %v, want the store error", err)
		}
	})
}

func TestListReservationsReturnsOwnBookingsOnly(t *testing.T) {
	store := newFakeStore()
	service, _ := newTestService(store)

	mine := &model.Reservation{UserID: uuid.MustParse(owner().UserID), EventID: uuid.New(), BookingRef: "TICK-1-0001"}
	other := &model.Reservation{UserID: uuid.MustParse(otherUser().UserID), EventID: uuid.New(), BookingRef: "TICK-1-0002"}
	for _, reservation := range []*model.Reservation{mine, other} {
		if err := store.Create(reservation); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	got, err := service.ListReservations(authz.Caller{UserID: mine.UserID.String()}, 20)
	if err != nil {
		t.Fatalf("ListReservations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("reservations = %d, want only mine", len(got))
	}
	if got[0].ID != mine.ID {
		t.Errorf("id = %s, want %s", got[0].ID, mine.ID)
	}

	if _, err := service.ListReservations(authz.Caller{}, 20); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput without a token subject", err)
	}
}

func TestGetEventTicketStatusValidatesEventID(t *testing.T) {
	service, _ := newTestService(newFakeStore())

	if _, err := service.GetEventTicketStatus("nope"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}

	store := newFakeStore()
	store.ticketStatus = []repository.TicketStatus{{CategoryID: uuid.New(), Reserved: 7}}
	service, _ = newTestService(store)

	got, err := service.GetEventTicketStatus(uuid.NewString())
	if err != nil {
		t.Fatalf("GetEventTicketStatus: %v", err)
	}
	if len(got) != 1 || got[0].Reserved != 7 {
		t.Errorf("statuses = %+v, want one with 7 reserved", got)
	}
}
