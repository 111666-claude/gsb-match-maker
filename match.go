// Package matchmaker 排队匹配：把队列里的玩家凑成 6 人一局、两队各 3 人。
// 门槛按「同队最大分差」判断；职业规则两队都查；超时放宽随等待时长逐级递增。
package matchmaker

import "sort"

// Player 是排队中的玩家。
type Player struct {
	Name string
	MMR  int
	Role string // tank / healer / dps
	Wait int    // 已等待秒数
}

// Game 是一局匹配结果。
type Game struct {
	A          []Player
	B          []Player
	MaxGap     int
	Violations int
}

func teamGap(team []Player) int {
	if len(team) == 0 {
		return 0
	}
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

// teamViolations 检查一队是否恰好 1 坦克、1 治疗、1 输出，不合格计 1。
func teamViolations(team []Player) int {
	if len(team) != 3 {
		return 1
	}
	counts := map[string]int{}
	for _, player := range team {
		counts[player.Role]++
	}
	if counts["tank"] != 1 || counts["healer"] != 1 || counts["dps"] != 1 {
		return 1
	}
	return 0
}

// waitLimit 按等待时长计算放宽后的分差上限：
// 等待超过 widenAfter 秒后，每多等 10 秒把上限放宽 widenStep，逐级递增。
func waitLimit(maxWait int, maxAverageGap int, widenAfter int, widenStep int) int {
	if maxWait <= widenAfter {
		return maxAverageGap
	}
	return maxAverageGap + widenStep*((maxWait-widenAfter)/10)
}

// Build 按 MMR 与名字稳定排序后每 6 人一局、A/B 交替分组；
// 两队的同队最大分差都必须不超过按等待时长放宽后的上限；
// 职业规则对 A、B 两队都校验，不合格的队伍计入 Violations。
func Build(players []Player, maxAverageGap int, widenAfter int, widenStep int) []Game {
	sorted := append([]Player(nil), players...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].MMR != sorted[j].MMR {
			return sorted[i].MMR < sorted[j].MMR
		}
		return sorted[i].Name < sorted[j].Name
	})
	games := []Game{}
	for index := 0; index+6 <= len(sorted); index += 6 {
		group := sorted[index : index+6]
		maxWait := 0
		for _, player := range group {
			if player.Wait > maxWait {
				maxWait = player.Wait
			}
		}
		limit := waitLimit(maxWait, maxAverageGap, widenAfter, widenStep)
		game := Game{A: []Player{}, B: []Player{}}
		for position, player := range group {
			if position%2 == 0 {
				game.A = append(game.A, player)
			} else {
				game.B = append(game.B, player)
			}
		}
		if teamGap(game.A) > limit || teamGap(game.B) > limit {
			continue // 同队最大分差超过（放宽后的）上限，本组不出局
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

// Report 汇总验收指标。
func Report(games []Game) (maxGap int, violations int, matched int) {
	for _, game := range games {
		if game.MaxGap > maxGap {
			maxGap = game.MaxGap
		}
		violations += game.Violations
	}
	return maxGap, violations, len(games)
}
