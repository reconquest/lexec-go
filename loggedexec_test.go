package lexec

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReturnsEmptyOutputWhenCommandReturnsNothing(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`true`},
		``,
		``,
		[]string{
			`{true} <exec> ["true"] start`,
			`{true} <exec> ["true"] exit code 0`,
		},
		nil,
	)
}

func TestReturnsAndLogsLineOnStdout(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`echo`, `1`},
		"1\n",
		``,
		[]string{
			`{echo} <exec> ["echo" "1"] start`,
			"{echo} <stdout> 1",
			`{echo} <exec> ["echo" "1"] exit code 0`,
		},
		nil,
	)
}

func TestReturnsAndLogsLineOnStderr(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`sh`, `-c`, `echo 1 >&2`},
		``,
		"1\n",
		[]string{
			`{sh} <exec> ["sh" "-c" "echo 1 >&2"] start`,
			"{sh} <stderr> 1",
			`{sh} <exec> ["sh" "-c" "echo 1 >&2"] exit code 0`,
		},
		nil,
	)
}

func TestReturnsAndLogsLineWithoutNewline(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`echo`, `-n`, `1`},
		"1",
		``,
		[]string{
			`{echo} <exec> ["echo" "-n" "1"] start`,
			"{echo} <stdout> 1",
			`{echo} <exec> ["echo" "-n" "1"] exit code 0`,
		},
		nil,
	)
}

func TestCanPassStdinToCommand(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`sed`, `s/^/xxx /`},
		"xxx test",
		``,
		[]string{
			`{sed} <exec> ["sed" "s/^/xxx /"] start`,
			"{sed} <stdout> xxx test",
			`{sed} <exec> ["sed" "s/^/xxx /"] exit code 0`,
		},
		bytes.NewBufferString(`test`),
	)
}

func assertCommandOutput(
	t *testing.T,
	command []string,
	stdout string,
	stderr string,
	logged []string,
	stdin io.Reader,
) {
	log := []string{}

	logger := func(format string, data ...interface{}) {
		log = append(log, fmt.Sprintf(format, data...))
	}

	execution := NewExec(
		Loggerf(logger),
		exec.Command(
			command[0],
			command[1:]...,
		),
	)

	if stdin != nil {
		execution.SetStdin(stdin)
	}

	actualStdout := &bytes.Buffer{}
	actualStderr := &bytes.Buffer{}

	execution.SetStdout(actualStdout)
	execution.SetStderr(actualStderr)

	err := execution.Run()
	assert.NoError(t, err)

	assert.Equal(t, stdout, actualStdout.String())
	assert.Equal(t, stderr, actualStderr.String())
	assert.Equal(t, logged, log)
}
