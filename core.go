package fastcdc

// ChunkerCore implements zero-allocation content-defined chunking using the Gear hash algorithm.
// It provides a low-level FindBoundary API for performance-critical code where managing buffers
// manually is acceptable.
//
// For a more convenient streaming API with minimal allocations, use Chunker instead.
type ChunkerCore struct {
	// Hot path fields (frequently accessed together)
	table       [256]uint64 // Gear hash lookup table (2048 bytes)
	fingerprint uint64      // Current rolling hash value

	// Config fields (read-only after initialization)
	minSize   uint32 // Minimum chunk size
	normSize  uint32 // Normalization boundary (minSize + normalized region)
	maxSize   uint32 // Maximum chunk size
	maskS     uint64 // Small mask for [minSize, normSize) region
	maskL     uint64 // Large mask for [normSize, maxSize) region
	bits      uint8  // Number of bits in target size
	normLevel uint8  // Normalization level (0-8)

	// State
	position uint32 // Current position within chunk
}

// NewChunkerCore creates a new ChunkerCore with the given options.
// This is a zero-allocation API - the caller manages all buffers.
func NewChunkerCore(opts ...Option) (*ChunkerCore, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	maskS, maskL, normSize, bits := cfg.computeMasks()

	return &ChunkerCore{
		table:       generateTable(cfg.seed),
		fingerprint: 0,
		minSize:     cfg.minSize,
		normSize:    normSize,
		maxSize:     cfg.maxSize,
		maskS:       maskS,
		maskL:       maskL,
		bits:        bits,
		normLevel:   cfg.normLevel,
		position:    0,
	}, nil
}

// Reset resets the chunker state for processing a new stream.
// This allows reusing the same ChunkerCore instance.
func (c *ChunkerCore) Reset() {
	c.fingerprint = 0
	c.position = 0
}

// FindBoundary scans the provided data for a chunk boundary.
// It returns:
//   - boundary: the index of the chunk boundary (exclusive)
//   - hash: the final Gear hash value at the boundary
//   - found: true if a boundary was found, false if data exhausted
//
// This is a zero-allocation API. The caller is responsible for:
//  1. Providing the data buffer
//  2. Tracking absolute position across multiple calls
//  3. Handling data at chunk boundaries
//
// The chunker maintains state between calls, so calling FindBoundary
// multiple times continues scanning from where the previous call left off.
//
// Example usage:
//
//	core := NewChunkerCore(WithTargetSize(64*1024))
//	buf := make([]byte, 1*1024*1024)
//
//	for {
//	    n, _ := reader.Read(buf)
//	    if n == 0 {
//	        break
//	    }
//	    boundary, hash, found := core.FindBoundary(buf[:n])
//	    if found {
//	        processChunk(buf[:boundary], hash)
//	        // Continue with remaining data: buf[boundary:n]
//	    }
//	}
// FindBoundary scans the provided data for a chunk boundary.
// It returns:
//   - boundary: the index of the chunk boundary (exclusive)
//   - hash: the final Gear hash value at the boundary
//   - found: true if a boundary was found, false if data exhausted
//
// This is a zero-allocation API. The caller is responsible for:
//  1. Providing the data buffer
//  2. Tracking absolute position across multiple calls
//  3. Handling data at chunk boundaries
//
// The chunker maintains state between calls, so calling FindBoundary
// multiple times continues scanning from where the previous call left off.
//
// Example usage:
//
//	core := NewChunkerCore(WithTargetSize(64*1024))
//	buf := make([]byte, 1*1024*1024)
//
//	for {
//	    n, _ := reader.Read(buf)
//	    if n == 0 {
//	        break
//	    }
//	    boundary, hash, found := core.FindBoundary(buf[:n])
//	    if found {
//	        processChunk(buf[:boundary], hash)
//	        // Continue with remaining data: buf[boundary:n]
//	    }
//	}
func (c *ChunkerCore) FindBoundary(data []byte) (boundary int, hash uint64, found bool) {
	dataLen := len(data)
	if dataLen == 0 {
		return 0, c.fingerprint, false
	}

	// Capture state into local variables (CPU registers)
	fp := c.fingerprint
	pos := int(c.position)
	minSize := int(c.minSize)
	normSize := int(c.normSize)
	maxSize := int(c.maxSize)
	maskS := c.maskS
	maskL := c.maskL
	// We don't capture table as it's an array and would be copied.
	// We access it directly via pointer receiver which is fast.

	// Phase 0: Skip to minimum size WITHOUT computing hash
	if pos < minSize {
		skip := minSize - pos
		if skip > dataLen {
			skip = dataLen
		}

		pos += skip
		data = data[skip:]
		dataLen -= skip
	}

	// Phase 1: Normalized chunking [minSize, normSize)
	// Uses smaller mask (maskS)
	if pos < normSize && dataLen > 0 {
		end := normSize - pos
		if end > dataLen {
			end = dataLen
		}

		i := 0
		// Unroll loop 8x for Phase 1
		for ; i+8 <= end; i += 8 {
			// 1
			fp = (fp << 1) + c.table[data[i]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 1, fp, true
			}
			// 2
			fp = (fp << 1) + c.table[data[i+1]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 2, fp, true
			}
			// 3
			fp = (fp << 1) + c.table[data[i+2]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 3, fp, true
			}
			// 4
			fp = (fp << 1) + c.table[data[i+3]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 4, fp, true
			}
			// 5
			fp = (fp << 1) + c.table[data[i+4]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 5, fp, true
			}
			// 6
			fp = (fp << 1) + c.table[data[i+5]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 6, fp, true
			}
			// 7
			fp = (fp << 1) + c.table[data[i+6]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 7, fp, true
			}
			// 8
			fp = (fp << 1) + c.table[data[i+7]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 8, fp, true
			}
		}

		// Handle remaining bytes for Phase 1
		for ; i < end; i++ {
			fp = (fp << 1) + c.table[data[i]]
			if (fp & maskS) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 1, fp, true
			}
		}

		pos += end
		data = data[end:]
		dataLen -= end
	}

	// Phase 2: Standard chunking [normSize, maxSize)
	// Uses larger mask (maskL)
	if pos < maxSize && dataLen > 0 {
		end := maxSize - pos
		if end > dataLen {
			end = dataLen
		}

		i := 0
		// Unroll loop 8x for Phase 2
		for ; i+8 <= end; i += 8 {
			// 1
			fp = (fp << 1) + c.table[data[i]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 1, fp, true
			}
			// 2
			fp = (fp << 1) + c.table[data[i+1]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 2, fp, true
			}
			// 3
			fp = (fp << 1) + c.table[data[i+2]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 3, fp, true
			}
			// 4
			fp = (fp << 1) + c.table[data[i+3]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 4, fp, true
			}
			// 5
			fp = (fp << 1) + c.table[data[i+4]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 5, fp, true
			}
			// 6
			fp = (fp << 1) + c.table[data[i+5]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 6, fp, true
			}
			// 7
			fp = (fp << 1) + c.table[data[i+6]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 7, fp, true
			}
			// 8
			fp = (fp << 1) + c.table[data[i+7]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 8, fp, true
			}
		}

		// Handle remaining bytes for Phase 2
		for ; i < end; i++ {
			fp = (fp << 1) + c.table[data[i]]
			if (fp & maskL) == 0 {
				c.fingerprint = fp
				c.position = 0
				return pos + i + 1, fp, true
			}
		}

		pos += end
		data = data[end:]
		dataLen -= end
	}

	// Phase 3: Hard limit at maxSize
	if pos >= maxSize {
		c.fingerprint = fp
		c.position = 0 // Reset for next chunk
		return pos, fp, true
	}

	// No boundary found, save state for next call
	c.fingerprint = fp
	c.position = uint32(pos)
	return pos, fp, false
}

// Position returns the current position within the chunk being processed.
// This can be used to determine how much data has been consumed.
func (c *ChunkerCore) Position() uint32 {
	return c.position
}

// Fingerprint returns the current rolling hash value.
func (c *ChunkerCore) Fingerprint() uint64 {
	return c.fingerprint
}

// MinSize returns the minimum chunk size.
func (c *ChunkerCore) MinSize() uint32 {
	return c.minSize
}

// MaxSize returns the maximum chunk size.
func (c *ChunkerCore) MaxSize() uint32 {
	return c.maxSize
}

// NormSize returns the normalization boundary.
func (c *ChunkerCore) NormSize() uint32 {
	return c.normSize
}
