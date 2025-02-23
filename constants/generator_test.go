package constants

import (
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	config := DefaultConfig()
	generator := NewGenerator(config)

	if generator == nil {
		t.Error("NewGenerator returned nil")
	}
	if generator.config != config {
		t.Error("Config not properly set")
	}
	if generator.logger == nil {
		t.Error("Logger not initialized")
	}
	if generator.ctx == nil {
		t.Error("Context not initialized")
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantErr  bool
		validate func(*testing.T, *GenerationResult)
	}{
		{
			name:    "Valid generation",
			config:  DefaultConfig(),
			wantErr: false,
			validate: func(t *testing.T, result *GenerationResult) {
				if result.SelectedP.Value == 0 {
					t.Error("SelectedP not generated")
				}
				if result.SelectedQ.Value == 0 {
					t.Error("SelectedQ not generated")
				}
				if result.Duration == 0 {
					t.Error("Duration not recorded")
				}
			},
		},
		{
			name: "Invalid config",
			config: Config{
				NumCandidates:   0,
				ParallelWorkers: 0,
			},
			wantErr:  true,
			validate: func(t *testing.T, result *GenerationResult) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.config)
			result, err := generator.Generate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestGenerateCandidate(t *testing.T) {
	generator := NewGenerator(DefaultConfig())

	tests := []struct {
		name     string
		validate func(*testing.T, ConstantCandidate)
	}{
		{
			name: "Basic candidate generation",
			validate: func(t *testing.T, c ConstantCandidate) {
				if c.Value == 0 {
					t.Error("Generated value is zero")
				}
				if c.BitDistribution < 0 || c.BitDistribution > 1 {
					t.Error("Invalid bit distribution")
				}
				if c.AvalancheScore < 0 || c.AvalancheScore > 1 {
					t.Error("Invalid avalanche score")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidate, err := generator.generateCandidate()
			if err != nil {
				t.Errorf("generateCandidate() error = %v", err)
				return
			}
			tt.validate(t, candidate)
		})
	}
}

func TestWorker(t *testing.T) {
	config := DefaultConfig()
	config.NumCandidates = 10
	generator := NewGenerator(config)

	candidateChan := make(chan ConstantCandidate, 10)
	errorChan := make(chan error, 10)

	go generator.worker(0, candidateChan, errorChan, 5)

	// Collect results with timeout
	timeout := time.After(5 * time.Second)
	candidates := 0
	errors := 0

	for candidates < 5 {
		select {
		case <-candidateChan:
			candidates++
		case err := <-errorChan:
			if err != nil {
				t.Errorf("Worker error: %v", err)
			}
			errors++
		case <-timeout:
			t.Fatal("Worker test timed out")
		}
	}
}

// Benchmark tests
func BenchmarkGenerateCandidate(b *testing.B) {
	generator := NewGenerator(DefaultConfig())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := generator.generateCandidate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsPrime(b *testing.B) {
	generator := NewGenerator(DefaultConfig())
	value := uint32(104729) // A prime number
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		generator.isPrime(value)
	}
}
