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
	return &NotificationHandler{
		service: service,
	}
}

// SendTicketConfirmation handles POST /api/v1/notifications/ticket-confirmation
// Validates payload, enqueues the delivery job, and immediately returns 202 Accepted.
func (h *NotificationHandler) SendTicketConfirmation(c *gin.Context) {
	var req dto.TicketConfirmationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
			Success: false,
			Message: "Validation failed",
			Error:   err.Error(),
		})
		return
	}

	jobID, err := h.service.SendTicketConfirmation(c.Request.Context(), req)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
				Success: false,
				Message: "Validation failed",
				Error:   err.Error(),
			})
			return
		}
		if errors.Is(err, queue.ErrQueueFull) {
			c.JSON(http.StatusServiceUnavailable, dto.BaseResponse[any]{
				Success: false,
				Message: "Notification queue is full, please try again later",
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.BaseResponse[any]{
			Success: false,
			Message: "Failed to queue ticket confirmation notification",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, dto.BaseResponse[dto.QueuedResponse]{
		Success: true,
		Message: "Ticket confirmation notification queued for delivery",
		Data: dto.QueuedResponse{
			JobID: jobID,
		},
	})
}

// SendPaymentFailed handles POST /api/v1/notifications/payment-failed
// Validates payload, enqueues the failure notification job, and immediately returns 202 Accepted.
func (h *NotificationHandler) SendPaymentFailed(c *gin.Context) {
	var req dto.PaymentFailedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
			Success: false,
			Message: "Validation failed",
			Error:   err.Error(),
		})
		return
	}

	jobID, err := h.service.SendPaymentFailed(c.Request.Context(), req)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
				Success: false,
				Message: "Validation failed",
				Error:   err.Error(),
			})
			return
		}
		if errors.Is(err, queue.ErrQueueFull) {
			c.JSON(http.StatusServiceUnavailable, dto.BaseResponse[any]{
				Success: false,
				Message: "Notification queue is full, please try again later",
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.BaseResponse[any]{
			Success: false,
			Message: "Failed to queue payment failed notification",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, dto.BaseResponse[dto.QueuedResponse]{
		Success: true,
		Message: "Payment failed notification queued for delivery",
		Data: dto.QueuedResponse{
			JobID: jobID,
		},
	})
}

// SendCustomEmail handles POST /api/v1/notifications/send-custom
// Validates payload, enqueues the custom email job, and immediately returns 202 Accepted.
func (h *NotificationHandler) SendCustomEmail(c *gin.Context) {
	var req dto.CustomEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
			Success: false,
			Message: "Validation failed",
			Error:   err.Error(),
		})
		return
	}

	jobID, err := h.service.SendCustomEmail(c.Request.Context(), req)
	if err != nil {
		if isValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.BaseResponse[any]{
				Success: false,
				Message: "Validation failed",
				Error:   err.Error(),
			})
			return
		}
		if errors.Is(err, queue.ErrQueueFull) {
			c.JSON(http.StatusServiceUnavailable, dto.BaseResponse[any]{
				Success: false,
				Message: "Notification queue is full, please try again later",
				Error:   err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.BaseResponse[any]{
			Success: false,
			Message: "Failed to queue custom email notification",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, dto.BaseResponse[dto.QueuedResponse]{
		Success: true,
		Message: "Custom email notification queued for delivery",
		Data: dto.QueuedResponse{
			JobID: jobID,
		},
	})
}

// isValidationError determines if the error represents a client validation failure.
func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrInvalidEmail) ||
		errors.Is(err, service.ErrEmptyBookingRef) ||
		errors.Is(err, service.ErrEmptyCustomerName) ||
		errors.Is(err, service.ErrEmptyEventTitle) ||
		errors.Is(err, service.ErrEmptySeats) ||
		errors.Is(err, service.ErrEmptyRecipients) ||
		errors.Is(err, service.ErrEmptySubject) ||
		errors.Is(err, service.ErrEmptyBody) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "empty") ||
		strings.Contains(msg, "validation")
}
