package goexec

import "context"

// Backend defines the common interface for all execution backends
type Backend interface {
	// RunCommand executes a command and returns a Result struct
	RunCommand(ctx context.Context, cmd Command) Result

	// RunStream executes a command and streams output chunks via callback
	RunStream(ctx context.Context, cmd Command, delim byte, callback func(ctx context.Context, chunk string) error) error
}
