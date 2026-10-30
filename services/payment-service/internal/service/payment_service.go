package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrForbidden           = errors.New("Anda tidak punya akses ke pembayaran ini.")
	ErrInvalidInput        = errors.New("invalid input")
	ErrReservationNotFound = errors.New("reservasi tidak ditemukan")
)

type Reservation struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	TotalAmount float64
	Currency    string
	ExpiredAt   time.Time
	Status      string
}

type BookingClient interface {
	GetReservation(reservationID uuid.UUID) (*Reservation, error)
	ConfirmReservation(reservationID uuid.UUID) error
}

// paymentStore is the slice of the repository this service depends on.
// Keeping it an interface lets tests substitute an in-memory fake.
type paymentStore interface {
	Create(payment *model.Payment) error
	FindByID(id uuid.UUID) (*model.Payment, error)
	FindByReservationID(reservationID uuid.UUID) (*model.Payment, error)
	Update(payment *model.Payment) error
}

type PaymentService struct {
	repository   paymentStore
	qrisProvider provider.QRISProvider
	booking      BookingClient
	notifier     notify.Notifier
}

func NewPaymentService(repository paymentStore, qrisProvider provider.QRISProvider, booking BookingClient, notifier notify.Notifier) *PaymentService {
	if notifier == nil {
		notifier = notify.Noop()
	}
	return &PaymentService{
		repository:   repository,
		qrisProvider: qrisProvider,
		booking:      booking,
		notifier:     notifier,
	}
}

// ensureOwnership rejects a non-internal caller that is not the owner of the
// given user id. This is the only gate between two users' payments.
func ensureOwnership(caller authz.Caller, owner uuid.UUID) error {
	if caller.Internal {
		return nil
	}
	if caller.UserID == "" || caller.UserID != owner.String() {
		return ErrForbidden
	}
	return nil
}

func (s *PaymentService) CreatePayment(req *dto.CreatePaymentRequest, caller authz.Caller) (*model.Payment, error) {

	reservationID, err := uuid.Parse(req.ReservationID)
	if err != nil {
		return nil, fmt.Errorf("%w: reservation id", ErrInvalidInput)
	}
	if reservationID == uuid.Nil {
		return nil, fmt.Errorf("%w: reservation id", ErrInvalidInput)
	}

	if strings.ToUpper(req.PaymentMethod) != "QRIS" {
		return nil, fmt.Errorf("%w: metode pembayaran harus QRIS", ErrInvalidInput)
	}

	reservation, err := s.booking.GetReservation(reservationID)
	if err != nil {
		return nil, err
	}

	// The reservation belongs to whoever holds its token: a caller may only
	// pay for their own reservation.
	if err := ensureOwnership(caller, reservation.UserID); err != nil {
		return nil, err
	}

	if reservation.Status != "LOCKED" {
		return nil, fmt.Errorf("%w: reservasi tidak dalam status LOCKED", ErrInvalidInput)
	}

	if time.Now().After(reservation.ExpiredAt) {
		return nil, fmt.Errorf("%w: reservasi sudah kedaluwarsa", ErrInvalidInput)
	}

	existingPayment, err := s.repository.FindByReservationID(reservationID)

	if err == nil && existingPayment != nil {
		return existingPayment, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		return nil, err
	}

	payment := &model.Payment{
		ID:            uuid.New(),
		ReservationID: reservation.ID,
		Amount:        reservation.TotalAmount,
		Currency:      reservation.Currency,
		Status:        model.PaymentPending,
		ExpiredAt:     reservation.ExpiredAt,
		Gateway:       model.PaymentGatewayMidtrans,
		UserID:        reservation.UserID,
	}
	qrisResponse, err := s.qrisProvider.CreatePayment(provider.QRISRequest{
		OrderID: payment.ID.String(),
		Amount:  payment.Amount,
	})

	if err != nil {
		return nil, err
	}

	payment.ProviderPaymentID = qrisResponse.ProviderPaymentID
	payment.QRISReference = qrisResponse.QRISReference
	payment.QRString = qrisResponse.QRString
	payment.QRURL = qrisResponse.QRURL
	payment.ExpiredAt = qrisResponse.ExpiredAt

	if err := s.repository.Create(payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) GetPayment(id string, caller authz.Caller) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: payment id", ErrInvalidInput)
	}

	payment, err := s.repository.FindByID(paymentID)
	if err != nil {
		return nil, err
	}

	if err := ensureOwnership(caller, payment.UserID); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) ConfirmPayment(id string, caller authz.Caller) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("%w: payment id", ErrInvalidInput)
	}

	payment, err := s.repository.FindByID(paymentID)
	if err != nil {
		return nil, err
	}

	if err := ensureOwnership(caller, payment.UserID); err != nil {
		return nil, err
	}

	if payment.Status == model.PaymentSucceeded || payment.Status == model.PaymentConfirmed {
		return payment, nil
	}

	if payment.Status != model.PaymentPending {
		return nil, fmt.Errorf("%w: pembayaran tidak dalam status PENDING", ErrInvalidInput)
	}

	if time.Now().After(payment.ExpiredAt) {
		payment.Status = model.PaymentExpired
		_ = s.repository.Update(payment)
		return nil, fmt.Errorf("%w: pembayaran sudah kedaluwarsa", ErrInvalidInput)
	}

	if err := s.booking.ConfirmReservation(payment.ReservationID); err != nil {
		return nil, err
	}

	payment.Status = model.PaymentSucceeded
	if err := s.repository.Update(payment); err != nil {
		return nil, err
	}

	s.notifier.Send(notify.Message{
		UserID:    payment.UserID.String(),
		Kind:      notify.KindPaymentSucceeded,
		Title:     "Pembayaran berhasil",
		Body:      fmt.Sprintf("Pembayaran %s %.0f (%s) untuk reservasi %s diterima.", payment.Currency, payment.Amount, payment.ID, payment.ReservationID),
		Reference: payment.ID.String(),
	})

	return payment, nil
}
