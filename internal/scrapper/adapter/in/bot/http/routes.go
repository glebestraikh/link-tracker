package httpadapter

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tg-chat/{id}", h.handleRegisterChat)
	mux.HandleFunc("DELETE /tg-chat/{id}", h.handleDeleteChat)
	mux.HandleFunc("GET /links", h.handleGetLinks)
	mux.HandleFunc("POST /links", h.handleAddLink)
	mux.HandleFunc("DELETE /links", h.handleDeleteLink)
}
