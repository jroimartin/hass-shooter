package main

import (
	"log"
	"os"
)

// Logger is used for debug messages. If it is nil, logging is disabled.
var Logger *log.Logger

// logf logs a message if the global Logger is not nil.
func logf(format string, v ...interface{}) {
	if Logger == nil {
		return
	}
	Logger.Printf(format, v...)
}

// fatalf is equivalent to logf() followed by a call to os.Exit(1).
func fatalf(format string, v ...interface{}) {
	logf(format, v...)
	os.Exit(1)
}
