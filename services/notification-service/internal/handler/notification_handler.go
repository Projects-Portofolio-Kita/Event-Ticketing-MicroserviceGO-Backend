package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/httperr"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/model"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/notification-service/internal/service"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// POST /api/v1/notifications — entry point for libs/common/notify.
func (h *NotificationHandler) Create(c *gin.Context) {
	var req dto.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	notification, err := h.service.Create(&req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			httperr.BadRequest(c, err.Error())
			return
		}
		httperr.Internal(c, "Notifikasi gagal disimpan.")
		return
	}

	c.JSON(http.StatusCreated, toResponse(notification))
}

// GET /api/v1/notifications?userId=&limit=
func (h *NotificationHandler) List(c *gin.Context) {
	var query dto.ListNotificationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	notifications, err := h.service.List(&query)
	if err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	items := make([]dto.NotificationResponse, 0, len(notifications))
	for i := range notifications {
		items = append(items, toResponse(&notifications[i]))
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items})
}

// POST /api/v1/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notification, err := h.service.MarkRead(c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			httperr.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrNotFound):
			httperr.NotFound(c, err.Error())
		default:
			httperr.Internal(c, "Notifikasi gagal diperbarui.")
		}
		return
	}

	c.JSON(http.StatusOK, toResponse(notification))
}

func toResponse(notification *model.Notification) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:        notification.ID.String(),
		UserID:    notification.UserID.String(),
		Kind:      notification.Kind,
		Title:     notification.Title,
		Body:      notification.Body,
		Reference: notification.Reference,
		Read:      notification.ReadAt != nil,
		ReadAt:    notification.ReadAt,
		CreatedAt: notification.CreatedAt,
	}
}
