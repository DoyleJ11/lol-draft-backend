package engine

func NewEmptyState() State {
	s := State{
		Picks:    map[Team][]int{TeamBlue: {}, TeamRed: {}},
		Bans:     map[Team][]int{TeamBlue: {}, TeamRed: {}},
		Fearless: map[int]bool{},
		Hover:    map[string]int{},
		Rules:    Rules{PickTimerSec: 25, BanTimerSec: 25},
		Cursor:   0,
	}
	s.Phase = DerivePhase(s.Cursor) // Ensure "ban1" shows up on join
	return s
}

func ContainsEvent(events []Event, eventType EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func DerivePhase(cursor int) Phase {
	if cursor >= len(GameOrder) {
		return PhaseDone
	} else if cursor >= 0 && cursor <= 5 {
		return PhaseBan1
	} else if cursor > 5 && cursor <= 11 {
		return PhasePick1
	} else if cursor > 11 && cursor <= 15 {
		return PhaseBan2
	} else {
		return PhasePick2
	}
}

var chooseRandomLegal = func(s State, team Team) (int, bool) {
	// real impl later; for tests, you’ll stub this.
	return 0, false
}
