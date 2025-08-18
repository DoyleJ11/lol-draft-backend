package lobby

import (
	"context"
	"log"
	"time"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
	"github.com/DoyleJ11/lol-draft-backend/internal/types"
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
	ClientID string
	Cmd      engine.Command
}

func (FromClient) isLobbyMsg() {}

type Join struct {
	ClientID string
	Outbox   chan types.ServerMessage // changed: envelope channel
}

func (Join) isLobbyMsg() {}

type Leave struct{ ClientID string }

func (Leave) isLobbyMsg() {}

type TimerFired struct{ Gen int }

func (TimerFired) isLobbyMsg() {}

type Shutdown struct{}

func (Shutdown) isLobbyMsg() {}

// Test-only messages
type GetState struct{ Reply chan View }

func (GetState) isLobbyMsg() {}

type PrimeTimer struct{}

func (PrimeTimer) isLobbyMsg() {}

// (Kept for tests that might use it; not used for WS anymore)
type Snapshot struct {
	Version int
	State   engine.State
}

// Introspection view for tests
type View struct {
	Version    int
	NumClients int
	State      engine.State
}

type Lobby struct {
	inbox     chan Msg
	state     engine.State
	version   int
	clients   map[string]chan types.ServerMessage
	turnTimer *time.Timer
	timerGen  int
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewLobby(parent context.Context, initial engine.State) *Lobby {
	ctx, cancel := context.WithCancel(parent)

	// Optional (nice): make the very first snapshot show a real phase
	initial.Phase = engine.DerivePhase(initial.Cursor)

	l := &Lobby{
		inbox:   make(chan Msg, 64),
		state:   initial,
		version: 0,
		clients: make(map[string]chan types.ServerMessage),
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
				// Register client and immediately send current snapshot
				l.clients[msg.ClientID] = msg.Outbox
				l.sendTo(msg.ClientID, types.ServerMessage{
					Type:    "StateSnapshot",
					Version: l.version,
					State:   &l.state,
				})

			case Leave:
				if ch, ok := l.clients[msg.ClientID]; ok {
					close(ch) // let WS writer goroutine exit
					delete(l.clients, msg.ClientID)
				}

			case FromClient:
				log.Printf("FromClient: cursor=%d cmd=%s", l.state.Cursor, msg.Cmd.Type)
				events, newState, err := engine.Apply(l.state, msg.Cmd)
				if err != nil {
					log.Printf("ApplyError: client=%s err=%v", msg.ClientID, err)
					// Send error ONLY to this client; don't broadcast
					l.sendTo(msg.ClientID, types.ServerMessage{
						Type:  "Error",
						Error: err.Error(),
					})
					break
				}

				// Success path: update state/cursor/phase, version++, broadcast snapshot
				l.state = newState
				for _, e := range events {
					switch e.Type {
					case engine.EvtTurnAdvanced:
						l.state.Cursor++
					case engine.EvtGameCompleted:
						l.stopTurnTimer()
					case engine.EvtChampionPicked, engine.EvtChampionBanned:
						// Clear any hovers that now point to a taken/banned champ
						for seat, champ := range l.state.Hover {
							if champ == e.ChampionID {
								delete(l.state.Hover, seat)
							}
						}
					}
				}
				l.state.Phase = engine.DerivePhase(l.state.Cursor)
				l.version++
				l.broadcastState()

				// (Re)arm timer if turn advanced and game not completed
				if hasEvent(events, engine.EvtTurnAdvanced) && !hasEvent(events, engine.EvtGameCompleted) {
					l.armTurnTimer()
				}

			case TimerFired:
				log.Printf("timer: fired gen=%d (current=%d) cursor=%d", msg.Gen, l.timerGen, l.state.Cursor)
				if msg.Gen != l.timerGen {
					// stale fire — ignore
					break
				}
				cmd := engine.Command{Type: engine.CmdTimeoutAdvance, SeatID: ""} // TODO: seat auth later
				events, newState, err := engine.Apply(l.state, cmd)
				if err != nil {
					// should not really happen; ignore in v1
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
				l.broadcastState()

				if hasEvent(events, engine.EvtTurnAdvanced) && !hasEvent(events, engine.EvtGameCompleted) {
					l.armTurnTimer()
				}

			case PrimeTimer:
				l.armTurnTimer()

			case GetState:
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
		close(ch)
		delete(l.clients, id)
	}
	l.cancel()
}

// ---- Outbound helpers ----

func (l *Lobby) sendTo(clientID string, m types.ServerMessage) {
	ch, ok := l.clients[clientID]
	if !ok {
		return
	}
	select {
	case ch <- m:
	default:
		// slow client: drop & remove
		close(ch)
		delete(l.clients, clientID)
	}
}

func (l *Lobby) broadcastState() {
	msg := types.ServerMessage{
		Type:    "StateSnapshot",
		Version: l.version,
		State:   &l.state,
	}
	for id := range l.clients {
		l.sendTo(id, msg)
	}
}

// ---- Timers ----

func (l *Lobby) armTurnTimer() {
	step := engine.GameOrder[l.state.Cursor]
	var sec int
	if step.Action == engine.ActionPick {
		sec = l.state.Rules.PickTimerSec
	} else {
		sec = l.state.Rules.BanTimerSec
	}

	// Guard: don’t arm zero/negative timers
	if sec <= 0 {
		l.stopTurnTimer() // ensure any previous timer is stopped

		log.Printf("timer: not arming (sec=%d) cursor=%d phase=%s", sec, l.state.Cursor, l.state.Phase)
		return
	}

	dur := time.Duration(sec) * time.Second
	l.timerGen++
	gen := l.timerGen

	if l.turnTimer != nil {
		l.turnTimer.Stop()
	}
	l.turnTimer = time.AfterFunc(dur, func() {
		select {
		case l.inbox <- TimerFired{Gen: gen}:
			log.Printf("timer: armed %ds cursor=%d phase=%s gen=%d", sec, l.state.Cursor, l.state.Phase, gen)
		case <-l.ctx.Done():
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
