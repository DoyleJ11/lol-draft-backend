package lobby

import (
	"context"
	"testing"
	"time"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
)

// helper: receive one snapshot with a timeout so tests never hang
func recvSnapshot(t *testing.T, ch <-chan Snapshot, within time.Duration) Snapshot {
	t.Helper()
	select {
	case snap, ok := <-ch:
		if !ok {
			t.Fatalf("client outbox closed unexpectedly")
		}
		return snap
	case <-time.After(within):
		t.Fatalf("timed out waiting for snapshot")
		return Snapshot{} // unreachable
	}
}

func recvNoSnapshot(t *testing.T, ch <-chan Snapshot, within time.Duration) {
	t.Helper()
	select {
	case s, ok := <-ch:
		if !ok {
			// channel closed → that's fine; no further snapshots possible
			return
		}
		t.Fatalf("expected no snapshot within %v, but got: %+v", within, s)
	case <-time.After(within):
		// good: no snapshot
	}
}

func recvView(t *testing.T, ch <-chan View, within time.Duration) View {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(within):
		t.Fatalf("timed out waiting for view")
		return View{} // unreachable
	}
}

func TestLobby_Pick_BroadcastsSnapshotAndVersionIncrements(t *testing.T) {
	// 1) initial state at a pick step (Blue’s first pick is cursor 6 in our order)
	init := engine.NewEmptyState()
	init.Cursor = 6

	// 2) start lobby with a cancellable context so we can shut it down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewLobby(ctx, init)

	// 3) create a client "connection": an outbox channel the lobby will write snapshots to
	clientOut := make(chan Snapshot, 2) // small buffer so broadcast doesn’t block
	l.Inbox() <- Join{ClientID: "ch1", Outbox: clientOut}

	// 4) on join, lobby should immediately send the current snapshot (version 0, no picks)
	first := recvSnapshot(t, clientOut, 100*time.Millisecond)
	if first.Version != 0 {
		t.Fatalf("after join: want version=0, got %d", first.Version)
	}
	if len(first.State.Picks[engine.TeamBlue]) != 0 {
		t.Fatalf("after join: expected no picks yet, got %+v", first.State.Picks)
	}

	// 5) send a legal pick as a FromClient message
	cmd := engine.Command{Type: engine.CmdLockPick, Team: engine.TeamBlue, ChampionID: 266}
	l.Inbox() <- FromClient{Cmd: cmd}

	// 6) expect a new snapshot with version=1 and the pick applied
	next := recvSnapshot(t, clientOut, 100*time.Millisecond)
	if next.Version != 1 {
		t.Fatalf("after pick: want version=1, got %d", next.Version)
	}
	picks := next.State.Picks[engine.TeamBlue]
	if len(picks) != 1 || picks[0] != 266 {
		t.Fatalf("after pick: expected Blue picks [266], got %+v", next.State.Picks)
	}

	// 7) clean shutdown (optional in this test since defer cancel() will close l.ctx)
	l.Inbox() <- Shutdown{}
}

func TestLobby_DropSlowClient(t *testing.T) {
	init := engine.NewEmptyState()
	init.Cursor = 6

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewLobby(ctx, init)

	clientOut := make(chan Snapshot, 1)
	l.Inbox() <- Join{ClientID: "ch1", Outbox: clientOut}

	cmd := engine.Command{Type: engine.CmdLockPick, Team: engine.TeamBlue, ChampionID: 266}
	l.Inbox() <- FromClient{Cmd: cmd}

	reply := make(chan View, 1)
	l.Inbox() <- GetState{Reply: reply}
	view := recvView(t, reply, 100*time.Millisecond)

	if view.NumClients != 0 {
		t.Fatalf("expected slow client to be dropped; NumClients=%d", view.NumClients)
	}
}

func TestLobby_TimerFires_TimeoutAdvanceEmitsSnapshot(t *testing.T) {
	init := engine.NewEmptyState()
	init.Cursor = 5
	init.Rules.PickTimerSec = 0
	init.Rules.BanTimerSec = 0

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewLobby(ctx, init)

	clientOut := make(chan Snapshot, 1)
	l.Inbox() <- Join{ClientID: "ch1", Outbox: clientOut}
	first := recvSnapshot(t, clientOut, 100*time.Millisecond)
	if first.Version != 0 {
		t.Fatalf("after join: want version=0, got %d", first.Version)
	}

	l.Inbox() <- PrimeTimer{}
	next := recvSnapshot(t, clientOut, 500*time.Millisecond)
	if next.Version != 1 {
		t.Fatalf("after prime timer: want version=1, got %d", next.Version)
	}

	if next.State.Phase != engine.PhasePick1 {
		t.Fatalf("after prime timer: want phase Pick1, got %v", next.State.Phase)
	}
}

func TestLobby_TimerGen_DropsStaleFires(t *testing.T) {
	init := engine.NewEmptyState()
	init.Cursor = 5 // Ban Step
	init.Rules.BanTimerSec = 1
	init.Rules.PickTimerSec = 2

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewLobby(ctx, init)

	out := make(chan Snapshot, 4)
	l.Inbox() <- Join{ClientID: "ch1", Outbox: out}

	_ = recvSnapshot(t, out, 100*time.Millisecond) // version 0

	// Arm timer #1 for BAN
	l.Inbox() <- PrimeTimer{}

	// BEFORE #1 fires, advance via a legal BAN
	l.Inbox() <- FromClient{Cmd: engine.Command{
		Type:       engine.CmdBanChampion,
		Team:       engine.GameOrder[init.Cursor].Team,
		ChampionID: 55,
	}}

	// Drain the action snapshot (version 1 from the ban)
	postBan := recvSnapshot(t, out, 500*time.Millisecond)

	step := engine.GameOrder[postBan.State.Cursor]
	l.Inbox() <- FromClient{Cmd: engine.Command{
		Type:       engine.CmdHoverChampion,
		Team:       step.Team, // must match current pick team
		SeatID:     "",        // important: TimerFired uses "", so we hover under ""
		ChampionID: 99,        // choose something not banned/picked/fearless
	}}

	hoverSnap := recvSnapshot(t, out, 200*time.Millisecond)
	if hoverSnap.Version != 2 {
		t.Fatalf("expected version=2 on hover broadcast, got %d", hoverSnap.Version)
	}

	recvNoSnapshot(t, out, 1200*time.Millisecond)

	next := recvSnapshot(t, out, 1500*time.Millisecond)
	if next.Version != 3 {
		t.Fatalf("want version=3 after pick-timer fires, got %d", next.Version)
	}

	l.Inbox() <- Shutdown{}
}

func TestLobby_Shutdown_StopsTimer_NoFire(t *testing.T) {
	init := engine.NewEmptyState()
	init.Cursor = 6
	init.Rules.PickTimerSec = 1

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewLobby(ctx, init)

	out := make(chan Snapshot, 2)
	l.Inbox() <- Join{ClientID: "c1", Outbox: out}
	_ = recvSnapshot(t, out, 500*time.Millisecond) // drain join snapshot

	// Arm timer and immediately shut down
	l.Inbox() <- PrimeTimer{}
	l.Inbox() <- Shutdown{}

	// Now assert no *new* snapshot shows up (or channel is closed)
	recvNoSnapshot(t, out, 700*time.Millisecond) // < PickTimerSec (1s)
}
