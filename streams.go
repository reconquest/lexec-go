package lexec

import (
	"io"
	"sync"
)

// Stream represents execution output stream.
type Stream string

const (
	// Stdout is ID for execution stdout.
	Stdout Stream = `stdout`

	// Stdout is ID for execution stderr.
	Stderr Stream = `stderr`

	// Start is ID for execution start.
	Launch Stream = `launch`

	// Fininsh is ID for execution finish.
	Finish Stream = `finish`
)

// StreamData represents execution output stream data.
type StreamData struct {
	Stream Stream

	// Data represents output that has been written into given stream.
	Data []byte
}

type streamWriter struct {
	output *[]StreamData
	stream Stream
	mutex  *sync.Mutex
}

func (writer *streamWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	indirected := make([]byte, len(data))
	copy(indirected, data)

	*writer.output = append(*writer.output, StreamData{
		Stream: writer.stream,
		Data:   indirected,
	})

	return len(indirected), nil
}

func newStreamWriter(
	output *[]StreamData,
	mutex *sync.Mutex,
	stream Stream,
) io.Writer {
	return &streamWriter{
		output: output,
		stream: stream,
		mutex:  mutex,
	}
}
