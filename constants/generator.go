package constants

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/bits"
	"os"
	"sort"
	"sync"
	"time"
)

type Generator struct {
	config Config
	logger *Logger
	ctx    context.Context
	cancel context.CancelFunc
}

func NewGenerator(config Config) *Generator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Generator{
		config: config,
		logger: NewLogger(config.DetailedLogging),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (g *Generator) Cleanup() {
	if g.cancel != nil {
		g.cancel()
	}
}

func (g *Generator) Generate() (*GenerationResult, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Validate configuration
	if err := ValidateConfig(&g.config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize channels
	workerCount := g.config.ParallelWorkers
	batchSize := g.config.NumCandidates / workerCount
	bufferSize := workerCount * 2

	candidateChan := make(chan ConstantCandidate, bufferSize)
	errorChan := make(chan error, bufferSize)

	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			g.worker(workerID, candidateChan, errorChan, batchSize)
		}(i)
	}

	// Collect results
	var candidates []ConstantCandidate
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	// Handle completion or timeout
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("generation timed out: %v", ctx.Err())
	case <-done:
		// Process any remaining errors
		close(errorChan)
		for err := range errorChan {
			if err != nil {
				return nil, fmt.Errorf("worker error: %v", err)
			}
		}

		// Collect remaining candidates
		close(candidateChan)
		for candidate := range candidateChan {
			candidates = append(candidates, candidate)
		}
	}

	// Validate we have enough candidates
	if len(candidates) < 2 {
		return nil, fmt.Errorf("insufficient valid candidates generated: got %d, need at least 2", len(candidates))
	}

	// Process results and create final output
	result, err := g.processResults(candidates, start)
	if err != nil {
		return nil, fmt.Errorf("processing results: %w", err)
	}

	// Save results if configured
	if g.config.ResultsFile != "" {
		if err := g.saveResults(result); err != nil {
			g.logger.Error("Failed to save results:", err)
		}
	}

	return result, nil
}

func (g *Generator) processResults(candidates []ConstantCandidate, startTime time.Time) (*GenerationResult, error) {
	// Select best constants
	bestP, bestQ := g.selectBestConstants(candidates)

	// Validate selected constants
	if err := g.validateSelectedConstants(bestP, bestQ); err != nil {
		return nil, fmt.Errorf("invalid selected constants: %w", err)
	}

	// Create result
	result := &GenerationResult{
		SelectedP:       bestP,
		SelectedQ:       bestQ,
		TotalCandidates: len(candidates),
		Duration:        time.Since(startTime),
		StartTime:       startTime,
		EndTime:         time.Now(),
		Config:          g.config,
	}

	// Run final validation tests
	if err := g.runFinalValidation(result); err != nil {
		return nil, fmt.Errorf("final validation failed: %w", err)
	}

	return result, nil
}

func (g *Generator) validateSelectedConstants(p, q ConstantCandidate) error {
	// Check for nil or zero values
	if p.Value == 0 || q.Value == 0 {
		return fmt.Errorf("zero value constant selected")
	}

	// Validate minimum scores
	if p.AvalancheScore < g.config.MinAvalancheScore ||
		q.AvalancheScore < g.config.MinAvalancheScore {
		return fmt.Errorf("constants do not meet minimum avalanche score")
	}

	// Validate bit distribution
	if !g.isValidBitDistribution(p) || !g.isValidBitDistribution(q) {
		return fmt.Errorf("constants do not meet bit distribution requirements")
	}

	// Validate primality
	if !g.isPrime(p.Value) || !g.isPrime(q.Value) {
		return fmt.Errorf("selected constants are not prime")
	}

	return nil
}

func (g *Generator) isValidBitDistribution(c ConstantCandidate) bool {
	return c.BitDistribution >= g.config.MinBitDistribution &&
		c.BitDistribution <= g.config.MaxBitDistribution
}

func (g *Generator) runFinalValidation(result *GenerationResult) error {
	// Perform final statistical tests
	pTests := g.runAllStatisticalTests(result.SelectedP.Value)
	qTests := g.runAllStatisticalTests(result.SelectedQ.Value)

	// Update results with final tests
	result.SelectedP.TestResults.StatisticalTests = pTests
	result.SelectedQ.TestResults.StatisticalTests = qTests

	// Verify minimum test passing requirements
	if !g.verifyTestResults(pTests) || !g.verifyTestResults(qTests) {
		return fmt.Errorf("final statistical tests failed")
	}

	// Verify constants are sufficiently different
	if !g.areSufficientlyDifferent(result.SelectedP, result.SelectedQ) {
		return fmt.Errorf("selected constants are not sufficiently different")
	}

	return nil
}

func (g *Generator) verifyTestResults(tests []StatisticalTest) bool {
	failedTests := 0
	for _, test := range tests {
		if !test.Passed {
			failedTests++
		}
	}
	// Allow up to 20% of tests to fail
	return failedTests <= len(tests)/5
}

func (g *Generator) rc6Transform(input, constant uint32) uint32 {
	// Simplified RC6-like transformation
	x := input
	x = ((x << 5) | (x >> 27)) // ROL by 5
	x *= constant
	x = ((x << 3) | (x >> 29)) // ROL by 3
	return x
}

func (g *Generator) worker(workerID int, candidates chan<- ConstantCandidate, errors chan<- error, batchSize int) {
	// Use context from Generator struct
	for i := 0; i < batchSize; i++ {
		// Check for context cancellation
		select {
		case <-g.ctx.Done():
			errors <- fmt.Errorf("worker %d cancelled: %v", workerID, g.ctx.Err())
			return
		default:
			// Continue processing
		}

		candidate, err := g.generateCandidate()
		if err != nil {
			errors <- fmt.Errorf("worker %d error: %v", workerID, err)
			continue
		}

		if g.validateCandidate(candidate) {
			candidates <- candidate
		}
	}
}

func (g *Generator) generateCandidate() (ConstantCandidate, error) {
	start := time.Now()

	value, err := g.generate32BitPrime()
	if err != nil {
		return ConstantCandidate{}, err
	}

	bitDist := g.calculateBitDistribution(value)
	avalanche := g.testAvalancheEffect(value)
	entropy := g.calculateEntropy(value)
	hammingWeight := bits.OnesCount32(value)

	candidate := ConstantCandidate{
		Value:           value,
		BitDistribution: bitDist,
		AvalancheScore:  avalanche,
		HammingWeight:   hammingWeight,
		EntropyScore:    entropy,
		TestDuration:    time.Since(start),
		GenerationTime:  start,
	}

	// Perform additional tests
	candidate.TestResults = g.runTests(candidate)

	return candidate, nil
}

func (g *Generator) generate32BitPrime() (uint32, error) {
	for attempt := 0; attempt < g.config.MaxPrimeAttempts; attempt++ {
		var b [4]byte
		n, err := rand.Read(b[:])
		if err != nil {
			return 0, fmt.Errorf("random generation failed: %w", err)
		}
		if n != 4 {
			return 0, fmt.Errorf("incomplete random read: got %d bytes", n)
		}

		value := binary.BigEndian.Uint32(b[:])

		// Avoid overflow in primality testing
		if value > math.MaxUint32-100 {
			continue
		}

		if g.isPrime(value) {
			return value, nil
		}
	}
	return 0, fmt.Errorf("prime generation failed after %d attempts",
		g.config.MaxPrimeAttempts)
}

func (g *Generator) isPrime(n uint32) bool {
	if n <= 1 || n == 4 {
		return false
	}
	if n <= 3 {
		return true
	}

	// Miller-Rabin test bases for 32-bit integers
	bases := []uint32{2, 7, 61}

	// Find d such that n-1 = d * 2^r
	d := n - 1
	r := uint32(0)
	for d%2 == 0 {
		d /= 2
		r++
	}

	for _, a := range bases {
		if !g.millerRabinTest(n, d, r, a) {
			return false
		}
	}
	return true
}

func (g *Generator) millerRabinTest(n, d, r, a uint32) bool {
	if n == a {
		return true
	}
	x := g.modPow(a, d, n)
	if x == 1 || x == n-1 {
		return true
	}
	for j := uint32(0); j < r-1; j++ {
		x = (x * x) % n
		if x == n-1 {
			return true
		}
		if x == 1 {
			return false
		}
	}
	return false
}

func (g *Generator) modPow(base, exp, mod uint32) uint32 {
	if mod == 0 {
		panic("modulus cannot be zero")
	}

	result := uint64(1)
	b := uint64(base) % uint64(mod)
	e := uint64(exp)

	for e > 0 {
		if e&1 == 1 {
			result = (result * b) % uint64(mod)
		}
		b = (b * b) % uint64(mod)
		e >>= 1
	}

	return uint32(result)
}

func (g *Generator) calculateBitDistribution(n uint32) float64 {
	ones := 0
	for i := 0; i < 32; i++ {
		if n&(1<<uint(i)) != 0 {
			ones++
		}
	}
	return float64(ones) / 32.0
}

func (g *Generator) testAvalancheEffect(constant uint32) float64 {
	var totalChanges float64
	testCases := g.config.AvalancheTestCases

	for i := 0; i < testCases; i++ {
		// Generate random input
		var input uint32
		for j := 0; j < 4; j++ {
			b := make([]byte, 1)
			rand.Read(b)
			input = (input << 8) | uint32(b[0])
		}

		// Test each bit position
		for bitPos := 0; bitPos < 32; bitPos++ {
			// Flip one bit
			modifiedInput := input ^ (1 << uint(bitPos))

			// Apply RC6-like transformation
			result1 := g.rc6Transform(input, constant)
			result2 := g.rc6Transform(modifiedInput, constant)

			// Count changed bits in output
			changes := bits.OnesCount32(result1 ^ result2)
			totalChanges += float64(changes)
		}
	}

	// Average changes per bit flip (normalize to 0-1 range)
	return totalChanges / float64(testCases*32*32)
}

func (g *Generator) compareOutputs(input1, input2 []byte, constant uint32) int {
	result1 := g.encryptionTest(input1, constant)
	result2 := g.encryptionTest(input2, constant)

	differences := 0
	for i := 0; i < len(result1); i++ {
		diff := result1[i] ^ result2[i]
		differences += bits.OnesCount8(uint8(diff))
	}
	return differences
}

func (g *Generator) encryptionTest(input []byte, constant uint32) []byte {
	output := make([]byte, len(input))
	for i := 0; i < len(input); i++ {
		output[i] = input[i] ^ byte(constant>>(uint(i%4)*8))
	}
	return output
}

func (g *Generator) validateCandidate(candidate ConstantCandidate) bool {
	if candidate.BitDistribution < g.config.MinBitDistribution ||
		candidate.BitDistribution > g.config.MaxBitDistribution {
		return false
	}

	if candidate.AvalancheScore < g.config.MinAvalancheScore {
		return false
	}

	if candidate.HammingWeight < 12 || candidate.HammingWeight > 20 {
		return false
	}

	if candidate.EntropyScore < 1.5 {
		return false
	}

	for _, test := range candidate.TestResults.WeakKeyTests {
		if !test.Passed {
			return false
		}
	}

	return true
}

func (g *Generator) selectBestConstants(candidates []ConstantCandidate) (ConstantCandidate, ConstantCandidate) {
	if len(candidates) < 2 {
		panic("insufficient candidates")
	}

	// Pre-calculate scores to avoid repeated calculations
	type scoredCandidate struct {
		candidate ConstantCandidate
		score     float64
	}

	// Allocate with capacity
	scored := make([]scoredCandidate, 0, len(candidates))

	// Calculate scores once
	for _, c := range candidates {
		scored = append(scored, scoredCandidate{
			candidate: c,
			score:     g.calculateScore(c),
		})
	}

	// Sort with pre-calculated scores
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Select best pair
	bestP := scored[0].candidate
	for _, sc := range scored[1:] {
		if g.areSufficientlyDifferent(bestP, sc.candidate) {
			return bestP, sc.candidate
		}
	}

	// Fallback if no sufficiently different pair found
	return bestP, scored[1].candidate
}

func (g *Generator) calculateScore(candidate ConstantCandidate) float64 {
	// Weighted scoring of various properties
	bitDistScore := 1.0 - math.Abs(0.5-candidate.BitDistribution)
	avalancheWeight := 2.0 // Give more weight to avalanche effect
	entropyWeight := 1.5

	return (bitDistScore +
		avalancheWeight*candidate.AvalancheScore +
		entropyWeight*(candidate.EntropyScore/2.0)) /
		(1.0 + avalancheWeight + entropyWeight)
}

func (g *Generator) areSufficientlyDifferent(a, b ConstantCandidate) bool {
	// Check if constants are sufficiently different
	diff := a.Value ^ b.Value
	hammingDistance := bits.OnesCount32(diff)

	// Should have at least 12 bits different
	if hammingDistance < 12 {
		return false
	}

	// Should not be related by simple shifts
	for i := 1; i < 32; i++ {
		if a.Value == b.Value<<uint(i) || a.Value == b.Value>>uint(i) {
			return false
		}
	}

	return true
}

func (g *Generator) saveResults(result *GenerationResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %v", err)
	}

	err = os.WriteFile(g.config.ResultsFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write results file: %v", err)
	}

	return nil
}

func (g *Generator) runTests(candidate ConstantCandidate) TestResults {
	results := TestResults{
		PrimalityTests: g.runPrimalityTests(candidate.Value),
		AvalancheTests: g.runAvalancheTests(candidate.Value),
		WeakKeyTests:   g.runWeakKeyTests(candidate.Value),
	}

	// Add statistical tests when enabled in config
	if g.config.StatisticalAnalysis {
		results.StatisticalTests = g.runAllStatisticalTests(candidate.Value)
	}

	return results
}

func (g *Generator) runPrimalityTests(value uint32) []PrimalityTest {
	tests := []PrimalityTest{
		{
			Method:  "Miller-Rabin",
			Passed:  g.isPrime(value),
			Details: fmt.Sprintf("Tested with bases [2, 7, 61]"),
		},
	}
	return tests
}

func (g *Generator) runAvalancheTests(value uint32) []AvalancheTest {
	start := time.Now()
	score := g.testAvalancheEffect(value)

	return []AvalancheTest{
		{
			Score:    score,
			Changes:  int(score * float64(g.config.AvalancheTestCases*128)),
			Total:    g.config.AvalancheTestCases * 128,
			Duration: time.Since(start),
		},
	}
}

func (g *Generator) runWeakKeyTests(value uint32) []WeakKeyTest {
	tests := []WeakKeyTest{
		{
			Pattern: "Low Hamming Weight",
			Passed:  bits.OnesCount32(value) >= 12,
		},
		{
			Pattern: "Simple Bit Pattern",
			Passed:  !g.hasSimpleBitPattern(value),
		},
	}
	return tests
}

func (g *Generator) hasSimpleBitPattern(value uint32) bool {
	// Check for simple repeating patterns
	patterns := []uint32{
		0xAAAAAAAA, // alternating bits
		0x55555555, // alternating bits
		0x33333333, // repeating pairs
		0xCCCCCCCC, // repeating pairs
		0x0F0F0F0F, // repeating quads
		0xF0F0F0F0, // repeating quads
	}

	for _, pattern := range patterns {
		if value == pattern || value == ^pattern {
			return true
		}
	}

	return false
}

func (g *Generator) testConstantCorrelation(p, q uint32) float64 {
	// Convert to bit arrays
	pBits := make([]int, 32)
	qBits := make([]int, 32)

	for i := 0; i < 32; i++ {
		if p&(1<<uint(i)) != 0 {
			pBits[i] = 1
		}
		if q&(1<<uint(i)) != 0 {
			qBits[i] = 1
		}
	}

	// Calculate correlation coefficient
	var sum, pSum, qSum, pSqSum, qSqSum float64
	n := float64(32)

	for i := 0; i < 32; i++ {
		pVal := float64(pBits[i])
		qVal := float64(qBits[i])
		sum += pVal * qVal
		pSum += pVal
		qSum += qVal
		pSqSum += pVal * pVal
		qSqSum += qVal * qVal
	}

	numerator := sum - (pSum * qSum / n)
	denominator := ((pSqSum - (pSum * pSum / n)) * (qSqSum - (qSum * qSum / n)))

	if denominator == 0 {
		return 1.0
	}
	return numerator / denominator
}

func (g *Generator) testCombinedAvalancheEffect(p, q uint32) float64 {
	var totalChanges int
	testCases := g.config.AvalancheTestCases

	for i := 0; i < testCases; i++ {
		// Test how changes in input affect both P and Q operations
		input := uint32(i)
		modified := input ^ 1 // Flip lowest bit

		result1 := (input * p) ^ (input * q)
		result2 := (modified * p) ^ (modified * q)

		changes := 0
		diff := result1 ^ result2
		for diff != 0 {
			changes += int(diff & 1)
			diff >>= 1
		}

		totalChanges += changes
	}

	return float64(totalChanges) / float64(testCases*32)
}
