package logflags

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"
)

// textHandler implements slog.Handler.
type textHandler struct {
	out               io.WriteCloser
	opts              slog.HandlerOptions
	attrs             []slog.Attr
	preformattedAttrs string // preformatted version of attrs for optimization purposes
}

func newTextHandler(out io.WriteCloser, opts *slog.HandlerOptions) *textHandler {
	return &textHandler{
		out:  out,
		opts: *opts,
	}
}

// Enabled check if log level `l` is enabled.
func (h *textHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.opts.Level.Level()
}

// WithAttrs preformats the attrs before really writing to the log.
func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.attrs = append(h2.attrs, attrs...)
	m := map[string]slog.Value{}
	keys := []string{}
	for i := range attrs {
		m[attrs[i].Key] = attrs[i].Value
		keys = append(keys, attrs[i].Key)
	}
	sort.Strings(keys)
	b := new(bytes.Buffer)
	for _, key := range keys {
		appendAttr(b, key, m[key])
	}
	b.Truncate(b.Len() - 1)
	h2.preformattedAttrs = b.String()
	return &h2
}

func appendAttr(b *bytes.Buffer, key string, val slog.Value) {
	b.WriteString(key)
	b.WriteByte('=')
	stringVal := val.String()
	if needsQuoting(stringVal) {
		fmt.Fprintf(b, "%q", stringVal)
	} else {
		b.WriteString(stringVal)
	}
	b.WriteByte(',')
}

func (h *textHandler) WithGroup(group string) slog.Handler {
	// group not handled
	return h
}

// Handle rewrites the format of logger message.
func (h *textHandler) Handle(_ context.Context, entry slog.Record) error {
	b := &bytes.Buffer{}

	b.WriteString(entry.Time.Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(entry.Level.String()))
	b.WriteByte(' ')
	b.WriteString(h.preformattedAttrs)

	if entry.NumAttrs() > 0 {
		if len(h.preformattedAttrs) > 0 {
			b.WriteByte(' ')
		}
		entry.Attrs(func(attr slog.Attr) bool {
			appendAttr(b, attr.Key, attr.Value)
			return true
		})
		b.Truncate(b.Len() - 1)
	}

	if len(h.preformattedAttrs) > 0 || entry.NumAttrs() > 0 {
		b.WriteByte(' ')
	}

	b.WriteString(entry.Message)
	b.WriteByte('\n')
	_, err := h.out.Write(b.Bytes())
	return err
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

// create a slog logger based on the given flag and attributes.
func makeLogger(flag bool, attrs ...interface{}) Logger {
	if f := loggerFactory; f != nil {
		fields := make(Fields)
		for i := 0; i < len(attrs); i += 2 {
			fields[attrs[i].(string)] = attrs[i+1]
		}
		return f(flag, fields, logOut)
	}

	var out io.WriteCloser = os.Stderr
	if logOut != nil {
		out = logOut
	}
	level := slog.LevelError
	if flag {
		level = slog.LevelDebug
	}
	logger := slog.New(newTextHandler(out, &slog.HandlerOptions{Level: level})).With(attrs...)
	return slogLogger{logger}
}
