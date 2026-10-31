// Package handler provides HTTP transport controllers for reservation and booking endpoints.
//
// File: booking_handler_create.go
// Usability: Implements the CreateReservation endpoint handler for seat locking and booking creation.
package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
)

// CreateReservation handles POST /api/v1/reservations
func (h *BookingHandler) CreateReservation(c *gin.Context) {
	var req dto.CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}
	res, err := h.bookingService.CreateReservation(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrSeatsUnavailable) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "seats_unavailable",
				Message: "One or more selected seats are currently held or sold.",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "reservation_failed",
			Message: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, res)
}
