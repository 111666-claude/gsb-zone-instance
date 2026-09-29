package zone

import (
	"fmt"
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

func TestCapacityExactFit(t *testing.T) {
	manager := New(1, 2)
	id, err := manager.Assign([]string{"a", "b"})
	if err != nil || id != 1 {
		t.Fatalf("正好坐满应该成功：%d %v", id, err)
	}
	if len(manager.Instances[0].Members) != 2 {
		t.Fatalf("成员数不对：%v", manager.Instances[0].Members)
	}
	if _, err := manager.Assign([]string{"c"}); err != ErrFull {
		t.Fatalf("坐满后再分应该报 ErrFull：%v", err)
	}
}

func TestCapacityStrictReject(t *testing.T) {
	manager := New(1, 2)
	id, err := manager.Assign([]string{"a", "b", "c"})
	if err != ErrFull || id != 0 {
		t.Fatalf("超员应该整队拒绝：%d %v", id, err)
	}
	if len(manager.Instances[0].Members) != 0 {
		t.Fatalf("拒绝后实例里不该有成员：%v", manager.Instances[0].Members)
	}
}

func TestTeamNeverSplit(t *testing.T) {
	manager := New(2, 1)
	if _, err := manager.Assign([]string{"a", "b"}); err != ErrFull {
		t.Fatalf("两个容量 1 的实例放不下 2 人队，应该拒绝：%v", err)
	}
	for index, instance := range manager.Instances {
		if len(instance.Members) != 0 {
			t.Fatalf("队伍绝不能被拆开：实例 %d 有成员 %v", index, instance.Members)
		}
	}
}

func TestTeamStaysTogether(t *testing.T) {
	manager := New(2, 3)
	id, err := manager.Assign([]string{"a", "b"})
	if err != nil {
		t.Fatalf("应该能放下：%v", err)
	}
	id2, err := manager.Assign([]string{"c"})
	if err != nil {
		t.Fatalf("应该能放下：%v", err)
	}
	if id != id2 {
		t.Fatalf("两批都应该进第一个还有位置的实例：%d %d", id, id2)
	}
	if len(manager.Instances[0].Members) != 3 {
		t.Fatalf("成员应该都在实例 1：%v", manager.Instances[0].Members)
	}
}

func TestReleaseNotReusable(t *testing.T) {
	manager := New(1, 2)
	manager.Release(1)
	if _, err := manager.Assign([]string{"a"}); err != ErrFull {
		t.Fatalf("回收后的实例不能再被分配：%v", err)
	}
	if len(manager.Instances[0].Members) != 0 {
		t.Fatalf("回收后的实例不该有成员：%v", manager.Instances[0].Members)
	}
}

func TestReleaseSkipsToNext(t *testing.T) {
	manager := New(2, 2)
	manager.Release(1)
	id, err := manager.Assign([]string{"a"})
	if err != nil || id != 2 {
		t.Fatalf("应该跳过回收的实例分到实例 2：%d %v", id, err)
	}
}

func TestDeterministic(t *testing.T) {
	run := func() []int {
		manager := New(4, 3)
		results := []int{}
		teams := [][]string{{"a", "b"}, {"c"}, {"d", "e", "f"}, {"g"}, {"h", "i"}}
		for _, team := range teams {
			id, err := manager.Assign(team)
			if err != nil {
				results = append(results, -1)
			} else {
				results = append(results, id)
			}
		}
		manager.Release(1)
		id, err := manager.Assign([]string{"j"})
		if err != nil {
			results = append(results, -1)
		} else {
			results = append(results, id)
		}
		return results
	}
	first, second := run(), run()
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("同一批请求重复执行结果必须相同：%v vs %v", first, second)
		}
	}
}

// refManager 是参考实现：整队装箱 + 容量校验 + 回收不可复用，逐实例扫描。
type refManager struct {
	caps    []int
	members []int
	closed  []bool
}

func newRefManager(count, capacity int) *refManager {
	ref := &refManager{caps: make([]int, count), members: make([]int, count), closed: make([]bool, count)}
	for index := range ref.caps {
		ref.caps[index] = capacity
	}
	return ref
}

func (r *refManager) assign(teamSize int) int {
	for index := range r.caps {
		if r.closed[index] || r.members[index]+teamSize > r.caps[index] {
			continue
		}
		r.members[index] += teamSize
		return index + 1
	}
	return 0
}

func (r *refManager) release(id int) {
	if id >= 1 && id <= len(r.caps) {
		r.closed[id-1] = true
	}
}

func TestDifferentialRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	for trial := 0; trial < 200; trial++ {
		count := rng.Intn(8)
		capacity := 1 + rng.Intn(5)
		manager := New(count, capacity)
		ref := newRefManager(count, capacity)
		ops := 1 + rng.Intn(30)
		for op := 0; op < ops; op++ {
			switch rng.Intn(3) {
			case 0, 1: // 分配
				teamSize := 1 + rng.Intn(6)
				team := make([]string, teamSize)
				for index := range team {
					team[index] = fmt.Sprintf("t%dp%d", trial, index)
				}
				id, err := manager.Assign(team)
				refID := ref.assign(teamSize)
				if (err == ErrFull) != (refID == 0) {
					t.Fatalf("第 %d 组第 %d 步：拒绝口径不一致 err=%v ref=%d", trial, op, err, refID)
				}
				if err == nil && id != refID {
					t.Fatalf("第 %d 组第 %d 步：实例号不一致 got=%d want=%d", trial, op, id, refID)
				}
			case 2: // 回收
				id := 1 + rng.Intn(count+1)
				manager.Release(id)
				ref.release(id)
			}
		}
		for index, instance := range manager.Instances {
			if len(instance.Members) != ref.members[index] {
				t.Fatalf("第 %d 组：实例 %d 成员数不一致 got=%d want=%d",
					trial, instance.ID, len(instance.Members), ref.members[index])
			}
			if len(instance.Members) > instance.Cap {
				t.Fatalf("第 %d 组：实例 %d 超员 %d > %d", trial, instance.ID, len(instance.Members), instance.Cap)
			}
			if instance.Closed != ref.closed[index] {
				t.Fatalf("第 %d 组：实例 %d 回收标记不一致", trial, instance.ID)
			}
		}
	}
}

func TestScale(t *testing.T) {
	const instances = 100000
	const assigns = 100000
	manager := New(instances, 4)
	start := time.Now()
	for index := 0; index < assigns; index++ {
		team := []string{fmt.Sprintf("p%d", index)}
		if _, err := manager.Assign(team); err != nil {
			t.Fatalf("第 %d 次分配失败：%v", index, err)
		}
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("10 万次分配应在 1 秒内完成，实际 %v", elapsed)
	}
}
