package main

import (
	"bytes"
	"errors"
	"io"
)

func matchLine(qs [][]byte, line []byte) bool {
	for _, q := range qs {
		if !bytes.Contains(line, q) {
			return false
		}
	}
	return true
}

func matchInLines(qs [][]byte, lines [][]byte) bool {
	for _, line := range lines {
		if matchLine(qs, line) {
			return true
		}
	}
	return false
}

func grepProcessBlock(lines [][]byte, qsArr [][][]byte,
	out func(index int, lines [][]byte) error) (resErr error) {
	for i, qs := range qsArr {
		if matchInLines(qs, lines) {
			if err := out(i, lines); err != nil {
				resErr = err
			}
		}
	}
	return resErr
}

var (
	STOP      = errors.New("Stop")
	NEXT_FILE = errors.New("Next file")
)

func grep(qsArr [][][]byte, in io.Reader, out func(index int, lines [][]byte) error) error {
	var buffer []byte
	var lines [][]byte
	for {
		lineLen := -1
		for i := range buffer {
			if buffer[i] == '\n' {
				lineLen = i
				break
			}
		}

		if lineLen < 0 {
			// Didn't find a line feed, try read more data
			l := len(buffer)
			BUFFER := make([]byte, l+10*1024*1024)
			copy(BUFFER, buffer)

			n, err := in.Read(BUFFER[l:])
			if err != nil && err != io.EOF {
				return err
			}
			if n <= 0 {
				break
			}
			buffer = BUFFER[:l+n]
			continue
		}

		line := buffer[:lineLen]
println(line)
		if canParse(line) {
			if err := grepProcessBlock(lines, qsArr, out); err != nil {
				return err
			}

			lines = lines[:0]
		}
		lines = append(lines, line)
		buffer = buffer[lineLen+1:]
	}

	if len(buffer) > 0 {
		line := buffer
		if canParse(line) {
			if err := grepProcessBlock(lines, qsArr, out); err != nil {
				return err
			}

			lines = lines[:0]
		}
		lines = append(lines, line)
	}

	if len(lines) > 0 {
		if err := grepProcessBlock(lines, qsArr, out); err != nil {
			return err
		}
	}

	return nil
}
