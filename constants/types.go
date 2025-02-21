package constants

import (
    "time"
)

type Config struct {
    NumCandidates        int
    AvalancheTestCases   int
    MinPrimeAttempts     int
    MaxPrimeAttempts     int
    ParallelWorkers      int
    MinBitDistribution   float64
    MaxBitDistribution   float64
    MinAvalancheScore    float64
    ResultsFile          string
    DetailedLogging      bool
    StatisticalAnalysis  bool
}

type ConstantCandidate struct {
    Value           uint32
    BitDistribution float64
    AvalancheScore  float64
    HammingWeight   int
    EntropyScore    float64
    TestDuration    time.Duration
    GenerationTime  time.Time
    TestResults     TestResults
}

type TestResults struct {
    PrimalityTests     []PrimalityTest
    AvalancheTests     []AvalancheTest
    StatisticalTests   []StatisticalTest
    WeakKeyTests       []WeakKeyTest
}

type PrimalityTest struct {
    Passed    bool
    Duration  time.Duration
    Method    string
    Details   string
}

type AvalancheTest struct {
    Score     float64
    Changes   int
    Total     int
    Duration  time.Duration
}

type StatisticalTest struct {
    Name      string
    Score     float64
    Passed    bool
    Details   string
}

type WeakKeyTest struct {
    Passed    bool
    Pattern   string
    Details   string
}

type GenerationResult struct {
    SelectedP        ConstantCandidate
    SelectedQ        ConstantCandidate
    TotalCandidates  int
    Duration         time.Duration
    StartTime        time.Time
    EndTime          time.Time
    Config           Config
}