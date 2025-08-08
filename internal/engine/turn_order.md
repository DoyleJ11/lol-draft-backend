Turn Order:
Team 1 = Blue
Team 2 = Red

Ban Phase 1:
Blue = 0
Red = 1
Blue = 2
Red = 3
Blue = 4
Red = 5

Pick Phase 1:
Blue = 0
Red = 1
Red = 2
Blue = 3
Blue = 4
Red = 5

Ban Phase 2:
Red = 0
Blue = 1
Red = 2
Blue = 3

Pick Phase 2:
Red = 0
Blue = 1
Blue = 2
Red = 3

Socratic prompt: How will you represent this sequence so the engine can just “advance to next step” without if/else spaghetti?
I think the way I would represent this sequence would be with a "turn order" object or
something like that. Maybe each phase has an object, could be something like:

Ban_Phase_1 = {
0: "team1"
1: "team2"
2: "team1"
3: "team2"
4: "team1"
5: "team2"
}

Then on each "turn" we could increment a turn counter by 1 and do something like
Ban_Phase_1[counter] to find which team's turn it is.
