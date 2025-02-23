package constants

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

type testLogger struct {
	buffer *bytes.Buffer
	logger *Logger
}

func newTestLogger(detailed bool) *testLogger {
	buffer := new(bytes.Buffer)
	logger := &Logger{
		detailed: detailed,
		log:      log.New(buffer, "", log.LstdFlags),
	}
	return &testLogger{
		buffer: buffer,
		logger: logger,
	}
}

func TestLoggerInfo(t *testing.T) {
	tests := []struct {
		name     string
		detailed bool
		message  string
		want     string
	}{
		{
			name:     "Detailed logging enabled",
			detailed: true,
			message:  "test message",
			want:     "INFO: test message",
		},
		{
			name:     "Detailed logging disabled",
			detailed: false,
			message:  "test message",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := newTestLogger(tt.detailed)
			tl.logger.Info(tt.message)
			got := tl.buffer.String()

			if tt.detailed {
				if !strings.Contains(got, tt.want) {
					t.Errorf("Info() = %v, want %v", got, tt.want)
				}
			} else {
				if got != "" {
					t.Errorf("Info() = %v, want empty string", got)
				}
			}
		})
	}
}

func TestLoggerError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "Basic error message",
			message: "error message",
			want:    "ERROR: error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := newTestLogger(true)
			tl.logger.Error(tt.message)
			got := tl.buffer.String()

			if !strings.Contains(got, tt.want) {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoggerDebug(t *testing.T) {
	tests := []struct {
		name     string
		detailed bool
		message  string
		want     string
	}{
		{
			name:     "Debug with detailed logging",
			detailed: true,
			message:  "debug message",
			want:     "DEBUG: debug message",
		},
		{
			name:     "Debug without detailed logging",
			detailed: false,
			message:  "debug message",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := newTestLogger(tt.detailed)
			tl.logger.Debug(tt.message)
			got := tl.buffer.String()

			if tt.detailed {
				if !strings.Contains(got, tt.want) {
					t.Errorf("Debug() = %v, want %v", got, tt.want)
				}
			} else {
				if got != "" {
					t.Errorf("Debug() = %v, want empty string", got)
				}
			}
		})
	}
}
