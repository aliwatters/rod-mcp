package types

import "testing"

func TestLocatorInFrame_BoundsCheck(t *testing.T) {
	// Create a snapshot with one frame (index 0)
	s := &Snapshot{
		frames: nil, // empty frames slice
	}

	// Any frame reference should fail on empty snapshot
	_, err := s.LocatorInFrame("f0e1")
	if err == nil {
		t.Error("LocatorInFrame with frameIndex=0 on empty frames should return error")
	}

	// Test with non-existent frame index that would pass with > but fail with >=
	_, err = s.LocatorInFrame("f1e1")
	if err == nil {
		t.Error("LocatorInFrame with frameIndex=1 on empty frames should return error")
	}
}
