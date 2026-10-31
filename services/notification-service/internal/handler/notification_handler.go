// Package handler manages REST controllers, JSON unmarshaling, and HTTP responses.
//
// File: notification_handler.go
// Usability: Defines NotificationHandler struct, constructor, and unified HTTP error mapper.
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/pkg/queue"
	"github.com/gin-gonic/gin"
)

// NotificationHandler handles HTTP requests for transactional notifications.
type NotificationHandler struct {
	service service.NotificationService
}

// NewNotificationHandler creates a new NotificationHandler with the given NotificationService.
func NewNotificationHandler(service service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

func (h *NotificationHandler) handleError(c *gin.Context, err error, fallback string) {
	if isValidationError(err) {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{Success: false, Message: "Validation failed", Error: err.Error()})
		return
	}
	if errors.Is(err, queue.ErrQueueFull) {
		c.JSON(http.StatusServiceUnavailable, dto.BaseResponse[any]{Success: false, Message: "Notification queue is full, please try again later", Error: err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, dto.BaseResponse[any]{Success: false, Message: fallback, Error: err.Error()})
}

func isValidationError(err error) bool {
	if err == nil { return false }
	if errors.Is(err, service.ErrInvalidEmail) || errors.Is(err, service.ErrEmptyBookingRef) ||
		errors.Is(err, service.ErrEmptyCustomerName) || errors.Is(err, service.ErrEmptyEventTitle) ||
		errors.Is(err, service.ErrEmptySeats) || errors.Is(err, service.ErrEmptyRecipients) ||
		errors.Is(err, service.ErrEmptySubject) || errors.Is(err, service.ErrEmptyBody) { return true }
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") || strings.Contains(msg, "invalid") || strings.Contains(msg, "empty") || strings.Contains(msg, "validation")
}