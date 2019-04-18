// Package llogger simplifies printing messages to CloudWatch logs from AWS Lambda.
package llogger

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// var (
// 	w = time.Duration(0)
// 	c = time.Duration(0)
// )

// Client struct contains the state of the Client as well
// as channels for Warning and Critical time left until
// lambda deadline is reached.
type Client struct {
	data     Input
	context  context.Context
	start    time.Time
	deadline time.Time
	// w        time.Duration
	// c        time.Duration

	// The field names for loglevel, message, duration,
	// time left and resource field names. Can be changed
	// by setting llogger tfn-, llogger-llfn, llogger-mfn,
	// llogger-dur, llogger-tl and llogger-res keys
	// respectaviley in the inp when creating the client.
	// If not set it will default to loglevel, message,
	// duration, timeLeft and resource.
	tfn  string // time fieldname
	llfn string // loglevel fieldname
	mfn  string // message fieldname
	dfn  string // duration fieldname
	tlfn string // time left fieldname
	rfn  string // resource fieldname

	// Prefix and suffixes
	pre string // Prefix
	suf string // Suffix

	// The warning and critical log levels. Can be
	// set by setting the llogger-wm and llogger-cm
	// keys in inp when creating the client.
	// If not set it will default to warning and
	// error.
	wm string // warning log level message
	cm string // critical log level message

	// The format used for the time field.
	// Defaults to 2006-01-02 15:04:05.999999
	// and can be overwritten with llogger-tf
	// in Input.
	tf string // Time format to use

	// Warning  chan<- time.Duration
	// Critical chan<- time.Duration
}

// Input is used by the Print function to print information
// to stdout in JSON format. The JSON field will be called
// exactly as the name of the keys supplied.
type Input map[string]interface{}

type output map[string]interface{}

type resource struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Row      int    `json:"row"`
}

// Print takes inp and prints it as a JSON to stdout.
// All fields left empty will be omitted in the JSON output.
// If ctx was set to nil in *Client Duration and TimeLeft will
// not be set.
func (l *Client) Print(inp Input) {
	// Creates a basic output that merges data form l and inp.
	out := l.createOutput(inp)

	// Fetch and set the calling function filename and line.
	// This call will never fail since skip is 1 and there
	// is always a caller. So skip ok variable.
	fptr, file, row, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(fptr).Name()
	out[l.rfn] = resource{
		Function: funcName,
		File:     file,
		Row:      row,
	}

	raw, err := json.Marshal(out)
	switch {
	// If JSON Marshal fails print a error message about failing JSON Marshal.
	// Don't print the original error message since it probably contains not so
	// good data that possibly could break other things.
	case err != nil:
		l.Print(Input{l.llfn: l.cm, l.mfn: "Couldn't JSON marshal the error message"})

	default:
		fmt.Printf("%s%s%s\n", l.pre, raw, l.suf)
	}
}

// createOutput will return output that contains the
// merged data from l.data and inp. If l.context is
// set duration and time_left will also be set based
// on data from the lambda context.
// Returns output.
func (l *Client) createOutput(inp Input) output {
	out := output{}

	switch l.tf {
	case "Unix":
		out[l.tfn] = time.Now().Unix()

	case "UnixNano":
		out[l.tfn] = time.Now().UnixNano()

	default:
		out[l.tfn] = time.Now().Format(l.tf)
	}

	// Merge Input from l and Input.
	for k, v := range l.data {
		out[k] = v
	}
	for k, v := range inp {
		out[k] = v
	}

	// Set duration and time_left if context is set.
	if l.context != nil {
		out[l.dfn] = time.Now().Sub(l.start).Seconds()
		out[l.tlfn] = l.deadline.Sub(time.Now()).Seconds()
	}

	return out
}

// Create takes context ctx and Input inp and creates a llogger client. The llogger
// client can then be used to print JSON messages to CloudWatch logs.
// ctx should be a valid context created by AWS Lambda. If set to nil all additional
// functionality that requires context will be disabled such as getting lambda duration,
// time left. The channels for time left warnings will also never trigger.
// All data specified in inp will be added to each message sent by the client.
// You can also specify the "log level" and "message" field name by adding the special
// variables llfn for "log level field name" anf mfn "message field name" to inp.
// If context as set and as a valid AWS Lambda context there will be events on the
// l.Warning and l.Critical channels when the lambda detects that only 25% and 10%
// respectively of runtime is left before it will self terminate.
// Returns *Client.
func Create(ctx context.Context, inp Input) *Client {
	l := &Client{
		data:    inp,
		start:   time.Now().UTC(),
		context: ctx,
	}

	// Set the loglevel and message field names.
	l.setFieldNames()

	// Set the warning and critical error messages..
	l.setErrorMessages()

	// Set the format to use for time.
	l.setTimeFormat()

	// If context is nil we can just return the *Client.
	// This is so we support using this logger without
	// having to use the context from lambda.
	// In most cases the context can and should
	// be included. There is practically no overhead
	// for using it.
	if l.context == nil {
		return l
	}

	// If we can't get Deadline from context set context to nil and
	// print an error message.
	d, ok := l.context.Deadline()
	switch {
	case !ok:
		l.context = nil
		l.Print(Input{l.llfn: l.cm, l.mfn: "Couldn't get Deadline from context"})
		return l

	default:
		l.deadline = d.UTC()
	}

	// Set duration, warning and critical levels.
	// And create the channels for sending messages
	// back to the calling function.
	// dur := l.deadline.Sub(l.start)

	// w = 0
	// c = 0

	// w = dur * 3 / 4
	// c = dur * 9 / 19

	// fmt.Println("w", l.w)
	// fmt.Println("c", l.c)

	// l.Warning = make(chan<- time.Duration)
	// l.Critical = make(chan<- time.Duration)

	// fmt.Println(runtime.NumGoroutine())

	// go l.warning(w)
	// go l.critical(c)

	return l
}

// func (l *Client) Close() {
// 	l.
// }

// func (l *Client) warning(w time.Duration) {
// 	select {
// 	default:
// 		time.Sleep(time.Duration(100*time.Millisecond))
// 	}

// 	time.Sleep(w)
// 	l.Print(Input{l.llfn: l.wm, l.mfn: "Only 25% of execution time left"})
// 	l.Warning <- l.deadline.Sub(time.Now())
// }

// func (l *Client) critical(c time.Duration) {
// 	time.Sleep(c)
// 	l.Print(Input{l.llfn: l.cm, l.mfn: "Only 10% of execution time left"})
// 	l.Critical <- l.deadline.Sub(time.Now())
// }

// setFieldNames will set the default key names for the log level and message
// field. If not specified by env variables it will default to "loglevel"
// and "message".
func (l *Client) setFieldNames() {
	// Try and get Time Field Name from l.data as a string.
	if tfn, ok := l.data["llogger-tfn"]; ok {
		if str, ok := tfn.(string); ok {
			l.tfn = str
		}
		delete(l.data, "llogger-tfn")
	}

	// Try and get Log Level Field Name from l.data as a string.
	if llfn, ok := l.data["llogger-llfn"]; ok {
		if str, ok := llfn.(string); ok {
			l.llfn = str
		}
		delete(l.data, "llogger-llfn")
	}

	// Try and get Message Field Name from l.data as a string.
	if mfn, ok := l.data["llogger-mfn"]; ok {
		if str, ok := mfn.(string); ok {
			l.mfn = str
		}
		delete(l.data, "llogger-mfn")
	}

	// Try and get Duration Field Name from l.data as a string.
	if dfn, ok := l.data["llogger-dfn"]; ok {
		if str, ok := dfn.(string); ok {
			l.dfn = str
		}
		delete(l.data, "llogger-dfn")
	}

	// Try and get Time Left Field Name from l.data as a string.
	if tlfn, ok := l.data["llogger-tlfn"]; ok {
		if str, ok := tlfn.(string); ok {
			l.tlfn = str
		}
		delete(l.data, "llogger-tlfn")
	}

	// Try and get Resource Field Name from l.data as a string.
	if rfn, ok := l.data["llogger-rfn"]; ok {
		if str, ok := rfn.(string); ok {
			l.rfn = str
		}
		delete(l.data, "llogger-rfn")
	}

	// Add prefix to output if supplied.
	if pre, ok := l.data["llogger-prefix"]; ok {
		if str, ok := pre.(string); ok {
			l.pre = str
		}
		delete(l.data, "llogger-prefix")
	}

	// Add suffix to output if supplied.
	if suf, ok := l.data["llogger-suffix"]; ok {
		if str, ok := suf.(string); ok {
			l.suf = str
		}
		delete(l.data, "llogger-suffix")
	}

	// Check that Log Level and Message is not empty. If they are empty
	// default to field names "time", loglevel", "message", "duration",
	// "timeLeft" and "resource".
	if l.tfn == "" {
		l.tfn = "time"
	}
	if l.llfn == "" {
		l.llfn = "loglevel"
	}
	if l.mfn == "" {
		l.mfn = "message"
	}
	if l.dfn == "" {
		l.dfn = "duration"
	}
	if l.tlfn == "" {
		l.tlfn = "timeLeft"
	}
	if l.rfn == "" {
		l.rfn = "resource"
	}
}

// setErrorMessages will set the default log level warning and error messages
// If not specified by env variables it will default to "warning"
// and "error".
func (l *Client) setErrorMessages() {
	// Try and get Warning Message from l.data as a string.
	if wm, ok := l.data["llogger-wm"]; ok {
		if str, ok := wm.(string); ok {
			l.wm = str
		}
		delete(l.data, "llogger-wm")
	}

	// Try and get Critical Message from l.data as a string.
	if cm, ok := l.data["llogger-cm"]; ok {
		if str, ok := cm.(string); ok {
			l.cm = str
		}
		delete(l.data, "llogger-cm")
	}

	// Check that Warning and Critical Messages are not empty. If they are empty
	// default to field names "warning" and "error".
	if l.wm == "" {
		l.wm = "warning"
	}
	if l.cm == "" {
		l.cm = "error"
	}
}

// setTimeFormat will set the format to use for showing "time". Will default
// to "2006-01-02 15:04:05.999999". All golang time formats can be used.
// For list and manual parse see https://golang.org/src/time/format.go
func (l *Client) setTimeFormat() {
	// Try and get Warning Message from l.data as a string.
	if tf, ok := l.data["llogger-tf"]; ok {
		if str, ok := tf.(string); ok {
			l.tf = str
		}
		delete(l.data, "llogger-tf")
	}

	// Check that format was set. If empty set to default
	// 2006-01-02 15:04:05.999999.
	if l.tf == "" {
		l.tf = "2006-01-02 15:04:05.999999"
	}
}
