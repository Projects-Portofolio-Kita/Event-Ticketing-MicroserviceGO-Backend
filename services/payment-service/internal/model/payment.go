package model

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "PENDING"
	PaymentConfirmed PaymentStatus = "CONFIRMED"
	PaymentExpired   PaymentStatus = "EXPIRED"
	PaymentSucceeded PaymentStatus = "SUCCEEDED"
	PaymentCanceled  PaymentStatus = "CANCELED"
	PaymentFailed    PaymentStatus = "FAILED"
)

type PaymentGateway string

const (
	PaymentGatewayXendit PaymentGateway = "XENDIT_QRIS"
	PaymentGatewayMidtrans PaymentGateway = "MIDTRANS_QRIS"
)

type Payment struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ReservationID      uuid.UUID      `gorm:"type:uuid;not null" json:"reservation_id"`
	UserID             uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	ProviderPaymentID  string         `gorm:"type:varchar(255);uniqueIndex" json:"provider_payment_id"`
	QRISReference      string         `gorm:"type:varchar(255)" json:"qris_reference"`
	QRString           string         `gorm:"type:text" json:"qr_string"`
	QRURL              string         `gorm:"type:text" json:"qr_url"`
	Amount             float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	Currency            string         `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	FailureDescription string         `gorm:"type:text" json:"failure_description"`
	Gateway            PaymentGateway `gorm:"type:varchar(30);not null" json:"gateway"`
	Status             PaymentStatus  `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	ExpiredAt          time.Time      `json:"expired_at"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}