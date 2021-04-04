package main

import (
	"fmt"
	"os"
	"time"

	"github.com/harshrpg/go-blockchain-tut/database"
)

func main() {
	cwd, _ := os.Getwd()
	state, err := database.NewStateFromDisk(cwd)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	defer state.Close()

	block0 := database.NewBlock(
		database.Hash{},
		uint64(time.Now().Unix()),
		[]database.Tx{
			database.NewTx("owner", "owner", 3, ""),
			database.NewTx("owner", "owner", 700, "reward"),
		},
	)

	state.AddBlock(block0)
	block0Hash, _ := state.Persist()

	block1 := database.NewBlock(
		block0Hash,
		uint64(time.Now().Unix()),
		[]database.Tx{
			database.NewTx("owner", "harsh", 2000, ""),
			database.NewTx("owner", "owner", 100, "reward"),
			database.NewTx("harsh", "owner", 1, ""),
			database.NewTx("harsh", "ishan", 1000, ""),
			database.NewTx("harsh", "owner", 50, ""),
			database.NewTx("owner", "owner", 100, "reward"),
		},
	)

	state.AddBlock(block1)
	state.Persist()
}
