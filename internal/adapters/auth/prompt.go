package auth

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"elydelva/one/internal/core"
)

// promptFn reads a secret from the user. Default uses no-echo terminal input;
// tests override via the package-level var.
var promptFn = defaultPrompt

func defaultPrompt(label string, in io.Reader, out io.Writer) (core.Secret, error) {
	fmt.Fprint(out, label)
	// If stdin is a TTY, use no-echo. Otherwise read a single line (piped input).
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		raw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return core.Secret{}, err
		}
		s := strings.TrimSpace(string(raw))
		if s == "" {
			return core.Secret{}, errors.New("empty input")
		}
		return core.NewSecret(s), nil
	}

	r := bufio.NewReader(in)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return core.Secret{}, err
	}
	s := strings.TrimSpace(line)
	if s == "" {
		return core.Secret{}, errors.New("empty input")
	}
	return core.NewSecret(s), nil
}

// promptSecret prompts the user with the given label and returns the entered Secret.
func promptSecret(label string) (core.Secret, error) {
	return promptFn(label, os.Stdin, os.Stderr)
}
