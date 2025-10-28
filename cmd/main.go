package main

import (
	"context"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/orhayat/goexec"
)

func main() {

	r, err := goexec.NewSystemdRunner()
	if err != nil {
		panic(err)
	}
	defer r.Close()

	res := r.RunCommand(context.TODO(), goexec.Command{
		Cmd: "lsblk",
	})
	spew.Dump(res)
	time.Sleep(time.Minute)

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
