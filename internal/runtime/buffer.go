package runtime

import "sync"

// RingBuffer 是一个固定大小的环形缓冲区，用于存储输出历史记录。
// 该实现使用批量拷贝优化写入性能，适用于高吞吐量场景。
// 注意：RingBuffer 只存储原始字节，不对 \r、\n 等控制字符做特殊处理。
type RingBuffer struct {
	mu    sync.RWMutex
	data  []byte
	size  int
	start int // 数据起始位置（最旧的数据）
	end   int // 数据结束位置（下一个写入位置）
	full  bool
}

// NewRingBuffer 创建一个新的环形缓冲区。
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 50000 // 默认 ~50KB
	}
	return &RingBuffer{
		data: make([]byte, size),
		size: size,
	}
}

// Write 将数据追加到缓冲区，如果缓冲区已满则覆盖最旧的数据。
// 使用批量拷贝优化性能，功能与逐字节写入完全等价。
func (r *RingBuffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	n = len(p)

	// 如果写入数据大于等于缓冲区容量，只保留最后 size 字节
	if n >= r.size {
		copy(r.data, p[n-r.size:])
		r.start = 0
		r.end = 0
		r.full = true
		return n, nil
	}

	// 计算写入前的有效数据长度
	oldLen := r.lenLocked()

	// 写入数据（可能需要分两段拷贝）
	if r.end+n <= r.size {
		// 单段拷贝：不需要绕回
		copy(r.data[r.end:], p)
		r.end += n
		if r.end == r.size {
			r.end = 0
		}
	} else {
		// 双段拷贝：需要绕回
		firstPart := r.size - r.end
		copy(r.data[r.end:], p[:firstPart])
		copy(r.data[0:], p[firstPart:])
		r.end = n - firstPart
	}

	// 判断是否发生覆盖，需要更新 start 指针
	newLen := oldLen + n
	if newLen >= r.size {
		// 缓冲区已满或溢出，start 移动到 end 位置
		r.full = true
		r.start = r.end
	}

	return n, nil
}

// lenLocked 返回当前缓冲区中的字节数（调用前需持有锁）。
func (r *RingBuffer) lenLocked() int {
	if r.full {
		return r.size
	}
	if r.end >= r.start {
		return r.end - r.start
	}
	return r.size - r.start + r.end
}

// Bytes 返回缓冲区中的所有数据（按顺序）。
// 返回的切片是数据的拷贝，可安全使用。
func (r *RingBuffer) Bytes() []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.start == r.end && !r.full {
		return nil
	}

	var result []byte
	if r.full || r.end <= r.start {
		// 缓冲区已绕回，需要拼接两段
		totalLen := r.size
		if !r.full {
			totalLen = r.size - r.start + r.end
		}
		result = make([]byte, 0, totalLen)
		result = append(result, r.data[r.start:]...)
		result = append(result, r.data[:r.end]...)
	} else {
		// 未绕回，直接拷贝
		result = make([]byte, r.end-r.start)
		copy(result, r.data[r.start:r.end])
	}
	return result
}

// Len 返回当前缓冲区中的字节数。
func (r *RingBuffer) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lenLocked()
}

// Cap 返回缓冲区的容量。
func (r *RingBuffer) Cap() int {
	return r.size
}

// Reset 清空缓冲区。
func (r *RingBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.start = 0
	r.end = 0
	r.full = false
}
