// Package zone 是副本实例管理：按队伍分配实例、限制人数、回收实例。
// 缺陷：容量判断用 >（差一）、按人分配（同队可能被拆开）、回收后实例仍能被分配。
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
}

// New 建管理器，预置 count 个容量 cap 的实例。
func New(count, capacity int) *Manager {
	manager := &Manager{}
	for index := 0; index < count; index++ {
		manager.NextID++
		manager.Instances = append(manager.Instances, &Instance{ID: manager.NextID, Cap: capacity})
	}
	return manager
}

// Assign 给队伍分配实例。缺陷：逐人找位置（同队会被拆开）、容量判断差一、不看 Closed。
func (m *Manager) Assign(team []string) (int, error) {
	assigned := 0
	for _, member := range team {
		for _, instance := range m.Instances {
			if len(instance.Members) > instance.Cap {
				continue
			}
			instance.Members = append(instance.Members, member)
			assigned = instance.ID
			break
		}
	}
	if assigned == 0 {
		return 0, ErrFull
	}
	return assigned, nil
}

// Release 回收实例。缺陷：只打标记，分配时不看这个标记。
func (m *Manager) Release(id int) {
	for _, instance := range m.Instances {
		if instance.ID == id {
			instance.Closed = true
		}
	}
}
