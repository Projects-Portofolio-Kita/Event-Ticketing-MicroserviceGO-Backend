// Package handler manages REST controllers, JSON unmarshaling, and HTTP responses.
//
// File: notif_handler_ticket.go
// Usability: Controller action for accepting and queuing ticket confirmation dispatch requests.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/gin-gonic/gin"
)

// SendTicketConfirmation handles POST /api/v1/notifications/ticket-confirmation
func (h *NotificationHandler) SendTicketConfirmation(c *gin.Context) {
	var req dto.TicketConfirmationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
			Success: false, Message: "Validation failed", Error: err.Error(),
		})
		return
	}
	jobID, err := h.service.SendTicketConfirmation(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err, "Failed to queue ticket confirmation notification")
		return
	}
	c.JSON(http.StatusAccepted, dto.BaseResponse[dto.QueuedResponse]{
		Success: true,
		Message: "Ticket confirmation notification queued for delivery",
		Data:    dto.QueuedResponse{JobID: jobID},
	})
}