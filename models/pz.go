package models

import "sync"

// 定义数组队列的数据结构
type PzArrayQueue struct {
	Array []string
	Size  int
	Lock  sync.Mutex
}

// 1. 入队操作
func (q *PzArrayQueue) ArrayAdd(v string) {
	q.Lock.Lock()
	defer q.Lock.Unlock()

	q.Array = append(q.Array, v)
	q.Size++
}

// 2. 出队操作
func (q *PzArrayQueue) ArrayRemove() string {
	q.Lock.Lock()
	defer q.Lock.Unlock()

	if q.Size == 0 {
		return ""
	}

	v := q.Array[0]
	q.Array = q.Array[1:]
	q.Size--
	return v
}
