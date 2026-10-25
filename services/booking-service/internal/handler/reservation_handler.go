package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
)

type ReservationHandler struct {
	service *service.ReservationService
}

func NewReservationHandler(service *service.ReservationService) *ReservationHandler {
	return &ReservationHandler{service: service}
}

func (h *ReservationHandler) CreateReservation(c *gin.Context) {
	var req dto.CreateReservationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	reservation, err := h.service.CreateReservation(&req)
	if err != nil {
		if errors.Is(err, service.ErrSeatsUnavailable) {
			httperr.JSON(c, http.StatusConflict, err.Error())
			return
		}
		httperr.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, toResponse(reservation))
}

func (h *ReservationHandler) GetReservation(c *gin.Context) {
	reservation, err := h.service.GetReservation(c.Param("id"))
	if err != nil {
		httperr.NotFound(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, toResponse(reservation))
}

func (h *ReservationHandler) GetEventSeatStatus(c *gin.Context) {
	statuses, err := h.service.GetSeatStatus(c.Param("eventId"))
	if err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"seats": statuses})
}

func (h *ReservationHandler) ConfirmReservation(c *gin.Context) {
	reservation, err := h.service.ConfirmReservation(c.Param("id"))
	if err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, toResponse(reservation))
}

func (h *ReservationHandler) CancelReservation(c *gin.Context) {
	reservation, err := h.service.CancelReservation(c.Param("id"))
	if err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, toResponse(reservation))
}

func toResponse(reservation *model.Reservation) dto.ReservationResponse {
	seats := make([]dto.SeatItemResponse, 0, len(reservation.Seats))
	for _, seat := range reservation.Seats {
		seats = append(seats, dto.SeatItemResponse{
			SeatID:  seat.SeatID.String(),
			Section: seat.Section,
			Row:     seat.Row,
			Number:  seat.Number,
			Price:   seat.Price,
		})
	}

	return dto.ReservationResponse{
		ReservationID: reservation.ID.String(),
		UserID:        reservation.UserID.String(),
		EventID:       reservation.EventID.String(),
		BookingRef:    reservation.BookingRef,
		Status:        string(reservation.Status),
		ExpiresAt:     reservation.ExpiresAt,
		TotalAmount:   reservation.TotalAmount,
		Currency:      reservation.Currency,
		Seats:         seats,
		CreatedAt:     reservation.CreatedAt,
	}
}
