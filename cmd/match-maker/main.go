// Command match-maker 生成一支队列并输出匹配汇总。
package main

import (
	"flag"
	"fmt"
	"math/rand"

	matchmaker "example.com/matchmaker"
)

var roles = []string{"tank", "healer", "dps"}

// generate 造一支确定性的队列：MMR 集中在相邻分段，职业按 MMR 分段相关，
// 因此按 MMR 就近凑出的队伍经常缺职业，正好暴露职业校验口径。
func generate(count int, seed int64) []matchmaker.Player {
	rng := rand.New(rand.NewSource(seed))
	players := make([]matchmaker.Player, count)
	for index := range players {
		mmr := 800 + rng.Intn(560)
		players[index] = matchmaker.Player{
			Name: fmt.Sprintf("player-%d", index),
			MMR:  mmr,
			Role: roles[(mmr/100)%len(roles)],
			Wait: rng.Intn(30),
		}
	}
	return players
}

func main() {
	players := flag.Int("players", 60, "排队玩家数")
	seed := flag.Int64("seed", 1, "随机种子")
	maxAverageGap := flag.Int("max-average-gap", 60, "同队最大分差上限")
	widenAfter := flag.Int("widen-after", 30, "等待超过该秒数后开始放宽")
	widenStep := flag.Int("widen-step", 50, "每多等 10 秒放宽的分差")
	flag.Parse()

	games := matchmaker.Build(generate(*players, *seed), *maxAverageGap, *widenAfter, *widenStep)
	maxGap, violations, matched := matchmaker.Report(games)
	fmt.Printf("players=%d matched=%d teams=%d\n", *players, matched, matched*2)
	fmt.Printf("max_gap=%d role_violations=%d\n", maxGap, violations)
}
