package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/domain"
	"github.com/fabiankaraben/go-core-banking-platform/account-service/internal/ports"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Handler holds HTTP handler dependencies.
type Handler struct {
	service ports.AccountService
	logger  *zap.Logger
}

// NewHandler creates a new Handler.
func NewHandler(service ports.AccountService, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// CreateAccountRequest is the request body for POST /accounts.
type CreateAccountRequest struct {
	CustomerID     string `json:"customer_id"`
	Currency       string `json:"currency"`
	InitialBalance string `json:"initial_balance"`
}

// CreateAccount handles POST /accounts.
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid customer_id")
		return
	}
	initialBalance, err := decimal.NewFromString(req.InitialBalance)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid initial_balance")
		return
	}

	account, err := h.service.CreateAccount(r.Context(), customerID, req.Currency, initialBalance)
	if err != nil {
		if errors.Is(err, domain.ErrNegativeBalance) {
			h.writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		h.logger.Error("create account", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.writeJSON(w, http.StatusCreated, account)
}

// GetAccount handles GET /accounts/{accountID}.
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "accountID")
	id, err := uuid.Parse(rawID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}
	account, err := h.service.GetAccount(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrAccountNotFound) {
			h.writeError(w, http.StatusNotFound, "account not found")
			return
		}
		h.logger.Error("get account", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	h.writeJSON(w, http.StatusOK, account)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("encoding response", zap.Error(err))
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
