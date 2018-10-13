# lloger

Easy logs to CloudWatch.

## About

Used for sending logs to CloudWatch Logs in Lambda functions
written in golang.

## Usage

Just import the package and use `llogger.Create()` function to
create a new llogger. You can then use the `llogger.Print()` to print
logs. `llogger.Print` will only return error when one of the two
required fields (LogLevel and Message) are missing.
It will try and handle any other errors by doing it's best to
guess how to output the log as JSON.

Use the exported `llogger.Print` function with the exported `Input` struct.
`LogLevel` and `Message` is required. `RequestID`, `SourceIP`
and `UserAgent` is optional.

See example below.

```go
func handler(ctx context.Context) {
    l, err := llogger.Create(ctx, "service" ,"env")
    if err != nil {
        log.Fatal(err)
    }

    err := l.Print(&llogger.Input{
        LogLevel:  "critical",
        Text:      "This is just an example text",
        RequestID: "1234567890",
        SourceIP:  "127.0.0.1",
        UserAgent: "Example v1.0",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

The example above would result in an output to stdout that looks like

```json
{"loglevel":"critical","time":"2018-01-01 00:00:00.000001","message":"This is just an example text","service":"service","env":"env","request_id":"1234567890","source_ip":"127.0.0.1","user_agent":"Example v1.0","duration":0.000123,"time_left":2.999877,"resource":{"function":"main.main","file":"/go/src/github.com/nuttmeister/example/example.go","row":8}}
```

We use stdout for logging since all messages to stdout and stderr are sent to cloudwatch logs.

## Tests

To run package tests simple run.

```bash
go test
```

## Package return error

We only return an error if either `LogLevel` or `Message` is missing. Otherwise the
package will always try and find a way to print the message. If it encounters an error
it will try and handle it as well as output an critical logevel message about the
error it encounterd. So it can be searched in CloudWatch.

## Package error messages

This package can produce two different errors. Either way the original message sent to Print
will be printed to stdout. However two error messages can be sent to stdout before the invoked
message has been printed.

### Caller function couldn't be retrieved

If go can't get the caller (file, function, row etc) we will print a critical message before the invoked
message that looks like this. The `message` part of the message will then be set to `Couldn't get caller function`.

### Message couldn't get JSON Marshaled

If go can't JSON Marshal the Input we will print a critical message before the invoked message that
has the following `message` `Error unmarshalling JSON`.

