package handler

import (
	"errors"
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/libs/common/authz"
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

// writeError maps service errors onto HTTP statuses using the shared envelope.
func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden):
		httperr.JSON(c, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrNotFound):
		httperr.NotFound(c, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		httperr.JSON(c, http.StatusBadRequest, err.Error())
	default:
		httperr.Internal(c, "Terjadi kesalahan pada server.")
	}
}

// Create handles POST /api/v1/notifications, the entry point for
// libs/common/notify. It is mounted behind authz.RequireService, so only
// services inside the network reach it.
func (h *NotificationHandler) Create(c *gin.Context) {
	var req dto.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	notification, err := h.service.Create(&req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(notification))
}

// List handles GET /api/v1/notifications?userId=&limit=. The caller may only
// list their own inbox: the service compares userId with the token subject.
func (h *NotificationHandler) List(c *gin.Context) {
	var query dto.ListNotificationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httperr.BadRequest(c, err.Error())
		return
	}

	notifications, err := h.service.List(&query, authz.CallerFrom(c))
	if err != nil {
		writeError(c, err)
		return
	}

	items := make([]dto.NotificationResponse, 0, len(notifications))
	for i := range notifications {
		items = append(items, toResponse(&notifications[i]))
	}
	c.JSON(http.StatusOK, gin.H{"notifications": items})
}

// MarkRead handles POST /api/v1/notifications/:id/read. Only the owner of the
// notification may mark it read.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notification, err := h.service.MarkRead(c.Param("id"), authz.CallerFrom(c))
	if err != nil {
		writeError(c, err)
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
