package service

import (
	"errors"
	"testing"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// fakeStore is an in-memory stand-in for PaymentRepository.
type fakeStore struct {
	payments map[uuid.UUID]*model.Payment
	byReserv map[uuid.UUID]*model.Payment
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		payments: map[uuid.UUID]*model.Payment{},
		byReserv: map[uuid.UUID]*model.Payment{},
	}
}

func (f *fakeStore) Create(payment *model.Payment) error {
	if payment.ID == uuid.Nil {
		payment.ID = uuid.New()
	}
	f.payments[payment.ID] = payment
	f.byReserv[payment.ReservationID] = payment
	return nil
}

func (f *fakeStore) FindByID(id uuid.UUID) (*model.Payment, error) {
	payment, ok := f.payments[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return payment, nil
}

func (f *fakeStore) FindByReservationID(reservationID uuid.UUID) (*model.Payment, error) {
	payment, ok := f.byReserv[reservationID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return payment, nil
}

func (f *fakeStore) Update(payment *model.Payment) error {
	f.payments[payment.ID] = payment
	f.byReserv[payment.ReservationID] = payment
	return nil
}

// fakeBookingClient stands in for booking-service.
type fakeBookingClient struct {
	reservation *Reservation
	getErr      error
	confirmErr  error
	confirmed   []uuid.UUID
}

func (f *fakeBookingClient) GetReservation(uuid.UUID) (*Reservation, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.reservation, nil
}

func (f *fakeBookingClient) ConfirmReservation(id uuid.UUID) error {
	if f.confirmErr != nil {
		return f.confirmErr
	}
	f.confirmed = append(f.confirmed, id)
	return nil
}

type fakeNotifier struct {
	sent []notify.Message
}

func (f *fakeNotifier) Send(msg notify.Message) { f.sent = append(f.sent, msg) }

type fixture struct {
	service  *PaymentService
	store    *fakeStore
	booking  *fakeBookingClient
	notifier *fakeNotifier
	caller   authz.Caller
}

// newFixture wires a service with a LOCKED reservation owned by the caller.
func newFixture(t *testing.T) *fixture {
	t.Helper()

	caller := authz.Caller{UserID: uuid.NewString()}
	store := newFakeStore()
	booking := &fakeBookingClient{
		reservation: &Reservation{
			ID:          uuid.New(),
			UserID:      uuid.MustParse(caller.UserID),
			TotalAmount: 250000,
			Currency:    "IDR",
			ExpiredAt:   time.Now().Add(5 * time.Minute),
			Status:      "LOCKED",
		},
	}
	notifier := &fakeNotifier{}

	return &fixture{
		service:  NewPaymentService(store, provider.NewMockQRISProvider(), booking, notifier),
		store:    store,
		booking:  booking,
		notifier: notifier,
		caller:   caller,
	}
}

func (f *fixture) validRequest() *dto.CreatePaymentRequest {
	return &dto.CreatePaymentRequest{
		ReservationID: f.booking.reservation.ID.String(),
		PaymentMethod: "QRIS",
	}
}

func TestCreatePaymentIssuesAQRForTheOwnersReservation(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	if payment.Status != model.PaymentPending {
		t.Errorf("status = %s, want PENDING", payment.Status)
	}
	if payment.Amount != 250000 {
		t.Errorf("amount = %v, want 250000", payment.Amount)
	}
	if payment.UserID.String() != f.caller.UserID {
		t.Errorf("owner = %s, want %s", payment.UserID, f.caller.UserID)
	}
	if payment.QRString == "" {
		t.Error("expected a QR string from the provider")
	}
	if payment.ID == uuid.Nil {
		t.Error("expected a generated payment id")
	}
}

func TestCreatePaymentRefusesForeignReservation(t *testing.T) {
	f := newFixture(t)

	if _, err := f.service.CreatePayment(f.validRequest(), authz.Caller{UserID: uuid.NewString()}); !errors.Is(err, ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
	if len(f.store.payments) != 0 {
		t.Errorf("payments created = %d, want 0", len(f.store.payments))
	}
}

func TestCreatePaymentValidatesInput(t *testing.T) {
	f := newFixture(t)

	t.Run("malformed reservation id", func(t *testing.T) {
		req := &dto.CreatePaymentRequest{ReservationID: "nope", PaymentMethod: "QRIS"}
		if _, err := f.service.CreatePayment(req, f.caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("unsupported method", func(t *testing.T) {
		req := f.validRequest()
		req.PaymentMethod = "CARD"
		if _, err := f.service.CreatePayment(req, f.caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("reservation not LOCKED", func(t *testing.T) {
		other := newFixture(t)
		other.booking.reservation.Status = "CONFIRMED"
		if _, err := other.service.CreatePayment(other.validRequest(), other.caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("reservation expired", func(t *testing.T) {
		other := newFixture(t)
		other.booking.reservation.ExpiredAt = time.Now().Add(-time.Minute)
		if _, err := other.service.CreatePayment(other.validRequest(), other.caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})

	t.Run("booking unreachable", func(t *testing.T) {
		other := newFixture(t)
		other.booking.getErr = errors.New("booking-service returned status 500")
		if _, err := other.service.CreatePayment(other.validRequest(), other.caller); err == nil {
			t.Error("expected the booking error to be propagated")
		}
	})
}

func TestCreatePaymentIsIdempotentPerReservation(t *testing.T) {
	f := newFixture(t)

	first, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("first CreatePayment: %v", err)
	}

	second, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("second CreatePayment: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("second payment id = %s, want the existing %s", second.ID, first.ID)
	}
	if len(f.store.payments) != 1 {
		t.Errorf("payments stored = %d, want 1", len(f.store.payments))
	}
}

func TestGetPaymentEnforcesOwnership(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	id := payment.ID.String()

	t.Run("owner", func(t *testing.T) {
		if _, err := f.service.GetPayment(id, f.caller); err != nil {
			t.Errorf("GetPayment: %v", err)
		}
	})

	t.Run("stranger", func(t *testing.T) {
		if _, err := f.service.GetPayment(id, authz.Caller{UserID: uuid.NewString()}); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("empty subject", func(t *testing.T) {
		if _, err := f.service.GetPayment(id, authz.Caller{}); !errors.Is(err, ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("internal service", func(t *testing.T) {
		if _, err := f.service.GetPayment(id, authz.Caller{Internal: true}); err != nil {
			t.Errorf("GetPayment: %v", err)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		if _, err := f.service.GetPayment(uuid.NewString(), f.caller); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("err = %v, want gorm.ErrRecordNotFound", err)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		if _, err := f.service.GetPayment("nope", f.caller); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("err = %v, want ErrInvalidInput", err)
		}
	})
}

func TestConfirmPaymentMarksReservationConfirmed(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	confirmed, err := f.service.ConfirmPayment(payment.ID.String(), f.caller)
	if err != nil {
		t.Fatalf("ConfirmPayment: %v", err)
	}
	if confirmed.Status != model.PaymentSucceeded {
		t.Errorf("status = %s, want SUCCEEDED", confirmed.Status)
	}
	if len(f.booking.confirmed) != 1 || f.booking.confirmed[0] != payment.ReservationID {
		t.Errorf("booking confirmed = %v, want [%s]", f.booking.confirmed, payment.ReservationID)
	}
	if len(f.notifier.sent) != 1 || f.notifier.sent[0].Kind != notify.KindPaymentSucceeded {
		t.Errorf("notifications = %+v, want one payment-succeeded message", f.notifier.sent)
	}
	if f.notifier.sent[0].UserID != f.caller.UserID {
		t.Errorf("notified user = %q, want %q", f.notifier.sent[0].UserID, f.caller.UserID)
	}
}

func TestConfirmPaymentEnforcesOwnership(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	if _, err := f.service.ConfirmPayment(payment.ID.String(), authz.Caller{UserID: uuid.NewString()}); !errors.Is(err, ErrForbidden) {
		t.Errorf("err = %v, want ErrForbidden", err)
	}
	if len(f.booking.confirmed) != 0 {
		t.Errorf("a stranger triggered booking confirmation: %v", f.booking.confirmed)
	}
}

func TestConfirmPaymentIsIdempotent(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if _, err := f.service.ConfirmPayment(payment.ID.String(), f.caller); err != nil {
		t.Fatalf("first ConfirmPayment: %v", err)
	}

	again, err := f.service.ConfirmPayment(payment.ID.String(), f.caller)
	if err != nil {
		t.Fatalf("second ConfirmPayment: %v", err)
	}
	if again.Status != model.PaymentSucceeded {
		t.Errorf("status = %s, want SUCCEEDED", again.Status)
	}
	if len(f.booking.confirmed) != 1 {
		t.Errorf("booking confirmed calls = %d, want 1", len(f.booking.confirmed))
	}
	if len(f.notifier.sent) != 1 {
		t.Errorf("notifications = %d, want 1 (no duplicate on retry)", len(f.notifier.sent))
	}
}

func TestConfirmPaymentRejectsExpiredPayment(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	payment.ExpiredAt = time.Now().Add(-time.Minute)

	if _, err := f.service.ConfirmPayment(payment.ID.String(), f.caller); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if f.store.payments[payment.ID].Status != model.PaymentExpired {
		t.Errorf("status = %s, want EXPIRED persisted", f.store.payments[payment.ID].Status)
	}
	if len(f.booking.confirmed) != 0 {
		t.Error("an expired payment must not confirm its reservation")
	}
}

func TestConfirmPaymentPropagatesBookingFailure(t *testing.T) {
	f := newFixture(t)

	payment, err := f.service.CreatePayment(f.validRequest(), f.caller)
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	f.booking.confirmErr = errors.New("reservasi tidak dalam status LOCKED")

	if _, err := f.service.ConfirmPayment(payment.ID.String(), f.caller); err == nil {
		t.Fatal("expected the booking error to propagate")
	}
	if f.store.payments[payment.ID].Status != model.PaymentPending {
		t.Errorf("status = %s, want the payment to stay PENDING", f.store.payments[payment.ID].Status)
	}
	if len(f.notifier.sent) != 0 {
		t.Errorf("a failed confirmation must not notify, got %d", len(f.notifier.sent))
	}
}
