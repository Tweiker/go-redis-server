package redis

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

func parseRequest(r *bufio.Reader) (*Request, error) {
	// first line of redis request should be:
	// *<number of arguments>CRLF
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	// note that this line also protects us from negative integers
	var argsCount int
	if _, err := fmt.Sscanf(line, "*%d\r", &argsCount); err != nil {
		if req, err := readInline(line); err != nil {
			return nil, malformed("*<numberOfArguments>", line)
		} else {
			return req, err
		}
	}

	// All next lines are pairs of:
	//$<number of bytes of argument 1> CR LF
	//<argument data> CR LF
	// first argument is a command name, so just convert
	firstArg, err := readArgument(r)
	if err != nil {
		return nil, err
	}

	args := make([][]byte, argsCount-1)
	for i := 0; i < argsCount-1; i += 1 {
		if args[i], err = readArgument(r); err != nil {
			return nil, err
		}
	}

	return &Request{name: strings.ToLower(string(firstArg)), args: args}, nil
}

func readInline(buf string) (*Request, error) {
	tab := strings.Split(strings.Trim(buf, "\r\n"), " ")

	var args [][]byte
	if len(tab) > 1 {
		for _, arg := range tab[1:] {
			args = append(args, []byte(arg))
		}
	}
	return &Request{name: strings.ToLower(string(tab[0])), args: args}, nil
}

func readArgument(r *bufio.Reader) ([]byte, error) {

	line, err := r.ReadString('\n')
	if err != nil {
		return nil, malformed("$<argumentLength>", line)
	}
	var argSize int
	if _, err := fmt.Sscanf(line, "$%d\r", &argSize); err != nil {
		return nil, malformed("$<argumentSize>", line)
	}

	// I think int is safe here as the max length of request
	// should be less then max int value?
	data, err := ioutil.ReadAll(io.LimitReader(r, int64(argSize)))
	if err != nil {
		return nil, err
	}

	if len(data) != argSize {
		return nil, malformedLength(argSize, len(data))
	}

	// Now check for trailing CR
	if b, err := r.ReadByte(); err != nil || b != '\r' {
		return nil, malformedMissingCRLF()
	}

	// And LF
	if b, err := r.ReadByte(); err != nil || b != '\n' {
		return nil, malformedMissingCRLF()
	}

	return data, nil
}

func malformed(expected string, got string) error {
	Debugf("Mailformed request:'%s does not match %s\\r\\n'", got, expected)
	return fmt.Errorf("Mailformed request:'%s does not match %s\\r\\n'", got, expected)
}

func malformedLength(expected int, got int) error {
	return fmt.Errorf(
		"Mailformed request: argument length '%d does not match %d\\r\\n'",
		got, expected)
}

func malformedMissingCRLF() error {
	return fmt.Errorf("Mailformed request: line should end with \\r\\n")
}
