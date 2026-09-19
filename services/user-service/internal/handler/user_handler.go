package handler

import (
	"net/http"

	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/dto"
	"github.com/Projects-Portofolio-Kita/Event-Ticketing-MicroserviceGO-Backend/services/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Unauthorized", "ERR_UNAUTHORIZED"))
		return
	}

	userID := userIDVal.(uuid.UUID)
	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse(err.Error(), "ERR_USER_NOT_FOUND"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse("Profile retrieved successfully", profile))
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Unauthorized", "ERR_UNAUTHORIZED"))
		return
	}

	userID := userIDVal.(uuid.UUID)

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_BAD_REQUEST"))
		return
	}

	updated, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse(err.Error(), "ERR_UPDATE_FAILED"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse("Profile updated successfully", updated))
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	targetID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("Invalid user ID format", "ERR_INVALID_UUID"))
		return
	}

	user, err := h.userService.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse(err.Error(), "ERR_USER_NOT_FOUND"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse("User found", user))
}

func (h *UserHandler) DeactivateAccount(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse("Unauthorized", "ERR_UNAUTHORIZED"))
		return
	}

	userID := userIDVal.(uuid.UUID)
	if err := h.userService.DeactivateAccount(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse(err.Error(), "ERR_INTERNAL"))
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse("Account deactivated successfully", nil))
}
