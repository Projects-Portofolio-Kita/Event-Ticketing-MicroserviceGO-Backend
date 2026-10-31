// Package handler provides HTTP transport controllers for payment endpoints.
//
// File: payment_handler.go
// Usability: Defines PaymentHandler struct, constructor, and CreatePayment endpoint handling.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/gin-gonic/gin"
)

// PaymentHandler handles incoming HTTP requests for payment processing.
type PaymentHandler struct {
	service *service.PaymentService
}

// NewPaymentHandler initializes a new PaymentHandler instance.
func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

// CreatePayment handles HTTP POST requests to initialize a payment transaction.
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req dto.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	payment, err := h.service.CreatePayment(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":             payment.ID,
		"reservation_id": payment.ReservationID,
		"qr_string":      payment.QRString,
		"qr_url":         payment.QRURL,
		"status":         payment.Status,
		"amount":         payment.Amount,
		"currency":       payment.Currency,
		"expired_at":     payment.ExpiredAt,
	})
}
