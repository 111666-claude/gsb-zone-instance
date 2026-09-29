package zone

import (
	"container/heap"
	"math/rand"
	"testing"
	"time"
)

func TestAssignSingleInstance(t *testing.T) {
	manager := New(1, 3)
	id, err := manager.Assign([]string{"a", "b"})
	if err != nil || id != 1 {
		t.Fatalf("两个玩家应该进同一个实例：%d %v", id, err)
	}
	if len(manager.Instances[0].Members) != 2 {
		t.Fatalf("成员数不对：%v", manager.Instances[0].Members)
	}
}

func TestNoInstance(t *testing.T) {
	manager := New(0, 2)
	if _, err := manager.Assign([]string{"a"}); err != ErrFull {
		t.Fatalf("没有实例应该报 ErrFull：%v", err)
	}
}

func TestReleaseMarksClosed(t *testing.T) {
	manager := New(1, 2)
	manager.Release(1)
	if !manager.Instances[0].Closed {
		t.Fatal("回收应该打上标记")
	}
}

func TestAssignWritesMembers(t *testing.T) {
	manager := New(2, 1)
	manager.Assign([]string{"a"})
	if len(manager.Instances[0].Members) != 1 {
		t.Fatalf("成员没写进实例：%v", manager.Instances[0].Members)
	}
}

func TestCapacityBoundary(t *testing.T) {
	manager := New(1, 2)
	if _, err := manager.Assign([]string{"a", "b"}); err != nil {
		t.Fatalf("正好装满应该成功：%v", err)
	}
	if _, err := manager.Assign([]string{"c"}); err != ErrFull {
		t.Fatalf("满了再分应该报 ErrFull：%v", err)
	}
	if len(manager.Instances[0].Members) != 2 {
		t.Fatalf("成员数不能超过容量：%v", manager.Instances[0].Members)
	}

	oversized := New(1, 2)
	if _, err := oversized.Assign([]string{"a", "b", "c"}); err != ErrFull {
		t.Fatalf("超员应该整队拒绝：%v", err)
	}
	if len(oversized.Instances[0].Members) != 0 {
		t.Fatalf("拒绝时不应该写入任何成员：%v", oversized.Instances[0].Members)
	}
}

func TestTeamStaysTogether(t *testing.T) {
	manager := New(2, 1)
	if _, err := manager.Assign([]string{"a", "b"}); err != ErrFull {
		t.Fatalf("两个容量 1 的实例装不下 2 人队伍，应该拒绝：%v", err)
	}
	for _, instance := range manager.Instances {
		if len(instance.Members) != 0 {
			t.Fatalf("拒绝时不应该拆队写入：%v", instance.Members)
		}
	}

	roomy := New(2, 3)
	id, err := roomy.Assign([]string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("容量够应该整队进同一个实例：%v", err)
	}
	if len(roomy.Instances[id-1].Members) != 3 {
		t.Fatalf("整队应该在同一个实例：%v", roomy.Instances[id-1].Members)
	}
}

func TestReleaseNotReusable(t *testing.T) {
	manager := New(2, 1)
	manager.Release(1)
	id, err := manager.Assign([]string{"a"})
	if err != nil || id != 2 {
		t.Fatalf("回收过的实例不能再分配：%d %v", id, err)
	}
	manager.Release(2)
	if _, err := manager.Assign([]string{"b"}); err != ErrFull {
		t.Fatalf("全部回收后应该报 ErrFull：%v", err)
	}
}

// refInstance 是参考实现：整队装箱 + 容量校验，线性扫描选空闲位最多
// （并列取 ID 最小）的未回收实例，装不下整队就拒绝。
type refInstance struct {
	capacity int
	members  int
	closed   bool
}

func refAssign(instances []refInstance, teamSize int) (int, bool) {
	best := -1
	bestFree := -1
	for index := range instances {
		if instances[index].closed {
			continue
		}
		free := instances[index].capacity - instances[index].members
		if free < teamSize {
			continue
		}
		if free > bestFree {
			best, bestFree = index, free
		}
	}
	if best == -1 {
		return 0, false
	}
	instances[best].members += teamSize
	return best + 1, true
}

// newManagerWithCaps 按每个实例各自的容量建管理器（测试辅助，与 New 同逻辑）。
func newManagerWithCaps(capacities []int) *Manager {
	manager := &Manager{
		byID:  make(map[int]*Instance, len(capacities)),
		free:  make(map[int]int, len(capacities)),
		avail: instanceHeap{pos: make(map[int]int, len(capacities))},
	}
	for _, capacity := range capacities {
		manager.NextID++
		instance := &Instance{ID: manager.NextID, Cap: capacity}
		manager.Instances = append(manager.Instances, instance)
		manager.byID[instance.ID] = instance
		manager.free[instance.ID] = capacity
		if capacity > 0 {
			manager.avail.entries = append(manager.avail.entries, heapEntry{id: instance.ID, free: capacity})
		}
	}
	for index, entry := range manager.avail.entries {
		manager.avail.pos[entry.id] = index
	}
	heap.Init(&manager.avail)
	return manager
}

func TestAgainstReference(t *testing.T) {
	const sequences = 200
	for seed := int64(0); seed < sequences; seed++ {
		runSequence(t, seed)
	}
}

func runSequence(t *testing.T, seed int64) {
	t.Helper()
	var firstRun []int
	// 每个序列跑两遍，验证同一批分配请求重复执行结果相同。
	for run := 0; run < 2; run++ {
		rng := rand.New(rand.NewSource(seed))
		instanceCount := 1 + rng.Intn(5)
		capacities := make([]int, instanceCount)
		for index := range capacities {
			capacities[index] = rng.Intn(5)
		}
		manager := newManagerWithCaps(capacities)
		reference := make([]refInstance, instanceCount)
		for index, capacity := range capacities {
			reference[index] = refInstance{capacity: capacity}
		}
		var results []int
		for step := 0; step < 50; step++ {
			if rng.Intn(4) == 0 {
				id := 1 + rng.Intn(instanceCount)
				manager.Release(id)
				reference[id-1].closed = true
				continue
			}
			teamSize := 1 + rng.Intn(4)
			team := make([]string, teamSize)
			for index := range team {
				team[index] = "p"
			}
			id, err := manager.Assign(team)
			got := id
			if err == ErrFull {
				got = 0
			} else if err != nil {
				t.Fatalf("seed=%d 意外错误：%v", seed, err)
			}
			refID, ok := refAssign(reference, teamSize)
			want := 0
			if ok {
				want = refID
			}
			if got != want {
				t.Fatalf("seed=%d step=%d 分配结果不一致：got=%d want=%d", seed, step, got, want)
			}
			results = append(results, got)
			for index := range reference {
				if len(manager.Instances[index].Members) != reference[index].members {
					t.Fatalf("seed=%d step=%d 实例 %d 成员数不一致：got=%d want=%d",
						seed, step, index+1, len(manager.Instances[index].Members), reference[index].members)
				}
				if len(manager.Instances[index].Members) > manager.Instances[index].Cap {
					t.Fatalf("seed=%d step=%d 实例 %d 超员", seed, step, index+1)
				}
			}
		}
		if run == 0 {
			firstRun = results
			continue
		}
		for index := range results {
			if results[index] != firstRun[index] {
				t.Fatalf("seed=%d 重复执行结果不同：%v vs %v", seed, results, firstRun)
			}
		}
	}
}

func TestAssignScales(t *testing.T) {
	const instanceCount = 100000
	manager := New(instanceCount, 1)
	start := time.Now()
	for index := 0; index < instanceCount; index++ {
		if _, err := manager.Assign([]string{"p"}); err != nil {
			t.Fatalf("第 %d 次分配失败：%v", index, err)
		}
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("10 万次分配耗时 %v，超过 1 秒", elapsed)
	}
}
