// Package color IS A MODIFIED VERSION of the original https://github.com/dusted-go/logging
package color

import (
	"bytes"
	"context"
	"fmt"
	"github.com/kapetan-io/tackle/set"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
)

type Attribute int

const (
	Reset Attribute = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

const (
	ResetBold Attribute = iota + 22
	ResetItalic
	ResetUnderline
	ResetBlinking
	_
	ResetReversed
	ResetConcealed
	ResetCrossedOut
)

// Foreground text colors
const (
	FgBlack Attribute = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Foreground Hi-Intensity text colors
const (
	FgHiBlack Attribute = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

// Background text colors
const (
	BgBlack Attribute = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

// Background Hi-Intensity text colors
const (
	BgHiBlack Attribute = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

const (
	timeFormat = "[15:04:05.000]"
)

type Func func(_ Attribute, value string) string

func Colorize(colorCode Attribute, v string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", colorCode, v)
}

func NoColor(_ Attribute, value string) string {
	return value
}

type LogOptions struct {
	slog.HandlerOptions

	Writer    io.Writer
	ColorFunc Func
	MsgColor  Attribute
}

func NewLog(opts *LogOptions) *Handler {
	set.Default(&opts, &LogOptions{})
	set.Default(&opts.Writer, os.Stdout)

	if opts.ColorFunc == nil {
		opts.ColorFunc = Colorize
	}

	buf := &bytes.Buffer{}
	handler := &Handler{
		text: slog.NewTextHandler(buf, &slog.HandlerOptions{
			Level:     opts.Level,
			AddSource: opts.AddSource,
			ReplaceAttr: suppressAttrs(opts.ReplaceAttr,
				[]string{slog.TimeKey, slog.LevelKey, slog.MessageKey}),
		}),
		replace: opts.ReplaceAttr,
		mutex:   &sync.Mutex{},
		opts:    opts,
		buf:     buf,
	}
	return handler
}

type Handler struct {
	replace func([]string, slog.Attr) slog.Attr
	buf     *bytes.Buffer
	text    slog.Handler
	mutex   *sync.Mutex
	opts    *LogOptions
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.text.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		text:    h.text.WithAttrs(attrs),
		replace: h.replace,
		mutex:   h.mutex,
		opts:    h.opts,
		buf:     h.buf,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		text:    h.text.WithGroup(name),
		replace: h.replace,
		mutex:   h.mutex,
		opts:    h.opts,
		buf:     h.buf,
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var level string
	levelAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(r.Level),
	}
	if h.replace != nil {
		levelAttr = h.replace([]string{}, levelAttr)
	}

	if !levelAttr.Equal(slog.Attr{}) {
		level = levelAttr.Value.String() + ":"

		if r.Level <= slog.LevelDebug {
			level = h.opts.ColorFunc(FgHiBlack, level)
		} else if r.Level == slog.LevelDebug+1 {
			level = h.opts.ColorFunc(FgWhite, level)
		} else if r.Level == slog.LevelDebug+2 {
			level = h.opts.ColorFunc(FgYellow, level)
		} else if r.Level == slog.LevelDebug+3 {
			level = h.opts.ColorFunc(FgBlue, level)
		} else if r.Level < slog.LevelInfo {
			level = h.opts.ColorFunc(FgWhite, level)
		} else if r.Level < slog.LevelWarn {
			level = h.opts.ColorFunc(FgCyan, level)
		} else if r.Level < slog.LevelError {
			level = h.opts.ColorFunc(FgHiBlue, level)
		} else if r.Level == slog.LevelError {
			level = h.opts.ColorFunc(FgHiYellow, level)
		} else if r.Level == slog.LevelError+1 {
			level = h.opts.ColorFunc(FgHiMagenta, level)
		} else if r.Level > slog.LevelError+1 {
			level = h.opts.ColorFunc(FgHiRed, level)
		}
	}

	var timestamp string
	timeAttr := slog.Attr{
		Key:   slog.TimeKey,
		Value: slog.StringValue(r.Time.Format(timeFormat)),
	}
	if h.replace != nil {
		timeAttr = h.replace([]string{}, timeAttr)
	}
	if !timeAttr.Equal(slog.Attr{}) {
		timestamp = h.opts.ColorFunc(FgWhite, timeAttr.Value.String())
	}

	var msg string
	msgAttr := slog.Attr{
		Key:   slog.MessageKey,
		Value: slog.StringValue(r.Message),
	}
	if h.replace != nil {
		msgAttr = h.replace([]string{}, msgAttr)
	}
	if !msgAttr.Equal(slog.Attr{}) {
		msg = h.opts.ColorFunc(h.opts.MsgColor, msgAttr.Value.String())
	}

	attrs, err := h.formatAttrs(ctx, r)
	if err != nil {
		return err
	}

	out := strings.Builder{}
	if len(timestamp) > 0 {
		out.WriteString(timestamp)
		out.WriteString(" ")
	}
	if len(level) > 0 {
		out.WriteString(level)
		out.WriteString(" ")
	}
	if len(msg) > 0 {
		out.WriteString(msg)
		out.WriteString(" ")
	}
	if len(attrs) > 0 {
		out.WriteString(h.opts.ColorFunc(FgHiBlack, attrs))
	}

	_, err = io.WriteString(h.opts.Writer, out.String())
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) formatAttrs(ctx context.Context, r slog.Record) (string, error) {
	h.mutex.Lock()
	defer func() {
		h.buf.Reset()
		h.mutex.Unlock()
	}()
	if err := h.text.Handle(ctx, r); err != nil {
		return "", fmt.Errorf("error when calling slog.TextHandler: %w", err)
	}
	return h.buf.String(), nil
}

type ReplaceFunc func(groups []string, a slog.Attr) slog.Attr

func SuppressAttrs(attrs ...string) ReplaceFunc {
	return suppressAttrs(nil, attrs)
}

func suppressAttrs(wrap ReplaceFunc, attrs []string) ReplaceFunc {
	return func(groups []string, a slog.Attr) slog.Attr {
		if slices.Contains(attrs, a.Key) {
			return slog.Attr{}
		}
		if wrap == nil {
			return a
		}
		return wrap(groups, a)
	}
}
