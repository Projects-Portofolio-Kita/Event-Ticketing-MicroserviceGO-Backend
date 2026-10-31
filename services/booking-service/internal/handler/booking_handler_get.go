// Package handler provides HTTP transport controllers for reservation and booking endpoints.
//
// File: booking_handler_get.go
// Usability: Implements the GetReservation endpoint handler to retrieve a reservation by UUID.
package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
)

// GetReservation handles GET /api/v1/reservations/:id
func (h *BookingHandler) GetReservation(c *gin.Context) {
	resID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	res, err := h.bookingService.GetReservation(c.Request.Context(), resID)
	if err != nil {
		if errors.Is(err, service.ErrReservationNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: "Reservation not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}
