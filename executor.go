package goexec

import (
	"bufio"
	"context"
	"fmt"
	"strconv"

	"al.essio.dev/pkg/shellescape"
)

// Executor provides a convenient wrapper around Backend for easier command execution
type Executor struct {
	runner Backend
}

// NewExecutor creates a new Executor with the given backend
func NewExecutor(backend Backend) *Executor {
	return &Executor{runner: backend}
}

// RunW executes a command with variadic arguments, automatically converting different types to strings
// Usage: executor.RunW(ctx, "ls", "-la", "/tmp")
//
//	executor.RunW(ctx, "echo", "port:", 8080, "enabled:", true)
//	executor.RunW(ctx, "ls", "-la", "/tmp", ">", "output.txt")
func (e *Executor) RunW(ctx context.Context, cmd string, args ...any) Result {
	if cmd == "" {
		return Result{Err: ErrEmptyCommand}
	}

	argsStr := make([]string, len(args))
	for i, arg := range args {
		argStr, err := e.convertToString(arg)
		if err != nil {
			return Result{Err: err}
		}
		argsStr[i] = argStr
	}
	fmt.Println("Args=", argsStr)
	return e.runner.RunCommand(ctx, Command{Cmd: cmd, Args: argsStr})
}

// RunWStream executes a streaming command with variadic arguments
func (e *Executor) RunWStream(ctx context.Context, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error, cmd string, args ...any) error {
	if cmd == "" {
		return ErrEmptyCommand
	}
	argsStr := make([]string, len(args))
	for i, arg := range args {
		argStr, err := e.convertToString(arg)
		if err != nil {
			return err
		}
		argsStr[i] = argStr
	}
	return e.runner.RunStream(ctx, Command{Cmd: cmd, Args: argsStr}, delim, callback)
}

// RunCommand provides direct access to the underlying backend's RunCommand method
func (e *Executor) RunCommand(ctx context.Context, cmd Command) Result {
	return e.runner.RunCommand(ctx, cmd)
}

// RunStream provides direct access to the underlying backend's RunStream method
func (e *Executor) RunStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	return e.runner.RunStream(ctx, cmd, delim, callback)
}

// convertToString converts various types to their string representation
func (e *Executor) convertToString(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return shellescape.Quote(v), nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("unsupported argument type: %T", val)
	}
}
