package engine

import (
	"errors"
	"slices"
)

var ErrWrongTurn = errors.New("invalid turn")
var ErrIllegalPick = errors.New("illegal champion")
var ErrIllegalBan = errors.New("illegal ban")
var ErrUnsupportedCommand = errors.New("unsupported command")
var ErrGameAlreadyCompleted = errors.New("game already completed")

type Team string

const (
	TeamBlue Team = "blue"
	TeamRed  Team = "red"
)

type Action string

const (
	ActionBan  Action = "ban"
	ActionPick Action = "pick"
)

type Phase string

const (
	PhaseBan1  Phase = "ban1"
	PhasePick1 Phase = "pick1"
	PhaseBan2  Phase = "ban2"
	PhasePick2 Phase = "pick2"
	PhaseDone  Phase = "done"
)

type TurnStep struct {
	Team   Team
	Action Action
}

type State struct {
	Phase    Phase
	Cursor   int
	Picks    map[Team][]int
	Bans     map[Team][]int
	Fearless map[int]bool
	Hover    map[string]int
	Rules    Rules
}

type Rules struct {
	Fearless     bool
	PickTimerSec int
	BanTimerSec  int
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
	Type       CommandType
	Team       Team
	SeatID     string
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
	Type       EventType
	Team       Team
	SeatID     string
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
		if s.Cursor == len(GameOrder)-1 {
			events = append(events, Event{Type: EvtGameCompleted})
		}
		return events, newState, nil

	case CmdBanChampion:
		if step.Team != cmd.Team || step.Action != ActionBan {
			return nil, s, ErrWrongTurn
		}

		// Legality
		if !canBan(s, cmd.ChampionID) {
			return nil, s, ErrIllegalBan
		}

		events := []Event{
			{Type: EvtChampionBanned, Team: step.Team, ChampionID: cmd.ChampionID},
			{Type: EvtTurnAdvanced},
		}

		// Mutate new state for convenience
		newState.Bans[cmd.Team] = append(newState.Bans[cmd.Team], cmd.ChampionID)
		return events, newState, nil

	case CmdHoverChampion:
		if step.Team != cmd.Team || step.Action != ActionPick {
			return nil, s, ErrWrongTurn
		}

		newState.Hover[cmd.SeatID] = cmd.ChampionID
		return nil, newState, nil

	case CmdTimeoutAdvance:
		hoveredChamp, ok := s.Hover[cmd.SeatID]
		events := []Event{}

		// Conditions:
		// Haven't hovered -> if picking: random, if banning: skip
		// Have hovered -> if picking: canPick true -> pick, else ErrIllegalPick, if banning: canBan true -> ban, else ErrIllegalBan

		// Haven't hovered
		if !ok {
			if step.Action != ActionPick {
				// If we're banning & we haven't hovered, skip ban (advance turn)
				events = []Event{
					{Type: EvtTurnAdvanced},
				}
				return events, s, nil
			} else {
				// We're picking but haven't hovered, random champ
				c_id, valid := chooseRandomLegal(s, step.Team)

				if c_id == 0 || !valid {
					err := ErrIllegalPick
					return nil, s, err
				}

				events = []Event{
					{Type: EvtChampionPicked, Team: step.Team, ChampionID: c_id},
					{Type: EvtTurnAdvanced},
				}

				if s.Cursor == len(GameOrder)-1 {
					events = append(events, Event{Type: EvtGameCompleted})
				}

				newState.Picks[step.Team] = append(newState.Picks[step.Team], c_id)

				return events, newState, nil
			}
		}

		// Have hovered (ok = true):
		if step.Action != ActionPick {
			if !canBan(s, hoveredChamp) {
				// Champ can't be banned
				err := ErrIllegalBan
				return nil, s, err
			}

			// If we're not picking, have hovered, & hovered champ can be banned
			events = []Event{
				{Type: EvtChampionBanned, Team: step.Team, ChampionID: hoveredChamp},
				{Type: EvtTurnAdvanced},
			}
			newState.Bans[step.Team] = append(newState.Bans[step.Team], hoveredChamp)
			delete(newState.Hover, cmd.SeatID)

		} else {
			// Picking & hovered exists
			if !canPick(s, step.Team, hoveredChamp) {
				// Picking & hovered exists but is invalid
				err := ErrIllegalPick
				return nil, s, err
			}

			events = []Event{
				{Type: EvtChampionPicked, Team: step.Team, ChampionID: hoveredChamp},
				{Type: EvtTurnAdvanced},
			}
			if s.Cursor == len(GameOrder)-1 {
				events = append(events, Event{Type: EvtGameCompleted})
			}
			newState.Picks[step.Team] = append(newState.Picks[step.Team], hoveredChamp)
			delete(newState.Hover, cmd.SeatID)
		}

		return events, newState, nil

	default:
		return nil, s, ErrUnsupportedCommand
	}
}

func Reduce(events []Event) State {
	s := NewEmptyState()
	s.Cursor = 0
	for _, event := range events {
		switch event.Type {
		case EvtChampionPicked:
			s.Picks[event.Team] = append(s.Picks[event.Team], event.ChampionID)
		case EvtChampionBanned:
			s.Bans[event.Team] = append(s.Bans[event.Team], event.ChampionID)
		case EvtTurnAdvanced:
			s.Cursor++
		case EvtGameCompleted:
			s.Phase = PhaseDone
		}
	}

	s.Phase = DerivePhase(s.Cursor)
	return s
}

func hasPick(s State, id int) bool {
	exists := slices.Contains(s.Picks[TeamBlue], id) || slices.Contains(s.Picks[TeamRed], id)
	return exists
}

func canPick(s State, team Team, id int) bool {
	if slices.Contains(s.Bans[TeamBlue], id) || slices.Contains(s.Bans[TeamRed], id) {
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

func hasBan(s State, id int) bool {
	exists := slices.Contains(s.Bans[TeamBlue], id) || slices.Contains(s.Bans[TeamRed], id)
	return exists
}

func canBan(s State, id int) bool {
	if s.Fearless[id] {
		return false
	}
	if hasBan(s, id) {
		return false
	}
	if hasPick(s, id) {
		return false
	}

	return true
}

func currentStep(s State) (TurnStep, bool) {
	if s.Cursor >= len(GameOrder) {
		return TurnStep{}, true
	}
	return GameOrder[s.Cursor], false
}
