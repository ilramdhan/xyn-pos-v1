package http

import (
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/application/command"
)

// midtransNotification is the Midtrans webhook POST body.
type midtransNotification struct {
	OrderID           string `json:"order_id"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
}

// WebhookHandler handles Midtrans payment notification callbacks.
type WebhookHandler struct {
	handleWebhookH *command.HandleWebhookHandler
	serverKey      string
}

// NewWebhookHandler wires the HTTP handler.
func NewWebhookHandler(h *command.HandleWebhookHandler, serverKey string) *WebhookHandler {
	return &WebhookHandler{handleWebhookH: h, serverKey: serverKey}
}

// ServeHTTP implements http.Handler for POST /webhook/payment.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var notif midtransNotification
	if err := json.Unmarshal(body, &notif); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Verify Midtrans signature: sha512(order_id + status_code + gross_amount + server_key)
	if !h.verifySignature(notif) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Midtrans sends our internal payment ID as order_id (we pass p.ID.String() in CreateTransaction).
	paymentID, err := uuid.Parse(notif.OrderID)
	if err != nil {
		http.Error(w, "invalid order_id", http.StatusBadRequest)
		return
	}

	in := command.WebhookInput{
		PaymentID:   paymentID,
		ExternalID:  notif.TransactionID,
		StatusCode:  notif.StatusCode,
		FraudStatus: notif.FraudStatus,
	}

	if err := h.handleWebhookH.Handle(r.Context(), in); err != nil {
		slog.ErrorContext(r.Context(), "webhook: HandleWebhook failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// verifySignature checks the Midtrans webhook signature using constant-time comparison.
// sha512(order_id + status_code + gross_amount + server_key) must match the notification's signature_key.
func (h *WebhookHandler) verifySignature(notif midtransNotification) bool {
	plain := notif.OrderID + notif.StatusCode + notif.GrossAmount + h.serverKey
	sum := sha512.Sum512([]byte(plain))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(notif.SignatureKey)) == 1
}
