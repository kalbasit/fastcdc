package fastcdc

import (
	"io"
	"sync"
)

// ChunkerPool is a pool of Chunker instances for reuse in high-throughput scenarios.
// It reduces allocations by recycling chunkers instead of creating new ones.
type ChunkerPool struct {
	pool sync.Pool
	opts []Option
}

// NewChunkerPool creates a new ChunkerPool with the given options.
// All chunkers created from this pool will use these options.
func NewChunkerPool(opts ...Option) (*ChunkerPool, error) {
	// Validate options by creating a test chunker
	_, err := NewChunker(nil, opts...)
	if err != nil {
		return nil, err
	}

	return &ChunkerPool{
		opts: opts,
	}, nil
}

// Get retrieves a Chunker from the pool, or creates a new one if the pool is empty.
// The chunker is configured with the given reader and ready to use.
func (p *ChunkerPool) Get(r io.Reader) (*Chunker, error) {
	if v := p.pool.Get(); v != nil {
		chunker := v.(*Chunker)
		chunker.Reset(r)
		return chunker, nil
	}

	return NewChunker(r, p.opts...)
}

// Put returns a Chunker to the pool for reuse.
// The chunker should not be used after being returned to the pool.
func (p *ChunkerPool) Put(c *Chunker) {
	// Clear the reader to avoid holding references
	c.reader = nil
	p.pool.Put(c)
}

// ChunkerCorePool is a pool of ChunkerCore instances for reuse.
type ChunkerCorePool struct {
	pool sync.Pool
	opts []Option
}

// NewChunkerCorePool creates a new ChunkerCorePool with the given options.
// All chunker cores created from this pool will use these options.
func NewChunkerCorePool(opts ...Option) (*ChunkerCorePool, error) {
	// Validate options by creating a test core
	_, err := NewChunkerCore(opts...)
	if err != nil {
		return nil, err
	}

	return &ChunkerCorePool{
		opts: opts,
	}, nil
}

// Get retrieves a ChunkerCore from the pool, or creates a new one if the pool is empty.
func (p *ChunkerCorePool) Get() (*ChunkerCore, error) {
	if v := p.pool.Get(); v != nil {
		core := v.(*ChunkerCore)
		core.Reset()
		return core, nil
	}

	return NewChunkerCore(p.opts...)
}

// Put returns a ChunkerCore to the pool for reuse.
// The core should not be used after being returned to the pool.
func (p *ChunkerCorePool) Put(c *ChunkerCore) {
	c.Reset()
	p.pool.Put(c)
}
