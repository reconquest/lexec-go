package lexec

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/reconquest/callbackwriter-go"
	"github.com/reconquest/executil-go"
	"github.com/reconquest/lineflushwriter-go"
	"github.com/reconquest/nopio-go"
	"github.com/reconquest/ser-go"
)

// Execution represents command prepared for the run.
type Execution struct {
	command Command

	stdin  io.ReadWriteCloser
	stdout io.ReadWriter
	stderr io.ReadWriter

	combinedStreams []StreamData

	logger Logger

	closer func()
}

type Command interface {
	Run() error
	Start() error
	Wait() error
	SetStdin(io.Reader)
	SetStdout(io.Writer)
	SetStderr(io.Writer)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.Reader, error)
	StderrPipe() (io.Reader, error)

	GetArgs() []string
}

var (
	_ Command = (*command)(nil)
)

type command struct {
	*exec.Cmd
}

func (command *command) GetArgs() []string {
	return command.Args
}

func (command *command) SetStdout(target io.Writer) {
	command.Stdout = target
}

func (command *command) SetStderr(target io.Writer) {
	command.Stderr = target
}

func (command *command) SetStdin(target io.Reader) {
	command.Stdin = target
}

func (command *command) StdoutPipe() (io.Reader, error) {
	pipe, err := command.Cmd.StdoutPipe()

	return pipe.(io.Reader), err
}

func (command *command) StderrPipe() (io.Reader, error) {
	pipe, err := command.Cmd.StderrPipe()

	return pipe.(io.Reader), err
}

// Logger represents type of function, which is considered logger by `New`.
type Logger func(command []string, stream Stream, data []byte)

// Loggerf will turn typical Somethingf() logger function into acceptible
// Logger function.
func Loggerf(logger func(string, ...interface{})) Logger {
	return func(command []string, stream Stream, data []byte) {
		if stream == InternalDebug {
			logger(`{%s} <exec> %q %s`, command[0], command, string(data))
		} else {
			logger(`{%s} <%s> %s`, command[0], stream, string(data))
		}
	}
}

// NewExec creates new execution object, that is used to start command and
// setupStreams stdout/stderr/stdin streams.
//
// stdout/stderr will be duplicated to specified logger. Each logged line will be
// prefixed with `<stdXXX> {command} `. Prefix can be overrided via likely
// named methods.
func NewExec(logger Logger, cmd *exec.Cmd) *Execution {
	return New(logger, &command{cmd})
}

// New same as NewExec but second argument must implement interface Command.
func New(logger Logger, cmd Command) *Execution {
	if logger == nil {
		logger = Loggerf(func(string, ...interface{}) {})
	}

	execution := &Execution{
		command: cmd,
		logger:  logger,
	}

	execution.stdout = &bytes.Buffer{}
	execution.stderr = &bytes.Buffer{}

	execution.combinedStreams = []StreamData{}

	return execution
}

// SetStdout sets writer to store stdout.
//
// If not called, internal buffer will be used.
func (execution *Execution) SetStdout(target io.Writer) *Execution {
	execution.stdout = struct {
		io.Reader
		io.Writer
	}{
		Writer: target,
	}

	return execution
}

// SetStderr sets writer to store stderr.
//
// If not called, internal buffer will be used.
func (execution *Execution) SetStderr(target io.Writer) *Execution {
	execution.stderr = struct {
		io.Reader
		io.Writer
	}{
		Writer: target,
	}

	return execution
}

func (execution *Execution) StdoutPipe() (io.Reader, error) {
	pipe, err := execution.command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	execution.stdout = nil

	return pipe, nil
}

func (execution *Execution) StderrPipe() (io.Reader, error) {
	pipe, err := execution.command.StderrPipe()
	if err != nil {
		return nil, err
	}

	execution.stderr = nil

	return pipe, nil
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
func (execution *Execution) SetStdin(source io.Reader) *Execution {
	execution.stdin = struct {
		io.WriteCloser
		io.Reader
	}{
		Reader: source,
	}

	return execution
}

// Starts will start command, but will not wait for execution.
func (execution *Execution) Start() error {
	if execution.logger != nil {
		execution.logger(
			execution.command.GetArgs(),
			InternalDebug,
			[]byte(`start`),
		)
	}

	err := execution.setupStreams()
	if err != nil {
		return err
	}

	if err := execution.command.Start(); err != nil {
		return ser.Errorf(
			err,
			`can't start command: %s`,
			execution.String(),
		)
	}

	return nil
}

// Wait will wait for command to finish.
func (execution *Execution) Wait() error {
	err := execution.command.Wait()
	if err != nil {
		if !executil.IsExitError(err) {
			return ser.Errorf(
				err,
				`can't finish command execution: %s`,
				execution.String(),
			)
		}

		if execution.logger != nil {
			execution.logger(
				execution.command.GetArgs(),
				InternalDebug,
				[]byte(fmt.Sprintf(
					`exit code %d`,
					executil.GetExitStatus(err),
				)),
			)
		}

		return &executil.Error{
			RunErr: err,
			Cmd:    execution.command,
		}
	}

	if execution.closer != nil {
		execution.closer()
	}

	if execution.logger != nil {
		execution.logger(
			execution.command.GetArgs(),
			InternalDebug,
			[]byte(`exit code 0`),
		)
	}

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

func (execution *Execution) Output() ([]byte, []byte, error) {
	err := execution.Run()

	var stdout []byte
	var stderr []byte

	{
		var err error

		stdout, err = ioutil.ReadAll(execution.stdout)
		if err != nil {
			return nil, nil, ser.Errorf(
				err,
				`can't read execution stdout: %s`,
				execution.String(),
			)
		}

		stderr, err = ioutil.ReadAll(execution.stderr)
		if err != nil {
			return nil, nil, ser.Errorf(
				err,
				`can't read execution stderr: %s`,
				execution.String(),
			)
		}
	}

	return stdout, stderr, err
}

// String returns string representation of command.
func (execution *Execution) String() string {
	return fmt.Sprintf(`%q`, execution.command.GetArgs())
}

func (execution *Execution) NoLog() *Execution {
	execution.logger = nil

	return execution
}

func (execution *Execution) setupStreams() error {
	var (
		streamMutex   = &sync.Mutex{}
		combinedMutex = &sync.Mutex{}
	)

	loggerize := func(
		stream Stream,
		output io.Writer,
	) (io.Writer, func() error) {
		logger := lineflushwriter.New(
			callbackwriter.New(
				nopio.NopWriteCloser{},
				func(data []byte) {
					execution.logger(
						execution.command.GetArgs(),
						stream,
						bytes.TrimRight(data, "\n"),
					)
				},
				nil,
			),
			streamMutex,
			true,
		)

		return io.MultiWriter(
			newStreamWriter(
				&execution.combinedStreams,
				combinedMutex,
				stream,
			),
			output, logger,
		), logger.Close
	}

	if execution.logger != nil {
		var (
			stdout, stderr io.Writer

			stdoutCloser, stderrCloser func() error
		)

		if execution.stdout != nil {
			stdout, stdoutCloser = loggerize(
				Stdout,
				execution.stdout,
			)

			execution.command.SetStdout(stdout)
		}

		if execution.stderr != nil {
			stderr, stderrCloser = loggerize(
				Stderr,
				execution.stderr,
			)

			execution.command.SetStderr(stderr)
		}

		execution.closer = func() {
			if stdoutCloser != nil {
				_ = stdoutCloser()
			}

			if stderrCloser != nil {
				_ = stderrCloser()
			}
		}
	} else {
		if execution.stdout != nil {
			execution.command.SetStdout(execution.stdout)
		}

		if execution.stderr != nil {
			execution.command.SetStderr(execution.stderr)
		}
	}

	if execution.stdin == nil {
		stdin, err := execution.command.StdinPipe()
		if err != nil {
			return ser.Errorf(
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
		execution.command.SetStdin(execution.stdin)
	}

	return nil
}

func (execution *Execution) Process() *os.Process {
	// this wrapper needs only in case when instead of exec.Command has been
	// passed runcmd.Remote
	if cmd, ok := execution.command.(*command); ok {
		return cmd.Process
	}

	return nil
}

func (execution *Execution) ProcessState() *os.ProcessState {
	if cmd, ok := execution.command.(*command); ok {
		return cmd.ProcessState
	}

	return nil
}

func (execution *Execution) SysProcAttr() *syscall.SysProcAttr {
	if cmd, ok := execution.command.(*command); ok {
		return cmd.SysProcAttr
	}

	return nil
}

func (execution *Execution) GetStreamsData() []StreamData {
	return execution.combinedStreams
}
