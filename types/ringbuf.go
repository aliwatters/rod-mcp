package types

// RingBuffer is a fixed-capacity circular buffer that overwrites the oldest
// entry when full. Add and Len are O(1); Slice is O(n).
type RingBuffer[T any] struct {
	items []T
	head  int // index of the oldest item
	count int // number of valid items
}

// NewRingBuffer creates a RingBuffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{items: make([]T, capacity)}
}

// Add appends an item, overwriting the oldest entry if the buffer is full.
func (r *RingBuffer[T]) Add(item T) {
	r.AddWithIndex(item)
}

// AddWithIndex appends an item and returns the internal slice index where it was stored.
// The returned index is stable for the lifetime of the item (until it is overwritten
// by a future Add when the buffer is full). Use UpdateAt to modify the item later.
func (r *RingBuffer[T]) AddWithIndex(item T) int {
	cap := len(r.items)
	var idx int
	if r.count < cap {
		// Buffer not yet full — write at tail position.
		idx = (r.head + r.count) % cap
		r.items[idx] = item
		r.count++
	} else {
		// Buffer full — overwrite oldest entry and advance head.
		idx = r.head
		r.items[idx] = item
		r.head = (r.head + 1) % cap
	}
	return idx
}

// UpdateAt updates the item at the given internal index in-place.
// The index must have been returned by a prior AddWithIndex call.
func (r *RingBuffer[T]) UpdateAt(idx int, fn func(*T)) {
	fn(&r.items[idx])
}

// Len returns the number of items currently stored.
func (r *RingBuffer[T]) Len() int {
	return r.count
}

// Slice returns all stored items in insertion order (oldest first).
func (r *RingBuffer[T]) Slice() []T {
	if r.count == 0 {
		return nil
	}
	cap := len(r.items)
	out := make([]T, r.count)
	for i := range r.count {
		out[i] = r.items[(r.head+i)%cap]
	}
	return out
}

// Clear resets the ring buffer to empty.
func (r *RingBuffer[T]) Clear() {
	r.head = 0
	r.count = 0
}
