package vm

import (
	"errors"
	"math/big"
)

var ErrMemoryLimit = errors.New("memory size exceeds limit")

const DefaultMaxMemorySize = 1 << 20 // 1 MB

type Memory struct {
	data    []byte
	maxSize int
}

func NewMemory() *Memory {
	return &Memory{
		data:    make([]byte, 0, 4096),
		maxSize: DefaultMaxMemorySize,
	}
}

func (m *Memory) Read(offset, length int) []byte {
	if length == 0 {
		return nil
	}
	result := make([]byte, length)
	if offset < len(m.data) {
		n := len(m.data) - offset
		if n > length {
			n = length
		}
		copy(result, m.data[offset:offset+n])
	}
	return result
}

func (m *Memory) Write(offset int, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	end := offset + len(data)
	if end > m.maxSize {
		return ErrMemoryLimit
	}
	if end > len(m.data) {
		m.expand(end)
	}
	copy(m.data[offset:], data)
	return nil
}

func (m *Memory) Expand(newSize int) (uint64, error) {
	if newSize <= len(m.data) {
		return 0, nil
	}
	if newSize > m.maxSize {
		return 0, ErrMemoryLimit
	}
	oldWords := toWordCount(len(m.data))
	newWords := toWordCount(newSize)
	gasCost := memoryGasCost(newWords) - memoryGasCost(oldWords)
	m.expand(newSize)
	return gasCost, nil
}

func (m *Memory) Size() int { return len(m.data) }

func (m *Memory) expand(newSize int) {
	rounded := int(toWordCount(newSize)) * 32
	if rounded <= len(m.data) {
		return
	}
	expanded := make([]byte, rounded)
	copy(expanded, m.data)
	m.data = expanded
}

func toWordCount(byteSize int) uint64 {
	return uint64((byteSize + 31) / 32)
}

func memoryGasCost(words uint64) uint64 {
	return 3*words + words*words/512
}

func (m *Memory) GetByte(offset int) byte {
	if offset < len(m.data) {
		return m.data[offset]
	}
	return 0
}

func (m *Memory) SetByte(offset int, b byte) error {
	end := offset + 1
	if end > m.maxSize {
		return ErrMemoryLimit
	}
	if end > len(m.data) {
		m.expand(end)
	}
	m.data[offset] = b
	return nil
}

func (m *Memory) Slice(offset, length int) []byte {
	if length == 0 {
		return nil
	}
	end := offset + length
	if end > len(m.data) {
		return m.Read(offset, length)
	}
	return m.data[offset:end]
}

func (m *Memory) MStore(offset int, val *big.Int) error {
	b := WordToBytes32(val)
	return m.Write(offset, b[:])
}

func (m *Memory) MLoad(offset int) *big.Int {
	data := m.Read(offset, 32)
	return new(big.Int).SetBytes(data)
}
