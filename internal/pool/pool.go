package pool

import "sync"

// Resettable — ограничение generic-параметра: тип должен иметь метод Reset().
type Resettable interface {
	Reset()
}

// Pool — потокобезопасный пул объектов типа T.
type Pool[T Resettable] struct {
	p sync.Pool
}

// New создаёт пул, в котором новые объекты создаются функцией fn.
func New[T Resettable](fn func() T) *Pool[T] {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any { return fn() },
		},
	}
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put сбрасывает состояние объекта и возвращает его в пул.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}
