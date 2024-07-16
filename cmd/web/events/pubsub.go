package events

import "sync"

type PubSub[T any] struct {
	// Might be able to use sync.RWMutex here
	mu     sync.Mutex
	subs   map[string][]chan T
	quit   chan struct{}
	closed bool
}

// NewPubSub creates a new PubSub with generic type T
func NewPubSub[T any]() *PubSub[T] {
	return &PubSub[T]{
		subs: make(map[string][]chan T),
		quit: make(chan struct{}),
	}
}

func (b *PubSub[T]) Publish(topic string, msg T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	for _, ch := range b.subs[topic] {
		ch <- msg
	}
}

func (b *PubSub[T]) Subscribe(topic string) chan T {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	ch := make(chan T)
	b.subs[topic] = append(b.subs[topic], ch)
	return ch
}

func (b *PubSub[T]) Unsubscribe(topic string, closing chan T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, ch := range b.subs[topic] {
		if ch == closing {
			b.subs[topic] = append(b.subs[topic][:i], b.subs[topic][i+1:]...)
			close(closing)
		}
	}
}

func (b *PubSub[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.closed = true
	close(b.quit)

	for _, ch := range b.subs {
		for _, sub := range ch {
			close(sub)
		}
	}
}
