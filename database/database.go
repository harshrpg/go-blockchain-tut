package database

import (
	"bufio"
	"encoding/json"
	"os"
)

func GetBlocksAfter(blockHash Hash, dataDir string) ([]Block, error) {
	f, err := os.OpenFile(getBlocksDbFilePath(dataDir), os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}

	blocks := make([]Block, 0)
	shouldStartCollecing := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		var blockFs BlockFS
		err = json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		if shouldStartCollecing {
			blocks = append(blocks, blockFs.Value)
			continue
		}

		if blockHash == blockFs.Key {
			shouldStartCollecing = true
		}
	}

	return blocks, nil
}
