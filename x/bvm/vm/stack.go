package vm

import (
	"errors"
	"math/big"
)

var (
	ErrStackOverflow  = errors.New("stack overflow")
	ErrStackUnderflow = errors.New("stack underflow")
)

const MaxStackDepth = 1024

type Stack struct {
	items    []*big.Int
	maxDepth int
}

func NewStack() *Stack {
	return &Stack{
		items:    make([]*big.Int, 0, 64),
		maxDepth: MaxStackDepth,
	}
}

func (s *Stack) Push(v *big.Int) error {
	if len(s.items) >= s.maxDepth {
		return ErrStackOverflow
	}
	s.items = append(s.items, v)
	return nil
}

func (s *Stack) Pop() (*big.Int, error) {
	if len(s.items) == 0 {
		return nil, ErrStackUnderflow
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, nil
}

func (s *Stack) Peek(depth int) (*big.Int, error) {
	idx := len(s.items) - 1 - depth
	if idx < 0 {
		return nil, ErrStackUnderflow
	}
	return s.items[idx], nil
}

func (s *Stack) Dup(depth int) error {
	val, err := s.Peek(depth)
	if err != nil {
		return err
	}
	return s.Push(new(big.Int).Set(val))
}

func (s *Stack) Swap(depth int) error {
	topIdx := len(s.items) - 1
	swapIdx := topIdx - depth
	if swapIdx < 0 {
		return ErrStackUnderflow
	}
	s.items[topIdx], s.items[swapIdx] = s.items[swapIdx], s.items[topIdx]
	return nil
}

func (s *Stack) Depth() int {
	return len(s.items)
}
