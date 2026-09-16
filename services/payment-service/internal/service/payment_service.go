package service

import (
	"errors"
	"time"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Reservation struct {
	ID		uuid.UUID
	UserID		uuid.UUID
	TotalAmount	float64
	Currency	string
	ExpiredAt	time.Time
	Status		string
}

type BookingClient interface {
	GetReservation(reservationID uuid.UUID) (*Reservation, error)
}

type MockBookingClient struct{}

func NewMockBookingClient() *MockBookingClient {
	return &MockBookingClient{}
}

func (c *MockBookingClient) GetReservation(reservationID uuid.UUID) (*Reservation, error) {
	if reservationID == uuid.Nil {
		return nil, errors.New("invalid reservation ID")
	}
	return &Reservation{
		ID:          reservationID,
		UserID:      uuid.New(),
		TotalAmount: 100.00,
		Currency:    "USD",
		ExpiredAt:   time.Now().Add(24 * time.Hour),
		Status:      "LOCKED",
	}, nil
}

type PaymentService struct {
	repository *repository.PaymentRepository
	qrisProvider *provider.MockQRISProvider
	booking BookingClient
}

func NewPaymentService(repository *repository.PaymentRepository, qrisProvider *provider.MockQRISProvider, booking BookingClient) *PaymentService {
	return &PaymentService{
		repository: repository,
		qrisProvider: qrisProvider,
		booking: booking,
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

	if req.PaymentMethod != "QRIS" {
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
		ID: 		  uuid.New(),
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