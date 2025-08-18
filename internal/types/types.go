package types

import "github.com/DoyleJ11/lol-draft-backend/internal/engine"

type ClientMessage struct {
	Type       string `json:"type"`
	Team       string `json:"team,omitempty"`
	SeatID     string `json:"seat_id,omitempty"`
	ChampionID int    `json:"champion_id,omitempty"`
}

type ServerMessage struct {
	Type    string        `json:"type"` // "StateSnapshot" | "Error"
	Version int           `json:"version,omitempty"`
	State   *engine.State `json:"state,omitempty"`
	Error   string        `json:"error,omitempty"`
}
