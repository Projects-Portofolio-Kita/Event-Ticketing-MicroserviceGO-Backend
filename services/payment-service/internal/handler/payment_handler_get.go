// Package handler provides HTTP transport controllers for payment endpoints.
//
// File: payment_handler_get.go
// Usability: Implements the GetPayment endpoint handler to retrieve payment status and QR details.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetPayment handles HTTP GET requests to fetch payment record details by ID.
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	id := c.Param("id")
	payment, err := h.service.GetPayment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
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
