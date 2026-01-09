package runtime

import "sync"

// RingBuffer is a fixed-size circular buffer for storing output history.
type RingBuffer struct {
	mu    sync.RWMutex
	data  []byte
	size  int
	start int
	end   int
	full  bool
}

// NewRingBuffer creates a new ring buffer with the given capacity.
func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write appends data to the buffer, overwriting oldest data if full.
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n = len(p)
	for _, b := range p {
		r.data[r.end] = b
		r.end = (r.end + 1) % r.size

		if r.full {
			r.start = (r.start + 1) % r.size
		}

		if r.end == r.start {
			r.full = true
		}
	}
	return n, nil
}

// Bytes returns all data in the buffer in order.
func (r *RingBuffer) Bytes() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.start == r.end && !r.full {
		return nil
	}

	var result []byte
	if r.full || r.end <= r.start {
		// Buffer has wrapped
		result = make([]byte, 0, r.size)
		result = append(result, r.data[r.start:]...)
		result = append(result, r.data[:r.end]...)
	} else {
		result = make([]byte, r.end-r.start)
		copy(result, r.data[r.start:r.end])
	}
	return result
}

// Len returns the current number of bytes in the buffer.
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.full {
		return r.size
	}
	if r.end >= r.start {
		return r.end - r.start
	}
	return r.size - r.start + r.end
}

// Reset clears the buffer.
func (r *RingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.start = 0
	r.end = 0
	r.full = false
}
