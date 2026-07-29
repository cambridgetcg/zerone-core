package types

import "testing"

func TestApplyBPSDecay(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		bps    uint64
		expect string
	}{
		{"full power", "1000000", 10000, "1000000"},
		{"half power", "1000000", 5000, "500000"},
		{"twenty percent decay", "1000000", 8000, "800000"},
		{"floor division", "999999", 5000, "499999"},
		{"zero bps", "1000000", 0, "0"},
		{"zero weight", "0", 10000, "0"},
		{"large stake", "50000000000000000000", 8000, "40000000000000000000"},
		{"invalid value", "not-a-number", 10000, "0"},
		{"negative value", "-10", 5000, "0"},
		{"over-scale cannot amplify", "1000000", 12000, "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyBPSDecay(tt.value, tt.bps); got != tt.expect {
				t.Errorf("ApplyBPSDecay(%q, %d) = %q, want %q", tt.value, tt.bps, got, tt.expect)
			}
		})
	}
}
