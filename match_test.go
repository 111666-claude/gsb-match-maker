package matchmaker

import (
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

// TestTeamGapThreshold 门槛按同队最大分差判断，不是两队平均分之差。
func TestTeamGapThreshold(t *testing.T) {
	// A={0,0,100}、B={0,100,100}：两队平均分仅差 33，但同队最大分差是 100。
	players := []Player{}
	for index, mmr := range []int{0, 0, 0, 100, 100, 100} {
		players = append(players, Player{Name: string(rune('a' + index)), MMR: mmr, Role: "dps"})
	}
	if games := Build(players, 60, 30, 0); len(games) != 0 {
		t.Fatalf("同队分差 100 超过上限 60，不该出局：%d", len(games))
	}
	games := Build(players, 100, 30, 0)
	if len(games) != 1 {
		t.Fatalf("上限放宽到 100 后应该出局：%d", len(games))
	}
	if games[0].MaxGap != 100 {
		t.Fatalf("MaxGap 应取两队同队分差的最大值 100：%d", games[0].MaxGap)
	}
}

// TestBothTeamsRoleChecked 职业规则对 A、B 两队都校验。
func TestBothTeamsRoleChecked(t *testing.T) {
	players := []Player{}
	for index := 0; index < 6; index++ {
		players = append(players, Player{Name: string(rune('a' + index)), MMR: 1000 + index, Role: "dps"})
	}
	games := Build(players, 5000, 30, 0)
	if len(games) != 1 {
		t.Fatalf("应该凑出一局：%d", len(games))
	}
	if games[0].Violations != 2 {
		t.Fatalf("两队都缺坦克与治疗，应该报 2 队：%d", games[0].Violations)
	}

	valid := []Player{
		{Name: "a", MMR: 1000, Role: "tank"}, {Name: "b", MMR: 1001, Role: "healer"},
		{Name: "c", MMR: 1002, Role: "dps"}, {Name: "d", MMR: 1003, Role: "tank"},
		{Name: "e", MMR: 1004, Role: "healer"}, {Name: "f", MMR: 1005, Role: "dps"},
	}
	games = Build(valid, 5000, 30, 0)
	if len(games) != 1 || games[0].Violations != 0 {
		t.Fatalf("两队职业都齐，不该有违规：%+v", games)
	}
}

// TestWideningCurve 超时放宽随等待时长逐级递增，不是一次性放宽。
func TestWideningCurve(t *testing.T) {
	// 同队分差固定为 70：上限 60 不够，放宽一级（+20）到 80 才够。
	group := func(wait int) []Player {
		players := []Player{}
		for index, mmr := range []int{0, 0, 0, 70, 70, 70} {
			players = append(players, Player{Name: string(rune('a' + index)), MMR: mmr, Role: "dps", Wait: wait})
		}
		return players
	}
	cases := []struct {
		wait    int
		matched int
	}{
		{30, 0},  // 未超过 widenAfter，不放宽
		{39, 0},  // 超过 9 秒，不足 10 秒，仍不放宽
		{40, 1},  // 多等 10 秒，放宽一级到 80
		{49, 1},  // 仍是一级
		{50, 1},  // 放宽两级到 100
		{100, 1}, // 逐级递增，远超过也能匹配
	}
	for _, testCase := range cases {
		games := Build(group(testCase.wait), 60, 30, 20)
		if len(games) != testCase.matched {
			t.Fatalf("等待 %d 秒应该匹配 %d 局，实际 %d", testCase.wait, testCase.matched, len(games))
		}
	}

	// 逐级曲线：分差 95 需要放宽两级（上限 100）才够，一级（80）不够。
	players := []Player{}
	for index, mmr := range []int{0, 0, 0, 95, 95, 95} {
		players = append(players, Player{Name: string(rune('a' + index)), MMR: mmr, Role: "dps", Wait: 49})
	}
	if games := Build(players, 60, 30, 20); len(games) != 0 {
		t.Fatalf("只放宽一级到 80，分差 95 不该出局：%d", len(games))
	}
	for index := range players {
		players[index].Wait = 50
	}
	if games := Build(players, 60, 30, 20); len(games) != 1 {
		t.Fatalf("放宽两级到 100，分差 95 应该出局：%d", len(games))
	}
}

// referenceBuild 参考实现：按 MMR 就近分组 + 双向职业校验 + 逐级放宽。
// 刻意写得很直白（每组重新排序、逐级循环加上放宽步长），用于对照 Build。
func referenceBuild(players []Player, maxAverageGap int, widenAfter int, widenStep int) []Game {
	games := []Game{}
	remaining := append([]Player(nil), players...)
	for len(remaining) >= 6 {
		sort.SliceStable(remaining, func(i, j int) bool {
			if remaining[i].MMR != remaining[j].MMR {
				return remaining[i].MMR < remaining[j].MMR
			}
			return remaining[i].Name < remaining[j].Name
		})
		group := remaining[:6]
		remaining = remaining[6:]

		maxWait := 0
		for _, player := range group {
			if player.Wait > maxWait {
				maxWait = player.Wait
			}
		}
		limit := maxAverageGap
		for wait := maxWait; wait > widenAfter && wait-widenAfter >= 10; {
			limit += widenStep
			wait -= 10
		}

		game := Game{A: []Player{}, B: []Player{}}
		for position, player := range group {
			if position%2 == 0 {
				game.A = append(game.A, player)
			} else {
				game.B = append(game.B, player)
			}
		}
		if teamGap(game.A) > limit || teamGap(game.B) > limit {
			continue
		}
		game.MaxGap = teamGap(game.A)
		if other := teamGap(game.B); other > game.MaxGap {
			game.MaxGap = other
		}
		game.Violations = teamViolations(game.A) + teamViolations(game.B)
		games = append(games, game)
	}
	return games
}

// TestAgainstReference 随机 200 份队列，与参考实现逐局比对，差异必须为 0。
func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	roles := []string{"tank", "healer", "dps"}
	for trial := 0; trial < 200; trial++ {
		size := rng.Intn(200)
		players := make([]Player, 0, size)
		for index := 0; index < size; index++ {
			players = append(players, Player{
				Name: string(rune('a'+index%26)) + string(rune('A'+index/26)),
				MMR:  rng.Intn(3000),
				Role: roles[rng.Intn(3)],
				Wait: rng.Intn(300),
			})
		}
		maxAverageGap := rng.Intn(200)
		widenAfter := rng.Intn(60)
		widenStep := rng.Intn(50)

		got := Build(players, maxAverageGap, widenAfter, widenStep)
		want := referenceBuild(players, maxAverageGap, widenAfter, widenStep)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("第 %d 份队列与参考实现不一致：got %+v want %+v", trial, got, want)
		}

		// 口径：每局的同队最大分差不超过按等待时长放宽后的上限。
		for _, game := range got {
			maxWait := 0
			for _, player := range append(append([]Player{}, game.A...), game.B...) {
				if player.Wait > maxWait {
					maxWait = player.Wait
				}
			}
			if limit := waitLimit(maxWait, maxAverageGap, widenAfter, widenStep); game.MaxGap > limit {
				t.Fatalf("第 %d 份队列 MaxGap=%d 超过放宽后上限 %d", trial, game.MaxGap, limit)
			}
		}

		// 不变量：一局中 6 个玩家互不重复；matched*6 等于实际出局人数。
		seen := map[string]int{}
		removed := 0
		for _, game := range got {
			for _, player := range append(append([]Player{}, game.A...), game.B...) {
				seen[player.Name]++
				removed++
			}
		}
		for name, count := range seen {
			if count != 1 {
				t.Fatalf("第 %d 份队列玩家 %s 重复出局 %d 次", trial, name, count)
			}
		}
		if _, _, matched := Report(got); matched*6 != removed {
			t.Fatalf("第 %d 份队列 matched*6=%d 与实际出局 %d 不等", trial, matched*6, removed)
		}

		// 同一份队列重复匹配给出同样的对局。
		again := Build(players, maxAverageGap, widenAfter, widenStep)
		if !reflect.DeepEqual(got, again) {
			t.Fatalf("第 %d 份队列重复匹配结果不一致", trial)
		}
	}
}

// TestLargeQueue 10 万排队玩家 1 秒内给出全部可匹配对局。
func TestLargeQueue(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	roles := []string{"tank", "healer", "dps"}
	players := make([]Player, 0, 100000)
	for index := 0; index < 100000; index++ {
		players = append(players, Player{
			Name: string(rune(index)),
			MMR:  rng.Intn(3000),
			Role: roles[rng.Intn(3)],
			Wait: rng.Intn(300),
		})
	}
	start := time.Now()
	games := Build(players, 60, 30, 20)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("10 万玩家匹配耗时 %v，超过 1 秒", elapsed)
	}
	if len(games) == 0 {
		t.Fatal("10 万玩家应该能凑出对局")
	}
}
