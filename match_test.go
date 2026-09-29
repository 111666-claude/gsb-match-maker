package matchmaker

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

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

func TestTeamGapIsTheThreshold(t *testing.T) {
	// 两队平均分几乎相同（1066 vs 1070），但同队最大分差是 190/200。
	players := []Player{
		{Name: "a", MMR: 1000, Role: "tank"},
		{Name: "b", MMR: 1000, Role: "healer"},
		{Name: "c", MMR: 1010, Role: "dps"},
		{Name: "d", MMR: 1010, Role: "tank"},
		{Name: "e", MMR: 1190, Role: "healer"},
		{Name: "f", MMR: 1200, Role: "dps"},
	}
	if games := Build(players, 60, 30, 0); len(games) != 0 {
		t.Fatalf("同队分差 200 超过上限 60，不该出局：%d", len(games))
	}
	games := Build(players, 250, 30, 0)
	if len(games) != 1 {
		t.Fatalf("上限放宽到 250 时该出局：%d", len(games))
	}
	if games[0].MaxGap != 200 {
		t.Fatalf("MaxGap 应该是同队最大分差 200：%d", games[0].MaxGap)
	}
}

func TestBothTeamsRoleChecked(t *testing.T) {
	// A 队恰好 1 坦克 1 治疗 1 输出；B 队 1 坦克 2 输出、缺治疗。
	players := []Player{
		{Name: "a", MMR: 1000, Role: "tank"},
		{Name: "b", MMR: 1001, Role: "tank"},
		{Name: "c", MMR: 1002, Role: "healer"},
		{Name: "d", MMR: 1003, Role: "dps"},
		{Name: "e", MMR: 1004, Role: "dps"},
		{Name: "f", MMR: 1005, Role: "dps"},
	}
	games := Build(players, 60, 30, 0)
	if len(games) != 1 {
		t.Fatalf("应该凑出一局：%d", len(games))
	}
	if games[0].Violations != 1 {
		t.Fatalf("B 队缺治疗要记 1 个违规队：%d", games[0].Violations)
	}
	// 6 个同职业玩家：两队都违规。
	six := []Player{}
	for index := 0; index < 6; index++ {
		six = append(six, Player{Name: fmt.Sprintf("t%d", index), MMR: 1000 + index, Role: "tank"})
	}
	games = Build(six, 60, 30, 0)
	if len(games) != 1 || games[0].Violations != 2 {
		t.Fatalf("两队都缺治疗与输出，应该记 2 个违规队：%+v", games)
	}
}

func TestWideningGrowsWithWait(t *testing.T) {
	// 每组 6 人、同队分差都是 100，上限 60、widenAfter 30、widenStep 15：
	// 等待 0/39/50/59 秒时上限是 60/60/90/90（不够），60/70 秒时是 105/120（够）。
	waits := []int{0, 39, 50, 59, 60, 70}
	players := []Player{}
	for group, wait := range waits {
		base := 2000 + group*1000
		for index := 0; index < 6; index++ {
			mmr := base
			if index >= 3 {
				mmr = base + 100
			}
			players = append(players, Player{
				Name: fmt.Sprintf("g%dp%d", group, index),
				MMR:  mmr,
				Role: "dps",
				Wait: wait,
			})
		}
	}
	games := Build(players, 60, 30, 15)
	if len(games) != 2 {
		t.Fatalf("只有等待 60s 与 70s 的组放宽后够格，应该 2 局：%d", len(games))
	}
	for _, game := range games {
		if game.A[0].Wait < 60 {
			t.Fatalf("等待 %d 秒的组不该在放宽后出局", game.A[0].Wait)
		}
	}
}

func randomQueue(rng *rand.Rand, size int) []Player {
	roles := []string{"tank", "healer", "dps"}
	players := make([]Player, size)
	for index := range players {
		players[index] = Player{
			Name: fmt.Sprintf("p%d", index),
			MMR:  rng.Intn(3000),
			Role: roles[rng.Intn(len(roles))],
			Wait: rng.Intn(200),
		}
	}
	return players
}

// referenceBuild 是口径的独立参考实现：按 MMR 就近分组、双向职业校验、逐级放宽。
func referenceBuild(players []Player, maxAverageGap, widenAfter, widenStep int) []Game {
	ordered := make([]Player, len(players))
	copy(ordered, players)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].MMR != ordered[j].MMR {
			return ordered[i].MMR < ordered[j].MMR
		}
		return ordered[i].Name < ordered[j].Name
	})
	gapOf := func(team []Player) int {
		low, high := team[0].MMR, team[0].MMR
		for _, player := range team[1:] {
			if player.MMR < low {
				low = player.MMR
			}
			if player.MMR > high {
				high = player.MMR
			}
		}
		return high - low
	}
	games := []Game{}
	for start := 0; start+6 <= len(ordered); start += 6 {
		chunk := ordered[start : start+6]
		teamA := []Player{chunk[0], chunk[2], chunk[4]}
		teamB := []Player{chunk[1], chunk[3], chunk[5]}
		longest := 0
		for _, player := range chunk {
			if player.Wait > longest {
				longest = player.Wait
			}
		}
		limit := maxAverageGap
		if longest > widenAfter {
			limit += widenStep * ((longest - widenAfter) / 10)
		}
		if gapOf(teamA) > limit || gapOf(teamB) > limit {
			continue
		}
		violations := 0
		for _, team := range [][]Player{teamA, teamB} {
			counts := map[string]int{}
			for _, player := range team {
				counts[player.Role]++
			}
			if counts["tank"] != 1 || counts["healer"] != 1 || counts["dps"] != 1 {
				violations++
			}
		}
		maxGap := gapOf(teamA)
		if other := gapOf(teamB); other > maxGap {
			maxGap = other
		}
		games = append(games, Game{A: teamA, B: teamB, MaxGap: maxGap, Violations: violations})
	}
	return games
}

func TestMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 200; trial++ {
		queue := randomQueue(rng, rng.Intn(150))
		maxGap := 20 + rng.Intn(180)
		widenAfter := rng.Intn(60)
		widenStep := rng.Intn(40)
		got := Build(queue, maxGap, widenAfter, widenStep)
		want := referenceBuild(queue, maxGap, widenAfter, widenStep)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("第 %d 份队列与参考实现不一致：got %+v want %+v", trial, got, want)
		}
	}
}

func TestInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 50; trial++ {
		queue := randomQueue(rng, rng.Intn(200))
		first := Build(queue, 60, 30, 15)
		second := Build(queue, 60, 30, 15)
		if !reflect.DeepEqual(first, second) {
			t.Fatal("同一份队列重复匹配必须给出同样的对局")
		}
		seen := map[string]int{}
		total := 0
		for _, game := range first {
			for _, player := range append(append([]Player{}, game.A...), game.B...) {
				seen[player.Name]++
				total++
			}
		}
		for name, count := range seen {
			if count != 1 {
				t.Fatalf("玩家 %s 出现在 %d 局，应该互不重复", name, count)
			}
		}
		if total != len(first)*6 {
			t.Fatalf("matched*6 应等于实际出局人数：%d != %d", len(first)*6, total)
		}
	}
}

func TestScalesTo100k(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	queue := randomQueue(rng, 100000)
	start := time.Now()
	games := Build(queue, 60, 30, 15)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("10 万玩家匹配应在 1 秒内完成：%v", elapsed)
	}
	if len(games) == 0 {
		t.Fatal("10 万玩家应该能凑出对局")
	}
}
