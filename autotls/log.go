package autotls

type StandardLogger interface {
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

type NoOpLogger struct{}

func (NoOpLogger) Error(msg string, args ...any) {}
func (NoOpLogger) Info(msg string, args ...any)  {}
func (NoOpLogger) Debug(msg string, args ...any) {}
func (NoOpLogger) Warn(msg string, args ...any)  {}
