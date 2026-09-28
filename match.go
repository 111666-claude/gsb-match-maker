// Package matchmaker 排队匹配：把队列里的玩家凑成 6 人一局、两队各 3 人。
// 缺陷：分差按「两队平均分」比较（不看同队最大分差）；超时放宽是固定值；只检查 A 队的职业。
package matchmaker

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

func average(team []Player) int {
	if len(team) == 0 {
		return 0
	}
	total := 0
	for _, player := range team {
		total += player.MMR
	}
	return total / len(team)
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

// teamViolations 缺陷：只数坦克，完全不管治疗与输出。
func teamViolations(team []Player) int {
	if len(team) != 3 {
		return 1
	}
	tanks := 0
	for _, player := range team {
		if player.Role == "tank" {
			tanks++
		}
	}
	if tanks != 1 {
		return 1
	}
	return 0
}

// Build 缺陷：按 MMR 排序后每 6 人一局、A/B 交替分组，只比两队平均分；
// 职业约束只查 A 队；超时放宽写死成「等待超过 widenAfter 秒就一律放宽 widenStep」。
func Build(players []Player, maxAverageGap int, widenAfter int, widenStep int) []Game {
	// 缺陷：直接按到达顺序分组，不看 MMR 远近
	sorted := append([]Player(nil), players...)
	games := []Game{}
	limit := maxAverageGap
	widened := false
	for index := 0; index+6 <= len(sorted); index += 6 {
		group := sorted[index : index+6]
		if !widened && group[5].Wait > widenAfter {
			limit += widenStep // 缺陷：一次性放宽，不随等待时长递增
			widened = true
		}
		game := Game{A: []Player{}, B: []Player{}}
		for position, player := range group {
			if position%2 == 0 {
				game.A = append(game.A, player)
			} else {
				game.B = append(game.B, player)
			}
		}
		if gap := average(game.A) - average(game.B); gap > limit || -gap > limit {
			continue // 缺陷：用平均分差做门槛
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
