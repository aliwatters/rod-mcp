package types

import (
	"testing"
)

func TestRingBuffer_AddAndSlice(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Add(1)
	rb.Add(2)
	rb.Add(3)

	got := rb.Slice()
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("Slice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Slice[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer[int](3)

	rb.Add(1)
	rb.Add(2)
	rb.Add(3)
	rb.Add(4) // overwrites 1
	rb.Add(5) // overwrites 2

	got := rb.Slice()
	want := []int{3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("Slice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Slice[%d] = %d, want %d", i, got[i], want[i])
		}
	}
	if rb.Len() != 3 {
		t.Errorf("Len = %d, want 3", rb.Len())
	}
}

func TestRingBuffer_AddWithIndex_UpdateAt(t *testing.T) {
	rb := NewRingBuffer[string](4)

	idx := rb.AddWithIndex("hello")
	rb.UpdateAt(idx, func(s *string) {
		*s = "updated"
	})

	got := rb.Slice()
	if got[0] != "updated" {
		t.Errorf("after UpdateAt: got %q, want %q", got[0], "updated")
	}
}

func TestRingBuffer_AddWithIndex_StableAfterMore(t *testing.T) {
	rb := NewRingBuffer[int](5)

	idx0 := rb.AddWithIndex(10)
	rb.Add(20)
	rb.Add(30)

	// idx0 should still point to the original item.
	rb.UpdateAt(idx0, func(v *int) {
		*v = 99
	})

	got := rb.Slice()
	if got[0] != 99 {
		t.Errorf("after UpdateAt on stable index: got %d, want 99", got[0])
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Add(1)
	rb.Add(2)
	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("Len after Clear = %d, want 0", rb.Len())
	}
	got := rb.Slice()
	if got != nil {
		t.Errorf("Slice after Clear = %v, want nil", got)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer[int](3)

	if rb.Len() != 0 {
		t.Errorf("Len = %d, want 0", rb.Len())
	}
	if got := rb.Slice(); got != nil {
		t.Errorf("Slice = %v, want nil", got)
	}
}

func TestRingBuffer_SliceOrder_AfterWrap(t *testing.T) {
	// Verify insertion order is preserved after multiple wraps.
	rb := NewRingBuffer[int](3)
	for i := 1; i <= 10; i++ {
		rb.Add(i)
	}
	// Should contain [8, 9, 10] in order.
	got := rb.Slice()
	want := []int{8, 9, 10}
	if len(got) != len(want) {
		t.Fatalf("Slice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Slice[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_ClearThenReuse(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Add(1)
	rb.Add(2)
	rb.Add(3)
	rb.Clear()

	rb.Add(10)
	rb.Add(20)

	got := rb.Slice()
	want := []int{10, 20}
	if len(got) != len(want) {
		t.Fatalf("Slice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Slice[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}
