package errors

import (
	"fmt"

	stderrors "errors"

	"go.yaml.in/yaml/v4"
)

type Err struct {
	Message string
	code    string
	details string
	cause   error
}

func New(message string) error {
	return Err{
		Message: message,
		code:    "error",
	}
}

func Newf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	return Err{
		Message: err.Error(),
		code:    "error",
	}
}

func (e Err) Format(s fmt.State, verb rune) {
	// implement %w
	switch verb {
	case 'W':
		if e.cause != nil {
			fmt.Fprintf(s, "%v", e.cause)
		}
	case 'w':
		if e.cause != nil {
			fmt.Fprintf(s, "%+v", e.cause)
		}
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%s\n", e.Message)
			if e.details != "" {
				fmt.Fprintf(s, "Details: %s\n", e.details)
			}
			if e.cause != nil {
				fmt.Fprintf(s, "Cause: %+v\n", e.cause)
			}
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(s, e.Message)
	case 'q':
		fmt.Fprintf(s, "%q", e.Message)
	}
}

type YamlErr struct {
	Err
	line   int
	column int
}

type YamlError interface {
	Line() int
	Column() int
}

type Cause interface {
	Cause() error
}

type Details interface {
	Details() string
}

func (e YamlErr) Line() int {
	return e.line
}

func (e YamlErr) Column() int {
	return e.column
}

func (e Err) Error() string {
	return e.Message
}

func (e Err) Details() string {
	return e.details
}

func (e Err) Cause() error {
	return e.cause
}

func (e Err) Is(target error) bool {
	if te, ok := target.(Err); ok {
		return e.code == te.code
	}
	return false
}

func WithDetails(err error, details string) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(Err); ok {
		e.details = details
		return e
	}

	return Err{
		Message: err.Error(),
		details: details,
		cause:   err,
		code:    "error",
	}
}

func WithCause(err error, cause error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(Err); ok {
		e.cause = cause
		return e
	}

	return Err{
		Message: err.Error(),
		cause:   cause,
		code:    "error",
	}
}

func WithYamlNode(err error, node *yaml.Node) error {
	if err == nil {
		return nil
	}

	if node == nil {
		return err
	}

	msg := fmt.Sprintf("%s on line %d, column %d", err.Error(), node.Line, node.Column)

	return YamlErr{
		line:   node.Line,
		column: node.Column,
		Err: Err{
			Message: msg,
			cause:   err,
			code:    "yaml-error",
		},
	}
}

func NewYamlError(node *yaml.Node, message string) error {
	line := 0
	column := 0

	if node != nil {
		line = node.Line
		column = node.Column

		message = fmt.Sprintf("%s on line %d, column %d", message, line, column)
	}

	return YamlErr{
		line:   line,
		column: column,
		Err: Err{
			Message: message,
		},
	}
}

func YamlErrorf(node *yaml.Node, format string, args ...interface{}) error {
	return NewYamlError(node, fmt.Sprintf(format, args...))
}

// Is reports whether any error in err's chain matches target.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error is considered to match a target if it is equal to that target or if
// it implements a method Is(error) bool such that Is(target) returns true.
func Is(err, target error) bool { return stderrors.Is(err, target) }

// As finds the first error in err's chain that matches target, and if so, sets
// target to that error value and returns true.
//
// The chain consists of err itself followed by the sequence of errors obtained by
// repeatedly calling Unwrap.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// As will panic if target is not a non-nil pointer to either a type that implements
// error, or to any interface type. As returns false if err is nil.
func As(err error, target interface{}) bool { return stderrors.As(err, target) }

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}
