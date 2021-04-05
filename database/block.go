package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	Number uint64 `json:"number"` // block height
	Time   uint64 `json:"time"`
}

type BlockFS struct {
	Key   Hash  `json:"hash"`
	Value Block `json:"block"`
}

func NewBlock(parent Hash, number uint64, time uint64, txs []Tx) Block {
	fmt.Printf("Number to be persisted: %d\n", number)
	return Block{BlockHeader{parent, number, time}, txs}
}

func (b Block) Hash() (Hash, error) {
	blockJson, err := json.Marshal(b)
	if err != nil {
		return Hash{}, err
	}
	return sha256.Sum256(blockJson), nil
}
