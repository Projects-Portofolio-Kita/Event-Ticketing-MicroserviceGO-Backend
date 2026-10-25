package provider

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MockQRISProvider struct{}

func NewMockQRISProvider() *MockQRISProvider {
	return &MockQRISProvider{}
}

func (p *MockQRISProvider) CreatePayment(request QRISRequest) (*QRISResponse, error) {
	paymentID := uuid.New().String()

	return &QRISResponse{
		ProviderPaymentID: "MOCK-" + paymentID,
		QRISReference:     "QRIS-" + paymentID,
		QRString: fmt.Sprintf(
			"00020101021226670016COM.MOCK.QRIS0115MOCK-%s520458125303360540%d5802ID5910TICKID6007SURABAYA6304MOCK",
			paymentID[:8],
			int(request.Amount),
		),
		QRURL:     "https://mock-qris.local/pay/" + paymentID,
		ExpiredAt: time.Now().Add(5 * time.Minute),
	}, nil
}
