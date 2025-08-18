package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
	"github.com/DoyleJ11/lol-draft-backend/internal/hub"
	"github.com/DoyleJ11/lol-draft-backend/internal/lobby"
)

func GenerateCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	code := make([]byte, 6)
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}
	return string(code), nil
}

func CreateLobby(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var code string
		for {
			c, err := GenerateCode()
			if err != nil {
				http.Error(w, "failed to generate code", http.StatusInternalServerError)
				return
			}
			reply := make(chan *lobby.Lobby, 1)
			h.Inbox() <- hub.GetLobby{Code: c, Reply: reply}
			if <-reply == nil {
				code = c
				break
			}
			fmt.Println("collision on code, regenerating")
		}

		reply := make(chan *lobby.Lobby, 1)
		h.Inbox() <- hub.EnsureLobby{Code: code, State: engine.NewEmptyState(), Reply: reply}
		if <-reply == nil {
			http.Error(w, "failed to create lobby", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(struct {
			Code string `json:"code"`
		}{Code: code})
	}
}

func Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
