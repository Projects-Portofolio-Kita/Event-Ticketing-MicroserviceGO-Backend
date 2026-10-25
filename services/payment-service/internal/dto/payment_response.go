package dto

type PaymentResponse struct {
	ID            string  `json:"id"`
	ReservationID string  `json:"reservation_id,omitempty"`
	QRString      string  `json:"qr_string"`
	QRURL         string  `json:"qr_url" `
	Status        string  `json:"status"`
	ExpiredAt     string  `json:"expired_at"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
}
