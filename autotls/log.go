package autotls

// StandardLogger is retained for backward compatibility. New code should use
// *slog.Logger, which satisfies this interface structurally and can be
// assigned to autotls.Config.Logger directly.
//
// Deprecated: use *slog.Logger.
type StandardLogger interface {
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// NoOpLogger is retained for backward compatibility. New code that needs a
// silent logger should use slog.New(slog.DiscardHandler).
//
// Deprecated: use slog.New(slog.DiscardHandler).
type NoOpLogger struct{}

func (NoOpLogger) Error(msg string, args ...any) {}
func (NoOpLogger) Info(msg string, args ...any)  {}
func (NoOpLogger) Debug(msg string, args ...any) {}
func (NoOpLogger) Warn(msg string, args ...any)  {}
