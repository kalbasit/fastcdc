package fastcdc

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidMinSize is returned when minSize is 0.
	ErrInvalidMinSize = errors.New("minSize must be greater than 0")

	// ErrInvalidTargetSize is returned when targetSize is 0.
	ErrInvalidTargetSize = errors.New("targetSize must be greater than 0")

	// ErrTargetSizeTooSmall is returned when targetSize is not greater than minSize.
	ErrTargetSizeTooSmall = errors.New("targetSize must be greater than minSize")

	// ErrInvalidMaxSize is returned when maxSize is 0.
	ErrInvalidMaxSize = errors.New("maxSize must be greater than 0")

	// ErrMaxSizeTooSmall is returned when maxSize is not greater than targetSize.
	ErrMaxSizeTooSmall = errors.New("maxSize must be greater than targetSize")

	// ErrInvalidNormLevel is returned when normLevel is not between 0 and 8.
	ErrInvalidNormLevel = errors.New("normLevel must be between 0 and 8")

	// ErrInvalidBufferSize is returned when bufferSize is 0.
	ErrInvalidBufferSize = errors.New("bufferSize must be greater than 0")
)

const (
	// DefaultMinSize is the default minimum chunk size (16 KiB).
	DefaultMinSize = 16 * 1024

	// DefaultTargetSize is the default target chunk size (64 KiB).
	DefaultTargetSize = 64 * 1024

	// DefaultMaxSize is the default maximum chunk size (256 KiB).
	DefaultMaxSize = 256 * 1024

	// DefaultNormLevel is the default normalization level (2)
	// Determines the size of the normalization region: (targetSize - minSize) / 2^normLevel.
	DefaultNormLevel = 2

	// DefaultBufferSize is the default internal buffer size for the streaming API (512 KiB).
	// This is 2x the default max chunk size, providing efficient buffering.
	DefaultBufferSize = 512 * 1024
)

// Option is a function that configures a Chunker or ChunkerCore.
type Option func(*config) error

// config holds the configuration for chunking.
type config struct {
	minSize    uint32
	targetSize uint32
	maxSize    uint32
	normLevel  uint8
	seed       uint64
	bufferSize int
}

// validate checks that the configuration is valid.
func (c *config) validate() error {
	if c.minSize == 0 {
		return ErrInvalidMinSize
	}

	if c.targetSize <= c.minSize {
		return fmt.Errorf("%w: targetSize (%d), minSize (%d)", ErrTargetSizeTooSmall, c.targetSize, c.minSize)
	}

	if c.maxSize <= c.targetSize {
		return fmt.Errorf("%w: maxSize (%d), targetSize (%d)", ErrMaxSizeTooSmall, c.maxSize, c.targetSize)
	}

	if c.normLevel > 8 {
		return fmt.Errorf("%w: got %d", ErrInvalidNormLevel, c.normLevel)
	}
	// Auto-adjust buffer size if needed
	if c.bufferSize < int(c.maxSize) {
		c.bufferSize = int(c.maxSize)
	}

	return nil
}

// computeMasks calculates the maskS and maskL for normalized chunking.
func (c *config) computeMasks() (maskS, maskL uint64, normSize uint32, bits uint8) {
	// Calculate bits from targetSize
	bits = 0

	tmp := c.targetSize
	for tmp > 1 {
		tmp >>= 1
		bits++
	}

	// Base mask (for targetSize)
	maskL = (uint64(1) << bits) - 1

	// Smaller mask for normalization region (more aggressive cutting)
	// maskS has fewer bits set, making it easier to match
	if bits > 0 {
		maskS = (uint64(1) << (bits - 1)) - 1
	} else {
		maskS = 0
	}

	// Calculate normalization boundary
	// normSize = minSize + (targetSize - minSize) / 2^normLevel
	normRange := c.targetSize - c.minSize
	normSize = c.minSize + (normRange >> c.normLevel)

	return maskS, maskL, normSize, bits
}

// WithMinSize sets the minimum chunk size.
func WithMinSize(size uint32) Option {
	return func(c *config) error {
		if size == 0 {
			return ErrInvalidMinSize
		}

		c.minSize = size

		return nil
	}
}

// WithTargetSize sets the target chunk size.
func WithTargetSize(size uint32) Option {
	return func(c *config) error {
		if size == 0 {
			return ErrInvalidTargetSize
		}

		c.targetSize = size

		return nil
	}
}

// WithMaxSize sets the maximum chunk size.
func WithMaxSize(size uint32) Option {
	return func(c *config) error {
		if size == 0 {
			return ErrInvalidMaxSize
		}

		c.maxSize = size

		return nil
	}
}

// WithNormalization sets the normalization level.
// Level 0 disables normalization (single-mask behavior).
// Higher levels create a larger normalization region.
func WithNormalization(level uint8) Option {
	return func(c *config) error {
		if level > 8 {
			return fmt.Errorf("%w: got %d", ErrInvalidNormLevel, level)
		}

		c.normLevel = level

		return nil
	}
}

// WithSeed sets a custom seed for the Gear hash table.
// Using a non-zero seed will allocate a per-instance table (2 KiB).
func WithSeed(seed uint64) Option {
	return func(c *config) error {
		c.seed = seed

		return nil
	}
}

// WithBufferSize sets the internal buffer size for the streaming API.
// Must be at least as large as maxSize.
func WithBufferSize(size int) Option {
	return func(c *config) error {
		if size <= 0 {
			return ErrInvalidBufferSize
		}

		c.bufferSize = size

		return nil
	}
}
