// Package service provides business logic and integration clients for payment operations.
//
// File: payment_service.go
// Usability: Defines the PaymentService core struct, constructor, and payment retrieval queries.
package service

import (
	"errors"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/repository"
	"github.com/google/uuid"
)

// PaymentService coordinates payment workflows, provider integrations, and persistence.
type PaymentService struct {
	repository   *repository.PaymentRepository
	qrisProvider *provider.MockQRISProvider
	booking      BookingClient
}

// NewPaymentService initializes a new PaymentService with required dependencies.
func NewPaymentService(repository *repository.PaymentRepository, qrisProvider *provider.MockQRISProvider, booking BookingClient) *PaymentService {
	return &PaymentService{
		repository:   repository,
		qrisProvider: qrisProvider,
		booking:      booking,
	}
}

// GetPayment fetches a payment record by its unique identifier.
func (s *PaymentService) GetPayment(id string) (*model.Payment, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid payment id")
	}
	return s.repository.FindByID(paymentID)
}
