package simulation

import (
	"io"
	"log"
)

var (
	debugLogger = log.New(io.Discard, "", 0)
	infoLogger  = log.New(io.Discard, "", 0)
)

func ConfigureLoggers(debug, info *log.Logger) {
	if debug != nil {
		debugLogger = debug
	}
	if info != nil {
		infoLogger = info
	}
}
