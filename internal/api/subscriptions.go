package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/storage"
)

type SubscriptionStorage interface {
	Create(ctx context.Context, sub *models.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	List(ctx context.Context, filter storage.ListFilter) ([]models.Subscription, error)
	Update(ctx context.Context, sub *models.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	TotalCost(ctx context.Context, filter storage.SummaryFilter) (int, error)
}

type summaryResponse struct {
	From        string     `json:"from"`
	To          string     `json:"to"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	ServiceName *string    `json:"service_name,omitempty"`
	TotalCost   int        `json:"total_cost"`
}

type SubscriptionHandler struct {
	log     *slog.Logger
	storage SubscriptionStorage
}

func NewSubscriptionHandler(log *slog.Logger, storage SubscriptionStorage) *SubscriptionHandler {
	return &SubscriptionHandler{log: log, storage: storage}
}

func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	req := bodyFrom[models.SubscriptionRequest](r.Context())

	sub := req.ToSubscription()

	if err := h.storage.Create(r.Context(), sub); err != nil {
		h.serverError(w, r, "create subscription", err)
		return
	}

	h.log.Info("subscription created", "id", sub.ID, "user_id", sub.UserID, "service_name", sub.ServiceName)
	writeJSON(w, http.StatusCreated, sub)
}

func (h *SubscriptionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	sub, err := h.storage.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		h.serverError(w, r, "get subscription", err)
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := storage.ListFilter{Limit: 50}
	query := r.URL.Query()

	if value := query.Get("user_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "user_id must be a valid UUID")
			return
		}
		filter.UserID = &id
	}

	if value := query.Get("service_name"); value != "" {
		filter.ServiceName = &value
	}

	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, http.StatusBadRequest, "limit must be an integer between 1 and 100")
			return
		}
		filter.Limit = limit
	}

	if value := query.Get("offset"); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil || offset < 0 {
			writeError(w, http.StatusBadRequest, "offset must be a non-negative integer")
			return
		}
		filter.Offset = offset
	}

	subs, err := h.storage.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, "list subscriptions", err)
		return
	}

	writeJSON(w, http.StatusOK, subs)
}

func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	req := bodyFrom[models.SubscriptionRequest](r.Context())

	sub := req.ToSubscription()
	sub.ID = id

	if err := h.storage.Update(r.Context(), sub); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		h.serverError(w, r, "update subscription", err)
		return
	}

	h.log.Info("subscription updated", "id", sub.ID)
	writeJSON(w, http.StatusOK, sub)
}

func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.storage.Delete(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		h.serverError(w, r, "delete subscription", err)
		return
	}

	h.log.Info("subscription deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SubscriptionHandler) Summary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	from, err := models.ParseMonthYear(query.Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "from is required, format MM-YYYY")
		return
	}

	to, err := models.ParseMonthYear(query.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "to is required, format MM-YYYY")
		return
	}

	if to.Before(from.Time) {
		writeError(w, http.StatusBadRequest, "to must not be before from")
		return
	}

	filter := storage.SummaryFilter{From: from.Time, To: to.Time}

	if value := query.Get("user_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "user_id must be a valid UUID")
			return
		}
		filter.UserID = &id
	}

	if value := query.Get("service_name"); value != "" {
		filter.ServiceName = &value
	}

	total, err := h.storage.TotalCost(r.Context(), filter)
	if err != nil {
		h.serverError(w, r, "sum subscriptions cost", err)
		return
	}

	h.log.Info("subscriptions cost calculated",
		"from", query.Get("from"), "to", query.Get("to"), "total_cost", total)

	writeJSON(w, http.StatusOK, summaryResponse{
		From:        query.Get("from"),
		To:          query.Get("to"),
		UserID:      filter.UserID,
		ServiceName: filter.ServiceName,
		TotalCost:   total,
	})
}

func (h *SubscriptionHandler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid UUID")
		return uuid.Nil, false
	}

	return id, true
}

func (h *SubscriptionHandler) serverError(w http.ResponseWriter, r *http.Request, action string, err error) {
	h.log.Error(action, "error", err, "path", r.URL.Path)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
