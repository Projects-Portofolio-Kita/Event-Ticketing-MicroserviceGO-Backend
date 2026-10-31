// Package handler provides HTTP transport controllers for reservation and booking endpoints.
//
// File: booking_handler_confirm.go
// Usability: Implements the ConfirmReservation endpoint handler to transition a booking to confirmed status.
package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
)

// ConfirmReservation handles POST /api/v1/reservations/:id/confirm
func (h *BookingHandler) ConfirmReservation(c *gin.Context) {
	resID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	res, err := h.bookingService.ConfirmReservation(c.Request.Context(), resID)
	if err != nil {
		if errors.Is(err, service.ErrReservationNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "not_found", Message: "Reservation not found"})
			return
		}
		if errors.Is(err, service.ErrReservationExpired) {
			c.JSON(http.StatusGone, dto.ErrorResponse{Error: "reservation_expired", Message: "Reservation expired before confirmation"})
			return
		}
		if errors.Is(err, service.ErrInvalidStatusChange) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Error: "invalid_status", Message: "Reservation is not in LOCKED status"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: "internal_error", Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
