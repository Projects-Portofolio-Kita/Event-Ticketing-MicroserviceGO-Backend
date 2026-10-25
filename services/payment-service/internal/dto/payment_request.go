package dto

type CreatePaymentRequest struct {
	ReservationID string `json:"reservation_id" binding:"required"`
	PaymentMethod string `json:"payment_method" binding:"required"`
}
