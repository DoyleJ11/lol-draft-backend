package engine

import (
	"errors"
	"slices"
)

var ErrWrongTurn = errors.New("invalid turn")
var ErrIllegalPick = errors.New("illegal champion")
var ErrUnsupportedCommand = errors.New("unsupported command")
var ErrGameAlreadyCompleted = errors.New("game already completed")

type Team string
const (
    TeamBlue Team = "blue"
    TeamRed  Team = "red"
)

type Action string
const (
	ActionBan Action = "ban"
	ActionPick Action = "pick"
)

type Phase string
const (
	PhaseBan1 Phase = "ban1"
	PhasePick1 Phase = "pick1"
	PhaseBan2 Phase = "ban2"
	PhasePick2 Phase = "pick2"
	PhaseDone Phase = "done"
)

type TurnStep struct {
	Team Team
	Action Action
}

type State struct {
	Phase Phase
	Cursor int
	Picks map[Team][]int
	Bans map[Team][]int
	Fearless map[int]bool
	Hover map[string]int
	Rules Rules
}

type Rules struct {
	Fearless bool
	PickTimerSec int
	BanTimerSec int
}

type CommandType string
const (
	CmdLockPick       CommandType = "LockPick"
    CmdBanChampion    CommandType = "BanChampion"
    CmdHoverChampion  CommandType = "HoverChampion"
    CmdTimeoutAdvance CommandType = "TimeoutAdvance"
    CmdStartGame      CommandType = "StartGame"
)

/*
	CmdLockPick      -> EvtChampionPicked -> EvtTurnAdvanced -> EvtTimerStarted
    CmdBanChampion    -> EvtChampionBanned -> EvtTurnAdvanced -> EvtTimerStarted
    CmdHoverChampion  -> I don't think we have an event for hovering a champion because I believe hovers are meant to be in memory, not persistent
    CmdTimeoutAdvance  -> EvtTimerExpired-> EvtChampionPicked -> EvtTurnAdvanced or EvtGameCompleted
	^ My logic here is that we send the event that the timer expires, then we lock in either a random or hovered champion (EvtChampionPicked),
	then we advance the turn
    CmdStartGame      -> EvtTimerStarted

*/


type Command struct {
	Type CommandType
	Team Team
	SeatID string
	ChampionID int
}

type EventType string
const (
	EvtChampionPicked EventType = "ChampionPicked"
    EvtChampionBanned EventType = "ChampionBanned"
    EvtTurnAdvanced   EventType = "TurnAdvanced"
    EvtTimerStarted   EventType = "TimerStarted"
    EvtTimerExpired   EventType = "TimerExpired"
    EvtGameCompleted  EventType = "GameCompleted"
)

type Event struct {
	Type EventType
	Team Team
	SeatID string
	ChampionID int
}

func Apply(s State, cmd Command) ([]Event, State, error) {

	if s.Cursor >= len(GameOrder) {
		return nil, s, ErrGameAlreadyCompleted
	}

	step := GameOrder[s.Cursor]
	newState := s

	switch cmd.Type {
	case CmdLockPick:
		// Turn must match BOTH team & action
		if step.Team != cmd.Team || step.Action != ActionPick {
			return nil, s, ErrWrongTurn
		}

		// Legality
		if !canPick(s, cmd.Team, cmd.ChampionID) {
			return nil, s, ErrIllegalPick
		}
		// Build Events

		events := []Event{
			{Type: EvtChampionPicked, Team: step.Team, ChampionID: cmd.ChampionID},
			{Type: EvtTurnAdvanced},
		}

		// Mutate new state for convenience
		newState.Picks[cmd.Team] = append(newState.Picks[cmd.Team], cmd.ChampionID)

		//Completion
		if s.Cursor == len(GameOrder) - 1 {
			events = append(events, Event{Type: EvtGameCompleted})
		}
		return events, newState, nil

	default:
		return nil, s, ErrUnsupportedCommand
	}
}

func Reduce(events []Event) State {
	return State{}
}

func hasPick(s State, id int) bool {
	exists := slices.Contains(s.Picks[TeamBlue], id) || slices.Contains(s.Picks[TeamRed], id)
	return exists
}

func canPick(s State, team Team, id int) bool {
	if slices.Contains(s.Bans[TeamBlue], id)  || slices.Contains(s.Bans[TeamRed], id){
		return false
	}

	if hasPick(s, id) {
		return false
	}

	if s.Rules.Fearless && s.Fearless[id] {
		return false
	}

	return true
}

func currentStep(s State) (TurnStep, bool) {
	if s.Cursor >= len(GameOrder) { return TurnStep{}, true }
	return GameOrder[s.Cursor], false
}