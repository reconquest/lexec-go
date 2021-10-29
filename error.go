package lexec

import "github.com/reconquest/karma-go"

// ExitStatusError is returned when a command exists with non-zero exit code.
type ExitStatusError struct {
	karma.Karma
	ExitStatus int
}

// IsExitStatus returns true if the given error is an instance of
// ExitStatusError.
func IsExitStatus(err error) bool {
	_, ok := err.(ExitStatusError)
	return ok
}

// GetExitStatus returns an exitcode of the given ExitStatusError.
func GetExitStatus(err error) int {
	if err, ok := err.(ExitStatusError); ok {
		return err.ExitStatus
	}
	return 0
}
