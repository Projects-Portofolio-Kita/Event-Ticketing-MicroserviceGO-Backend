package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
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
	createErr    error
	updateErr    error
}

func newFakeStore() *fakeStore {
	return &fakeStore{reservations: map[uuid.UUID]*model.Reservation{}}
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

func (f *fakeStore) FindByID(id uuid.UUID) (*model.Reservation, error) {
	reservation, ok := f.reservations[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return reservation, nil
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

// fakeNotifier records what the service tried to tell the user.
type fakeNotifier struct {
	sent []notify.Message
}

func (f *fakeNotifier) Send(msg notify.Message) { f.sent = append(f.sent, msg) }

func newTestService(store *fakeStore) (*ReservationService, *fakeNotifier) {
	notifier := &fakeNotifier{}
	return NewReservationService(store, notifier), notifier
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
