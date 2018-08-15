/*
Package log implements a reporter to send spans in V2 JSON format to the Go
standard Logger.
*/
package log

import (
	"encoding/json"
	"fmt"
	"github.com/qutoutiao/zipkin-go/model"
	"github.com/qutoutiao/zipkin-go/reporter"
	"github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

// logReporter will send spans to the default Go Logger.
type logReporter struct {
	logger *logrus.Logger
}

type ZipkinFormatter struct {
}

// Format renders a single log entry
func (f *ZipkinFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := fmt.Sprintf("%s\n", entry.Message)

	return []byte(data), nil
}

// NewReporter returns a new log reporter.
func NewReporter() reporter.Reporter {

	level, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		panic(err)
	}

	logfile := viper.GetString("log.trace.file")
	path := filepath.Join(viper.GetString("log.dir"), logfile)

	w, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		panic(err)
	}
	l := &logrus.Logger{
		Out:       w,
		Formatter: new(ZipkinFormatter),
		Level:     level,
	}

	return &logReporter{
		logger: l,
	}
}

// Send outputs a span to the Go logger.
/*func (r *logReporter) Send(s model.SpanModel) {
    if b, err := json.Marshal(s); err == nil {
        r.logger.Info(string(b))
    }
}*/
func (r *logReporter) Send(s model.SpanModel) {
	var t []model.SpanModel
	t = append(t, s)
	if b, err := json.Marshal(t); err == nil {
		r.logger.Info(string(b))
	}
}

// Close closes the reporter
func (*logReporter) Close() error { return nil }
