// Package service provides business logic and integration clients for payment operations.
//
// File: payment_service_create_helper.go
// Usability: Provides validation and mapping helpers for creating payment transactions.
package service

import (
	"errors"
	"time"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/provider"
	"github.com/google/uuid"
)

func validateReservation(reservation *Reservation) error {
	if reservation.Status != "LOCKED" {
		return errors.New("reservation is not in a valid state for payment")
	}
	if time.Now().After(reservation.ExpiredAt) {
		return errors.New("reservation has expired")
	}
	return nil
}

func buildNewPayment(res *Reservation, qrisResp *provider.QRISResponse, orderID uuid.UUID) *model.Payment {
	return &model.Payment{
		ID:                orderID,
		ReservationID:     res.ID,
		Amount:            res.TotalAmount,
		Currency:          res.Currency,
		Status:            model.PaymentPending,
		ExpiredAt:         qrisResp.ExpiredAt,
		Gateway:           model.PaymentGatewayMidtrans,
		UserID:            res.UserID,
		ProviderPaymentID: qrisResp.ProviderPaymentID,
		QRISReference:     qrisResp.QRISReference,
		QRString:          qrisResp.QRString,
		QRURL:             qrisResp.QRURL,
	}
}
