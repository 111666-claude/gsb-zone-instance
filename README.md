# zone-instance

副本实例管理：按队伍分配、人数上限与回收，只用 Go 标准库。

```
go test ./...
go run ./cmd/zone-instance --sample capacity
go run ./cmd/zone-instance --sample split
```

## 口径（README 为准）

- **人数上限严格**：实例成员数必须 `<= Cap`。`--sample capacity`（容量 2、队伍 3 人）按口径是
  `err=true members=0`（拒绝整队），现在给 `err=false members=3`。
- **同队必须在同一个实例**：整队一次分配，放不下就拒绝或另开实例，绝不能把队伍拆到两个实例。
  `--sample split`（两个容量 1 的实例、队伍 2 人）按口径是拒绝，现在给 `first=2 second=0`。
- **回收不可复用**：`Release` 之后该实例不能再被分配。
- 不变量：任何实例的成员数 ≤ Cap；同一队成员的实例号相同；同一批分配请求重复执行结果相同。
- 规模：10 万个实例、每秒 10 万次分配 1 秒内完成，内存 O(实例数 + 成员数)；不许每次分配都全表扫描。

## 现在的行为

- `Assign` 逐人找空位：队伍会被拆开。
- 容量判断是 `len(Members) > Cap`：正好超员一个也能进。
- `Assign` 完全不看 `Closed`，回收过的实例照样分。

## 输出

```
instance=0 err=true members=0
first=0 second=0
```

## 目录

```
zone.go             实例管理
cmd/zone-instance   命令行入口
zone_test.go        go test 用例
```
