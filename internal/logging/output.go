package logging

import (
	"io"
	"os"
)

// Out and Err are the destinations for verbose log lines. They default to the
// process stdout/stderr and may be overridden (e.g. in tests) to capture output.
var (
	Out io.Writer = os.Stdout
	Err io.Writer = os.Stderr
)
