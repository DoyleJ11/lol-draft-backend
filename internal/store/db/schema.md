    users (id, email unique, password_hash, display_name, created_at)
    lobbies (code PK, host_user_id, bans_on, series_type, fearless, pick_timer_sec, ban_timer_sec, status)
    lobby_members (lobby_code, user_id NULL, name, team, rank, role_primary, role_secondary)
    series (id, lobby_code, best_of, fearless, created_at, completed_at)
    games (id, series_id, index, created_at, completed_at)
    draft_events (id, game_id, ordinal, type, payload_json, at)

    For Day 1, I'd probably add lobbies, lobby_members, users. Maybe I'd need draft_events more but we'll see.
