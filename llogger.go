// Package llogger simplifies printing messages to CloudWatch logs.
package llogger

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

const (
	baseService = "lloger"
	basePath    = "/go/src/github.com/nuttmeister/llogger"
)

// Client struct contains the state of the Client as well
// as channels for Warning and Critical time left until
// lambda deadline is reached.
type Client struct {
	start    time.Time
	deadline time.Time
	service  string
	env      string
	context  context.Context
	Warning  chan<- time.Duration
	Critical chan<- time.Duration
}

// Input is used by the Print function to print information
// to stdout in JSON format. RequestID, Source IP
// and UserAgent will be omitted if left empty.
type Input struct {
	Loglevel  string /* Required */
	Message   string /* Required */
	RequestID string
	SourceIP  string
	UserAgent string
}

type output struct {
	Loglevel  string   `json:"loglevel"`
	Time      string   `json:"time"`
	Message   string   `json:"message"`
	Service   string   `json:"service"`
	Env       string   `json:"env"`
	RequestID string   `json:"request_id,omitempty"`
	SourceIP  string   `json:"source_ip,omitempty"`
	UserAgent string   `json:"user_agent,omitempty"`
	Duration  float64  `json:"duration"`
	TimeLeft  float64  `json:"time_left"`
	Resource  resource `json:"resource,omitempty"`
}

type resource struct {
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	Row      int    `json:"row,omitempty"`
}

// Print takes inp and prints it as a JSON to stdout.
func (l *Client) Print(inp *Input) error {
	// If the required variables aren't set, just return doing nothing.
	switch {
	case inp.Loglevel == "":
		return fmt.Errorf("LogLevel must be set")

	case inp.Message == "":
		return fmt.Errorf("Message must be set")
	}

	out := &output{
		Loglevel:  inp.Loglevel,
		Time:      time.Now().UTC().Format("2006-01-02 15:04:05.999999"),
		Message:   inp.Message,
		Service:   l.service,
		Env:       l.env,
		RequestID: inp.RequestID,
		SourceIP:  inp.SourceIP,
		UserAgent: inp.UserAgent,
		Duration:  time.Now().Sub(l.start).Seconds(),
		TimeLeft:  l.deadline.Sub(time.Now()).Seconds(),
	}

	// Fetch the calling function filename and line.
	fptr, file, row, ok := runtime.Caller(1)

	// Create an outputs slices that all will be marshaled and printed.
	outputs := []*output{}

	switch {
	// If we couldn't get Caller print error.
	case !ok:
		outputs = []*output{&output{
			Loglevel: "error",
			Time:     time.Now().UTC().Format("2006-01-02 15:04:05.999999"),
			Message:  "Couldn't get caller function",
			Service:  baseService,
			Env:      l.env,
			Duration: time.Now().Sub(l.start).Seconds(),
			TimeLeft: l.deadline.Sub(time.Now()).Seconds(),
			Resource: resource{
				Function: fmt.Sprintf("%s.Print", basePath),
				File:     fmt.Sprintf("%s.go", basePath),
				Row:      87,
			},
		}}

	// Set Caller info.
	default:
		funcName := runtime.FuncForPC(fptr).Name()
		out.Resource = resource{
			Function: funcName,
			File:     file,
			Row:      row,
		}
	}

	// Add the original output to the outputs slice.
	// This is just so any added errors will appear before
	// the originating error message.
	outputs = append(outputs, out)
	for _, o := range outputs {
		raw, err := json.Marshal(o)

		switch {
		// If JSON Marshal fails print a error message about failing JSON Marshal.
		// And do a best effort of printing a JSON representation of the original message.
		case err != nil:
			// Best effort print a error message about JSON marshaling failing.
			l.bestEffortPrint(&output{
				Loglevel: "error",
				Time:     time.Now().UTC().Format("2006-01-02 15:04:05.999999"),
				Message:  "Error unmarshalling JSON",
				Service:  baseService,
				Env:      l.env,
				Duration: time.Now().Sub(l.start).Seconds(),
				TimeLeft: l.deadline.Sub(time.Now()).Seconds(),
				Resource: resource{
					Function: fmt.Sprintf("%s.Print", basePath),
					File:     fmt.Sprintf("%s.go", basePath),
					Row:      125,
				},
			})

			// Best effort print the message as a JSON text.
			l.bestEffortPrint(out)

		default:
			fmt.Printf("%s\n", raw)
		}
	}

	return nil
}

// bestEffortPrint is used to, in a best effort manner, print a message as JSON when
// we can't marshal it to JSON using the json package for some reason. It will try and
// remove any special characters from the string so that it becomes a valid JSON.
// There is however no guarantee that it will be valid.
func (l *Client) bestEffortPrint(out *output) {
	format := `{"loglevel":"%s","time":"%s","message":"%s","service":"%s","env":"%s"`
	format += `,"duration":%f,"time_left":%f,"resource":{"function":"%s","file":"%s","row":%d}}%s`

	fmt.Printf(format, out.Loglevel, out.Time, out.Message, out.Service, out.Env, out.Duration,
		out.TimeLeft, out.Resource.Function, out.Resource.File, out.Resource.Row, "\n")
}

// Create will start a timer that keeps track of the time left
// and will print a warning when 25% of Deadline is left. And
// a error message when 10% of the Deadline is left.
// Returns *Client.
func Create(ctx context.Context, service string, env string) (*Client, error) {
	switch {
	case ctx == nil:
		return nil, fmt.Errorf("ctx must be set")

	case service == "":
		return nil, fmt.Errorf("service must be set")

	case env == "":
		return nil, fmt.Errorf("env must be set")

	}

	l := &Client{
		start:   time.Now().UTC(),
		service: service,
		env:     env,
		context: ctx,

		Warning:  make(chan<- time.Duration),
		Critical: make(chan<- time.Duration),
	}

	d, ok := ctx.Deadline()
	switch {
	case !ok:
		return nil, fmt.Errorf("Couldn't get Deadline from context")

	default:
		l.deadline = d.UTC()
	}

	dur := l.deadline.Sub(l.start)
	w := time.Tick(dur * (3 / 4))
	c := time.Tick(dur * (9 / 10))

	// Wait for Warning.
	go func() {
		<-w
		l.Print(&Input{Loglevel: "warning", Message: "Only 25% of execution time left"})
		l.Warning <- l.deadline.Sub(time.Now())
	}()

	// Wait for Critical.
	go func() {
		<-c
		l.Print(&Input{Loglevel: "error", Message: "Only 10% of execution time left"})
		l.Critical <- l.deadline.Sub(time.Now())
	}()

	return l, nil
}
