package logrusagent_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tengattack/logrus-agent-hook"
)

type simpleFmter struct{}

func (f simpleFmter) Format(e *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("msg: %#v", e.Message)), nil
}

func TestFire(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := logrusagent.New(buffer, simpleFmter{})

	entry := &logrus.Entry{
		Message: "my message",
		Data:    logrus.Fields{},
	}

	err := h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	expected := "msg: \"my message\""
	if buffer.String() != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, buffer.String())
	}
}

func TestDefaultFormatterWithEmptyFields(t *testing.T) {
	now := time.Now()
	formatter := logrusagent.DefaultFormatter(logrus.Fields{})

	entry := &logrus.Entry{
		Message: "message bla bla",
		Level:   logrus.DebugLevel,
		Time:    now,
		Data: logrus.Fields{
			"category": "test",
			"Key1":     "Value1",
		},
	}

	res, err := formatter.Format(entry)
	if err != nil {
		t.Errorf("expected Format not to return error: %s", err)
	}

	expected := []string{
		"\"message\":\"message bla bla Key1=Value1\"",
		"\"level\":\"DEBUG\"",
		"\"category\":\"test\"",
		"\"@version\":\"1\"",
		fmt.Sprintf("\"@timestamp\":\"%s\"", now.Format(logrusagent.TimeFormat)),
	}

	for _, exp := range expected {
		if !strings.Contains(string(res), exp) {
			t.Errorf("expected to have '%s' in '%s'", exp, string(res))
		}
	}
}
