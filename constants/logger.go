package constants

import (
    "fmt"
    "log"
    "os"
)

type Logger struct {
    detailed bool
    log      *log.Logger
}

func NewLogger(detailed bool) *Logger {
    return &Logger{
        detailed: detailed,
        log:      log.New(os.Stdout, "", log.LstdFlags),
    }
}

func (l *Logger) Info(v ...interface{}) {
    if l.detailed {
        l.log.Printf("INFO: %s", fmt.Sprint(v...))
    }
}

func (l *Logger) Error(v ...interface{}) {
    l.log.Printf("ERROR: %s", fmt.Sprint(v...))
}

func (l *Logger) Debug(v ...interface{}) {
    if l.detailed {
        l.log.Printf("DEBUG: %s", fmt.Sprint(v...))
    }
}
