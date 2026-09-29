package zone

import "testing"

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
