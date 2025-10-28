package goexec

import (
	"bufio"
	"context"
	"fmt"
)

var _ Backend = BashBackend{}

type BashBackend struct {
	subprocessExecutor *SubprocessBackend
}

//todo:improve constructor
func NewBashBackend() (BashBackend, error) {
	subproces, err := NewSubprocessBackend(SubprocessBackendConfig{})
	if err != nil {
		return BashBackend{}, err
	}
	return BashBackend{
		subprocessExecutor: &subproces,
	}, nil
}

func (b BashBackend) RunCommand(ctx context.Context, cmd Command) Result {
	if cmd.Cmd == "" {
		return Result{Err: ErrEmptyCommand}
	}
	cmd.Cmd = fmt.Sprintf("bash --noprofile --norc -c %s", cmd.Cmd)
	return b.subprocessExecutor.runCommand(ctx, cmd)
}

func (b BashBackend) RunStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	if cmd.Cmd == "" {
		return ErrEmptyCommand
	}
	cmd.Cmd = fmt.Sprintf("bash --noprofile --norc -c %s", cmd.Cmd)
	return b.subprocessExecutor.runStream(ctx, cmd, delim, callback)
}
