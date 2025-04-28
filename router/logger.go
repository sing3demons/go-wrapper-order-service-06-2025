package router

type Logger interface {
	Debugf(format string, args ...any)
	Debug(args ...any)
	Logf(format string, args ...any)
	Log(args ...any)
	Errorf(format string, args ...any)
	Error(args ...any)
}