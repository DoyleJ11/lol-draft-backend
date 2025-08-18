package hub

import (
	"context"
	"testing"

	"github.com/DoyleJ11/lol-draft-backend/internal/engine"
	"github.com/DoyleJ11/lol-draft-backend/internal/lobby"
)

func TestHub_Create_Get_SamePointer(t *testing.T) {
	ctx := context.Background()
	h := NewHub(ctx)
	reply := make(chan *lobby.Lobby, 1)

	state := engine.NewEmptyState()
	h.Inbox() <- CreateLobby{Code: "ZED123", State: state, Reply: reply}
	lb1 := <-reply

	h.Inbox() <- GetLobby{Code: "ZED123", Reply: reply}
	lb2 := <-reply

	if lb1 == nil || lb2 == nil || lb1 != lb2 {
		t.Fatalf("expected same lobby pointer")
	}
}
