package varying

import (
	"testing"
)

func TestInterpolated(t *testing.T) {
	x := NewInterpolated([]Point{
		{Time: 0.0, Value: 400.0},
		{Time: 1.0, Value: 500.0},
	})

	if x.Value() != 400.0 {
		t.Errorf("expected Value() = 400 at start, got %v", x.Value())
	}

	x.Advance(0.5)

	if x.Value() <= 449.0 || x.Value() >= 451.0 {
		t.Errorf("expected Value() in (449,451) in middle, got %v", x.Value())
	}

	x.Advance(0.5)

	if x.Value() <= 499.0 {
		t.Errorf("expected Value() ~= 500 at end, got %v", x.Value())
	}

	x.Advance(1.0)

	if x.Value() != 500.0 {
		t.Errorf("expected Value() = 500 at end, got %v", x.Value())
	}
}
