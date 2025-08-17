package lobby

import (
	"context"
	"time"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
)

func hasEvent(evts []engine.Event, t engine.EventType) bool {
	for _, e := range evts {
		if e.Type == t {
			return true
		}
	}
	return false
}

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

type TimerFired struct {
	Gen int
}

func (TimerFired) isLobbyMsg() {}

type Shutdown struct{}

func (Shutdown) isLobbyMsg() {}

// Test only types:

type GetState struct {
	Reply chan View
}

func (GetState) isLobbyMsg() {}

type PrimeTimer struct{}

func (PrimeTimer) isLobbyMsg() {}

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
	inbox     chan Msg
	state     engine.State
	version   int
	clients   map[string]chan Snapshot
	turnTimer *time.Timer
	timerGen  int
	ctx       context.Context
	cancel    context.CancelFunc
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
				events, newState, err := engine.Apply(l.state, msg.Cmd) // Add events later
				if err != nil {
					// v1: ignore or TODO: send an error to just that client (requires sender info)
					break
				}
				// TODO: persist events later
				l.state = newState
				for _, e := range events {
					switch e.Type {
					case engine.EvtTurnAdvanced:
						l.state.Cursor++
					case engine.EvtGameCompleted:
						l.stopTurnTimer()
					case engine.EvtChampionPicked, engine.EvtChampionBanned:
						for seat, champ := range l.state.Hover {
							if champ == e.ChampionID {
								delete(l.state.Hover, seat)
							}
						}
					}
				}

				l.state.Phase = engine.DerivePhase(l.state.Cursor)
				l.version++
				l.broadcast(Snapshot{Version: l.version, State: l.state})

				if hasEvent(events, engine.EvtTurnAdvanced) && !hasEvent(events, engine.EvtGameCompleted) {
					l.armTurnTimer()
				}

			case TimerFired:
				if msg.Gen != l.timerGen {
					break
				}

				cmd := engine.Command{Type: engine.CmdTimeoutAdvance, SeatID: ""} // TODO: switch to real seatID when we implement seat level auth
				events, newState, err := engine.Apply(l.state, cmd)
				if err != nil {
					// v1: ignore or TODO: send an error to just that client (requires sender info)
					break
				}

				l.state = newState
				for _, e := range events {
					switch e.Type {
					case engine.EvtTurnAdvanced:
						l.state.Cursor++
					case engine.EvtGameCompleted:
						l.stopTurnTimer()
					}
				}
				l.state.Phase = engine.DerivePhase(l.state.Cursor)
				l.version++
				l.broadcast(Snapshot{Version: l.version, State: l.state})
				if hasEvent(events, engine.EvtTurnAdvanced) && !hasEvent(events, engine.EvtGameCompleted) {
					l.armTurnTimer()
				}

			case PrimeTimer:
				// Test-only: prime timer
				l.armTurnTimer()

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
	l.stopTurnTimer()
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

func (l *Lobby) armTurnTimer() {
	// Decide duration based on current step
	step := engine.GameOrder[l.state.Cursor]
	var dur time.Duration
	if step.Action == engine.ActionPick {
		dur = time.Duration(l.state.Rules.PickTimerSec) * time.Second
	} else {
		dur = time.Duration(l.state.Rules.BanTimerSec) * time.Second
	}
	l.timerGen++
	gen := l.timerGen

	if l.turnTimer != nil {
		l.turnTimer.Stop()
	}

	l.turnTimer = time.AfterFunc(dur, func() {
		select {
		case l.inbox <- TimerFired{Gen: gen}:
		case <-l.ctx.Done(): // actor shutting down
			return
		}
	})
}

func (l *Lobby) stopTurnTimer() {
	if l.turnTimer != nil {
		l.turnTimer.Stop()
	}
}

// Expose the inbox so tests or WS layer can send messages.
func (l *Lobby) Inbox() chan<- Msg { return l.inbox }
