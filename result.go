package goexec

// Result holds the output and error of a command execution
type Result struct {
	Stdout   string
	Stderr   string
	Err      error
	ExitCode int // -1 if unknown
}
