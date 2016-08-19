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

	// InternalDebug is ID for logging internal debug messages.
	InternalDebug Stream = `debug`
)

// StreamData represents execution output stream data.
type StreamData struct {
	Stream Stream

	// Data represents output that has been written into given stream.
	Data []byte
}

type streamDataWriter struct {
	output *[]StreamData
	stream Stream
	mutex  *sync.Mutex
}

func (writer *streamDataWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()

	*writer.output = append(*writer.output, StreamData{
		Stream: writer.stream,
		Data:   data,
	})

	return len(data), nil
}

func getStreamWriter(
	output *[]StreamData, stream Stream, mutex *sync.Mutex,
) io.Writer {
	return &streamDataWriter{
		output: output,
		stream: stream,
		mutex:  mutex,
	}
}
