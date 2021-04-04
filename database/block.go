package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Hash [32]byte

// These methods override the marshalling and unmarshalling for byte array of name Hash
func (h Hash) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(h[:])), nil
}

func (h *Hash) UnmarshalText(data []byte) error {
	_, err := hex.Decode(h[:], data)
	return err
}

// Block has 2 attributes, Header and Payload
// Payload stores new transactions and Header stores the block's metadata
type Block struct {
	Header BlockHeader `json:"header"`
	TXs    []Tx        `json:"payload"` // nwe transactions only (payload)
}

type BlockHeader struct {
	Parent Hash   `json:"parent"` // parent block reference
	Time   uint64 `json:"time"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

func NewBlock(parent Hash, time uint64, txs []Tx) Block {
	return Block{BlockHeader{parent, time}, txs}
}

func (b Block) Hash() (Hash, error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(blockJson), nil
}
