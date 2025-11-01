package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/orhayat/goexec"
)

var _ goexec.Backend = mockBackend{}

type mockBackend struct{}

// RunCommand implements goexec.Backend.
func (m mockBackend) RunCommand(ctx context.Context, cmd goexec.Command) goexec.Result {
	fmt.Printf("cmd=%q args=%q\n", cmd.Cmd, cmd.Args)
	// args := []string{"-c", cmd.Cmd}
	// args = append(args, cmd.Args...)
	// cm := exec.CommandContext(ctx, "bash", args...)
	script := strings.Join(append([]string{cmd.Cmd}, cmd.Args...), " ")
	args := []string{"-c", script}
	cm := exec.CommandContext(ctx, "bash", args...)
	var stdout bytes.Buffer
	var stder bytes.Buffer
	cm.Stdout = &stdout
	cm.Stderr = &stder
	err := cm.Run()
	if err != nil {
		return goexec.Result{Err: err}
	}
	fmt.Println("ramming cmd done", cmd.Cmd, "args", cmd.Args)
	return goexec.Result{
		Stdout: stdout.String(),
		Stderr: stder.String(),
	}
}

// RunStream implements goexec.Backend.
func (m mockBackend) RunStream(ctx context.Context, cmd goexec.Command, delim bufio.SplitFunc, callback func(ctx context.Context, chunk string) error) error {
	panic("unimplemented")
}
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
func main() {

	r := goexec.NewExecutor(must(goexec.NewBashBackend(goexec.BashBackendConfig{})))

	// res := r.RunW(context.TODO(), "powershell.exe", "-c", "ls") //, "-a") // "2>", 5, true, "/dev/null")
	res := r.RunW(context.TODO(), "echo", "hello world;echo deleted files")
	// res := r.RunW(context.TODO(), "echo hello world > ./test.txt")

	fmt.Printf("res: %#v\n", res)

	// Handle the result as needed
	if res.Err != nil {
		fmt.Printf("error: %v\n", res.Err)
		return
	}
	fmt.Printf("stdout: %q\n", res.Stdout)
	fmt.Printf("stderr: %q\n", res.Stderr)
	res = r.RunW(context.TODO(), "echo", "test")

	fmt.Printf("stdout: %q\n", res.Stdout)
	fmt.Printf("stderr: %q\n", res.Stderr)

}

// func main() {
// 	xx := []byte("thisi\nsaverylongoutputwithoutanydelimiter")
// 	delim := byte('\n')
// 	callback := func(ch string) {
// 		fmt.Printf("chunk: %q\n", ch)
// 	}
// 	r := strings.NewReader(string(xx))
// 	buf := make([]byte, 3)
// 	var leftover []byte

// 	for {
// 		n, err := r.Read(buf)
// 		if n > 0 {
// 			data := append(leftover, buf[:n]...)
// 			chunks := bytes.Split(data, []byte{delim})

// 			// last chunk might be incomplete
// 			leftover = chunks[len(chunks)-1]

// 			for _, ch := range chunks[:len(chunks)-1] {
// 				if len(ch) == 0 {
// 					continue
// 				}
// 				// if err := callback( string(ch)); err != nil {
// 				callback(string(ch))
// 			}

// 			// leftover = nil
// 			// iter := bytes.Split(data, []byte{delim})
// 			// for ch := range iter {
// 			// 	if len(ch) == 0 {
// 			// 		continue
// 			// 	}
// 			// 	callback(string(ch))
// 			// }
// 		}
// 		if err != nil {
// 			if err == io.EOF {
// 				break
// 			}
// 			panic(err)
// 		}
// 	}
// 	if len(leftover) > 0 {
// 		callback(string(leftover))
// 	}

// }
