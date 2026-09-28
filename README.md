# match-maker

排队匹配：把队列里的玩家凑成 6 人一局、两队各 3 人。只用 Go 标准库。

```
go test ./...
go run ./cmd/match-maker --players 60 --seed 3
```

## 口径（README 为准）

- **同队分差才是硬门槛**：一局里任意一队的最大分差（`max(MMR) - min(MMR)`）必须 ≤ `--max-average-gap`。
  `--players 60 --seed 3` 现在按「两队平均分」比较，给 `max_gap=350`，按口径要 ≤ 60（或按放宽后的值）。
- **职业规则要查全**：每队必须恰好 1 坦克、1 治疗、1 输出。现在 `teamViolations` 只数坦克，
  `--players 60 --seed 3` 给 `role_violations=9`（而且只反映「缺坦克」），按口径 18 个队全都缺治疗与输出，
  必须报 18 队。
- **超时放宽要随等待时长递增**：等待超过 `widenAfter` 秒后，每多等 10 秒把分差上限放宽 `widenStep`，
  而不是一次性放宽一次。等待越久还没匹配上的玩家必须出现在下一轮更宽的池子里。
- 不变量：一局里的 6 个玩家互不重复；`matched * 6` 等于实际出局人数；
  同一份队列重复匹配给出同样的对局（按 MMR 与名字稳定排序）。
- 规模：10 万排队玩家 1 秒内给出全部可匹配对局，内存 O(queue)；不许每局重扫整条队列。

## 现在的行为

- `Build` 用 `average(A) - average(B)` 做门槛，两队平均分接近但队内差距可以很大。
- `teamViolations` 只对 A 队调用，B 队职业重复照样出局。
- 放宽逻辑是 `limit += widenStep` 只执行一次，等待 5 分钟与等待 40 秒没区别。

## 计划新增的能力

```
--players N / --seed N
--max-average-gap N / --widen-after N / --widen-step N
```

## 输出

```
players=60 matched=9 teams=18
max_gap=334 role_violations=9
```

## 目录

```
match.go              排队与组队
cmd/match-maker       命令行入口
match_test.go         go test 用例
```
