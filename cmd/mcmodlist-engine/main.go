// Command mcmodlist-engine serves the mcmodlist GUI protocol on stdin/stdout.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"mcmodlist/engine"
)

const maxLine = 1 << 20

func main() {
	eng := engine.New(engine.Options{})
	defer eng.Close()
	out := bufio.NewWriter(os.Stdout)
	emit := func(msg engine.Message) {
		data, err := json.Marshal(msg)
		if err != nil {
			return
		}
		_, _ = out.Write(data)
		_ = out.WriteByte('\n')
		_ = out.Flush()
	}
	reader := bufio.NewReaderSize(os.Stdin, 64*1024)
	for {
		line, err := readLine(reader)
		if len(line) > 0 {
			var req engine.Request
			if uerr := json.Unmarshal(line, &req); uerr != nil {
				emit(engine.Message{V: engine.ProtocolVersion, ID: 0, Error: &engine.ErrorBody{Code: engine.CodeInvalidRequest, Message: "malformed JSON request"}})
			} else {
				eng.Submit(req, emit)
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "mcmodlist-engine:", err)
			}
			return
		}
	}
}

func readLine(r *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		frag, err := r.ReadSlice('\n')
		buf = append(buf, frag...)
		if err == bufio.ErrBufferFull {
			if len(buf) > maxLine {
				for {
					_, e := r.ReadSlice('\n')
					if e == bufio.ErrBufferFull {
						continue
					}
					if e == io.EOF {
						return nil, io.EOF
					}
					break
				}
				return nil, fmt.Errorf("request line exceeds %d bytes", maxLine)
			}
			continue
		}
		if err != nil {
			return buf, err
		}
		if len(buf) > maxLine {
			return nil, fmt.Errorf("request line exceeds %d bytes", maxLine)
		}
		return buf, nil
	}
}
