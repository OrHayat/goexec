package goexec

import (
	"bufio"
	"context"
	"strings"
)

var _ Backend = BashBackend{}

type BashBackend struct {
	*SubprocessBackend
}

type BashBackendConfig struct {
	HighSlots   uint // max concurrent high-priority commands
	NormalSlots uint // max concurrent normal-priority commands
}

//todo:improve constructor
func NewBashBackend(cfg BashBackendConfig) (BashBackend, error) {

	SubprocessBackend, err := NewSubprocessBackend(SubprocessBackendConfig(cfg))
	if err != nil {
		return BashBackend{}, err
	}
	return BashBackend{SubprocessBackend: &SubprocessBackend}, nil
}

func (b BashBackend) RunCommand(ctx context.Context, cmd Command) Result {
	slot, err := b.commandsMutex.acquireSlot(ctx, cmd.HighPriority)
	if err != nil {
		return Result{Err: err}
	}
	defer b.commandsMutex.releaseSlot(slot)
	preparedCmd := []string{cmd.Cmd}
	preparedCmd = append(preparedCmd, cmd.Args...)

	args := []string{"--noprofile", "--norc", "-c", strings.Join(preparedCmd, " ")}
	return runSubprocessCommand(ctx, "bash", args...)
}

func (b BashBackend) RunStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	slot, err := b.commandsMutex.acquireSlot(ctx, cmd.HighPriority)
	if err != nil {
		return err
	}
	defer b.commandsMutex.releaseSlot(slot)
	preparedCmd := []string{cmd.Cmd}
	preparedCmd = append(preparedCmd, cmd.Args...)

	args := []string{"--noprofile", "--norc", "-c", strings.Join(preparedCmd, " ")}
	return runSubprocessStream(ctx, "bash", args, delim, callback)
}
