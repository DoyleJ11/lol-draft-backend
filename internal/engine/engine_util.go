package engine

func NewEmptyState() State {
    return State{
        Picks:  map[Team][]int{TeamBlue: {}, TeamRed: {}},
        Bans:  map[Team][]int{TeamBlue: {}, TeamRed: {}},
        Rules: Rules{},
        Fearless: map[int]bool{},
        Hover: map[string]int{},
    }
}

func ContainsEvent(events []Event, eventType EventType) bool {
    for _, event := range(events) {
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