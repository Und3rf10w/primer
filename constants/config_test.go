package constants

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"NumCandidates", config.NumCandidates, 1000},
		{"AvalancheTestCases", config.AvalancheTestCases, 10000},
		{"MinPrimeAttempts", config.MinPrimeAttempts, 100},
		{"MaxPrimeAttempts", config.MaxPrimeAttempts, 10000},
		{"ParallelWorkers", config.ParallelWorkers, 8},
		{"MinBitDistribution", config.MinBitDistribution, 0.35},
		{"MaxBitDistribution", config.MaxBitDistribution, 0.65},
		{"MinAvalancheScore", config.MinAvalancheScore, 0.25},
		{"ResultsFile", config.ResultsFile, "rc6_constants.json"},
		{"DetailedLogging", config.DetailedLogging, true},
		{"StatisticalAnalysis", config.StatisticalAnalysis, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v; want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "Valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "Invalid NumCandidates",
			config: Config{
				NumCandidates: 0,
			},
			wantErr: true,
		},
		{
			name: "Invalid ParallelWorkers",
			config: Config{
				NumCandidates:   1000,
				ParallelWorkers: 0,
			},
			wantErr: true,
		},
		{
			name: "Invalid bit distribution range",
			config: Config{
				NumCandidates:      1000,
				ParallelWorkers:    8,
				MinBitDistribution: 0.6,
				MaxBitDistribution: 0.5,
			},
			wantErr: true,
		},
		{
			name: "Invalid avalanche score",
			config: Config{
				NumCandidates:      1000,
				ParallelWorkers:    8,
				MinBitDistribution: 0.45,
				MaxBitDistribution: 0.55,
				MinAvalancheScore:  1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		setup    func() string
		wantErr  bool
		validate func(*testing.T, Config)
	}{
		{
			name: "Empty path returns default config",
			setup: func() string {
				return ""
			},
			wantErr: false,
			validate: func(t *testing.T, c Config) {
				default_config := DefaultConfig()
				if c != default_config {
					t.Errorf("Expected default config, got %+v", c)
				}
			},
		},
		{
			name: "Valid config file",
			setup: func() string {
				path := filepath.Join(tmpDir, "valid_config.json")
				config := DefaultConfig()
				config.NumCandidates = 2000
				data, _ := json.Marshal(config)
				os.WriteFile(path, data, 0644)
				return path
			},
			wantErr: false,
			validate: func(t *testing.T, c Config) {
				if c.NumCandidates != 2000 {
					t.Errorf("Expected NumCandidates=2000, got %d", c.NumCandidates)
				}
			},
		},
		{
			name: "Invalid JSON file",
			setup: func() string {
				path := filepath.Join(tmpDir, "invalid_config.json")
				os.WriteFile(path, []byte("{invalid json}"), 0644)
				return path
			},
			wantErr:  true,
			validate: func(t *testing.T, c Config) {},
		},
		{
			name: "Non-existent file",
			setup: func() string {
				return filepath.Join(tmpDir, "nonexistent.json")
			},
			wantErr:  true,
			validate: func(t *testing.T, c Config) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			config, err := LoadConfig(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				tt.validate(t, config)
			}
		})
	}
}
