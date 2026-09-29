// Package zone 是副本实例管理：按队伍分配实例、限制人数、回收实例。
// 口径：人数上限严格（成员数 <= Cap）、整队一次性分配（放不下就拒绝）、
// 回收后的实例不再参与分配。分配策略是按实例号首次适配（first-fit），
// 用线段树维护剩余容量的区间最大值，单次分配 O(log N)，不做全表扫描。
package zone

import "errors"

// ErrFull 表示没有可用实例。
var ErrFull = errors.New("zone: 没有可用实例")

// Instance 是一个副本实例。
type Instance struct {
	ID      int
	Cap     int
	Members []string
	Closed  bool
}

// Manager 管理实例。
type Manager struct {
	Instances []*Instance
	NextID    int

	size int         // 线段树叶子的槽位数（>= 实例数的最小 2 的幂）
	tree []int       // 线段树：每个区间剩余容量的最大值，回收的实例记 -1
	byID map[int]int // 实例 ID -> Instances 下标
}

// New 建管理器，预置 count 个容量 cap 的实例。
func New(count, capacity int) *Manager {
	manager := &Manager{byID: make(map[int]int, count)}
	for index := 0; index < count; index++ {
		manager.NextID++
		manager.Instances = append(manager.Instances, &Instance{ID: manager.NextID, Cap: capacity})
		manager.byID[manager.NextID] = index
	}
	manager.size = 1
	for manager.size < count {
		manager.size *= 2
	}
	manager.tree = make([]int, 2*manager.size)
	for index := range manager.Instances {
		manager.tree[manager.size+index] = capacity
	}
	for node := manager.size - 1; node >= 1; node-- {
		manager.tree[node] = max(manager.tree[2*node], manager.tree[2*node+1])
	}
	return manager
}

// Assign 给队伍分配实例：整队一次性放进同一个实例，任何实例放不下就
// 整体拒绝（ErrFull），绝不拆队，也不会选中已回收的实例。
func (m *Manager) Assign(team []string) (int, error) {
	if len(team) == 0 {
		return 0, ErrFull
	}
	index := m.findFirst(len(team))
	if index < 0 {
		return 0, ErrFull
	}
	instance := m.Instances[index]
	instance.Members = append(instance.Members, team...)
	m.update(index, instance.Cap-len(instance.Members))
	return instance.ID, nil
}

// Release 回收实例：打上 Closed 标记，并把它的剩余容量置为 -1，
// 之后的分配不会再选中它。
func (m *Manager) Release(id int) {
	index, ok := m.byID[id]
	if !ok {
		return
	}
	m.Instances[index].Closed = true
	m.update(index, -1)
}

// findFirst 返回剩余容量 >= need 的最小实例下标，没有则返回 -1。
func (m *Manager) findFirst(need int) int {
	if m.tree[1] < need {
		return -1
	}
	node := 1
	for node < m.size {
		node *= 2
		if m.tree[node] < need {
			node++ // 左子树放不下，去右子树
		}
	}
	return node - m.size
}

// update 把下标 index 的剩余容量改为 free，并沿路径刷新区间最大值。
func (m *Manager) update(index, free int) {
	node := m.size + index
	m.tree[node] = free
	for node /= 2; node >= 1; node /= 2 {
		m.tree[node] = max(m.tree[2*node], m.tree[2*node+1])
	}
}
