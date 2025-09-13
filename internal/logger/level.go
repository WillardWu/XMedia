package logger

// Level is a log level.
type Level int

// Log levels.
const (
	Info Level = iota + 1
	Warn
	Error
)
