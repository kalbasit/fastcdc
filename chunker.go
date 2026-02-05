package fastcdc

import (
	"errors"
	"io"
)

// Chunk represents a content-defined chunk with its metadata.
type Chunk struct {
	Offset uint64 // Absolute offset in the stream
	Length uint32 // Chunk size in bytes
	Hash   uint64 // Gear fingerprint at boundary
	Data   []byte // Chunk data (points into internal buffer)
}

// Chunker provides a convenient streaming API for content-defined chunking.
// It wraps an io.Reader and returns chunks via the Next() method.
//
// This API allocates minimally and is suitable for most use cases.
// For zero-allocation performance-critical code, use ChunkerCore.
type Chunker struct {
	core   ChunkerCore // Core chunking algorithm (embedded to avoid pointer allocation)
	reader io.Reader   // Input stream

	buf    []byte // Internal buffer
	cursor int    // Current position in buffer
	offset uint64 // Absolute offset in stream
	eof    bool   // EOF reached
}

// NewChunker creates a new Chunker that reads from the given io.Reader.
func NewChunker(r io.Reader, opts ...Option) (*Chunker, error) {
	// Use stack-allocated config to avoid heap allocation
	cfg := config{
		minSize:    DefaultMinSize,
		targetSize: DefaultTargetSize,
		maxSize:    DefaultMaxSize,
		normLevel:  DefaultNormLevel,
		seed:       0,
		bufferSize: DefaultBufferSize,
	}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	// Validate and adjust config
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Use internal function to avoid duplicate config allocation
	core := newChunkerCoreWithConfig(&cfg)

	return &Chunker{
		core:   core, // Embed by value to avoid heap allocation
		reader: r,
		buf:    make([]byte, cfg.bufferSize),
		cursor: cfg.bufferSize, // Start with empty buffer (triggers initial read)
		offset: 0,
		eof:    false,
	}, nil
}

// fillBuffer ensures the buffer has enough data for chunking.
// It moves unconsumed data to the front and reads more from the reader.
func (c *Chunker) fillBuffer() error {
	n := len(c.buf) - c.cursor
	if n >= int(c.core.MaxSize()) {
		return nil
	}

	// Move unconsumed data to the front of buffer
	copy(c.buf[:n], c.buf[c.cursor:])
	c.cursor = 0

	if c.eof {
		c.buf = c.buf[:n]

		return nil
	}

	// Fill the rest of the buffer
	m, err := io.ReadFull(c.reader, c.buf[n:])
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		c.buf = c.buf[:n+m]
		c.eof = true
	} else if err != nil {
		return err
	}

	return nil
}

// Next returns the next chunk from the stream.
// Returns io.EOF when the stream is exhausted.
//
// The returned Chunk.Data slice is valid until the next call to Next().
// If you need to keep the data, copy it to your own buffer.
func (c *Chunker) Next() (Chunk, error) {
	if err := c.fillBuffer(); err != nil {
		return Chunk{}, err
	}

	if len(c.buf) == 0 {
		return Chunk{}, io.EOF
	}

	// Find boundary in available data
	available := c.buf[c.cursor:]
	boundary, hash, found := c.core.FindBoundary(available)

	if !found {
		// No boundary found - this should only happen at EOF with remaining data
		// Return all remaining data as final chunk
		boundary = len(available)
	}

	chunk := Chunk{
		Offset: c.offset,
		Length: uint32(boundary), //nolint:gosec // G115
		Hash:   hash,
		Data:   available[:boundary],
	}

	c.cursor += boundary
	c.offset += uint64(boundary) //nolint:gosec // G115
	c.core.Reset()

	return chunk, nil
}

// Reset resets the chunker to start processing a new stream.
// The reader is replaced with the provided one, and all state is cleared.
func (c *Chunker) Reset(r io.Reader) {
	c.reader = r
	c.core.Reset()
	c.buf = c.buf[:cap(c.buf)] // Restore buffer to full capacity
	c.cursor = len(c.buf)      // Start with empty buffer
	c.offset = 0
	c.eof = false
}

// Offset returns the current absolute offset in the stream.
func (c *Chunker) Offset() uint64 {
	return c.offset
}
