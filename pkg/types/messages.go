package types

// Client -> Server
// Hello:
//   lobby_code: string
//   display_name: string
//   last_seen_version: number
//
// LockPick:
//   champion_id: number
//
// BanChampion:
//   champion_id: number
//
// HoverChampion (transient):
//   champion_id: number
//
// SetRules:
//   bans_on: boolean
//   series_type: "none" | "bo3" | "bo5"
//   fearless: boolean
//   pick_timer_sec: number
//   ban_timer_sec: number
//
// StartSeries:
//   best_of: 1 | 3 | 5
//
// StartGame: {}
//
// AssignTeamsBalanced:
//   priority: "rank" | "roles" | "mixed"

// Server -> Client
// StateSnapshot:
//   version: number
//   lobby_code: string
//   phase: "ban1" | "pick1" | "ban2" | "pick2" | "completed"
//   active_turn_index: number
//   active_team: "blue" | "red"
//   remaining_ms: number
//   picks: { blue: number[], red: number[] }
//   bans:  { blue: number[], red: number[] }
//   hovered: { [seatId]: number } // optional
//   teams: { blue: Seat[], red: Seat[] } // each Seat has user_id|name|rank|roles
//   fearless_used: number[] // champion_ids picked earlier in series
//
// Error:
//   code: string
//   message: string
//
// TimerTick (optional if you include remaining_ms in every snapshot):
//   active_turn_index: number
//   remaining_ms: number