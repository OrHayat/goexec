package goexec

import (
	"bytes"
	"context"
	"errors"
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

	if err := b.acquireSlot(ctx, cmd.HighPriority); err != nil {
		return Result{Err: err}
	}
	defer b.releaseSlot(cmd.HighPriority)

	return b.runDirect(ctx, cmd.Cmd)
}

// RunStream for direct execution
func (b SubprocessBackend) RunStream(ctx context.Context, cmd Command, delim byte, callback func(ctx context.Context, chunk string) error) error {
	if cmd.Cmd == "" {
		return errors.New("empty command")
	}

	if err := b.acquireSlot(ctx, cmd.HighPriority); err != nil {
		return err
	}
	defer b.releaseSlot(cmd.HighPriority)

	return b.streamDirect(ctx, cmd.Cmd, delim, callback)
}

// ---------------- Semaphore helpers ----------------
func (b SubprocessBackend) acquireSlot(ctx context.Context, high bool) error {
	if b.highPriorityCommands == nil && b.normalPriorityCommands == nil {
		return nil
	}
	if high {
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

func (b SubprocessBackend) releaseSlot(high bool) {
	if b.highPriorityCommands == nil && b.normalPriorityCommands == nil {
		return
	}
	if high {
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
	c := exec.CommandContext(ctx, command) // direct subprocess
	var stderr bytes.Buffer
	c.Stderr = &stderr
	out, err := c.Output()

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
func (b SubprocessBackend) streamDirect(ctx context.Context, command string, delim byte, callback func(ctx context.Context, chunk string) error) error {
	c := exec.CommandContext(ctx, command)
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	c.Stderr = &stderr

	if err := c.Start(); err != nil {
		return err
	}

	buf := make([]byte, 1024)
	for {
		n, err := stdoutPipe.Read(buf)
		if n > 0 {
			chunks := bytes.SplitSeq(buf[:n], []byte{delim})
			for ch := range chunks {
				if len(ch) == 0 {
					continue
				}
				if err := callback(ctx, string(ch)); err != nil {
					c.Process.Kill()
					return err
				}
			}
		}
		if err != nil {
			break
		}
	}

	return c.Wait()
}
