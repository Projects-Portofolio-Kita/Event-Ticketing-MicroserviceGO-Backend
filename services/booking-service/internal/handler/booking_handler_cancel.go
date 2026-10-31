// Package handler provides HTTP transport controllers for reservation and booking endpoints.
//
// File: booking_handler_cancel.go
// Usability: Implements the CancelReservation endpoint handler to release seats and cancel a locked booking.
package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
)

// CancelReservation handles POST /api/v1/reservations/:id/cancel
func (h *BookingHandler) CancelReservation(c *gin.Context) {
	resID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.bookingService.CancelReservation(c.Request.Context(), resID); err != nil {
		if errors.Is(err, service.ErrReservationNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "not_found", Message: "Reservation not found"})
			return
		}
		if errors.Is(err, service.ErrInvalidStatusChange) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "invalid_status", Message: "Only LOCKED reservations can be cancelled"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal_error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
