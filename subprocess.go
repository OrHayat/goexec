package goexec

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

var _ Backend = SubprocessBackend{}

type SubprocessBackend struct {
	highPriorityCommands   chan struct{}
	normalPriorityCommands chan struct{}
}

type SubprocessBackendConfig struct {
	HighSlots   int // max concurrent high-priority commands
	NormalSlots int // max concurrent normal-priority commands
}

func NewSubprocessBackend(cfg SubprocessBackendConfig) (SubprocessBackend, error) {
	if cfg.HighSlots > 0 && cfg.NormalSlots == 0 {
		cfg.NormalSlots = cfg.HighSlots
	}
	var highChan, normalChan chan struct{}
	if cfg.HighSlots > 0 {
		highChan = make(chan struct{}, cfg.HighSlots)
	}

	if cfg.NormalSlots > 0 {
		normalChan = make(chan struct{}, cfg.NormalSlots)
	}

	return SubprocessBackend{
		highPriorityCommands:   highChan,
		normalPriorityCommands: normalChan,
	}, nil
}

func (b SubprocessBackend) RunCommand(ctx context.Context, cmd Command) Result {
	if cmd.Cmd == "" {
		return Result{Err: ErrEmptyCommand}
	}
	return b.runCommand(ctx, cmd)
}

func (b SubprocessBackend) runCommand(ctx context.Context, cmd Command) Result {

	if err := b.acquireSlot(ctx, cmd.HighPriority); err != nil {
		return Result{Err: err}
	}
	defer b.releaseSlot(cmd.HighPriority)

	return b.runDirect(ctx, cmd.Cmd)
}

// RunStream for direct execution
func (b SubprocessBackend) RunStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	if cmd.Cmd == "" {
		return errors.New("empty command")
	}
	return b.runStream(ctx, cmd, delim, callback)
}

func (b SubprocessBackend) runStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	if err := b.acquireSlot(ctx, cmd.HighPriority); err != nil {
		return err
	}
	defer b.releaseSlot(cmd.HighPriority)

	return b.streamDirect(ctx, cmd.Cmd, delim, callback)
}

// ---------------- Semaphore helpers ----------------
func (b SubprocessBackend) acquireSlot(ctx context.Context, highPriority bool) error {
	if b.highPriorityCommands == nil && b.normalPriorityCommands == nil {
		return nil
	}
	if highPriority {
		select {
		case b.highPriorityCommands <- struct{}{}:
			return nil
		case b.normalPriorityCommands <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case b.normalPriorityCommands <- struct{}{}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (b SubprocessBackend) releaseSlot(highPriority bool) {
	if b.highPriorityCommands == nil && b.normalPriorityCommands == nil {
		return
	}
	if highPriority {
		select {
		case <-b.highPriorityCommands:
		default:
		}
	} else {
		select {
		case <-b.normalPriorityCommands:
		default:
		}
	}
}

// ---------------- Actual execution ----------------
func (b SubprocessBackend) runDirect(ctx context.Context, command string) Result {
	cmd := exec.CommandContext(ctx, command) // direct subprocess
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

	return Result{
		Stdout:   string(out),
		Stderr:   stderr.String(),
		Err:      err,
		ExitCode: exitCode,
	}
}

// streaming version
func (b SubprocessBackend) streamDirect(ctx context.Context, command string, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	cmd := exec.CommandContext(ctx, command)
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
