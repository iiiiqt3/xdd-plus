package models

import (
	"sync"
)

type tokenBucket struct {
	limitRate int
	tokenChan chan struct{}
	cap       int
	muLock    *sync.Mutex
	stop      bool
}

func NewTokenBucket(limitRate, cap int) *tokenBucket {
	if cap < 1 {
		panic("token bucket cap must be large 1")
	}
	return &tokenBucket{
		tokenChan: make(chan struct{}, cap),
		limitRate: limitRate,
		muLock:    new(sync.Mutex),
		cap:       cap,
	}
}

func (b *tokenBucket) Start() {
	go b.produce()
}

func (b *tokenBucket) produce() {
	b.muLock.Lock()
	if b.stop {
		close(b.tokenChan)
		b.muLock.Unlock()
		return
	}
	b.tokenChan <- struct{}{}
	b.muLock.Unlock()
}

func (b *tokenBucket) Consume() {
	<-b.tokenChan
}

func (b *tokenBucket) Stop() {
	b.muLock.Lock()
	defer b.muLock.Unlock()
	b.stop = true
}
