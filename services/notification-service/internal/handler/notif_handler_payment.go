// Package handler manages REST controllers, JSON unmarshaling, and HTTP responses.
//
// File: notif_handler_payment.go
// Usability: Controller action for queuing expired booking or failed payment email notices.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/gin-gonic/gin"
)

// SendPaymentFailed handles POST /api/v1/notifications/payment-failed
func (h *NotificationHandler) SendPaymentFailed(c *gin.Context) {
	var req dto.PaymentFailedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
			Success: false, Message: "Validation failed", Error: err.Error(),
		})
		return
	}
	jobID, err := h.service.SendPaymentFailed(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err, "Failed to queue payment failed notification")
		return
	}
	c.JSON(http.StatusAccepted, dto.BaseResponse[dto.QueuedResponse]{
		Success: true,
		Message: "Payment failed notification queued for delivery",
		Data:    dto.QueuedResponse{JobID: jobID},
	})
}