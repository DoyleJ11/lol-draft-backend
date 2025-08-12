package engine

import (
	"errors"
	"testing"
)

func newEmptyState() State {
    return State{
        Picks:  map[Team][]int{TeamBlue: {}, TeamRed: {}},
        Bans:  map[Team][]int{TeamBlue: {}, TeamRed: {}},
        Rules: Rules{},
        Fearless: map[int]bool{},
        Hover: map[string]int{},
    }
}

func containsEvent(events []Event, eventType EventType) bool {
    for _, event := range(events) {
        if event.Type == eventType {
            return true
        }
    }
    return false
}

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
    s := newEmptyState()
    s.Cursor = len(GameOrder) - 1
    cmd := Command{Type: CmdLockPick, Team: GameOrder[s.Cursor].Team, ChampionID: 99}

    events, _, err := Apply(s, cmd)
    if err != nil { t.Fatalf("unexpected err %v", err) }

    if !containsEvent(events, EvtGameCompleted) {
        t.Fatalf("expected EvtGameCompleted")
    }
}