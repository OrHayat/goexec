//go:build linux

package goexec

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var _ Backend = SystemdRunner{}

const defaultRunnerWorkdir = "go-exec-systemd"

func cleanupWorkdirExceptLock(workdir, lockPath string) error {
	entries, err := os.ReadDir(workdir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if filepath.Join(workdir, e.Name()) == lockPath {
			continue // skip lock file
		}
		if err := os.RemoveAll(filepath.Join(workdir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func NewSystemdRunner() (runner SystemdRunner, err error) {

	tmpDir := os.TempDir()
	workdir := filepath.Join(tmpDir, defaultRunnerWorkdir)

	//ensure workdir exists
	err = os.MkdirAll(workdir, 0700)
	if err != nil {
		return runner, err
	}
	fmt.Println("created workdir", workdir)

	lockPath := filepath.Join(workdir, ".lock")

	//open lockfile
	lockf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return runner, fmt.Errorf("open lock: %w", err)
	}
	fmt.Println("opened lockfile", lockf)

	defer func() {
		if err != nil {
			lockf.Close()
		}
	}()
	//lock the lockfile
	if err = syscall.Flock(int(lockf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return runner, fmt.Errorf("workdir already locked by another process")
	}
	fmt.Println("lockd lockfle")
	defer func() {
		if err != nil {
			syscall.Flock(int(lockf.Fd()), syscall.LOCK_UN)
		}
	}()
	// cleanup everything except lockfile
	if err = cleanupWorkdirExceptLock(workdir, lockPath); err != nil {
		return runner, fmt.Errorf("cleanup failed: %w", err)
	}
	fmt.Println("cleaned workdir")

	subProcess, err := NewSubprocessBackend(SubprocessBackendConfig{})
	if err != nil {
		return runner, err
	}
	return SystemdRunner{
		workDir:            workdir,
		subprocessExecutor: &subProcess,
		lockf:              lockf,
	}, nil
}

type SystemdRunner struct {
	workDir            string
	subprocessExecutor *SubprocessBackend
	lockf              *os.File
}

func (s SystemdRunner) RunCommand(ctx context.Context, cmd Command) Result {
	if cmd.Cmd == "" {
		return Result{Err: ErrEmptyCommand}
	}

	stdoutFile, err := os.CreateTemp(s.workDir, "*.stdout")
	if err != nil {
		return Result{Err: err}
	}
	defer os.Remove(stdoutFile.Name())

	stderrFile, err := os.CreateTemp(s.workDir, "*.stderr")
	if err != nil {
		return Result{Err: err}
	}
	defer os.Remove(stderrFile.Name())

	statusFile, err := os.CreateTemp(s.workDir, "*.exitcode")
	if err != nil {
		return Result{Err: err}
	}
	defer os.Remove(statusFile.Name())
	// // Whole block wrapped in subshell so redirection applies to all
	// script := fmt.Sprintf(`( %s ) >%s 2>%s; echo $? >%s`, cmd.Cmd, stdoutFile.Name(), stderrFile.Name(), statusFile.Name())
	// quotedScript := strconv.Quote(script)

	script := fmt.Sprintf(`( %s ) >%s 2>%s; echo $? >%s`, cmd.Cmd, stdoutFile.Name(), stderrFile.Name(), statusFile.Name())
	// s.subprocessExecutor.
	// cmd.Cmd = fmt.Sprintf("systemd-run  --wait --service-type=exec bash -c %s", script)
	// cmd.Cmd = fmt.Sprintf("systemd-run --wait --service-type-exec bash -c %s", cmd.Cmd)
	// script := cmd.Cmd
	// Use exec.CommandContext with program + args (do not pass the whole command-line as one string).
	execCmd := exec.CommandContext(ctx,
		"systemd-run",
		"--quiet",
		"--wait",
		"--service-type=exec",
		"bash",
		"-c",
		script,
	)
	err = execCmd.Run()
	if err != nil {
		return Result{
			Err: err,
		}
	}

	// cmd := exec.Command("systemd-run", "--quiet", "--wait", "--service-type=exec", "bash", "-c", script)
	// res := s.subprocessExecutor.runCommand(ctx, cmd)
	// if res.Err != nil {
	// 	return res
	// }
	var res Result
	stdout, err := os.ReadFile(stdoutFile.Name())
	if err != nil {
		if res.Err == nil {
			res.Err = fmt.Errorf("failed to read %s:%w", stdoutFile.Name(), err)
		} else {
			res.Err = fmt.Errorf("failed to read %s:%w:original error:%w", stdoutFile.Name(), err, res.Err)
		}
		return res
	}
	stderr, err := os.ReadFile(stderrFile.Name())
	if err != nil {
		if res.Err == nil {
			res.Err = fmt.Errorf("failed to read %s:%w", stderrFile.Name(), err)
		} else {
			res.Err = fmt.Errorf("failed to read %s:%w:original error:%w", stderrFile.Name(), err, res.Err)
		}
		return res
	}
	status, err := os.ReadFile(statusFile.Name())
	if err != nil {
		if res.Err == nil {
			res.Err = fmt.Errorf("failed to read %s:%w", statusFile.Name(), err)
		} else {
			res.Err = fmt.Errorf("failed to read %s:%w:original error:%w", statusFile.Name(), err, res.Err)
		}
		return res
	}
	exitCode := -1
	statusStr := strings.TrimSpace(string(status))
	if statusStr != "" {
		var parsedExitCode int
		parsedExitCode, err = strconv.Atoi(statusStr)
		if err == nil {
			exitCode = parsedExitCode
		}
	}
	return Result{
		Stdout:   string(stdout),
		Stderr:   string(stderr),
		ExitCode: exitCode,
		Err:      nil,
	}
}

func (s *SystemdRunner) Close() {
	if s.lockf != nil {
		_ = syscall.Flock(int(s.lockf.Fd()), syscall.LOCK_UN)
		_ = s.lockf.Close()
		s.lockf = nil
	}
}

func (s SystemdRunner) RunStream(ctx context.Context, cmd Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	return fmt.Errorf("stream command processing is not supported for systemd-runner")
}

/*
	outFile := "/tmp/out.log"
	errFile := "/tmp/err.log"
	statusFile := "/tmp/status.log"

	cmdStr := `echo "start"; echo "oops" >&2; sleep 1; echo "done"`

	// Whole block wrapped in subshell so redirection applies to all
	script := fmt.Sprintf(`( %s ) >%s 2>%s; echo $? >%s`, cmdStr, outFile, errFile, statusFile)

	cmd := exec.Command("systemd-run", "--quiet", "--wait", "--service-type=exec", "bash", "-c", script)

	if err := cmd.Run(); err != nil {
		fmt.Println("run err:", err)
	}

	dataOut, _ := os.ReadFile(outFile)
	dataErr, _ := os.ReadFile(errFile)
	status, _ := os.ReadFile(statusFile)

	fmt.Printf("OUT:\n%s\nERR:\n%s\nSTATUS:\n%s\n", dataOut, dataErr, status)

*/
