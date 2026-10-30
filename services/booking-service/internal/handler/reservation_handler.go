package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/booking-service/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReservationHandler struct {
	service *service.ReservationService
}

func NewReservationHandler(service *service.ReservationService) *ReservationHandler {
	return &ReservationHandler{service: service}
}

// caller reads the principal that authz put on the context. Every handler
// below is mounted behind RequireUser or RequireUserOrService, so it is
// always populated.
func caller(c *gin.Context) authz.Caller {
	return authz.CallerFrom(c)
}

// writeError maps service errors onto HTTP statuses using the shared envelope.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httperr.JSON(c, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrSeatsUnavailable), errors.Is(err, service.ErrTicketsUnavailable):
		httperr.JSON(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrEventNotFound):
		httperr.NotFound(c, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		httperr.JSON(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		httperr.NotFound(c, "reservasi tidak ditemukan")
	default:
		httperr.Internal(c, "Terjadi kesalahan pada server.")
	}
}

func (h *ReservationHandler) CreateReservation(c *gin.Context) {
	var req dto.CreateReservationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	reservation, err := h.service.CreateReservation(&req, caller(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(reservation))
}

func (h *ReservationHandler) GetReservation(c *gin.Context) {
	reservation, err := h.service.GetReservation(c.Param("id"), caller(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(reservation))
}

// ListReservations answers GET /reservations with the caller's own bookings
// ("Tiket Saya"), seats and tier items alike.
func (h *ReservationHandler) ListReservations(c *gin.Context) {
	var query struct {
		Limit int `form:"limit"`
	}
	_ = c.ShouldBindQuery(&query)

	reservations, err := h.service.ListReservations(caller(c), query.Limit)
	if err != nil {
		writeError(c, err)
		return
	}

	responses := make([]dto.ReservationResponse, 0, len(reservations))
	for i := range reservations {
		responses = append(responses, toResponse(&reservations[i]))
	}

	c.JSON(http.StatusOK, gin.H{"reservations": responses})
}

func (h *ReservationHandler) GetEventSeatStatus(c *gin.Context) {
	statuses, err := h.service.GetSeatStatus(c.Param("eventId"))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"seats": statuses})
}

// GetEventTicketStatus answers GET /events/:eventId/ticket-status: the tier
// availability feed catalog-service reads behind X-Service-Token.
func (h *ReservationHandler) GetEventTicketStatus(c *gin.Context) {
	statuses, err := h.service.GetEventTicketStatus(c.Param("eventId"))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": statuses})
}

func (h *ReservationHandler) ConfirmReservation(c *gin.Context) {
	reservation, err := h.service.ConfirmReservation(c.Param("id"), caller(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(reservation))
}

func (h *ReservationHandler) CancelReservation(c *gin.Context) {
	reservation, err := h.service.CancelReservation(c.Param("id"), caller(c))
	if err != nil {
		writeError(c, err)
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

	items := make([]dto.TicketItemResponse, 0, len(reservation.Items))
	for _, item := range reservation.Items {
		items = append(items, dto.TicketItemResponse{
			CategoryID:   item.CategoryID.String(),
			CategoryName: item.CategoryName,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			Subtotal:     item.Subtotal,
		})
	}

	mode := reservation.Mode
	if mode == "" {
		mode = model.ReservationModeSeated
	}

	return dto.ReservationResponse{
		ReservationID: reservation.ID.String(),
		UserID:        reservation.UserID.String(),
		EventID:       reservation.EventID.String(),
		BookingRef:    reservation.BookingRef,
		Status:        string(reservation.Status),
		Mode:          string(mode),
		ExpiresAt:     reservation.ExpiresAt,
		TotalAmount:   reservation.TotalAmount,
		Currency:      reservation.Currency,
		Seats:         seats,
		Items:         items,
		CreatedAt:     reservation.CreatedAt,
	}
}
