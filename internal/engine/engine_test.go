package engine

import (
	"errors"
	"reflect"
	"testing"
)

func TestDuplicatePickIsRejected(t *testing.T) {
    cases := []struct{
        name string
        setup State
        cmd   Command
        wantErr bool
    }{
        {
            name: "cannot pick a champ already picked by same team",
            setup: State{
                Cursor: 6, // first pick phase
                Picks: map[Team][]int{TeamBlue: {266}, TeamRed: {}},
                Bans:  map[Team][]int{TeamBlue: {}, TeamRed: {}},
                Rules: Rules{},
                Hover: map[string]int{},
            },
            cmd: Command{Type: CmdLockPick, Team: TeamBlue, ChampionID: 266},
            wantErr: true,
        },
        // add more cases...
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            _, _, err := Apply(tc.setup, tc.cmd)
            if tc.wantErr && err == nil { t.Fatalf("expected error") }
            if !tc.wantErr && err != nil { t.Fatalf("unexpected err: %v", err) }
        })
    }
}

// Write canPick tests with 3 cases: Legal pick, blocked by prior team pick, blocked by fearless

func TestCanPickChampion(t *testing.T) {
    cases := []struct{
        name string
        setup State
        cmd Command
        expectedBool bool
    } {
        {
            name: "Legal Pick",
            setup: State{
                Cursor: 7, // Red Team Pick 1
                Picks: map[Team][]int{TeamBlue: {266}, TeamRed: {}},
                Bans:  map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5, 6}},
                Rules: Rules{},
                Hover: map[string]int{},
            },
            cmd: Command{Type: CmdLockPick, Team: TeamRed, ChampionID: 15},
            expectedBool: true,
        },
        {
            name: "Blocked by prior team pick",
            setup: State{
                Cursor: 7, // Red Team Pick 1
                Picks: map[Team][]int{TeamBlue: {266}, TeamRed: {}},
                Bans:  map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5, 6}},
                Rules: Rules{},
                Hover: map[string]int{},
            },
            cmd: Command{Type: CmdLockPick, Team: TeamRed, ChampionID: 266},
            expectedBool: false,
        },
        {
            name: "Blocked by fearless",
            setup: State{
                Cursor: 7, // Red Team Pick 1
                Picks: map[Team][]int{TeamBlue: {266}, TeamRed: {}},
                Bans:  map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5, 6}},
                Rules: Rules{Fearless: true},
                Fearless: map[int]bool{222: true, 251: true, 45: true},
                Hover: map[string]int{},
            },
            cmd: Command{Type: CmdLockPick, Team: TeamRed, ChampionID: 222},
            expectedBool: false,
        },
    }

    for _, tc := range(cases) {
        t.Run(tc.name, func(t *testing.T) {
            valid := canPick(tc.setup, tc.cmd.Team, tc.cmd.ChampionID)
            if valid != tc.expectedBool {
                t.Fatalf("canPick: got %v, want %v", valid, tc.expectedBool)
            }
        })
    }
}

func TestOrderLookup(t *testing.T) {
    cases := []struct{
        name string
        setup State
        expectedVal TurnStep
    } {
        {
            name: "Red Team Pick 1",
            setup: State{Cursor: 7}, // Red team pick 1
            expectedVal: TurnStep{Team: TeamRed, Action: ActionPick},
        },
        {
            name: "Red Team Ban 1",
            setup: State{Cursor: 1}, // Red team ban 1
           expectedVal: TurnStep{Team: TeamRed, Action: ActionBan},
        },
        {
            name: "Red Team Pick 1",
            setup: State{Cursor: 14}, // Blue team ban 4 - ban phase 2
            expectedVal: TurnStep{Team: TeamRed, Action: ActionBan},
        },
        {
            name: "Red Team Pick 1",
            setup: State{Cursor: 9}, // Blue team pick 2
           expectedVal: TurnStep{Team: TeamBlue, Action: ActionPick},
        },
    }

    for _, tc := range(cases) {
        t.Run(tc.name, func(t *testing.T) {
            step, _ := currentStep(tc.setup)
            if step != tc.expectedVal {
                t.Fatalf("got %#v, want %#v", step, tc.expectedVal)
            }
        })
    }
}

func TestTurnOrder_RejectsOutOfOrderAction(t *testing.T) {
    s := State{
        Cursor: 0,
        Picks: map[Team][]int{TeamBlue: {}, TeamRed: {}},
        Bans: map[Team][]int{TeamBlue: {}, TeamRed: {}},
    }
    cmd := Command{Type: CmdLockPick, Team: TeamRed, ChampionID: 6}

    _, _, err := Apply(s, cmd)
    if err == nil || !errors.Is(err, ErrWrongTurn) {
        t.Fatalf("want ErrWrongTurn, got %v", err)
    }
}

func TestRejectsDuplicatePickSameTeam(t *testing.T) {
    s := State{
        Cursor: 9,
        Picks: map[Team][]int{TeamBlue: {266}, TeamRed: {121, 132}},
        Bans: map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5, 6}},
    }

    cmd := Command{Type: CmdLockPick, Team: TeamBlue, ChampionID: 266}
     _, _, err := Apply(s, cmd)
    if err == nil || !errors.Is(err, ErrIllegalPick) {
        t.Fatalf("want ErrIllegalPick, got %v", err)
    }
}

func TestApply_EmitsGameCompletedOnLastStep(t *testing.T) {
    s := NewEmptyState()
    s.Cursor = len(GameOrder) - 1
    cmd := Command{Type: CmdLockPick, Team: GameOrder[s.Cursor].Team, ChampionID: 99}

    events, _, err := Apply(s, cmd)
    if err != nil { t.Fatalf("unexpected err %v", err) }

    if !ContainsEvent(events, EvtGameCompleted) {
        t.Fatalf("expected EvtGameCompleted")
    }
}

func TestBan_RejectsDuplicateBan(t *testing.T) {
    s := NewEmptyState()
    s.Cursor = 5
    s.Bans = map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5}}
    cmd := Command{Type: CmdBanChampion, Team: GameOrder[s.Cursor].Team, ChampionID: 3}

    _, _, err := Apply(s, cmd)
    if !errors.Is(err, ErrIllegalBan) {
        t.Fatalf("want ErrIllegalBan, didn't get err")
    }
}

func TestBan_RejectsBanOfAlreadyPickedChampion(t *testing.T) {
    s := NewEmptyState()
    s.Cursor = 12
    s.Bans = map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5, 41}}
    s.Picks = map[Team][]int{TeamBlue: {6, 7, 8}, TeamRed: {9, 10, 11}}
    cmd := Command{Type: CmdBanChampion, Team: GameOrder[s.Cursor].Team, ChampionID: 7}

    _, _, err := Apply(s, cmd)
    if !errors.Is(err, ErrIllegalBan) {
        t.Fatalf("want ErrIllegalBan, didn't get err")
    }
}

func TestBan_AcceptsLegalBan(t *testing.T) {
    s := NewEmptyState()
    s.Cursor = 5
    s.Bans = map[Team][]int{TeamBlue: {1, 2, 3}, TeamRed: {4, 5}}
    cmd := Command{Type: CmdBanChampion, Team: GameOrder[s.Cursor].Team, ChampionID: 7}

    _, _, err := Apply(s, cmd)
    if err != nil {
        t.Fatalf("unexpected err, %v", err)
    }
}

func TestTurnOrder_RejectsOutOfOrderBan(t *testing.T) {
    s := NewEmptyState()
    s.Cursor = 5
    cmd := Command{Type: CmdBanChampion, Team: TeamBlue, ChampionID: 3}

     _, _, err := Apply(s, cmd)
    if err == nil || !errors.Is(err, ErrWrongTurn) {
        t.Fatalf("want ErrWrongTurn, got %v", err)
    }
}

func TestApply_BanEmitsTurnAdvanced(t *testing.T) {
    s := NewEmptyState()
    s.Cursor = 5
    cmd := Command{Type: CmdBanChampion, Team: GameOrder[s.Cursor].Team, ChampionID: 3}

    events, _, err := Apply(s, cmd)
    if err != nil { t.Fatalf("unexpected err %v", err) }

    if !ContainsEvent(events, EvtChampionBanned) || !ContainsEvent(events, EvtTurnAdvanced) {
        t.Fatalf("expected EvtChampionBanned & EvtTurnAdvanced")
    }
}

func TestReduce_RebuildsPicksBansAndCursor(t *testing.T) {
    events := []Event{
        {
            Type: EvtChampionBanned,
            Team: TeamBlue,
            ChampionID: 1,
        },
        {
            Type: EvtTurnAdvanced,
        },
        {
            Type: EvtChampionBanned,
            Team: TeamRed,
            ChampionID: 2,
        },
        {
            Type: EvtTurnAdvanced,
        },
        {
            Type: EvtChampionPicked,
            Team: TeamBlue,
            ChampionID: 3,
        },
        {
            Type: EvtTurnAdvanced,
        },
        {
            Type: EvtChampionPicked,
            Team: TeamRed,
            ChampionID: 4,
        },
        {
            Type: EvtTurnAdvanced,
        },
        {
            Type: EvtChampionPicked,
            Team: TeamRed,
            ChampionID: 5,
        },
        {
            Type: EvtTurnAdvanced,
        },
    }

    want := NewEmptyState()
    want.Bans  = map[Team][]int{TeamBlue: {1}, TeamRed: {2}}
    want.Picks = map[Team][]int{TeamBlue: {3}, TeamRed: {4, 5}}
    want.Cursor = 5
    want.Phase = PhaseBan1

    got := Reduce(events)

    if !reflect.DeepEqual(got, want) {
        t.Fatalf("state mismatch.\n got: %#v\nwant: %#v", got, want)
    }
}

func TestDerivePhase_FromCursorRanges(t *testing.T) {
    cases := []struct{
        name string
        cursor int
        wantPhase Phase
    }{
        { name: "Pick Phase 1", cursor: 7, wantPhase: PhasePick1 }, 
        { name: "Ban Phase 1", cursor: 3, wantPhase: PhaseBan1 },  
        { name: "Ban Phase 2", cursor: 14, wantPhase: PhaseBan2 },
        { name: "Pick Phase 2", cursor: 18, wantPhase: PhasePick2 },
    }

        for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
          phase := DerivePhase(tc.cursor)
          if phase != tc.wantPhase {
            t.Fatalf("Wanted %v, got %v", tc.wantPhase, phase)
          }
        })
    }
}