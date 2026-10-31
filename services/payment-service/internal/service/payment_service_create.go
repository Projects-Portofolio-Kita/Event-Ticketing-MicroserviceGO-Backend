// Package service provides business logic and integration clients for payment operations.
//
// File: payment_service_create.go
// Usability: Implements CreatePayment to initiate and record QRIS payments for locked reservations.
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreatePayment initiates a new QRIS payment transaction for a locked reservation.
func (s *PaymentService) CreatePayment(req *dto.CreatePaymentRequest) (*model.Payment, error) {
	resID, err := uuid.Parse(req.ReservationID)
	if err != nil || resID == uuid.Nil { return nil, errors.New("invalid reservation ID") }
	if req.PaymentMethod != "QRIS" { return nil, errors.New("unsupported payment method and must be QRIS") }
	res, err := s.booking.GetReservation(resID)
	if err != nil { return nil, err }
	if err := validateReservation(res); err != nil { return nil, err }
	if existing, err := s.repository.FindByReservationID(resID); err == nil && existing != nil {
		return existing, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	orderID := uuid.New()
	qrisResp, err := s.qrisProvider.CreatePayment(provider.QRISRequest{OrderID: orderID.String(), Amount: res.TotalAmount})
	if err != nil { return nil, err }
	payment := buildNewPayment(res, qrisResp, orderID)
	if err := s.repository.Create(payment); err != nil { return nil, err }
	return payment, nil
}