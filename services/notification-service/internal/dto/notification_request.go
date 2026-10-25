package dto

// CreateNotificationRequest mirrors the payload sent by libs/common/notify.
type CreateNotificationRequest struct {
	UserID    string `json:"userId" binding:"required,uuid"`
	Kind      string `json:"kind" binding:"required,max=40"`
	Title     string `json:"title" binding:"required,max=120"`
	Body      string `json:"body" binding:"required,max=1000"`
	Reference string `json:"reference,omitempty"`
}

// ListNotificationsQuery is bound from GET /api/v1/notifications?userId=&limit=.
type ListNotificationsQuery struct {
	UserID string `form:"userId" binding:"required,uuid"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
}
