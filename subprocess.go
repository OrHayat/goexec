package goexec

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

var _ Backend = SubprocessBackend{}

type SubprocessBackend struct {
	*commandsMutex
}

type SubprocessBackendConfig struct {
	HighSlots   uint // max concurrent high-priority commands
	NormalSlots uint // max concurrent normal-priority commands
}

func NewSubprocessBackend(cfg SubprocessBackendConfig) (SubprocessBackend, error) {
	mu := newCommandsMutex(cfg.HighSlots, cfg.NormalSlots)
	return SubprocessBackend{
		commandsMutex: mu,
	}, nil

}

func (b SubprocessBackend) RunCommand(ctx context.Context, cmd Command) Result {
	slot, err := b.acquireSlot(ctx, cmd.HighPriority)
	if err != nil {
		return Result{Err: err}
	}
	defer b.releaseSlot(slot)
	return runSubprocessCommand(ctx, cmd.Cmd, cmd.Args...)
}

func (b SubprocessBackend) RunStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	slot, err := b.acquireSlot(ctx, cmd.HighPriority)
	if err != nil {
		return err
	}
	defer b.releaseSlot(slot)

	return runSubprocessStream(ctx, cmd.Cmd, cmd.Args, delim, callback)
}

func runSubprocessCommand(ctx context.Context, command string, args ...string) Result {
	cmd := exec.CommandContext(ctx, command, args...) // direct subprocess
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	exitCode := -1
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	} else {
		exitCode = 0
	}
	fmt.Println("ran command result", cmd.String(), "err", err)
	return Result{
		Stdout:   string(out),
		Stderr:   stderr.String(),
		Err:      err,
		ExitCode: exitCode,
	}
}

func runSubprocessStream(
	ctx context.Context,
	command string,
	args []string,
	delim bufio.SplitFunc,
	callback func(ctx context.Context, chunk string) error,
) (err error) {
	cmd := exec.CommandContext(ctx, command, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdoutPipe)
	if delim == nil {
		delim = bufio.ScanLines
	}
	scanner.Split(delim)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		err = callback(ctx, line)
		if err != nil {
			_ = cmd.Cancel()
			return err
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w:stderr: %s", err, stderr.String())
	}
	return nil

}
