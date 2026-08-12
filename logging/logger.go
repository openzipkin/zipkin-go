// Copyright 2022 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package logging provides a level-aware wrapper around the standard library
log.Logger used by the library components.
*/
package logging

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
)

// Level represents a log severity level.
//
// Levels are ordered by severity, from InfoLevel (lowest) to ErrorLevel
// (highest). A Logger emits messages whose severity is equal to or higher
// than its configured level.
type Level int

const (
	// InfoLevel logs all messages. This is the default level.
	InfoLevel Level = iota
	// WarnLevel suppresses Info messages, logging only warnings and errors.
	WarnLevel
	// ErrorLevel suppresses Info and Warn messages, logging only errors.
	ErrorLevel
)

// String returns the name of the severity level.
func (l Level) String() string {
	switch l {
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return fmt.Sprintf("Level(%d)", int(l))
	}
}

// Logger wraps a standard library log.Logger with severity-level filtering.
//
// Logger is safe for concurrent use by multiple goroutines. The zero value is
// not usable; create one with NewLogger.
type Logger struct {
	l     *log.Logger
	level atomic.Int32
}

// NewLogger returns a Logger backed by the given *log.Logger. If l is nil, a
// default logger writing to os.Stderr with standard flags is used. The
// initial level is InfoLevel, which logs all messages.
func NewLogger(l *log.Logger) *Logger {
	if l == nil {
		l = log.New(os.Stderr, "", log.LstdFlags)
	}
	lg := &Logger{l: l}
	lg.level.Store(int32(InfoLevel))
	return lg
}

// Swap replaces the underlying *log.Logger while preserving the current
// level. If l is nil, the default logger writing to os.Stderr is used.
//
// Swap is intended to be used through option functions during initialization
// and must not be called concurrently with the logging methods.
func (lg *Logger) Swap(l *log.Logger) {
	if l == nil {
		l = log.New(os.Stderr, "", log.LstdFlags)
	}
	lg.l = l
}

// Level returns the current severity threshold of the logger.
func (lg *Logger) Level() Level {
	return Level(lg.level.Load())
}

// SetLevel sets the severity threshold of the logger. Messages with a
// severity lower than level are suppressed. SetLevel is safe for concurrent
// use. Setting a value outside the defined Level constants suppresses all
// output.
func (lg *Logger) SetLevel(level Level) {
	lg.level.Store(int32(level))
}

// enabled reports whether messages of the given severity level pass the
// logger's threshold.
func (lg *Logger) enabled(level Level) bool {
	return level >= lg.Level()
}

// Log logs a message at the given severity level.
func (lg *Logger) Log(level Level, v ...any) {
	if lg.enabled(level) {
		// Output with calldepth 3 so that the caller of the convenience
		// methods (Info, Warn, ...) is attributed in the log header when the
		// Lshortfile or Llongfile flags are set.
		lg.l.Output(3, fmt.Sprintln(v...))
	}
}

// Logf logs a formatted message at the given severity level.
func (lg *Logger) Logf(level Level, format string, v ...any) {
	if lg.enabled(level) {
		lg.l.Output(3, fmt.Sprintf(format, v...))
	}
}

// Info logs a message at Info level.
func (lg *Logger) Info(v ...any) { lg.Log(InfoLevel, v...) }

// Warn logs a message at Warn level.
func (lg *Logger) Warn(v ...any) { lg.Log(WarnLevel, v...) }

// Error logs a message at Error level.
func (lg *Logger) Error(v ...any) { lg.Log(ErrorLevel, v...) }

// Infof logs a formatted message at Info level.
func (lg *Logger) Infof(format string, v ...any) { lg.Logf(InfoLevel, format, v...) }

// Warnf logs a formatted message at Warn level.
func (lg *Logger) Warnf(format string, v ...any) { lg.Logf(WarnLevel, format, v...) }

// Errorf logs a formatted message at Error level.
func (lg *Logger) Errorf(format string, v ...any) { lg.Logf(ErrorLevel, format, v...) }
