// Package llogger simplifies printing messages to CloudWatch logs.
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

const fileName = "github.com/nuttmeister/llogger/llogger_test.go"

// Test will test all the coverage on the logger.go file.
func Test(t *testing.T) {
	startTime := time.Now().UTC()
	funcName1 := "github.com/nuttmeister/llogger.Test"
	funcName2 := "TestBestEffortPrint"

	// Create a context with time slightly after Test start.
	now := time.Now().UTC()
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Duration(3*time.Second)))

	// Fail to create *Client.
	if _, err := Create(nil, "test", "test"); err.Error() != "ctx must be set" {
		t.Fatalf("Expected error to be '%s' when ctx wasn't set but got '%s'", "ctx must be set", err.Error())
	}
	if _, err := Create(ctx, "", "test"); err.Error() != "service must be set" {
		t.Fatalf("Expected error to be '%s' when service wasn't set but got '%s'", "service must be set", err.Error())
	}
	if _, err := Create(ctx, "test", ""); err.Error() != "env must be set" {
		t.Fatalf("Expected error to be '%s' when env wasn't set but got '%s'", "env must be set", err.Error())
	}
	if _, err := Create(context.Background(), "test", "test"); err.Error() != "Couldn't get Deadline from context" {
		t.Fatalf("Expected error to be '%s' when ctx lacks Deadline but got '%s'",
			"Couldn't get Deadline from context", err.Error())
	}

	// Create logger.
	l, err := Create(ctx, "shplss/common/logger", "test")
	if err != nil {
		t.Fatalf("Couldn't create Client. Error %s", err.Error())
	}

	// Try to make prints that should return error.
	if err := l.Print(&Input{Message: "Testmessage"}); err.Error() != "LogLevel must be set" {
		t.Fatalf("Expected error to be '%s' when LogLevel wasn't set but got '%s'", "LogLevel must be set", err.Error())
	}
	if err := l.Print(&Input{Loglevel: "error"}); err.Error() != "Message must be set" {
		t.Fatalf("Expected error to be '%s' when Message wasn't set but got '%s'", "Message must be set", err.Error())
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Couldn't create new Pipe files. Error %s", err.Error())
	}
	os.Stdout = w

	// Message 1, successfull error message.
	if err := l.Print(&Input{Loglevel: "error", Message: "Testmessage"}); err != nil {
		t.Fatalf(err.Error())
	}
	// Message 2, test the best effort printing mechanic when an error has occured.
	l.bestEffortPrint(&output{
		Loglevel: "error",
		Time:     now.Format("2006-01-02 15:04:05.999999"),
		Message:  "Testmessage",
		Service:  "shplss/common/logger",
		Env:      "test",
		Duration: time.Duration(50 * time.Millisecond).Seconds(),
		TimeLeft: time.Duration(2950 * time.Millisecond).Seconds(),
		Resource: resource{
			Function: "TestBestEffortPrint",
			File:     fileName,
			Row:      0,
		},
	})

	raw := make(chan []byte)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		raw <- buf.Bytes()
	}()
	w.Close()

	// Get result from stdout.
	strs := strings.Split(string(<-raw), "\n")

	// Check that strs has length of 3 and that last str is a blank line.
	switch {
	case len(strs) != 3:
		t.Fatalf("Expected slice length from stdout to be 3 but got %d", len(strs))

	case strs[2] != "":
		t.Fatalf("Exepected last slice string from stdout to be a blank str but got %s", strs[2])
	}

	// Unmarshal Message 1
	msg1 := &output{}
	if err := json.Unmarshal([]byte(strs[0]), msg1); err != nil {
		t.Fatalf("Couldn't unmarshal the message 1. Error %s", err.Error())
	}

	// Unmarshal Message 2
	msg2 := &output{}
	if err := json.Unmarshal([]byte(strs[1]), msg2); err != nil {
		t.Fatalf("Couldn't unmarshal the message 2. Error %s", err.Error())
	}

	// Parse the time in Message 1
	msg1Time, err := time.Parse("2006-01-02 15:04:05.999999", msg1.Time)
	if err != nil {
		t.Fatalf("Couldn't parse time in message 1. Error %s", err.Error())
	}

	// Parse the time in Message 2
	msg2Time, err := time.Parse("2006-01-02 15:04:05.999999", msg2.Time)
	if err != nil {
		t.Fatalf("Couldn't parse time in message 2. Error %s", err.Error())
	}

	// Check that we have the correct values in msg1 and msg2.
	switch {
	// Check for correct loglevel.
	case msg1.Loglevel != "error":
		t.Fatalf("loglevel in msg1 not error")
	case msg2.Loglevel != "error":
		t.Fatalf("loglevel in msg2 not error")

	// Check for correct message.
	case msg1.Message != "Testmessage":
		t.Fatalf("message in msg1 not Testmessage")
	case msg2.Message != "Testmessage":
		t.Fatalf("message in msg2 not Testmessage")

	// Check that time is after starttime.
	case msg1Time.Before(startTime):
		t.Fatalf("Time in msg1 was before start time of test! Msg1 time: %s, Test start time: %s",
			msg1Time.String(), startTime.String())
	case msg2Time.Before(startTime):
		t.Fatalf("Time in msg2 was before start time of test! Msg2 time: %s, Test start time: %s",
			msg2Time.String(), startTime.String())

	// Check filename of function.
	case !strings.Contains(msg1.Resource.File, fileName):
		t.Fatalf("Expected Filename in msg1 to include %s but got %s", fileName, msg1.Resource.File)
	case !strings.Contains(msg2.Resource.File, fileName):
		t.Fatalf("Expected Filename in msg2 to include %s but got %s", fileName, msg2.Resource.File)

	// Check function name.
	case msg1.Resource.Function != funcName1:
		t.Fatalf("Expected Function in msg1 to be %s but got %s", funcName1, msg1.Resource.Function)
	case msg2.Resource.Function != funcName2:
		t.Fatalf("Expected Function in msg2 to be %s but got %s", funcName2, msg2.Resource.Function)

	// Check time left.
	case msg1.TimeLeft < 2.9 || msg1.TimeLeft > 3.0:
		t.Fatalf("Expected TimeLeft in msg1 to be between 2.9 and 3.0 seconds. But got %f", msg1.TimeLeft)
	case msg2.TimeLeft < 2.9 || msg2.TimeLeft > 3.0:
		t.Fatalf("Expected TimeLeft in msg2 to be between 2.9 and 3.0 seconds. But got %f", msg2.TimeLeft)
	}

	cancel()
}
