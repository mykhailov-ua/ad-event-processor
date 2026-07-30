package management

import (
	"net/http"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
)

type createPaymentIntentRequest struct {
	AmountMicro int64  `json:"amount_micro"`
	Currency    string `json:"currency"`
}

func (h *Handler) createCustomerPaymentIntent(w http.ResponseWriter, r *http.Request) {
	if h.payment == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "PAYMENT_UNAVAILABLE", "payment service not configured")
		return
	}

	customerID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}

	req, ok := coldpath.DecodeRequestOrBadRequest[createPaymentIntentRequest](w, r, 16*1024)
	if !ok {
		return
	}
	if req.AmountMicro <= 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "amount_micro must be greater than zero")
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	resp, err := h.payment.CreatePaymentIntent(r.Context(), customerID.String(), req.AmountMicro, currency, idempotencyKey, nil)
	if err != nil {
		httpresponse.WriteGRPCError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"intent_id":    resp.IntentId,
		"status":       resp.Status.String(),
		"checkout_url": resp.CheckoutUrl,
		"provider_ref": resp.ProviderRef,
	})
}
