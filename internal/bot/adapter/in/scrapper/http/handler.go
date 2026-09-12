package httpadapter

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot/service"
)

const maxUpdateBodyBytes = 1 * 1024 * 1024 // 1 MB

type Handler struct {
	notifier service.LinkUpdateNotifier
	logger   *slog.Logger
}

func NewHandler(notifier service.LinkUpdateNotifier, logger *slog.Logger) *Handler {
	return &Handler{
		notifier: notifier,
		logger:   logger,
	}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to encode json response", slog.Any("error", err))
	}
}

func (h *Handler) writeBadRequest(w http.ResponseWriter, description string) {
	h.writeJSON(w, http.StatusBadRequest, apiErrorResponse{
		Description: description,
	})
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if contentType := r.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		h.writeBadRequest(w, "Content-Type must be application/json")
		return
	}

	update, ok := h.decodeUpdate(w, r)
	if !ok {
		return
	}

	h.logger.Info("received link updates", slog.Int64("id", update.ID), slog.String("url", update.URL), slog.Int("chat_count", len(update.TgChatIDs)))

	if err := h.notifier.Notify(update); err != nil {
		h.handleNotifyError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) decodeUpdate(w http.ResponseWriter, r *http.Request) (model.LinkUpdate, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpdateBodyBytes)

	var req updateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		h.writeBadRequest(w, "invalid request body")
		return model.LinkUpdate{}, false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		h.writeBadRequest(w, "request body must contain a single JSON object")
		return model.LinkUpdate{}, false
	}

	return req.toModel(), true
}

func (h *Handler) handleNotifyError(w http.ResponseWriter, err error) {
	h.logger.Error("failed to notify chats from http updates", slog.Any("error", err))

	if errors.Is(err, model.ErrInvalidUpdate) || errors.Is(err, model.ErrDeliveryFailed) {
		h.writeBadRequest(w, err.Error())
		return
	}

	h.writeBadRequest(w, "failed to process updates")
}
