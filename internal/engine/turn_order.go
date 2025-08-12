package engine

var GameOrder = []TurnStep{
    // Ban Phase 1
    {Team: TeamBlue, Action: ActionBan}, // 0
    {Team: TeamRed,  Action: ActionBan}, // 1
    {Team: TeamBlue, Action: ActionBan}, // 2
    {Team: TeamRed,  Action: ActionBan}, // 3
    {Team: TeamBlue, Action: ActionBan}, // 4
    {Team: TeamRed,  Action: ActionBan}, // 5
    // Pick Phase 1
    {Team: TeamBlue, Action: ActionPick}, // 6
    {Team: TeamRed,  Action: ActionPick}, // 7
    {Team: TeamRed,  Action: ActionPick}, // 8
    {Team: TeamBlue, Action: ActionPick}, // 9
    {Team: TeamBlue, Action: ActionPick}, // 10
    {Team: TeamRed,  Action: ActionPick}, // 11
    // Ban Phase 2
    {Team: TeamRed,  Action: ActionBan}, // 12
    {Team: TeamBlue, Action: ActionBan}, // 13
    {Team: TeamRed,  Action: ActionBan}, // 14
    {Team: TeamBlue, Action: ActionBan}, // 15
    // Pick Phase 2
    {Team: TeamRed,  Action: ActionPick}, // 16
    {Team: TeamBlue, Action: ActionPick}, // 17
    {Team: TeamBlue, Action: ActionPick}, // 18
    {Team: TeamRed,  Action: ActionPick}, // 19
}