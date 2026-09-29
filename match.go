// Package matchmaker 排队匹配：把队列里的玩家凑成 6 人一局、两队各 3 人。
// 口径：同队最大分差是硬门槛；两队职业都校验；超时放宽随等待时长逐级递增。
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

// teamViolations 每队必须恰好 1 坦克、1 治疗、1 输出，违反记 1。
func teamViolations(team []Player) int {
	if len(team) != 3 {
		return 1
	}
	tanks, healers, damage := 0, 0, 0
	for _, player := range team {
		switch player.Role {
		case "tank":
			tanks++
		case "healer":
			healers++
		case "dps":
			damage++
		}
	}
	if tanks != 1 || healers != 1 || damage != 1 {
		return 1
	}
	return 0
}

// widenLimit 等待超过 widenAfter 秒后，每多等 10 秒把分差上限放宽 widenStep，
// 随等待时长递增，不是一次性放宽。
func widenLimit(maxAverageGap, widenAfter, widenStep, wait int) int {
	if wait <= widenAfter {
		return maxAverageGap
	}
	return maxAverageGap + widenStep*((wait-widenAfter)/10)
}

// Build 按 MMR（同分按名字）稳定排序后每 6 人一局、A/B 交替分组；
// 门槛是同队最大分差（max(MMR)-min(MMR)），两队都要过；
// 两队职业都计入 Violations；放宽上限随组内最长等待逐级递增。
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
		wait := 0
		for _, player := range group {
			if player.Wait > wait {
				wait = player.Wait
			}
		}
		limit := widenLimit(maxAverageGap, widenAfter, widenStep, wait)
		game := Game{A: []Player{}, B: []Player{}}
		for position, player := range group {
			if position%2 == 0 {
				game.A = append(game.A, player)
			} else {
				game.B = append(game.B, player)
			}
		}
		if teamGap(game.A) > limit || teamGap(game.B) > limit {
			continue // 同队最大分差超过当前上限，整组留队列
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
