package constants

import (
    "encoding/json"
    "os"
)

func DefaultConfig() Config {
    return Config{
        NumCandidates:       1000,
        AvalancheTestCases:  10000,
        MinPrimeAttempts:    100,
        MaxPrimeAttempts:    10000,
        ParallelWorkers:     8,
        MinBitDistribution:  0.45,
        MaxBitDistribution:  0.55,
        MinAvalancheScore:   0.49,
        ResultsFile:         "rc6_constants.json",
        DetailedLogging:     true,
        StatisticalAnalysis: true,
    }
}

func LoadConfig(path string) (Config, error) {
    config := DefaultConfig()
    
    if path == "" {
        return config, nil
    }

    data, err := os.ReadFile(path)
    if err != nil {
        return config, err
    }

    err = json.Unmarshal(data, &config)
    return config, err
}