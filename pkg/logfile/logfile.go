package logfile

import (
	"fmt"
	"os"
	"time"
)

const logFile = "error.log"

// Append writes the provided error to error.log in the current working directory.
func Append(err error) {
	f, ferr := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if ferr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %v\n", time.Now().Format(time.RFC3339), err)
}
