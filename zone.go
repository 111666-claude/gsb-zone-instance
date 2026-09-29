// Package zone 是副本实例管理：按队伍分配实例、限制人数、回收实例。
// 口径：整队一次分配（绝不拆队）、成员数严格 <= Cap、回收后的实例不再分配。
package zone

import (
	"container/heap"
	"errors"
)

// ErrFull 表示没有可用实例。
var ErrFull = errors.New("zone: 没有可用实例")

// Instance 是一个副本实例。
type Instance struct {
	ID      int
	Cap     int
	Members []string
	Closed  bool
}

// heapEntry 是可分配实例在堆中的条目，按空闲位降序、ID 升序取顶。
type heapEntry struct {
	id   int
	free int
}

// instanceHeap 是索引堆：每个实例最多出现一次，支持 O(log n) 取顶、更新与删除。
type instanceHeap struct {
	entries []heapEntry
	pos     map[int]int
}

func (h *instanceHeap) Len() int { return len(h.entries) }

func (h *instanceHeap) Less(i, j int) bool {
	a, b := h.entries[i], h.entries[j]
	if a.free != b.free {
		return a.free > b.free
	}
	return a.id < b.id
}

func (h *instanceHeap) Swap(i, j int) {
	h.entries[i], h.entries[j] = h.entries[j], h.entries[i]
	h.pos[h.entries[i].id] = i
	h.pos[h.entries[j].id] = j
}

func (h *instanceHeap) Push(x any) {
	entry := x.(heapEntry)
	h.pos[entry.id] = len(h.entries)
	h.entries = append(h.entries, entry)
}

func (h *instanceHeap) Pop() any {
	old := h.entries
	n := len(old)
	entry := old[n-1]
	h.entries = old[:n-1]
	delete(h.pos, entry.id)
	return entry
}

// Manager 管理实例。
type Manager struct {
	Instances []*Instance
	NextID    int
	byID      map[int]*Instance
	free      map[int]int
	avail     instanceHeap
}

// New 建管理器，预置 count 个容量 cap 的实例。
func New(count, capacity int) *Manager {
	manager := &Manager{
		byID:  make(map[int]*Instance, count),
		free:  make(map[int]int, count),
		avail: instanceHeap{pos: make(map[int]int, count)},
	}
	for index := 0; index < count; index++ {
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

// Assign 给整支队伍分配一个实例：所有成员进同一个实例，且成员数严格 <= Cap。
// 没有任何实例装得下整队时返回 ErrFull，不写任何成员。
func (m *Manager) Assign(team []string) (int, error) {
	if len(team) == 0 || m.avail.Len() == 0 {
		return 0, ErrFull
	}
	top := m.avail.entries[0]
	if top.free < len(team) {
		return 0, ErrFull
	}
	instance := m.byID[top.id]
	instance.Members = append(instance.Members, team...)
	remaining := top.free - len(team)
	m.free[top.id] = remaining
	if remaining == 0 {
		heap.Pop(&m.avail)
	} else {
		m.avail.entries[0].free = remaining
		heap.Fix(&m.avail, 0)
	}
	return instance.ID, nil
}

// Release 回收实例：打上 Closed 标记并移出可分配堆，之后不再被分配。
func (m *Manager) Release(id int) {
	instance, ok := m.byID[id]
	if !ok || instance.Closed {
		return
	}
	instance.Closed = true
	if index, ok := m.avail.pos[id]; ok {
		heap.Remove(&m.avail, index)
	}
}
