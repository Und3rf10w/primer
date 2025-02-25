package constants

import (
	"fmt"
	"math"
	"math/bits"
	"sync"
)

// Statistical test thresholds
const (
	// P-value thresholds
	minPValue = 0.01
	maxPValue = 0.99

	// Entropy thresholds
	minEntropyScore = 1.0
	maxEntropyScore = 2.5

	// Frequency test thresholds
	maxBitFrequencyDeviation = 0.15

	// Runs test thresholds
	minRunsZScore = -3.0
	maxRunsZScore = 3.0

	// Serial test thresholds
	maxSerialCorrelation = 0.5
)

// calculateEntropy calculates Shannon entropy of bit distribution
func (g *Generator) calculateEntropy(value uint32) float64 {
	// Count frequency of each bit
	counts := make(map[bool]int)
	for i := 0; i < 32; i++ {
		bit := (value & (1 << uint(i))) != 0
		counts[bit]++
	}

	// Calculate Shannon entropy
	entropy := 0.0
	for _, count := range counts {
		p := float64(count) / 32.0
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// runBitFrequencyTest performs the frequency (monobit) test
func (g *Generator) runBitFrequencyTest(value uint32) StatisticalTest {
	ones := 0
	for i := 0; i < 32; i++ {
		if value&(1<<uint(i)) != 0 {
			ones++
		}
	}

	proportion := float64(ones) / 32.0
	deviation := math.Abs(proportion - 0.5)

	return StatisticalTest{
		Name:    "Bit Frequency Test",
		Score:   1.0 - (deviation * 2), // Normalize to 0-1 scale
		Passed:  deviation <= maxBitFrequencyDeviation,
		Details: fmt.Sprintf("Proportion of ones: %.4f (deviation: %.4f)", proportion, deviation),
	}
}

// runRunsTest performs the runs test for randomness
func (g *Generator) runRunsTest(value uint32) StatisticalTest {
	var runs int
	var currentRun bool = value&1 != 0

	// Count runs
	for i := 1; i < 32; i++ {
		bit := value&(1<<uint(i)) != 0
		if bit != currentRun {
			runs++
			currentRun = bit
		}
	}
	runs++ // Count the last run

	// Calculate expected runs and variance
	n := 32
	n1 := bits.OnesCount32(value)
	n0 := n - n1
	expectedRuns := 1.0 + 2.0*float64(n0)*float64(n1)/float64(n)
	variance := (expectedRuns - 1.0) * (expectedRuns - 2.0) / float64(n-1)

	// Calculate Z-score
	zScore := (float64(runs) - expectedRuns) / math.Sqrt(variance)

	return StatisticalTest{
		Name:    "Runs Test",
		Score:   1.0 - math.Abs(zScore/6.0), // Normalize to 0-1 scale
		Passed:  zScore >= minRunsZScore && zScore <= maxRunsZScore,
		Details: fmt.Sprintf("Z-score: %.4f (runs: %d, expected: %.2f)", zScore, runs, expectedRuns),
	}
}

// runSerialTest performs the serial test for 2-bit patterns
func (g *Generator) runSerialTest(value uint32) StatisticalTest {
	// Count frequencies of 2-bit patterns
	patterns := make([]int, 4)
	for i := 0; i < 31; i++ {
		pattern := (value >> uint(i)) & 0x3
		patterns[pattern]++
	}

	// Calculate chi-square statistic
	expected := float64(31) / 4.0
	chiSquare := 0.0
	for _, count := range patterns {
		chiSquare += math.Pow(float64(count)-expected, 2) / expected
	}

	// Calculate p-value
	pValue := 1.0 - math.Exp(-chiSquare/2.0)

	return StatisticalTest{
		Name:    "Serial Test",
		Score:   1.0 - math.Abs(pValue-0.5)*2, // Normalize to 0-1 scale
		Passed:  pValue >= minPValue && pValue <= maxPValue,
		Details: fmt.Sprintf("Chi-square: %.4f (p-value: %.4f)", chiSquare, pValue),
	}
}

// runAutoCorrelationTest performs autocorrelation test
func (g *Generator) runAutoCorrelationTest(value uint32) StatisticalTest {
	maxCorrelation := 0.0

	// Test different shift values
	for shift := 1; shift < 16; shift++ {
		correlation := g.calculateAutocorrelation(value, shift)
		maxCorrelation = math.Max(maxCorrelation, math.Abs(correlation))
	}

	return StatisticalTest{
		Name:    "Autocorrelation Test",
		Score:   1.0 - maxCorrelation,
		Passed:  maxCorrelation <= maxSerialCorrelation,
		Details: fmt.Sprintf("Maximum correlation: %.4f", maxCorrelation),
	}
}

// calculateAutocorrelation calculates autocorrelation for a given shift
func (g *Generator) calculateAutocorrelation(value uint32, shift int) float64 {
	matches := 0
	total := 32 - shift

	for i := 0; i < total; i++ {
		bit1 := (value >> uint(i)) & 1
		bit2 := (value >> uint(i+shift)) & 1
		if bit1 == bit2 {
			matches++
		}
	}

	return math.Abs(float64(matches)/float64(total)-0.5) * 2
}

// runLinearComplexityTest estimates the linear complexity
func (g *Generator) runLinearComplexityTest(value uint32) StatisticalTest {
	complexity := g.calculateLinearComplexity(value)
	expectedComplexity := 16.0 // Half of 32 bits

	deviation := math.Abs(float64(complexity) - expectedComplexity)
	normalizedScore := 1.0 - (deviation / expectedComplexity)

	return StatisticalTest{
		Name:    "Linear Complexity Test",
		Score:   normalizedScore,
		Passed:  complexity >= 12, // At least 12 bits of complexity
		Details: fmt.Sprintf("Linear complexity: %d bits", complexity),
	}
}

// calculateLinearComplexity implements the Berlekamp-Massey algorithm
func (g *Generator) calculateLinearComplexity(value uint32) int {
	// Convert to bit sequence
	sequence := make([]int, 32)
	for i := 0; i < 32; i++ {
		if value&(1<<uint(i)) != 0 {
			sequence[i] = 1
		}
	}

	// Berlekamp-Massey algorithm
	L := 0
	m := -1
	d := 0
	C := make([]int, 32)
	B := make([]int, 32)
	C[0] = 1
	B[0] = 1

	for n := 0; n < 32; n++ {
		d = sequence[n]
		for i := 1; i <= L; i++ {
			d ^= C[i] & sequence[n-i]
		}
		if d == 1 {
			T := make([]int, 32)
			copy(T, C)
			for i := 0; i < 32-n+m; i++ {
				C[n-m+i] ^= B[i]
			}
			if L <= n/2 {
				L = n + 1 - L
				m = n
				copy(B, T)
			}
		}
	}

	return L
}

// runAllStatisticalTests runs all statistical tests on a value
func (g *Generator) runAllStatisticalTests(value uint32) []StatisticalTest {
	var mu sync.Mutex
	var tests []StatisticalTest

	var wg sync.WaitGroup
	testFuncs := []struct {
		name string
		fn   func(uint32) StatisticalTest
	}{
		{"BitFrequency", g.runBitFrequencyTest},
		{"Runs", g.runRunsTest},
		{"Serial", g.runSerialTest},
		{"AutoCorrelation", g.runAutoCorrelationTest},
		{"LinearComplexity", g.runLinearComplexityTest},
	}

	for _, tf := range testFuncs {
		wg.Add(1)
		go func(name string, testFn func(uint32) StatisticalTest) {
			defer wg.Done()
			result := testFn(value)
			mu.Lock()
			tests = append(tests, result)
			mu.Unlock()
		}(tf.name, tf.fn)
	}

	wg.Wait()
	return tests
}

// aggregateTestResults combines all test results into a single score
func (g *Generator) aggregateTestResults(tests []StatisticalTest) float64 {
	if len(tests) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, test := range tests {
		totalScore += test.Score
	}

	return totalScore / float64(len(tests))
}
