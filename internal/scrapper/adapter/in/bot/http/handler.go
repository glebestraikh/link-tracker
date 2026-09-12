package httpadapter

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/model"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/scrapper/service"
)

type Handler struct {
	tracker service.Tracker
	logger  *slog.Logger
}

const maxRequestBodyBytes = 1 * 1024 * 1024 // 1 MB

func NewHandler(tracker service.Tracker, logger *slog.Logger) *Handler {
	return &Handler{
		tracker: tracker,
		logger:  logger,
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

func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		h.writeBadRequest(w, "invalid request body")
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		h.writeBadRequest(w, "request body must contain a single JSON object")
		return false
	}

	return true
}

func (h *Handler) handleRegisterChat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeBadRequest(w, "invalid chat id")
		return
	}

	h.logger.Info("register chat", slog.Int64("chat_id", id))

	if err = h.tracker.RegisterChat(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.writeBadRequest(w, "invalid chat id")
		return
	}

	h.logger.Info("delete chat", slog.Int64("chat_id", id))

	if err = h.tracker.DeleteChat(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleGetLinks(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.Header.Get("Tg-Chat-Id"), 10, 64)
	if err != nil {
		h.writeBadRequest(w, "invalid chat id")
		return
	}

	h.logger.Info("get links", slog.Int64("chat_id", chatID))

	links, err := h.tracker.GetLinks(r.Context(), chatID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	resp := listLinksResponse{
		Links: make([]linkResponse, 0, len(links)),
		Size:  int32(len(links)),
	}
	for _, l := range links {
		resp.Links = append(resp.Links, linkResponse{
			ID:   l.ID,
			URL:  l.URL,
			Tags: l.Tags,
		})
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleAddLink(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.Header.Get("Tg-Chat-Id"), 10, 64)
	if err != nil {
		h.writeBadRequest(w, "invalid chat id")
		return
	}

	var req addLinkRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	h.logger.Info("add link", slog.Int64("chat_id", chatID), slog.String("link", req.Link))

	l, err := h.tracker.AddLink(r.Context(), chatID, req.Link, req.Tags)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, linkResponse{
		ID:   l.ID,
		URL:  l.URL,
		Tags: l.Tags,
	})
}

func (h *Handler) handleDeleteLink(w http.ResponseWriter, r *http.Request) {
	chatID, err := strconv.ParseInt(r.Header.Get("Tg-Chat-Id"), 10, 64)
	if err != nil {
		h.writeBadRequest(w, "invalid chat id")
		return
	}

	var req removeLinkRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	h.logger.Info("delete link", slog.Int64("chat_id", chatID), slog.String("link", req.Link))

	l, err := h.tracker.RemoveLink(r.Context(), chatID, req.Link)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, linkResponse{
		ID:   l.ID,
		URL:  l.URL,
		Tags: l.Tags,
	})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrChatAlreadyExists):
		h.writeJSON(w, http.StatusConflict, apiErrorResponse{Description: err.Error()})
	case errors.Is(err, model.ErrLinkAlreadyTracked):
		h.writeJSON(w, http.StatusConflict, apiErrorResponse{Description: err.Error()})
	case errors.Is(err, model.ErrChatNotFound):
		h.writeJSON(w, http.StatusNotFound, apiErrorResponse{Description: err.Error()})
	case errors.Is(err, model.ErrLinkNotFound):
		h.writeJSON(w, http.StatusNotFound, apiErrorResponse{Description: err.Error()})
	default:
		h.writeJSON(w, http.StatusBadRequest, apiErrorResponse{Description: err.Error()})
	}
}
