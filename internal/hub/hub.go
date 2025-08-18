package hub

import (
	"context"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
	"github.com/DoyleJ11/lol-draft-backend/internal/lobby"
)

type HubMsg interface{ isHubMsg() }

type CreateLobby struct {
	Code  string
	State engine.State
	Reply chan *lobby.Lobby
}

type GetLobby struct {
	Code  string
	Reply chan *lobby.Lobby
}

type EnsureLobby struct {
	Code  string
	State engine.State // only used if creation happens
	Reply chan *lobby.Lobby
}

type RemoveLobby struct {
	Code string
}

type Hub struct {
	inbox   chan HubMsg
	lobbies map[string]*lobby.Lobby
	ctx     context.Context
	cancel  context.CancelFunc
}

type ShutdownHub struct{}

func (CreateLobby) isHubMsg() {}
func (GetLobby) isHubMsg()    {}
func (EnsureLobby) isHubMsg() {}
func (RemoveLobby) isHubMsg() {}
func (ShutdownHub) isHubMsg() {}

func NewHub(parent context.Context) *Hub {
	ctx, cancel := context.WithCancel(parent)
	h := &Hub{
		inbox:   make(chan HubMsg, 64),
		lobbies: make(map[string]*lobby.Lobby),
		ctx:     ctx,
		cancel:  cancel,
	}
	go h.loop()
	return h
}

func (h *Hub) Inbox() chan<- HubMsg { return h.inbox }

func (h *Hub) loop() {
	for {
		select {
		case <-h.ctx.Done():
			// h.shutdown() <- Write helper function to shutdown hub
			return

		case m := <-h.inbox:
			switch msg := m.(type) {
			case CreateLobby:
				if lb := h.lobbies[msg.Code]; lb != nil {
					msg.Reply <- lb
					break
				}
				lb := lobby.NewLobby(h.ctx, msg.State)
				h.lobbies[msg.Code] = lb
				msg.Reply <- lb

			case GetLobby:
				msg.Reply <- h.lobbies[msg.Code] // May be nil

			case EnsureLobby:
				if lb := h.lobbies[msg.Code]; lb != nil {
					msg.Reply <- lb
					break
				}

				lb := lobby.NewLobby(h.ctx, msg.State)
				h.lobbies[msg.Code] = lb
				msg.Reply <- lb

			case RemoveLobby:
				delete(h.lobbies, msg.Code)

			case ShutdownHub:
				for _, lb := range h.lobbies {
					lb.Inbox() <- lobby.Shutdown{}
				}
				clear(h.lobbies)
				h.cancel()
			}

		}
	}
}
