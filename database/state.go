/**
The most important database component is the state
https://gist.github.com/josephspurrier/7686b139f29601c3b370

RESPONSIBILITIES:
• Adding new transactions to Mempool
• Validating transactions against the current State (sufficient
sender balance)
• Changing the state
• Persisting transactions to disk
• Calculating accounts balances by replaying all transactions
since Genesis in a sequence
*/

package database

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Snapshot [32]byte

type State struct {
	Balances  map[Account]uint
	txMempool []Tx
	dbFile    *os.File
	snapshot  Snapshot
}

// The state struct is constructed by reading the initial user balances from the genesis.json file
func NewStateFromDisk() (*State, error) {
	// get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	genFilePath := filepath.Join(cwd, "database", "genesis.json")
	gen, err := loadGenesis(genFilePath)
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range gen.Balances {
		balances[account] = balance
	}

	f, err := os.OpenFile(filepath.Join(cwd, "database", "tx.db"), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)

	state := &State{balances, make([]Tx, 0), f, Snapshot{}}

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		// Convert each indicidual json line into an object
		var tx Tx
		json.Unmarshal(scanner.Bytes(), &tx)

		// state.apply(tx) builds a state with the read transaction from the db file
		if err := state.apply(tx); err != nil {
			return nil, err
		}
	}

	err = state.doSnapshot()
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (s *State) LatestSnapshot() Snapshot {
	return s.snapshot
}

// Adding new transactions to the mempool
func (s *State) Add(tx Tx) error {
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMempool = append(s.txMempool, tx)

	return nil
}

// Persisting the transactions to the disk
func (s *State) Persist() (Snapshot, error) {
	// Make a copy of mempool because the s.txMempool will be modified
	// in the loop below
	mempool := make([]Tx, len(s.txMempool))
	copy(mempool, s.txMempool)

	for i := 0; i < len(mempool); i++ {
		txJson, err := json.Marshal(mempool[i])
		if err != nil {
			return Snapshot{}, err
		}

		fmt.Printf("Persisting new TX to disk:\n")
		fmt.Printf("\t%s\n", txJson)
		if _, err = s.dbFile.Write(append(txJson, '\n')); err != nil {
			return Snapshot{}, err
		}

		// Perform snapshot here
		err = s.doSnapshot()
		if err != nil {
			return Snapshot{}, err
		}
		fmt.Printf("New DB Snapshot: %x\n", s.snapshot)

		// Remove the Tx written to a file from the mempool
		s.txMempool = s.txMempool[1:]
	}

	return s.snapshot, nil
}

// Changing/ Validating the state

func (s *State) apply(tx Tx) error {
	if tx.isReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if tx.Value > s.Balances[tx.From] {
		return fmt.Errorf("insufficient balance")
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value
	return nil
}

// close the db file
func (s *State) Close() error {
	return s.dbFile.Close()
}

// creates a Snapshot of the transaction by using a new sha256 secure hashing function
func (s *State) doSnapshot() error {
	// Re-read the entire file from the first byte
	_, err := s.dbFile.Seek(0, 0)
	if err != nil {
		return err
	}

	txsData, err := ioutil.ReadAll(s.dbFile)
	if err != nil {
		return err
	}

	s.snapshot = sha256.Sum256(txsData)
	return nil
}
