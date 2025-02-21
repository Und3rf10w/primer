package constants

import (
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
}

func NewGenerator(config Config) *Generator {
    return &Generator{
        config: config,
        logger: NewLogger(config.DetailedLogging),
    }
}

func (g *Generator) Generate() (*GenerationResult, error) {
    start := time.Now()
    g.logger.Info("Starting RC6 constant generation")

    // Create channels for parallel processing
    candidateChan := make(chan ConstantCandidate, g.config.NumCandidates)
    errorChan := make(chan error, g.config.NumCandidates)
    var wg sync.WaitGroup

    // Start worker pool
    for i := 0; i < g.config.ParallelWorkers; i++ {
        wg.Add(1)
        go g.worker(i, candidateChan, errorChan, &wg)
    }

    // Wait for completion in separate goroutine
    go func() {
        wg.Wait()
        close(candidateChan)
        close(errorChan)
    }()

    // Collect results and handle errors
    var candidates []ConstantCandidate
    for {
        select {
        case err := <-errorChan:
            if err != nil {
                return nil, fmt.Errorf("generation error: %v", err)
            }
        case candidate, ok := <-candidateChan:
            if !ok {
                // Channel closed, all work complete
                goto ProcessResults
            }
            candidates = append(candidates, candidate)
        }
    }

ProcessResults:
    if len(candidates) < 2 {
        return nil, fmt.Errorf("insufficient valid candidates generated")
    }

    // Select best constants
    bestP, bestQ := g.selectBestConstants(candidates)
    
    result := &GenerationResult{
        SelectedP:       bestP,
        SelectedQ:       bestQ,
        TotalCandidates: len(candidates),
        Duration:        time.Since(start),
        StartTime:       start,
        EndTime:         time.Now(),
        Config:          g.config,
    }

    // Save results
    if err := g.saveResults(result); err != nil {
        g.logger.Error("Failed to save results:", err)
    }

    return result, nil
}

func (g *Generator) worker(id int, candidates chan<- ConstantCandidate, errors chan<- error, wg *sync.WaitGroup) {
    defer wg.Done()

    for i := 0; i < g.config.NumCandidates/g.config.ParallelWorkers; i++ {
        candidate, err := g.generateCandidate()
        if err != nil {
            errors <- err
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
        _, err := rand.Read(b[:])
        if err != nil {
            return 0, fmt.Errorf("failed to generate random number: %v", err)
        }
        n := binary.BigEndian.Uint32(b[:])
        if n > math.MaxUint32-100 {
            continue // Avoid overflow in primality testing
        }
        if g.isPrime(n) {
            return n, nil
        }
    }
    return 0, fmt.Errorf("failed to find prime number after %d attempts", g.config.MaxPrimeAttempts)
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
    result := uint32(1)
    base %= mod
    for exp > 0 {
        if exp&1 == 1 {
            result = (result * base) % mod
        }
        base = (base * base) % mod
        exp >>= 1
    }
    return result
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
    var totalChanges int
    
    for i := 0; i < g.config.AvalancheTestCases; i++ {
        var input [16]byte
        _, err := rand.Read(input[:])
        if err != nil {
            g.logger.Error("Failed to generate random input:", err)
            continue
        }

        input1 := make([]byte, 16)
        input2 := make([]byte, 16)
        copy(input1, input[:])
        copy(input2, input[:])

        bitPos := i % 128
        input2[bitPos/8] ^= 1 << uint(bitPos%8)

        changes := g.compareOutputs(input1, input2, constant)
        totalChanges += changes
    }

    return float64(totalChanges) / float64(g.config.AvalancheTestCases*128)
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

    // Check if all tests passed
    for _, test := range candidate.TestResults.StatisticalTests {
        if !test.Passed {
            return false
        }
    }

    for _, test := range candidate.TestResults.WeakKeyTests {
        if !test.Passed {
            return false
        }
    }

    return true
}

func (g *Generator) selectBestConstants(candidates []ConstantCandidate) (ConstantCandidate, ConstantCandidate) {
    // Sort candidates by score
    sort.Slice(candidates, func(i, j int) bool {
        scoreI := g.calculateScore(candidates[i])
        scoreJ := g.calculateScore(candidates[j])
        return scoreI > scoreJ
    })

    // Select best two candidates that are sufficiently different
    var bestP, bestQ ConstantCandidate
    bestP = candidates[0]

    for _, candidate := range candidates[1:] {
        if g.areSufficientlyDifferent(bestP, candidate) {
            bestQ = candidate
            break
        }
    }

    return bestP, bestQ
}

func (g *Generator) calculateScore(candidate ConstantCandidate) float64 {
    // Weighted scoring of various properties
    bitDistScore := 1.0 - math.Abs(0.5-candidate.BitDistribution)
    avalancheWeight := 2.0 // Give more weight to avalanche effect
    entropyWeight := 1.5

    return (bitDistScore +
            avalancheWeight * candidate.AvalancheScore +
            entropyWeight * (candidate.EntropyScore / 2.0)) /
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
    return TestResults{
        PrimalityTests:   g.runPrimalityTests(candidate.Value),
        AvalancheTests:   g.runAvalancheTests(candidate.Value),
        StatisticalTests: g.runStatisticalTests(candidate),
        WeakKeyTests:     g.runWeakKeyTests(candidate.Value),
    }
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