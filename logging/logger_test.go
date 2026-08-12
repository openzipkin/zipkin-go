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

package logging_test

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/openzipkin/zipkin-go/logging"
)

func captureLogger(t *testing.T) (*bytes.Buffer, *logging.Logger) {
	t.Helper()
	var buf bytes.Buffer
	return &buf, logging.NewLogger(log.New(&buf, "", 0))
}

func TestLoggerDefaultLevelShowsAll(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.Info("info")
	lg.Warn("warn")
	lg.Error("error")
	out := buf.String()
	if !strings.Contains(out, "info") {
		t.Error("expected Info to appear at default InfoLevel")
	}
	if !strings.Contains(out, "warn") {
		t.Error("expected Warn to appear at default InfoLevel")
	}
	if !strings.Contains(out, "error") {
		t.Error("expected Error to appear at default InfoLevel")
	}
}

func TestLoggerWarnLevelSuppressesInfo(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.WarnLevel)

	lg.Info("info")
	if buf.Len() != 0 {
		t.Error("expected Info to be suppressed at WarnLevel")
	}

	lg.Warn("warn")
	if buf.Len() == 0 {
		t.Error("expected Warn to appear at WarnLevel")
	}
}

func TestLoggerWarnLevelShowsWarnAndError(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.WarnLevel)

	lg.Warn("warn")
	lg.Error("error")
	out := buf.String()
	if !strings.Contains(out, "warn") {
		t.Error("expected Warn to appear at WarnLevel")
	}
	if !strings.Contains(out, "error") {
		t.Error("expected Error to appear at WarnLevel")
	}
}

func TestLoggerErrorLevelSuppressesWarn(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.ErrorLevel)

	lg.Warn("warn")
	if buf.Len() != 0 {
		t.Error("expected Warn to be suppressed at ErrorLevel")
	}

	lg.Error("error")
	if buf.Len() == 0 {
		t.Error("expected Error to appear at ErrorLevel")
	}
}

func TestLoggerErrorLevelSuppressesInfo(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.ErrorLevel)

	lg.Info("info")
	if buf.Len() != 0 {
		t.Error("expected Info to be suppressed at ErrorLevel")
	}
}

func TestLoggerInfof(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.Infof("formatted %s %d", "info", 1)
	if !strings.Contains(buf.String(), "formatted info 1") {
		t.Error("expected Infof to produce formatted output")
	}
}

func TestLoggerWarnf(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.Warnf("formatted %s %d", "warn", 2)
	if !strings.Contains(buf.String(), "formatted warn 2") {
		t.Error("expected Warnf to produce formatted output")
	}
}

func TestLoggerErrorf(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.Errorf("formatted %s %d", "error", 3)
	if !strings.Contains(buf.String(), "formatted error 3") {
		t.Error("expected Errorf to produce formatted output")
	}
}

func TestLoggerErrorfSuppressedAtErrorLevel(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.ErrorLevel)
	lg.Infof("should not appear")
	if buf.Len() != 0 {
		t.Error("expected Infof to be suppressed at ErrorLevel")
	}
	lg.Errorf("should appear")
	if buf.Len() == 0 {
		t.Error("expected Errorf to appear at ErrorLevel")
	}
}

func TestLoggerLevel(t *testing.T) {
	_, lg := captureLogger(t)
	if got := lg.Level(); got != logging.InfoLevel {
		t.Errorf("expected default Level InfoLevel, got %v", got)
	}

	lg.SetLevel(logging.WarnLevel)
	if got := lg.Level(); got != logging.WarnLevel {
		t.Errorf("expected Level WarnLevel after SetLevel, got %v", got)
	}

	lg.SetLevel(logging.ErrorLevel)
	if got := lg.Level(); got != logging.ErrorLevel {
		t.Errorf("expected Level ErrorLevel after SetLevel, got %v", got)
	}
}

func TestLoggerNewLoggerWithNilDefaults(t *testing.T) {
	lg := logging.NewLogger(nil)
	if lg == nil {
		t.Fatal("expected non-nil Logger")
	}
	lg.Info("works")
	lg.Warn("works")
	lg.Error("works")
}

func TestLoggerLogAndLogf(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.WarnLevel)

	lg.Log(logging.InfoLevel, "suppressed")
	if buf.Len() != 0 {
		t.Error("expected Log at InfoLevel to be suppressed at WarnLevel")
	}

	lg.Log(logging.WarnLevel, "warn via log")
	lg.Logf(logging.ErrorLevel, "error %d", 42)
	out := buf.String()
	if !strings.Contains(out, "warn via log") {
		t.Error("expected Log at WarnLevel to appear")
	}
	if !strings.Contains(out, "error 42") {
		t.Error("expected Logf at ErrorLevel to appear")
	}
}

func TestLoggerSwapPreservesLevel(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.ErrorLevel)
	lg.Info("suppressed")
	if buf.Len() != 0 {
		t.Error("expected Info to be suppressed before Swap")
	}

	var swapped bytes.Buffer
	lg.Swap(log.New(&swapped, "", 0))
	if got := lg.Level(); got != logging.ErrorLevel {
		t.Errorf("expected Swap to preserve ErrorLevel, got %v", got)
	}
	lg.Info("still suppressed")
	if swapped.Len() != 0 {
		t.Error("expected Info to be suppressed after Swap")
	}
	lg.Error("error")
	if !strings.Contains(swapped.String(), "error") {
		t.Error("expected Error to appear after Swap")
	}
}

func TestLoggerSwapNilUsesDefault(t *testing.T) {
	_, lg := captureLogger(t)
	lg.Swap(nil)
	lg.Error("works")
}

func TestLoggerSetLevelOutOfRangeSuppressesAll(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.Level(99))
	lg.Info("info")
	lg.Warn("warn")
	lg.Error("error")
	if buf.Len() != 0 {
		t.Error("expected out-of-range level to suppress all output")
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level logging.Level
		want  string
	}{
		{logging.InfoLevel, "info"},
		{logging.WarnLevel, "warn"},
		{logging.ErrorLevel, "error"},
		{logging.Level(42), "Level(42)"},
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("String() of %d = %q, want %q", int(tt.level), got, tt.want)
		}
	}
}

func TestLoggerAttributesCallerWithShortfile(t *testing.T) {
	var buf bytes.Buffer
	lg := logging.NewLogger(log.New(&buf, "", log.Lshortfile))
	lg.Error("boom")

	want := "logger_test.go:"
	if got := strings.TrimSpace(buf.String()); !strings.Contains(got, want) {
		t.Errorf("expected log header to attribute this test file, got %q", got)
	}
}

func TestLoggerConcurrentUse(t *testing.T) {
	buf, lg := captureLogger(t)
	lg.SetLevel(logging.WarnLevel)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				lg.Info("suppressed")
				lg.Warnf("warn %d-%d", id, j)
				_ = lg.Level()
			}
		}(i)
	}
	wg.Wait()

	out := buf.String()
	if strings.Contains(out, "suppressed") {
		t.Error("expected Info output to be suppressed at WarnLevel")
	}
	if got := strings.Count(out, "warn"); got != 8*50 {
		t.Errorf("expected 400 warn lines, got %d", got)
	}
}

func TestLoggerSetLevelConcurrent(t *testing.T) {
	_, lg := captureLogger(t)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			lg.SetLevel(logging.Level(i % 4))
		}
	}()
	for i := 0; i < 1000; i++ {
		lg.Error("boom")
	}
	wg.Wait()
}
