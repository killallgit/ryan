package stream

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	t.Run("HandlerFunc", func(t *testing.T) {
		var chunks [][]byte
		var finalContent string
		var capturedError error

		handler := HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				chunks = append(chunks, chunk)
				return nil
			},
			CompleteFunc: func(content string) error {
				finalContent = content
				return nil
			},
			ErrorFunc: func(err error) {
				capturedError = err
			},
		}

		// Test OnChunk
		err := handler.OnChunk([]byte("test chunk"))
		assert.NoError(t, err)
		assert.Len(t, chunks, 1)
		assert.Equal(t, []byte("test chunk"), chunks[0])

		// Test OnComplete
		err = handler.OnComplete("final")
		assert.NoError(t, err)
		assert.Equal(t, "final", finalContent)

		// Test OnError
		testErr := errors.New("test error")
		handler.OnError(testErr)
		assert.Equal(t, testErr, capturedError)
	})

	t.Run("ToStreamingFunc", func(t *testing.T) {
		var receivedChunk []byte
		var errorReceived error

		handler := HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				receivedChunk = chunk
				return nil
			},
			ErrorFunc: func(err error) {
				errorReceived = err
			},
		}

		streamFunc := ToStreamingFunc(handler)

		// Test normal chunk
		ctx := context.Background()
		err := streamFunc(ctx, []byte("test"))
		assert.NoError(t, err)
		assert.Equal(t, []byte("test"), receivedChunk)

		// Test cancelled context
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err = streamFunc(cancelCtx, []byte("should not process"))
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, errorReceived)
	})

}

func TestWriterHandler(t *testing.T) {
	t.Run("Basic Write", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewWriterHandler(&buf)

		// Write chunks
		err := handler.OnChunk([]byte("Hello "))
		assert.NoError(t, err)

		err = handler.OnChunk([]byte("World"))
		assert.NoError(t, err)

		// Check writer output
		assert.Equal(t, "Hello World", buf.String())

		// Check internal buffer
		assert.Equal(t, "Hello World", handler.GetContent())
	})

	t.Run("OnComplete with different content", func(t *testing.T) {
		var buf bytes.Buffer
		handler := NewWriterHandler(&buf)

		// Write initial chunks
		handler.OnChunk([]byte("partial"))

		// Complete with different content
		err := handler.OnComplete("partial content")
		assert.NoError(t, err)

		// Should write the difference
		assert.Equal(t, "partial content", buf.String())
	})
}

func TestMultiHandler(t *testing.T) {
	t.Run("Broadcasts to all handlers", func(t *testing.T) {
		var handler1Chunks []string
		var handler2Chunks []string
		var handler1Complete string
		var handler2Complete string

		h1 := HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				handler1Chunks = append(handler1Chunks, string(chunk))
				return nil
			},
			CompleteFunc: func(content string) error {
				handler1Complete = content
				return nil
			},
		}

		h2 := HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				handler2Chunks = append(handler2Chunks, string(chunk))
				return nil
			},
			CompleteFunc: func(content string) error {
				handler2Complete = content
				return nil
			},
		}

		multi := NewMultiHandler(h1, h2)

		// Send chunk
		err := multi.OnChunk([]byte("test"))
		assert.NoError(t, err)
		assert.Equal(t, []string{"test"}, handler1Chunks)
		assert.Equal(t, []string{"test"}, handler2Chunks)

		// Send completion
		err = multi.OnComplete("final")
		assert.NoError(t, err)
		assert.Equal(t, "final", handler1Complete)
		assert.Equal(t, "final", handler2Complete)
	})

	t.Run("Error in one handler stops processing", func(t *testing.T) {
		var handler2Called bool

		h1 := HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				return errors.New("handler1 error")
			},
		}

		h2 := HandlerFunc{
			ChunkFunc: func(chunk []byte) error {
				handler2Called = true
				return nil
			},
		}

		multi := NewMultiHandler(h1, h2)

		err := multi.OnChunk([]byte("test"))
		assert.Error(t, err)
		assert.False(t, handler2Called, "handler2 should not be called after handler1 error")
	})
}
