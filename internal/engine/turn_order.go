package engine

var GameOrder = []TurnStep{
    // Ban Phase 1
    {Team: TeamBlue, Action: ActionBan},
    {Team: TeamRed,  Action: ActionBan},
    {Team: TeamBlue, Action: ActionBan},
    {Team: TeamRed,  Action: ActionBan},
    {Team: TeamBlue, Action: ActionBan},
    {Team: TeamRed,  Action: ActionBan},
    // Pick Phase 1
    {Team: TeamBlue, Action: ActionPick},
    {Team: TeamRed,  Action: ActionPick},
    {Team: TeamRed,  Action: ActionPick},
    {Team: TeamBlue, Action: ActionPick},
    {Team: TeamBlue, Action: ActionPick},
    {Team: TeamRed,  Action: ActionPick},
    // Ban Phase 2
    {Team: TeamRed,  Action: ActionBan},
    {Team: TeamBlue, Action: ActionBan},
    {Team: TeamRed,  Action: ActionBan},
    {Team: TeamBlue, Action: ActionBan},
    // Pick Phase 2
    {Team: TeamRed,  Action: ActionPick},
    {Team: TeamBlue, Action: ActionPick},
    {Team: TeamBlue, Action: ActionPick},
    {Team: TeamRed,  Action: ActionPick},
}