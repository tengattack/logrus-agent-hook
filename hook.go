package logrusagent

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

// Hook represents a Logstash hook.
// It has two fields: writer to write the entry to Logstash and
// formatter to format the entry to a Logstash format before sending.
//
// To initialize it use the `New` function.
//
type Hook struct {
	writer    io.Writer
	formatter logrus.Formatter
}

const (
	// TimeFormat is the default @timestamp format, add millisecounds to time.RFC3339
	TimeFormat = "2006-01-02T15:04:05.999Z07:00"
)

// New returns a new logrus.Hook for Logstash.
//
// To create a new hook that sends logs to `tcp://logstash.corp.io:9999`:
//
// conn, _ := net.Dial("tcp", "logstash.corp.io:9999")
// hook := logrustash.New(conn, logrustash.DefaultFormatter())
func New(w io.Writer, f logrus.Formatter) logrus.Hook {
	return Hook{
		writer:    w,
		formatter: f,
	}
}

// Fire takes, formats and sends the entry to Logstash.
// Hook's formatter is used to format the entry into Logstash format
// and Hook's writer is used to write the formatted entry to the Logstash instance.
func (h Hook) Fire(e *logrus.Entry) error {
	dataBytes, err := h.formatter.Format(e)
	if err != nil {
		return err
	}
	_, err = h.writer.Write(dataBytes)
	return err
}

// Levels returns all logrus levels.
func (h Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Using a pool to re-use of old entries when formatting Logstash messages.
// It is used in the Fire function.
var entryPool = sync.Pool{
	New: func() interface{} {
		return &logrus.Entry{}
	},
}

// copyEntry copies the entry `e` to a new entry and then adds all the fields in `fields` that are missing in the new entry data.
// It uses `entryPool` to re-use allocated entries.
func copyEntry(e *logrus.Entry, fields logrus.Fields) *logrus.Entry {
	ne := entryPool.Get().(*logrus.Entry)
	ne.Message = e.Message
	ne.Level = e.Level
	ne.Time = e.Time
	ne.Data = logrus.Fields{}
	for k, v := range fields {
		ne.Data[k] = v
	}
	for k, v := range e.Data {
		ne.Data[k] = v
	}
	return ne
}

// releaseEntry puts the given entry back to `entryPool`. It must be called if copyEntry is called.
func releaseEntry(e *logrus.Entry) {
	entryPool.Put(e)
}

// LogAgentFormatter represents a Logstash format.
// It has logrus.Formatter which formats the entry and logrus.Fields which
// are added to the JSON message if not given in the entry data.
//
// Note: use the `DefaultFormatter` function to set a default Logstash formatter.
type LogAgentFormatter struct {
	logrus.FieldMap
	logrus.Fields
}

// fields
const (
	FieldKeyTime  = "@timestamp"
	FieldKeyMsg   = "message"
	FieldKeyLevel = "level"
)

var (
	logstashFields = logrus.Fields{"@version": "1"}
)

// Convert the Level to a string. E.g. ErrorLevel becomes "ERROR".
func getLevelString(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return "DEBUG"
	case logrus.InfoLevel:
		return "INFO"
	case logrus.WarnLevel:
		return "WARN"
	case logrus.ErrorLevel:
		return "ERROR"
	case logrus.FatalLevel:
		return "FATAL"
	case logrus.PanicLevel:
		return "PANIC"
	}

	return "UNKNOWN"
}

// DefaultFormatter returns a default Logstash formatter:
// A JSON format with "@version" set to "1" (unless set differently in `fields`)
// "@timestamp" to the log time and "message" to the log message.
//
// Note: to set a different configuration use the `LogAgentFormatter` structure.
func DefaultFormatter(fields logrus.Fields) logrus.Formatter {
	for k, v := range logstashFields {
		if _, ok := fields[k]; !ok {
			fields[k] = v
		}
	}

	return &LogAgentFormatter{
		Fields: fields,
	}
}

// Format formats an entry to a Logstash format according to the given Formatter and Fields.
//
// Note: the given entry is copied and not changed during the formatting process.
func (f *LogAgentFormatter) Format(e *logrus.Entry) ([]byte, error) {
	entry := copyEntry(e, f.Fields)
	data := make(logrus.Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	// defaultTimestampFormat
	data[FieldKeyTime] = entry.Time.Format(TimeFormat)
	data[FieldKeyLevel] = getLevelString(entry.Level)
	data[FieldKeyMsg] = entry.Message

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	dataBytes := append(serialized, '\n')
	releaseEntry(entry)
	return dataBytes, nil
}
