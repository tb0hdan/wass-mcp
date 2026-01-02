package types

import "testing"

func TestConstants(t *testing.T) {
	// Verify MaxDefaultLines is set to expected value
	if MaxDefaultLines != 200 {
		t.Errorf("expected MaxDefaultLines to be 200, got %d", MaxDefaultLines)
	}

	// Verify MaxAllowedLines is set to expected value
	if MaxAllowedLines != 100000 {
		t.Errorf("expected MaxAllowedLines to be 100000, got %d", MaxAllowedLines)
	}

	// Verify MaxAllowedLines > MaxDefaultLines
	if MaxAllowedLines <= MaxDefaultLines {
		t.Error("expected MaxAllowedLines to be greater than MaxDefaultLines")
	}
}

func TestMaxDefaultLines_Reasonable(t *testing.T) {
	// MaxDefaultLines should be reasonable for terminal output
	if MaxDefaultLines < 50 {
		t.Error("MaxDefaultLines seems too small for practical use")
	}
	if MaxDefaultLines > 1000 {
		t.Error("MaxDefaultLines seems too large for default view")
	}
}

func TestMaxAllowedLines_Reasonable(t *testing.T) {
	// MaxAllowedLines should allow large outputs but have a limit
	if MaxAllowedLines < 10000 {
		t.Error("MaxAllowedLines seems too small for full scan outputs")
	}
	if MaxAllowedLines > 1000000 {
		t.Error("MaxAllowedLines seems excessively large")
	}
}
