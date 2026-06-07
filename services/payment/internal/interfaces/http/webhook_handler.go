package http

import (
	"encoding/json"
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
}

// WebhookHandler handles Midtrans payment notification callbacks.
type WebhookHandler struct {
	handleWebhookH *command.HandleWebhookHandler
}

// NewWebhookHandler wires the HTTP handler.
func NewWebhookHandler(h *command.HandleWebhookHandler) *WebhookHandler {
	return &WebhookHandler{handleWebhookH: h}
}

// ServeHTTP implements http.Handler for POST /webhook/payment.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var notif midtransNotification
	if err := json.NewDecoder(r.Body).Decode(&notif); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
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
