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
