package ws

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
	"github.com/DoyleJ11/lol-draft-backend/internal/hub"
	"github.com/DoyleJ11/lol-draft-backend/internal/lobby"
	"github.com/DoyleJ11/lol-draft-backend/internal/types"
	"github.com/coder/websocket"
)

func Handler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		reply := make(chan *lobby.Lobby, 1)
		h.Inbox() <- hub.GetLobby{Code: code, Reply: reply}
		lb := <-reply
		if lb == nil {
			http.Error(w, "lobby not found", http.StatusNotFound)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// In dev ONLY, you can loosen origin checks:
			// OriginPatterns: []string{"http://localhost:*", "http://127.0.0.1:*"},
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "bye")

		out := make(chan lobby.Snapshot, 8)
		clientID := randID(6) // Implement simple rand id

		lb.Inbox() <- lobby.Join{ClientID: clientID, Outbox: out}
		defer func() { lb.Inbox() <- lobby.Leave{ClientID: clientID} }()

		// Writer goroutine
		writeCtx, writeCancel := context.WithCancel(r.Context())
		defer writeCancel()
		go func() {
			for snap := range out {
				msg := types.ServerMessage{Type: "StateSnapshot", Version: snap.Version, State: &snap.State}
				payload, _ := json.Marshal(msg)
				ctx, cancel := context.WithTimeout(writeCtx, 3*time.Second)
				_ = conn.Write(ctx, websocket.MessageText, payload)
				cancel()
			}
		}()

		// Reader loop
		for {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			_, data, err := conn.Read(ctx)
			cancel()
			if err != nil {
				// Treat clean close/going-away as normal:
				switch websocket.CloseStatus(err) {
				case websocket.StatusNormalClosure, websocket.StatusGoingAway:
					return
				}
				// Otherwise, just exit (lobby.Leave in defer):
				return
			}

			var cm types.ClientMessage
			if err := json.Unmarshal(data, &cm); err != nil {
				_ = conn.Write(r.Context(), websocket.MessageText,
					[]byte(`{"type":"Error","error":"bad json"}`))
				continue
			}

			cmd, ok := toEngineCommand(cm)
			if !ok {
				_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"type":"Error","error":"unknown type"}`))
				continue
			}

			lb.Inbox() <- lobby.FromClient{Cmd: cmd}
		}
	}
}

func toEngineCommand(m types.ClientMessage) (engine.Command, bool) {
	team, ok := parseTeam(m.Team)
	if !ok {
		return engine.Command{}, false
	}

	switch m.Type {
	case "LockPick":
		return engine.Command{Type: engine.CmdLockPick, Team: team, SeatID: m.SeatID, ChampionID: m.ChampionID}, true
	case "BanChampion":
		return engine.Command{Type: engine.CmdBanChampion, Team: team, ChampionID: m.ChampionID}, true
	case "HoverChampion":
		return engine.Command{Type: engine.CmdHoverChampion, Team: team, SeatID: m.SeatID, ChampionID: m.ChampionID}, true
	default:
		return engine.Command{}, false
	}
}

func parseTeam(team string) (engine.Team, bool) {
	switch team {
	case "blue":
		return engine.TeamBlue, true
	case "red":
		return engine.TeamRed, true
	default:
		return "", false
	}
}

func randID(length int) string {
	// Not sure how complex the clientID should be. Could make it a uuid but that may be too complicated for our purposes.
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
