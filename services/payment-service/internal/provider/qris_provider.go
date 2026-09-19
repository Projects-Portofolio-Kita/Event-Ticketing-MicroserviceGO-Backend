package provider

import "time"

type QRISRequest struct {
	OrderID string
	Amount  float64
}

type QRISResponse struct {
	ProviderPaymentID string
	QRISReference      string
	QRString           string
	QRURL              string
	ExpiredAt          time.Time
}

type QRISProvider interface {
	CreatePayment(request QRISRequest) (*QRISResponse, error)
}