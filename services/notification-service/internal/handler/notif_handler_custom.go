// Package handler manages REST controllers, JSON unmarshaling, and HTTP responses.
//
// File: notif_handler_custom.go
// Usability: Controller action for submitting raw custom emails to the background delivery queue.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/gin-gonic/gin"
)

// SendCustomEmail handles POST /api/v1/notifications/send-custom
func (h *NotificationHandler) SendCustomEmail(c *gin.Context) {
	var req dto.CustomEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
			Success: false, Message: "Validation failed", Error: err.Error(),
		})
		return
	}
	jobID, err := h.service.SendCustomEmail(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err, "Failed to queue custom email notification")
		return
	}
	c.JSON(http.StatusAccepted, dto.BaseResponse[dto.QueuedResponse]{
		Success: true,
		Message: "Custom email notification queued for delivery",
		Data:    dto.QueuedResponse{JobID: jobID},
	})
}