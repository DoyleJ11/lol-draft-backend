package httpapi

import (
	"net/http"

	"github.com/DoyleJ11/lol-draft-backend/internal/hub"
	"github.com/DoyleJ11/lol-draft-backend/internal/ws"
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(h *hub.Hub) http.Handler {
	r := chi.NewRouter()

	// Public routes
	r.Post("/lobbies", CreateLobby(h))
	r.Get("/healthz", Healthz)
	r.Get("/ws", ws.Handler(h))
	return r
}
