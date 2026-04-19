package logx

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	levelTrace = "trace"
	levelDebug = "debug"
	levelWarn  = "warn"
	levelError = "error"
)

func TraceEnabled() bool {
	return currentLevel() == levelTrace
}

func DebugEnabled() bool {
	level := currentLevel()
	return level == levelTrace || level == levelDebug
}

func Tracef(format string, args ...any) {
	if !TraceEnabled() {
		return
	}
	write(os.Stderr, levelTrace, format, args...)
}

func Debugf(format string, args ...any) {
	if !DebugEnabled() {
		return
	}
	write(os.Stderr, levelDebug, format, args...)
}

func Warnf(format string, args ...any) {
	write(os.Stderr, levelWarn, format, args...)
}

func Errorf(format string, args ...any) {
	write(os.Stderr, levelError, format, args...)
}

func write(f *os.File, level, format string, args ...any) {
	label := strings.ToUpper(level)
	if colorEnabled(f) {
		label = colorize(level, label)
	}
	_, _ = fmt.Fprintf(f, "[%s] %s\n", label, fmt.Sprintf(format, args...))
}

func currentLevel() string {
	for _, key := range []string{"CAST_LOG", "CAST_TRACE", "CAST_DEBUG"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "" {
			continue
		}
		switch value {
		case levelTrace, "2":
			return levelTrace
		case levelDebug, "1", "true", "yes", "on":
			return levelDebug
		case levelWarn, levelError:
			return value
		case "false", "off", "0":
			return ""
		}
	}
	return ""
}

func colorEnabled(f *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("CLICOLOR_FORCE")); v != "" && v != "0" {
		return true
	}
	if strings.TrimSpace(os.Getenv("TERM")) == "dumb" {
		return false
	}
	if v := strings.TrimSpace(os.Getenv("CLICOLOR")); v == "0" {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func colorize(level, s string) string {
	code := "36"
	switch level {
	case levelDebug:
		code = "34"
	case levelWarn:
		code = "33"
	case levelError:
		code = "31"
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}
