package vm

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

// MemoryStateDB is an in-memory StateDB implementation for testing.
type MemoryStateDB struct {
	storage  map[string]map[string][]byte
	balances map[string]*big.Int
	code     map[string][]byte
	nonces   map[string]uint64
}

func NewMemoryStateDB() *MemoryStateDB {
	return &MemoryStateDB{
		storage:  make(map[string]map[string][]byte),
		balances: make(map[string]*big.Int),
		code:     make(map[string][]byte),
		nonces:   make(map[string]uint64),
	}
}

func addrHex(addr []byte) string {
	return hex.EncodeToString(addr)
}

func (s *MemoryStateDB) GetStorage(contract []byte, key []byte) []byte {
	a := addrHex(contract)
	m, ok := s.storage[a]
	if !ok {
		return nil
	}
	return m[hex.EncodeToString(key)]
}

func (s *MemoryStateDB) SetStorage(contract []byte, key []byte, value []byte) {
	a := addrHex(contract)
	m, ok := s.storage[a]
	if !ok {
		m = make(map[string][]byte)
		s.storage[a] = m
	}
	m[hex.EncodeToString(key)] = value
}

func (s *MemoryStateDB) GetBalance(addr []byte) *big.Int {
	b, ok := s.balances[addrHex(addr)]
	if !ok {
		return new(big.Int)
	}
	return new(big.Int).Set(b)
}

func (s *MemoryStateDB) SetBalance(addr []byte, amount *big.Int) {
	s.balances[addrHex(addr)] = new(big.Int).Set(amount)
}

func (s *MemoryStateDB) GetCode(addr []byte) []byte {
	return s.code[addrHex(addr)]
}

func (s *MemoryStateDB) SetCode(addr []byte, bytecode []byte) {
	s.code[addrHex(addr)] = bytecode
}

func (s *MemoryStateDB) GetCodeSize(addr []byte) int {
	return len(s.code[addrHex(addr)])
}

func (s *MemoryStateDB) GetCodeHash(addr []byte) []byte {
	c := s.code[addrHex(addr)]
	if len(c) == 0 {
		return nil
	}
	h := sha256.Sum256(c)
	return h[:]
}

func (s *MemoryStateDB) Exists(addr []byte) bool {
	a := addrHex(addr)
	_, hasCode := s.code[a]
	_, hasBal := s.balances[a]
	_, hasStor := s.storage[a]
	return hasCode || hasBal || hasStor
}

func (s *MemoryStateDB) GetNonce(addr []byte) uint64 {
	return s.nonces[addrHex(addr)]
}

func (s *MemoryStateDB) SetNonce(addr []byte, nonce uint64) {
	s.nonces[addrHex(addr)] = nonce
}

func (s *MemoryStateDB) GetStorageSlotCount(contract []byte) uint64 {
	m, ok := s.storage[addrHex(contract)]
	if !ok {
		return 0
	}
	return uint64(len(m))
}
