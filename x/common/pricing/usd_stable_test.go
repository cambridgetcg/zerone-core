package pricing

import "testing"

func TestCalculateUSDStablePrice(t *testing.T) {
	tests := []struct {
		name             string
		targetMicroUSD   uint64
		zrnPriceMicroUSD uint64
		minUZRN          uint64
		maxUZRN          uint64
		want             uint64
	}{
		{
			name:             "basic $1/ZRN",
			targetMicroUSD:   10_000,          // $0.01
			zrnPriceMicroUSD: 1_000_000,       // $1.00
			minUZRN:          1_000,
			maxUZRN:          100_000_000,
			want:             10_000,           // 10000 * 1e6 / 1e6
		},
		{
			name:             "ZRN appreciates 10x ($10/ZRN)",
			targetMicroUSD:   10_000,
			zrnPriceMicroUSD: 10_000_000,      // $10.00
			minUZRN:          100,
			maxUZRN:          100_000_000,
			want:             1_000,            // 10000 * 1e6 / 10e6
		},
		{
			name:             "ZRN depreciates 0.1x ($0.001/ZRN)",
			targetMicroUSD:   10_000,
			zrnPriceMicroUSD: 1_000,           // $0.001
			minUZRN:          1_000,
			maxUZRN:          100_000_000,
			want:             10_000_000,       // 10000 * 1e6 / 1000
		},
		{
			name:             "floor clamp",
			targetMicroUSD:   10_000,
			zrnPriceMicroUSD: 1_000_000_000,   // $1000/ZRN — result would be 10
			minUZRN:          5_000,
			maxUZRN:          100_000_000,
			want:             5_000,            // clamped to min
		},
		{
			name:             "ceiling clamp",
			targetMicroUSD:   10_000,
			zrnPriceMicroUSD: 100,             // $0.0001/ZRN — result would be 100_000_000
			minUZRN:          1_000,
			maxUZRN:          50_000_000,
			want:             50_000_000,       // clamped to max
		},
		{
			name:             "oracle unavailable (price=0)",
			targetMicroUSD:   10_000,
			zrnPriceMicroUSD: 0,
			minUZRN:          1_000,
			maxUZRN:          100_000_000,
			want:             0,
		},
		{
			name:             "large target stress uint64",
			targetMicroUSD:   1_000_000_000,   // $1000 target
			zrnPriceMicroUSD: 1_000_000,       // $1/ZRN
			minUZRN:          1_000,
			maxUZRN:          10_000_000_000_000,
			want:             1_000_000_000,    // 1e9 * 1e6 / 1e6 = 1e9
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateUSDStablePrice(tt.targetMicroUSD, tt.zrnPriceMicroUSD, tt.minUZRN, tt.maxUZRN)
			if got != tt.want {
				t.Errorf("CalculateUSDStablePrice(%d, %d, %d, %d) = %d, want %d",
					tt.targetMicroUSD, tt.zrnPriceMicroUSD, tt.minUZRN, tt.maxUZRN, got, tt.want)
			}
		})
	}
}
