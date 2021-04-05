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
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type State struct {
	Balances        map[Account]uint
	txMempool       []Tx
	dbFile          *os.File
	latestBlockHash Hash
	latestBlock     Block
}

// The state struct is constructed by reading the initial user balances from the genesis.json file
func NewStateFromDisk(dataDir string) (*State, error) {
	err := initDataDirIfNotExists(dataDir)
	if err != nil {
		return nil, err
	}

	gen, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	balances := make(map[Account]uint)
	for account, balance := range gen.Balances {
		balances[account] = balance
	}

	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_APPEND|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)

	state := &State{balances, make([]Tx, 0), f, Hash{}, Block{}}

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		// Convert each indicidual json line into an object
		blockFsJson := scanner.Bytes()
		if len(blockFsJson) == 0 {
			break
		}
		var blockFs BlockFS
		err = json.Unmarshal(blockFsJson, &blockFs)
		if err != nil {
			return nil, err
		}

		// state.apply(tx) builds a state with the read transaction from the db file
		if err := state.applyBlock(blockFs.Value); err != nil {
			return nil, err
		}

		state.latestBlockHash = blockFs.Key
		state.latestBlock = blockFs.Value
	}

	return state, nil
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

// Adding new transactions to the mempool
func (s *State) AddBlock(b Block) error {
	fmt.Print("\tDEBUG::Adding a new block\n")
	for _, tx := range b.TXs {
		if err := s.AddTx(tx); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) AddTx(tx Tx) error {
	fmt.Print("\tDEBUG::Adding a new Transaction to Mempool\n")
	if err := s.apply(tx); err != nil {
		return err
	}

	s.txMempool = append(s.txMempool, tx)
	fmt.Print("\tDEBUG::Transaction added to Mempool\n")
	return nil
}

// Persisting the transactions to the disk
func (s *State) Persist() (Hash, error) {
	// Create a new block with only the new transactions
	latestBlockHash, err := s.latestBlock.Hash()
	if err != nil {
		return Hash{}, err
	}
	block := NewBlock(
		latestBlockHash,
		s.latestBlock.Header.Number+1,
		uint64(time.Now().Unix()),
		s.txMempool,
	)

	blockHash, err := block.Hash()
	if err != nil {
		return Hash{}, err
	}
	fmt.Printf("Blcock Hash calculated: %x\n", blockHash)

	blockFs := BlockFS{blockHash, block}
	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Print("Persisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFsJson)

	if _, err = s.dbFile.Write(append(blockFsJson, '\n')); err != nil {
		return Hash{}, err
	}

	s.latestBlockHash = blockHash
	s.latestBlock = block
	// Reset the mempool
	s.txMempool = []Tx{}

	return s.latestBlockHash, nil
}

func (s *State) applyBlock(b Block) error {
	for _, tx := range b.TXs {
		if err := s.apply(tx); err != nil {
			return err
		}
	}
	return nil
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
