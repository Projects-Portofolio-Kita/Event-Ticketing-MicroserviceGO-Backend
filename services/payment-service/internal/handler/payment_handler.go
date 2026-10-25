package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/payment-service/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		service: service,
	}
}

// writeError maps service errors onto HTTP statuses using the shared envelope.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httperr.JSON(c, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrReservationNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		httperr.NotFound(c, "pembayaran tidak ditemukan")
	case errors.Is(err, service.ErrInvalidInput):
		httperr.JSON(c, http.StatusBadRequest, err.Error())
	default:
		httperr.Internal(c, "Terjadi kesalahan pada server.")
	}
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req dto.CreatePaymentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	payment, err := h.service.CreatePayment(&req, authz.CallerFrom(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(payment))
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	id := c.Param("id")

	payment, err := h.service.GetPayment(id, authz.CallerFrom(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(payment))
}

func (h *PaymentHandler) ConfirmPayment(c *gin.Context) {
	id := c.Param("id")

	payment, err := h.service.ConfirmPayment(id, authz.CallerFrom(c))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(payment))
}

func toResponse(payment *model.Payment) gin.H {
	return gin.H{
		"id":             payment.ID,
		"reservation_id": payment.ReservationID,
		"qr_string":      payment.QRString,
		"qr_url":         payment.QRURL,
		"status":         payment.Status,
		"amount":         payment.Amount,
		"currency":       payment.Currency,
		"expired_at":     payment.ExpiredAt,
	}
}
