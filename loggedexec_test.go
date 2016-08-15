package loggedexec

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReturnsEmptyOutputWhenCommandReturnsNothing(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`true`},
		``,
		``,
		[]string{},
		nil,
	)
}

func TestReturnsAndLogsLineOnStdout(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`echo`, `1`},
		"1\n",
		``,
		[]string{"<stdout> {echo} 1\n"},
		nil,
	)
}

func TestReturnsAndLogsLineOnStderr(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`sh`, `-c`, `echo 1 >&2`},
		``,
		"1\n",
		[]string{"<stderr> {sh} 1\n"},
		nil,
	)
}

func TestReturnsAndLogsLineWithoutNewline(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`echo`, `-n`, `1`},
		"1",
		``,
		[]string{"<stdout> {echo} 1\n"},
		nil,
	)
}

func TestCanPassStdinToCommand(t *testing.T) {
	assertCommandOutput(
		t,
		[]string{`sed`, `s/^/xxx /`},
		"xxx test",
		``,
		[]string{"<stdout> {sed} xxx test\n"},
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

	logger := func (format string, data ...interface{}) {
		log = append(log, fmt.Sprintf(format, data...))
	}

	execution := New(
		Loggerf(logger),
		command[0],
		command[1:]...,
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
