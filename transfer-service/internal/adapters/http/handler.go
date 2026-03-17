package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/domain"
	"github.com/fabiankaraben/go-core-banking-platform/transfer-service/internal/ports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Handler holds HTTP handler dependencies.
type Handler struct {
	service ports.TransferService
	logger  *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(service ports.TransferService, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// CreateTransferRequest is the request body for POST /transfers.
type CreateTransferRequest struct {
	SourceAccountID string `json:"source_account_id"`
	DestAccountID   string `json:"dest_account_id"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
}

// CreateTransfer handles POST /transfers.
func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		h.writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	var req CreateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sourceID, err := uuid.Parse(req.SourceAccountID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid source_account_id")
		return
	}
	destID, err := uuid.Parse(req.DestAccountID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid dest_account_id")
		return
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid amount")
		return
	}

	transfer, err := h.service.CreateTransfer(r.Context(), idempotencyKey, sourceID, destID, amount, req.Currency)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidAmount):
			h.writeError(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, domain.ErrSameAccount):
			h.writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			h.logger.Error("create transfer", zap.Error(err))
			h.writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	h.writeJSON(w, http.StatusCreated, transfer)
}

// GetTransfer handles GET /transfers/{transferID}.
func (h *Handler) GetTransfer(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "transferID")
	id, err := uuid.Parse(rawID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid transfer id")
		return
	}
	transfer, err := h.service.GetTransfer(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			h.writeError(w, http.StatusNotFound, "transfer not found")
			return
		}
		h.logger.Error("get transfer", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.writeJSON(w, http.StatusOK, transfer)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
