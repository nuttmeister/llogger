// Package llogger simplifies printing messages to CloudWatch logs from AWS Lambda.
package llogger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

const fileName = "llogger_test.go"

type message1 struct {
	Time     int64    `json:"time"`
	Service  string   `json:"service"`
	Env      string   `json:"env"`
	Version  string   `json:"version"`
	LogLevel string   `json:"loglevel"`
	Message  string   `json:"message"`
	Duration float64  `json:"duration"`
	TimeLeft float64  `json:"timeLeft"`
	Resource resource `json:"resource"`
	Extra    string   `json:"extra"`
}

type message2 struct {
	Time     int64    `json:"custom-time"`
	Service  string   `json:"service"`
	Env      string   `json:"env"`
	Version  string   `json:"version"`
	LogLevel string   `json:"custom-loglevel"`
	Message  string   `json:"custom-message"`
	Duration float64  `json:"custom-duration"`
	TimeLeft float64  `json:"custom-timeLeft"`
	Resource resource `json:"custom-resource"`
}

type message3 struct {
	Time     string   `json:"time"`
	LogLevel string   `json:"loglevel"`
	Message  string   `json:"message"`
	Resource resource `json:"resource"`
}

var (
	startTime = time.Now().UTC()
	funcName1 = "github.com/nuttmeister/llogger.Test"
)

// Test will test all the coverage on the logger.go file.
func Test(t *testing.T) {
	// Create a context with time slightly after Test start.
	now := time.Now().UTC()
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Duration(3*time.Second)))

	// Create lloggers.
	client1 := Create(ctx, Input{
		"service":      "llogger-test",
		"env":          "test",
		"version":      "1.0.0",
		"llogger-tf":   "Unix",
		"llogger-tfn":  1,
		"llogger-llfn": 2,
		"llogger-mfn":  3,
		"llogger-dfn":  4,
		"llogger-tlfn": 5,
		"llogger-rfn":  6,
		"llogger-wm":   7,
		"llogger-cm":   8,
	})

	client2 := Create(nil, Input{
		"service":        "llogger-test",
		"env":            "test",
		"version":        "1.0.0",
		"llogger-tfn":    "custom-time",
		"llogger-tf":     "UnixNano",
		"llogger-llfn":   "custom-loglevel",
		"llogger-mfn":    "custom-message",
		"llogger-dfn":    "custom-duration",
		"llogger-tlfn":   "custom-timeLeft",
		"llogger-rfn":    "custom-resource",
		"llogger-wm":     "custom-warning",
		"llogger-cm":     "custom-error",
		"llogger-prefix": "prefix: ",
		"llogger-suffix": " suffix",
	})

	client3 := Create(nil, nil)

	client4 := Create(context.Background(), nil)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Couldn't create new Pipe files. Error %s", err.Error())
	}
	os.Stdout = w

	// Print 3 messages with the 3 different clients.
	client1.Print(Input{"loglevel": "verbose", "message": "Testmessage1", "extra": "extra test data"})
	client2.Print(Input{"custom-loglevel": "custom-warning", "custom-message": "Testmessage2"})
	client3.Print(Input{"loglevel": "error", "message": "Testmessage3"})
	client4.Print(Input{"this-should-fail": func() string { return "did-we-fail?" }})

	raw := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		raw <- buf.Bytes()
	}()
	w.Close()

	// Get result from stdout.
	strs := strings.Split(string(<-raw), "\n")

	// Check that strs has length of 4 and that last str is a blank line.
	switch {
	case len(strs) != 5:
		t.Fatalf("Expected slice length from stdout to be 5 but got %d", len(strs))

	case strs[4] != "":
		t.Fatalf("Exepected last slice string from stdout to be a blank str but got %s", strs[4])
	}

	// Test msg outputs
	msg1(strs[0], t)
	msg2(strs[1], t)
	msg3(strs[2], t)
	msg4(strs[3], t)

	cancel()
}

// Check that msg1 is correct.
func msg1(raw string, t *testing.T) {
	// Unmarshal Message
	msg := &message1{}
	if err := json.Unmarshal([]byte(raw), msg); err != nil {
		t.Fatalf("Couldn't unmarshal the message in msg1. Error %s", err.Error())
	}

	switch {
	// Check for correct loglevel.
	case msg.LogLevel != "verbose":
		t.Fatalf("loglevel in msg1 not error")

	// Check for correct message.
	case msg.Message != "Testmessage1":
		t.Fatalf("message in msg1 not Testmessage1")

	// Check that time.Now().UnixNano() is higher
	case time.Now().Unix() < msg.Time:
		t.Fatalf("time in msg1 is in the future")

	// Check for correct service.
	case msg.Service != "llogger-test":
		t.Fatalf("service in msg1 not llogger-test")

	// Check for correct env.
	case msg.Env != "test":
		t.Fatalf("env in msg1 not test")

	// Check for correct version.
	case msg.Version != "1.0.0":
		t.Fatalf("version in msg1 not 1.0.0")

	// Check filename of function.
	case !strings.Contains(msg.Resource.File, fileName):
		t.Fatalf("Expected Filename in msg1 to include %s but got %s", fileName, msg.Resource.File)

	// Check function name.
	case msg.Resource.Function != funcName1:
		t.Fatalf("Expected Function in msg1 to be %s but got %s", funcName1, msg.Resource.Function)

	// Check time left.
	case msg.TimeLeft < 2.9 || msg.TimeLeft > 3.0:
		t.Fatalf("Expected TimeLeft in msg1 to be between 2.9 and 3.0 seconds. But got %f", msg.TimeLeft)

	// Check Extra Data
	case msg.Extra != "extra test data":
		t.Fatalf("extra in msg1 not extra test data")
	}
}

// Check that msg2 is correct.
func msg2(raw string, t *testing.T) {
	// Unmarshal Message
	msg := &message2{}

	if err := json.Unmarshal([]byte(raw[8:len(raw)-7]), msg); err != nil {
		t.Fatalf("Couldn't unmarshal the message in msg2. Error %s", err.Error())
	}

	switch {
	// Check for correct prefix
	case raw[0:8] != "prefix: ":
		t.Fatalf("prefix wasn't 'prefix: ' in msg2")

	// Check for correct suffix.
	case raw[len(raw)-7:] != " suffix":
		t.Fatalf("suffix wasn't ' suffix' in msg2")

	// Check for correct loglevel.
	case msg.LogLevel != "custom-warning":
		t.Fatalf("loglevel in msg2 not custom-warning")

	// Check for correct message.
	case msg.Message != "Testmessage2":
		t.Fatalf("message in msg2 not Testmessage2")

	// Check that time.Now().UnixNano() is higher
	case time.Now().UnixNano() < msg.Time:
		t.Fatalf("time in msg2 is in the future")

	// Check for correct service.
	case msg.Service != "llogger-test":
		t.Fatalf("service in msg2 not llogger-test")

	// Check for correct env.
	case msg.Env != "test":
		t.Fatalf("env in msg2 not test")

	// Check for correct version.
	case msg.Version != "1.0.0":
		t.Fatalf("version in msg2 not 1.0.0")

	// Check filename of function.
	case !strings.Contains(msg.Resource.File, fileName):
		t.Fatalf("Expected Filename in msg2 to include %s but got %s", fileName, msg.Resource.File)

	// Check function name.
	case msg.Resource.Function != funcName1:
		t.Fatalf("Expected Function in msg2 to be %s but got %s", funcName1, msg.Resource.Function)
	}
}

// Check that msg3 is correct.
func msg3(raw string, t *testing.T) {
	// Unmarshal Message
	msg := &message3{}
	if err := json.Unmarshal([]byte(raw), msg); err != nil {
		t.Fatalf("Couldn't unmarshal the message in msg3. Error %s", err.Error())
	}

	// Parse the time in Message
	msgTime, err := time.Parse("2006-01-02 15:04:05.999999", msg.Time)
	if err != nil {
		t.Fatalf("Couldn't parse time in message in msg3. Error %s", err.Error())
	}

	switch {
	// Check for correct loglevel.
	case msg.LogLevel != "error":
		t.Fatalf("loglevel in msg3 not error")

	// Check for correct message.
	case msg.Message != "Testmessage3":
		t.Fatalf("message in msg3 not Testmessage3")

	// Check that time is after starttime.
	case msgTime.Before(startTime):
		t.Fatalf("Time in msg3 was before start time of test! Time: %s, Test start time: %s",
			msgTime.String(), startTime.String())

	// Check filename of function.
	case !strings.Contains(msg.Resource.File, fileName):
		t.Fatalf("Expected Filename in msg3 to include %s but got %s", fileName, msg.Resource.File)

	// Check function name.
	case msg.Resource.Function != funcName1:
		t.Fatalf("Expected Function in msg3 to be %s but got %s", funcName1, msg.Resource.Function)
	}
}

// Check that msg4 is correct.
func msg4(raw string, t *testing.T) {
	if !strings.Contains(raw, "Couldn't JSON marshal the error message") {
		t.Fatalf("Expected JSON Marshal to fail in msg4. But got %s", raw)
	}
}
