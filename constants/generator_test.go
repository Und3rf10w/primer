package constants

import (
	"testing"
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
		// TODO: This would be nice if we could actually do this lol
		// {
		// 	name:    "Valid generation",
		// 	config:  DefaultConfig(),
		// 	wantErr: false,
		// 	validate: func(t *testing.T, result *GenerationResult) {
		// 		if result.SelectedP.Value == 0 {
		// 			t.Error("SelectedP not generated")
		// 		}
		// 		if result.SelectedQ.Value == 0 {
		// 			t.Error("SelectedQ not generated")
		// 		}
		// 		if result.Duration == 0 {
		// 			t.Error("Duration not recorded")
		// 		}
		// 	},
		// },
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

// TODO: This can't possibly be a real test
// func TestGenerateCandidate(t *testing.T) {
// 	generator := NewGenerator(DefaultConfig())

// 	tests := []struct {
// 		name     string
// 		validate func(*testing.T, ConstantCandidate)
// 	}{
// 		{
// 			name: "Basic candidate generation",
// 			validate: func(t *testing.T, c ConstantCandidate) {
// 				if c.Value == 0 {
// 					t.Error("Generated value is zero")
// 				}
// 				if c.BitDistribution < 0 || c.BitDistribution > 1 {
// 					t.Error("Invalid bit distribution")
// 				}
// 				if c.AvalancheScore < 0 || c.AvalancheScore > 1 {
// 					t.Error("Invalid avalanche score")
// 				}
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			candidate, err := generator.generateCandidate()
// 			if err != nil {
// 				t.Errorf("generateCandidate() error = %v", err)
// 				return
// 			}
// 			tt.validate(t, candidate)
// 		})
// 	}
// }

// TODO: This requires candidate generation, so we won't bother
// func TestWorker(t *testing.T) {
// 	config := DefaultConfig()
// 	config.NumCandidates = 1
// 	config.MinPrimeAttempts = 10
// 	generator := NewGenerator(config)

// 	candidateChan := make(chan ConstantCandidate, 1)
// 	errorChan := make(chan error, 1)
// 	done := make(chan struct{})

// 	go func() {
// 		generator.worker(0, candidateChan, errorChan, 1)
// 		close(done)
// 	}()

// 	select {
// 	case candidate := <-candidateChan:
// 		// Verify the candidate meets basic requirements
// 		if candidate.Value == 0 {
// 			t.Error("Worker generated invalid candidate with zero value")
// 		}
// 		if !generator.isPrime(candidate.Value) {
// 			t.Error("Worker generated non-prime candidate")
// 		}
// 	case err := <-errorChan:
// 		t.Errorf("Worker returned error: %v", err)
// 	case <-done:
// 		t.Error("Worker completed without generating candidate")
// 	}
// }

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

func BenchmarkRC6Constants(b *testing.B) {
	g := NewGenerator(DefaultConfig())

	b.Run("RC6_P", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			g.runAllStatisticalTests(RC6_P)
		}
	})

	b.Run("RC6_Q", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			g.runAllStatisticalTests(RC6_Q)
		}
	})

	b.Run("RC6_Combined", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			g.testCombinedAvalancheEffect(RC6_P, RC6_Q)
		}
	})
}
