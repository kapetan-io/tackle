package color_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/kapetan-io/tackle/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"regexp"
	"testing"
)

func TestWithTimeAndLevel(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	log := slog.New(color.NewLog(&color.LogOptions{
		ColorFunc: color.NoColor,
		Writer:    w,
	}))

	log.Info("testing logger")
	require.NoError(t, w.Flush())
	assert.Contains(t, buf.String(), "testing logger \n")
	regExp := regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\.\d{3}\] INFO: testing logger \n$`)
	assert.True(t, regExp.MatchString(buf.String()))
}

func TestAttributes(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	log := slog.New(color.NewLog(&color.LogOptions{
		ColorFunc: color.NoColor,
		Writer:    w,
	}))

	log.Info("testing logger", "code", 2319, "type", "sock")
	require.NoError(t, w.Flush())
	assert.Contains(t, buf.String(), "testing logger code=2319 type=sock\n")
}

func TestSuppressAttrs(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	log := slog.New(color.NewLog(&color.LogOptions{
		HandlerOptions: slog.HandlerOptions{
			ReplaceAttr: color.SuppressAttrs(slog.TimeKey),
		},
		ColorFunc: color.NoColor,
		Writer:    w,
	}))
	log.Info("This is a test", "code", 2319, "type", "sock")
	require.NoError(t, w.Flush())
	assert.Equal(t, buf.String(), "INFO: This is a test code=2319 type=sock\n")
}

func TestColor(t *testing.T) {
	log := slog.New(color.NewLog(nil))
	fmt.Printf("\n--- Default Options ---\n")
	log.Debug("This is a debug", "attr1", 2319, "attr2", "foo")
	log.Info("This is a info", "attr1", 2319, "attr2", "foo")
	log.Warn("This is a warning", "attr1", 2319, "attr2", "foo")
	log.Error("This is an error", "attr1", 2319, "attr2", "foo")
	log.Log(context.Background(), slog.LevelError+1, "This is a error+1", "attr1", 2319, "attr2", "foo")
	log.Log(context.Background(), slog.LevelError+2, "This is a error+2", "attr1", 2319, "attr2", "foo")

	log = slog.New(color.NewLog(&color.LogOptions{MsgColor: color.FgHiWhite}))
	log.Info("This is color.FgHiWhite message", "attr1", 2319, "attr2", "foo")
	log = slog.New(color.NewLog(&color.LogOptions{MsgColor: color.FgHiBlue}))
	log.Info("This is color.FgHiBlue message", "attr1", 2319, "attr2", "foo")

	log = slog.New(color.NewLog(&color.LogOptions{
		HandlerOptions: slog.HandlerOptions{
			ReplaceAttr: color.SuppressAttrs(slog.TimeKey),
		},
	}))
	fmt.Printf("\n--- color.SupressAttrs(slog.TimeKey) ---\n")
	log.Debug("This is a debug", "attr1", 2319, "attr2", "foo")
	log.Info("This is a info", "attr1", 2319, "attr2", "foo")
	log.Warn("This is a warning", "attr1", 2319, "attr2", "foo")
	log.Error("This is an error", "attr1", 2319, "attr2", "foo")
	log.Log(context.Background(), slog.LevelError+1, "This is a error+1", "attr1", 2319, "attr2", "foo")
	log.Log(context.Background(), slog.LevelError+2, "This is a error+2", "attr1", 2319, "attr2", "foo")

}
