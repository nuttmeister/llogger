# llogger

Easy Go Lambda logs to CloudWatch Logs.

## Usage

Just import the package and use `llogger.Create()` function to
create a new llogger. You can then use the `llogger.Print()` to print
logs. `llogger.Print`.

Any values specified in `Input{}` when creating the llogger will be included
in all logs when issuing print.

Use the exported `llogger.Print` function with the exported `Input` struct.

See example below.

```go
package main

import (
    log "github.com/nuttmeister/llogger"
)
func handler(ctx context.Context) {
    l := log.Create(ctx, Input{"service": "myService", "env": "production", "llogger-llfn": "custom-loglevel"})

    l.Print(log.Input{
        "custom-loglevel":  "error",
        "message":          "We got an fatal error in the flux capacitor",
        "requestId":        "1337-1234567890",
        "sourceIp":         "127.0.0.1",
        "userAgent":        "FutureBrowser/2.0",
    })
}
```

The example above would result in an output to stdout that looks like

```json
{"custom-loglevel":"error","time":"2018-01-01 00:00:00.000001","message":"We got an fatal error in the flux capacitor","service":"myService","env":"production","requestId":"1337-1234567890","sourceIp":"127.0.0.1","userAgent":"FutureBrowser/2.0","duration":0.000123,"timeLeft":2.999877,"resource":{"function":"main.main","file":"/go/src/github.com/nuttmeister/example/example.go","row":8}}
```

We use stdout for logging since all messages to stdout and stderr are sent to cloudwatch logs.

## Overwriting standard field names

These standard field names are used by the logger `"time", "loglevel", "message", "duration", "timeLeft", "resource"`.  
However, these can all be overwritten by supplying the `Create` function with the following keys in the `Input{}` struct.

```text
time        llogger-tfn
loglevel    llogger-llfn
message     llogger-mfn
duration    llogger-dfn
timeLeft    llogger-tlfn
resource    llogger-rfn
```

## Overwriting internal log level messages

Internally we will sometimes need to print an error when for example Deadline() can't ge retrieved from the context
or when the Input can't be Marshaled to JSON. Or when timeLeft hits either 25% or 10% (Warning or Critical).

By default these loglevel messages are `"warning", "error"`, but by setting the keys below in the `Input{}` for the
`Create` function they can be overwritten.

```text
warning     llogger-wm
critical    llogger-cm
```

## Tests

To run package tests simple run.

```bash
go test
```

## Package error messages

This package can produce two different errors. Either way the original message sent to Print
will be printed to stdout. These messages are for when Deadline can't be determined from context
(indicating that it's not an context from an AWS Lambda function). Or when one of the values
supplied in `Input{}` can't be Marshaled to JSON.
