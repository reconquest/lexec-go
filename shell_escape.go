package lexec

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reSpecialChars       = regexp.MustCompile("[$`\"!'\\s]")
	reSpecialCharsEscape = regexp.MustCompile("[$`\"!]")
)

func FormatShellCommand(command []string) string {
	var safe []string

	for _, arg := range command {
		if reSpecialChars.MatchString(arg) {
			safe = append(safe, fmt.Sprintf(
				`"%s"`,
				reSpecialCharsEscape.ReplaceAllString(arg, `\$0`),
			))
		} else {
			safe = append(safe, arg)
		}
	}

	return strings.Join(safe, " ")
}
