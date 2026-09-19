package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BookingHandler struct {
	bookingService *service.BookingService
}

func NewBookingHandler(bookingService *service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

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

// GetReservation handles GET /api/v1/reservations/:id
func (h *BookingHandler) GetReservation(c *gin.Context) {
	idParam := c.Param("id")
	resID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Reservation ID must be a valid UUID",
		})
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

// ConfirmReservation handles POST /api/v1/reservations/:id/confirm
func (h *BookingHandler) ConfirmReservation(c *gin.Context) {
	idParam := c.Param("id")
	resID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Reservation ID must be a valid UUID",
		})
		return
	}

	res, err := h.bookingService.ConfirmReservation(c.Request.Context(), resID)
	if err != nil {
		if errors.Is(err, service.ErrReservationNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: "Reservation not found",
			})
			return
		}
		if errors.Is(err, service.ErrReservationExpired) {
			c.JSON(http.StatusGone, dto.ErrorResponse{
				Error:   "reservation_expired",
				Message: "Reservation expired before confirmation",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidStatusChange) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "invalid_status",
				Message: "Reservation is not in LOCKED status",
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

// CancelReservation handles POST /api/v1/reservations/:id/cancel
func (h *BookingHandler) CancelReservation(c *gin.Context) {
	idParam := c.Param("id")
	resID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Reservation ID must be a valid UUID",
		})
		return
	}

	if err := h.bookingService.CancelReservation(c.Request.Context(), resID); err != nil {
		if errors.Is(err, service.ErrReservationNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: "Reservation not found",
			})
			return
		}
		if errors.Is(err, service.ErrInvalidStatusChange) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Error:   "invalid_status",
				Message: "Only LOCKED reservations can be cancelled",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
