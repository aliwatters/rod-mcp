package tools

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestSplitTextChunksPreservesRunes(t *testing.T) {
	text := "abc世界def"
	chunks := splitTextChunks(text, 3)

	if got := strings.Join(chunks, ""); got != text {
		t.Fatalf("joined chunks = %q, want %q", got, text)
	}
	for _, chunk := range chunks {
		if count := len([]rune(chunk)); count > 3 {
			t.Fatalf("chunk %q has %d runes, want at most 3", chunk, count)
		}
	}
}

func TestIsRetryableInputError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "io eof", err: io.EOF, want: true},
		{name: "unexpected eof", err: io.ErrUnexpectedEOF, want: true},
		{name: "wrapped eof text", err: errors.New("cdp dispatch failed: EOF"), want: true},
		{name: "closed connection", err: errors.New("websocket: close 1006 abnormal closure"), want: true},
		{name: "validation", err: errors.New("missing required argument: text"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableInputError(tt.err); got != tt.want {
				t.Fatalf("isRetryableInputError(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestTypeTextInChunksRetriesRetryableErrorOnce(t *testing.T) {
	calls := 0
	var inserted []string

	err := typeTextInChunks(context.Background(), "hello", 0, func(text string) error {
		calls++
		if calls == 1 {
			return io.EOF
		}
		inserted = append(inserted, text)
		return nil
	})
	if err != nil {
		t.Fatalf("typeTextInChunks returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("insert calls = %d, want 2", calls)
	}
	if got := strings.Join(inserted, ""); got != "hello" {
		t.Fatalf("inserted text = %q, want hello", got)
	}
}

func TestTypeTextInChunksDoesNotRetryNonRetryableError(t *testing.T) {
	calls := 0
	err := typeTextInChunks(context.Background(), "hello", 0, func(text string) error {
		calls++
		return errors.New("invalid target")
	})

	if err == nil {
		t.Fatal("typeTextInChunks returned nil error")
	}
	if calls != 1 {
		t.Fatalf("insert calls = %d, want 1", calls)
	}
}

func TestTypeTextInChunksStopsOnContextCancelDuringDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := typeTextInChunks(ctx, "hello", time.Millisecond, func(text string) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("typeTextInChunks error = %v, want context.Canceled", err)
	}
}
