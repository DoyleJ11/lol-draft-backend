package types

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