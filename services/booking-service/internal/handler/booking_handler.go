// Package handler provides HTTP transport controllers for reservation and booking endpoints.
//
// File: booking_handler.go
// Usability: Defines BookingHandler struct, constructor, and shared HTTP param parsing helpers.
package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BookingHandler handles incoming HTTP requests for seat reservations.
type BookingHandler struct {
	bookingService *service.BookingService
}

// NewBookingHandler initializes a new BookingHandler.
func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

func parseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, bool) {
	idParam := c.Param(paramName)
	resID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Reservation ID must be a valid UUID",
		})
		return uuid.Nil, false
	}
	return resID, true
}
