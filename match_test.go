package matchmaker

import "testing"

func TestBuildReturnsGames(t *testing.T) {
	players := []Player{}
	for index := 0; index < 12; index++ {
		role := []string{"tank", "healer", "dps"}[index%3]
		players = append(players, Player{Name: string(rune('a' + index)), MMR: 1500 - index*10, Role: role})
	}
	games := Build(players, 200, 30, 100)
	if len(games) == 0 {
		t.Fatal("应该能凑出对局")
	}
}

func TestTeamSizes(t *testing.T) {
	players := []Player{}
	for index := 0; index < 6; index++ {
		players = append(players, Player{MMR: 1000 + index, Role: "dps"})
	}
	games := Build(players, 5000, 30, 100)
	for _, game := range games {
		if len(game.A) != 3 || len(game.B) != 3 {
			t.Fatalf("两队都应该是 3 人：%d %d", len(game.A), len(game.B))
		}
	}
}

func TestReportCountsGames(t *testing.T) {
	players := []Player{}
	for index := 0; index < 6; index++ {
		players = append(players, Player{MMR: 1000, Role: "dps"})
	}
	_, _, matched := Report(Build(players, 5000, 30, 100))
	if matched != 1 {
		t.Fatalf("6 个人应该凑成一局：%d", matched)
	}
}

func TestEmptyQueue(t *testing.T) {
	if games := Build(nil, 100, 30, 50); len(games) != 0 {
		t.Fatalf("空队列不该有对局：%d", len(games))
	}
}
