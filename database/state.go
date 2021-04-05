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
	"log"
	"os"
	"reflect"

	"github.com/harshrpg/go-blockchain-tut/fs"
)

type State struct {
	Balances        map[Account]uint
	txMempool       []Tx
	dbFile          *os.File
	latestBlockHash Hash
	latestBlock     Block
	hasGenesisBlock bool
}

// The state struct is constructed by reading the initial user balances from the genesis.json file
func NewStateFromDisk(dataDir string) (*State, error) {
	dataDir = fs.ExpandPath(dataDir)
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

	state := &State{balances, make([]Tx, 0), f, Hash{}, Block{}, false}

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
		err = applyTxs(blockFs.Value.TXs, state)
		if err != nil {
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

func (s *State) AddBlocks(blocks []Block) error {
	log.Println("Adding blocks into db")
	for i, b := range blocks {
		log.Printf("Adding block#%d\n", i)
		_, err := s.AddBlock(b)
		if err != nil {
			return err
		}
	}
	return nil
}

// Adding new transactions to the mempool
func (s *State) AddBlock(b Block) (Hash, error) {
	log.Println("Adding a new block")
	log.Println("Initializing state copy")
	pendingState := s.copy()
	log.Println("State copy completed")
	err := applyBlock(b, pendingState)
	if err != nil {
		return Hash{}, err
	}

	log.Println("Calculating Block Hash")
	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	log.Println("Making the blockfs wrapper")
	blockFs := BlockFS{blockHash, b}

	log.Println("Marshalling blockfs into a json object")
	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}
	log.Printf("Blockfs object marshalled: %x", blockFsJson)
	log.Println("Persisting new block to disk")
	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}
	log.Println("Updating State balances")
	s.Balances = pendingState.Balances
	log.Println("Updating State's latestBlock")
	s.latestBlock = b
	log.Println("Updating State's latestBlockHash")
	s.latestBlockHash = blockHash
	return blockHash, nil
}

func applyBlock(b Block, s State) error {
	log.Println("Validating if block can be added as a transaction")
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1
	log.Printf("Next Expected Block Number: %d\n", nextExpectedBlockNumber)

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		log.Fatalf("Next Expected Block Number must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	}

	log.Println("Checking if current block's parent is the latest block in the db")
	if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		log.Fatalf("Next Block parent hash must be '%x' and not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	log.Println("Block valid. Applying transactions")
	return applyTxs(b.TXs, &s)
}

func applyTxs(txs []Tx, s *State) error {
	for _, tx := range txs {
		err := applyTx(tx, s)
		if err != nil {
			return err
		}
	}
	log.Println("All transactions applied.")
	return nil
}

// Changing/ Validating the state

func applyTx(tx Tx, s *State) error {
	if tx.isReward() {
		s.Balances[tx.To] += tx.Value
		return nil
	}

	if tx.Value > s.Balances[tx.From] {
		log.Fatalf("Wrong Tx. Sender '%s' balance is %d TOK. Tx cost is %d TOK", tx.From, s.Balances[tx.From], tx.Value)
	}

	s.Balances[tx.From] -= tx.Value
	s.Balances[tx.To] += tx.Value
	return nil
}

// close the db file
func (s *State) Close() error {
	return s.dbFile.Close()
}

// Internal method to return a copy of the state for security reasons
func (s *State) copy() State {
	c := State{}
	c.hasGenesisBlock = s.hasGenesisBlock
	log.Println("Genesis Block status Copied Successfully")
	c.latestBlock = s.latestBlock
	log.Println("Block Copied Successfully")
	c.latestBlockHash = s.latestBlockHash
	log.Println("Block hash Copied Successfully")
	c.txMempool = make([]Tx, len(s.txMempool))
	log.Println("Block hash Copied Successfully")
	c.Balances = make(map[Account]uint)
	log.Println("Initializing account balance copy")
	for acc, balance := range s.Balances {
		c.Balances[acc] = balance
		log.Printf("Account=%s balance copied successfully", acc)
	}
	log.Println("All account balances copied successfully")
	log.Println("Initializing mempool copy")
	for i, tx := range s.txMempool {
		c.txMempool = append(c.txMempool, tx)
		log.Printf("DEV::Mempool transaction#%d copied successfully", i)
	}
	return c
}

func (s *State) NextBlockNumber() uint64 {
	if !s.hasGenesisBlock {
		return uint64(0)
	}

	return s.latestBlock.Header.Number + 1
}
