package loggedexec

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/reconquest/go-callbackwriter"
	"github.com/reconquest/go-lineflushwriter"
	"github.com/reconquest/go-nopio"
	"github.com/zazab/hierr"
)

// Executiion represents command prepared for the run.
type Execution struct {
	command *exec.Cmd

	stdin  io.ReadWriteCloser
	stdout io.ReadWriter
	stderr io.ReadWriter

	logger Logger

	closer func()
}

// Stream represents execution output stream.
type Stream string

const (
	// Stdout is ID for execution stdout.
	Stdout Stream = `stdout`

	// Stdout is ID for execution stderr.
	Stderr Stream = `stderr`
)

// Logger represents type of function, which is considered logger by `New`.
type Logger func(command []string, stream Stream, data []byte)

// Callbackf will turn typical Somethingf() logger function into acceptible
// Logger function.
func Loggerf(logger func(string, ...interface{})) Logger {
	return func(command []string, stream Stream, data []byte) {
		logger(`<%s> {%s} %s`, stream, command[0], string(data))
	}
}

// New creates new execution object, that is used to start command and setup
// stdout/stderr/stdin streams.
//
// stdout/stderr will be duplicated to specified logger. Each logged line will be
// prefixed with `<stdXXX> {command} `. Prefix can be overrided via likely
// named methods.
//
// Further arguments are symmetric to `exec.Command`.
func New(
	logger Logger,
	name string,
	args ...string,
) *Execution {
	execution := &Execution{
		command: exec.Command(name, args...),

		logger: logger,
	}

	execution.stdout = &bytes.Buffer{}
	execution.stderr = &bytes.Buffer{}

	return execution
}

// SetStdout sets writer to store stdout.
//
// If not called, internal buffer will be used.
func (execution *Execution) SetStdout(target io.Writer) {
	execution.stdout = struct {
		io.Reader
		io.Writer
	}{
		Writer: target,
	}
}

// SetStderr sets writer to store stderr.
//
// If not called, internal buffer will be used.
func (execution *Execution) SetStderr(target io.Writer) {
	execution.stderr = struct {
		io.Reader
		io.Writer
	}{
		Writer: target,
	}
}

// GetStdout returns reader which is linked to the program stdout.
func (execution *Execution) GetStdout() io.Reader {
	return execution.stdout.(io.Reader)
}

// GetStderr returns reader which is linked to the program stderr.
func (execution *Execution) GetStderr() io.Reader {
	return execution.stderr.(io.Reader)
}

// GetStdin returns writer which is linked to the program stdin.
func (execution *Execution) GetStdin() io.WriteCloser {
	return execution.stdin
}

// SetStdin sets reader which will be used as program stdin.
func (execution *Execution) SetStdin(source io.Reader) {
	execution.stdin = struct {
		io.Reader
		io.WriteCloser
	}{
		Reader: source,
	}
}

// Starts will start command, but will not wait for execution.
func (execution *Execution) Start() error {
	err := execution.setup()
	if err != nil {
		return err
	}

	if err := execution.command.Start(); err != nil {
		return hierr.Errorf(
			err,
			`can't start command: %s`,
			execution,
		)
	}

	return nil
}

// Wait will wait for command to finish.
func (execution *Execution) Wait() error {
	err := execution.command.Wait()
	if err != nil {
		return hierr.Errorf(
			err,
			`can't finish command execution: %s`,
			execution.String(),
		)
	}

	execution.closer()

	return nil
}

// Run starts command and waits for it execution.
func (execution *Execution) Run() error {
	err := execution.Start()
	if err != nil {
		return err
	}

	err = execution.Wait()
	if err != nil {
		return err
	}

	return nil
}

// String returns string representation of command.
func (execution *Execution) String() string {
	return fmt.Sprintf(`%#v`,
		append(
			[]string{execution.command.Path},
			execution.command.Args...,
		),
	)
}

func (execution *Execution) setup() error {
	lock := &sync.Mutex{}

	loggerize := func(
		stream Stream,
		output io.Writer,
	) (io.Writer, func() error) {
		logger := lineflushwriter.New(
			callbackwriter.New(
				nopio.NopWriteCloser{},
				func(data []byte) {
					execution.logger(execution.command.Args, stream, data)
				},
				nil,
			),
			lock,
			true,
		)

		return io.MultiWriter(output, logger), logger.Close
	}

	var (
		stdoutCloser func() error
		stderrCloser func() error
	)

	execution.command.Stdout, stdoutCloser = loggerize(
		Stdout,
		execution.stdout,
	)

	execution.command.Stderr, stderrCloser = loggerize(
		Stderr,
		execution.stderr,
	)

	execution.closer = func() {
		_ = stdoutCloser()
		_ = stderrCloser()
	}

	if execution.stdin == nil {
		stdin, err := execution.command.StdinPipe()
		if err != nil {
			return hierr.Errorf(
				err,
				`can't get stdin pipe from command: %s`,
				execution,
			)
		}

		execution.stdin = struct {
			io.WriteCloser
			io.Reader
		}{
			WriteCloser: stdin,
		}
	} else {
		execution.command.Stdin = execution.stdin
	}

	return nil
}
