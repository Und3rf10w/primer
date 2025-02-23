package constants

import (
	"math"
	"testing"
)

func TestCalculateEntropy(t *testing.T) {
	g := &Generator{}
	tests := []struct {
		name    string
		value   uint32
		wantMin float64
		wantMax float64
	}{
		{
			name:    "Zero value",
			value:   0,
			wantMin: 0,
			wantMax: 0.1,
		},
		{
			name:    "All ones",
			value:   0xFFFFFFFF,
			wantMin: 0,
			wantMax: 0.1,
		},
		{
			name:    "Alternating bits",
			value:   0xAAAAAAAA,
			wantMin: 0.9,
			wantMax: 1.1,
		},
		{
			name:    "Random-like value",
			value:   0x1B7DE952,
			wantMin: 1.5,
			wantMax: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.calculateEntropy(tt.value)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateEntropy() = %v, want between %v and %v",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRunBitFrequencyTest(t *testing.T) {
	g := &Generator{}
	tests := []struct {
		name      string
		value     uint32
		wantScore float64
		wantPass  bool
	}{
		{
			name:      "Balanced bits",
			value:     0xAAAAAAAA,
			wantScore: 1.0,
			wantPass:  true,
		},
		{
			name:      "All zeros",
			value:     0,
			wantScore: 0.0,
			wantPass:  false,
		},
		{
			name:      "All ones",
			value:     0xFFFFFFFF,
			wantScore: 0.0,
			wantPass:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.runBitFrequencyTest(tt.value)
			if math.Abs(result.Score-tt.wantScore) > 0.01 {
				t.Errorf("Score = %v, want %v", result.Score, tt.wantScore)
			}
			if result.Passed != tt.wantPass {
				t.Errorf("Passed = %v, want %v", result.Passed, tt.wantPass)
			}
		})
	}
}

func TestRunsTest(t *testing.T) {
	g := &Generator{}
	tests := []struct {
		name     string
		value    uint32
		wantPass bool
	}{
		{
			name:     "Alternating bits",
			value:    0xAAAAAAAA,
			wantPass: true,
		},
		{
			name:     "All zeros",
			value:    0,
			wantPass: false,
		},
		{
			name:     "All ones",
			value:    0xFFFFFFFF,
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.runRunsTest(tt.value)
			if result.Passed != tt.wantPass {
				t.Errorf("Passed = %v, want %v", result.Passed, tt.wantPass)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCalculateEntropy(b *testing.B) {
	g := &Generator{}
	value := uint32(0x1B7DE952)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.calculateEntropy(value)
	}
}

func BenchmarkRunAllStatisticalTests(b *testing.B) {
	g := &Generator{}
	value := uint32(0x1B7DE952)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.runAllStatisticalTests(value)
	}
}
