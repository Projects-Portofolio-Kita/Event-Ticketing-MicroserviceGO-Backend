package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/notify"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

type PaymentService struct {
	repository   *repository.PaymentRepository
	qrisProvider *provider.MockQRISProvider
	booking      BookingClient
	notifier     notify.Notifier
}

func NewPaymentService(repository *repository.PaymentRepository, qrisProvider *provider.MockQRISProvider, booking BookingClient, notifier notify.Notifier) *PaymentService {
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

func (s *PaymentService) CreatePayment(req *dto.CreatePaymentRequest) (*model.Payment, error) {

	reservationID, err := uuid.Parse(req.ReservationID)
	if err != nil {
		return nil, errors.New("invalid reservation ID")
	}
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}

	if strings.ToUpper(req.PaymentMethod) != "QRIS" {
		return nil, errors.New("unsupported payment method and must be QRIS")
	}

	reservation, err := s.booking.GetReservation(reservationID)
	if err != nil {
		return nil, err
	}

	if reservation.Status != "LOCKED" {
		return nil, errors.New("reservation is not in a valid state for payment")
	}

	if time.Now().After(reservation.ExpiredAt) {
		return nil, errors.New("reservation has expired")
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

func (s *PaymentService) GetPayment(id string) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid payment id")
	}

	return s.repository.FindByID(paymentID)
}

func (s *PaymentService) ConfirmPayment(id string) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid payment id")
	}

	payment, err := s.repository.FindByID(paymentID)
	if err != nil {
		return nil, err
	}

	if payment.Status == model.PaymentSucceeded || payment.Status == model.PaymentConfirmed {
		return payment, nil
	}

	if payment.Status != model.PaymentPending {
		return nil, errors.New("payment is not in a pending state")
	}

	if time.Now().After(payment.ExpiredAt) {
		payment.Status = model.PaymentExpired
		_ = s.repository.Update(payment)
		return nil, errors.New("payment has expired")
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
