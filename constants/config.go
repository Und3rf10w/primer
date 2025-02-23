package constants

import (
	"encoding/json"
	"fmt"
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

func ValidateConfig(config *Config) error {
	if config.NumCandidates < 1 {
		return fmt.Errorf("NumCandidates must be positive")
	}
	if config.ParallelWorkers < 1 {
		return fmt.Errorf("ParallelWorkers must be positive")
	}
	if config.MinBitDistribution >= config.MaxBitDistribution {
		return fmt.Errorf("invalid bit distribution range")
	}
	if config.MinAvalancheScore < 0 || config.MinAvalancheScore > 1 {
		return fmt.Errorf("invalid avalanche score threshold")
	}
	return nil
}

func LoadConfig(path string) (Config, error) {
	config := DefaultConfig()

	if path == "" {
		return config, ValidateConfig(&config)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("parsing config: %w", err)
	}

	return config, ValidateConfig(&config)
}
