package lobby

import (
	"context"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
)

type Msg interface{ isLobbyMsg() }

type FromClient struct {
	Cmd engine.Command
}

func (FromClient) isLobbyMsg() {}

type Join struct {
	ClientID string
	Outbox   chan Snapshot // where this client wants to receive snapshots
}

func (Join) isLobbyMsg() {}

type Leave struct{ ClientID string }

func (Leave) isLobbyMsg() {}

type Shutdown struct{}

func (Shutdown) isLobbyMsg() {}

type GetState struct {
	Reply chan View
}

func (GetState) isLobbyMsg() {}

// Export fields
type Snapshot struct {
	Version int
	State   engine.State
}

type View struct {
	Version    int
	NumClients int
	State      engine.State
}

type Lobby struct {
	inbox   chan Msg
	state   engine.State
	version int
	clients map[string]chan Snapshot
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewLobby(parent context.Context, initial engine.State) *Lobby {
	ctx, cancel := context.WithCancel(parent)

	l := &Lobby{
		inbox:   make(chan Msg, 64), // Small buffer
		state:   initial,
		version: 0,
		clients: make(map[string]chan Snapshot),
		ctx:     ctx,
		cancel:  cancel,
	}

	go l.loop()
	return l
}

func (l *Lobby) loop() {
	for {
		select {
		case <-l.ctx.Done():
			l.shutdown()
			return

		case m := <-l.inbox:
			switch msg := m.(type) {
			case Join:
				// Register client + send current snapshot immediately
				l.clients[msg.ClientID] = msg.Outbox
				msg.Outbox <- Snapshot{Version: l.version, State: l.state}
				// l.broadcast(Snapshot{Version: l.version, State: l.state})

			case Leave:
				delete(l.clients, msg.ClientID)

			case FromClient:
				// Call engine.Apply() on success: update state, increment version, broadcast.
				_, newState, err := engine.Apply(l.state, msg.Cmd) // Add events later
				if err != nil {
					// v1: ignore or TODO: send an error to just that client (requires sender info)
					break
				}
				// TODO: persist events later
				l.state = newState
				l.version++
				l.broadcast(Snapshot{Version: l.version, State: l.state})

			case GetState:
				// test-only: reflect internal state without data races
				msg.Reply <- View{
					Version:    l.version,
					NumClients: len(l.clients),
					State:      l.state,
				}

			case Shutdown:
				l.shutdown()
				return
			}
		}
	}
}

func (l *Lobby) shutdown() {
	for id, ch := range l.clients {
		close(ch) // Tell client no more snapshots
		delete(l.clients, id)
	}
	l.cancel()
}

func (l *Lobby) broadcast(snap Snapshot) {
	for id, ch := range l.clients {
		select {
		case ch <- snap:
			//ok
		default:
			// Client is slow/full - drop them.
			close(ch)
			delete(l.clients, id)
		}
	}
}

// Expose the inbox so tests or WS layer can send messages.
func (l *Lobby) Inbox() chan<- Msg { return l.inbox }
