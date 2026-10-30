package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	midtransSandboxBaseURL = "https://api.sandbox.midtrans.com"
	midtransProdBaseURL    = "https://api.midtrans.com"
)

type MidtransQRISProvider struct {
	baseURL    string
	serverKey  string
	acquirer   string
	httpClient *http.Client
}

func NewMidtransQRISProvider(serverKey string, isProduction bool, acquirer string) *MidtransQRISProvider {
	baseURL := midtransSandboxBaseURL
	if isProduction {
		baseURL = midtransProdBaseURL
	}

	acq := strings.TrimSpace(strings.ToLower(acquirer))
	if acq == "" {
		acq = "gopay"
	}

	return &MidtransQRISProvider{
		baseURL:    baseURL,
		serverKey:  strings.TrimSpace(serverKey),
		acquirer:   acq,
		httpClient: &http.Client{Timeout: 12 * time.Second},
	}
}

type midtransChargeRequest struct {
	PaymentType        string `json:"payment_type"`
	TransactionDetails struct {
		OrderID     string `json:"order_id"`
		GrossAmount int64  `json:"gross_amount"`
	} `json:"transaction_details"`
	QRIS struct {
		Acquirer string `json:"acquirer"`
	} `json:"qris"`
}

type midtransAction struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type midtransChargeResponse struct {
	StatusCode      string           `json:"status_code"`
	StatusMessage   string           `json:"status_message"`
	TransactionID   string           `json:"transaction_id"`
	OrderID         string           `json:"order_id"`
	TransactionTime string           `json:"transaction_time"`
	ExpiryTime      string           `json:"expiry_time"`
	Actions         []midtransAction `json:"actions"`
}

func (p *MidtransQRISProvider) CreatePayment(request QRISRequest) (*QRISResponse, error) {
	if p.serverKey == "" {
		return nil, fmt.Errorf("midtrans server key is required")
	}
	if strings.TrimSpace(request.OrderID) == "" {
		return nil, fmt.Errorf("order id is required")
	}
	if request.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	payload := midtransChargeRequest{PaymentType: "qris"}
	payload.TransactionDetails.OrderID = request.OrderID
	payload.TransactionDetails.GrossAmount = int64(request.Amount + 0.5)
	payload.QRIS.Acquirer = p.acquirer

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, p.baseURL+"/v2/charge", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(p.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans charge request failed: %w", err)
	}
	defer resp.Body.Close()

	var result midtransChargeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid midtrans response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(result.StatusMessage)
		if msg == "" {
			msg = "failed to create midtrans qris"
		}
		return nil, fmt.Errorf("midtrans error (%d): %s", resp.StatusCode, msg)
	}

	expiredAt := time.Now().Add(15 * time.Minute)
	if result.ExpiryTime != "" {
		if parsed, err := parseMidtransTime(result.ExpiryTime); err == nil {
			expiredAt = parsed
		}
	}

	qrURL := ""
	for _, action := range result.Actions {
		name := strings.ToLower(strings.TrimSpace(action.Name))
		if strings.Contains(name, "qr") && action.URL != "" {
			qrURL = action.URL
			break
		}
	}

	providerID := strings.TrimSpace(result.TransactionID)
	if providerID == "" {
		providerID = strings.TrimSpace(result.OrderID)
	}
	if providerID == "" {
		providerID = request.OrderID
	}

	statusCode, _ := strconv.Atoi(result.StatusCode)
	qrisReference := fmt.Sprintf("MIDTRANS-%d-%s", statusCode, request.OrderID)

	return &QRISResponse{
		ProviderPaymentID: providerID,
		QRISReference:     qrisReference,
		QRString:          providerID,
		QRURL:             qrURL,
		ExpiredAt:         expiredAt,
	}, nil
}

func parseMidtransTime(raw string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported midtrans time format")
}
