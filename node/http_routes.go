package node

import (
	"net/http"

	"github.com/harshrpg/go-blockchain-tut/database"
)

type TxAddReq struct {
	From  string `json:"from"`
	To    string `json:"string"`
	Value uint   `json:"value"`
	Data  string `json:"data"`
}

type TxAddRes struct {
	Hash database.Hash `json:"block_hash"`
}

type ErrRes struct {
	Error string `json:"error"`
}

type BalancesRes struct {
	Hash     database.Hash             `json:"block_hash"`
	Balances map[database.Account]uint `json:"balances"`
}

type StatusRes struct {
	Hash       database.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	KnownPeers map[string]PeerNode `json:"peers_known"` // tell me all your peers
}

type SyncRes struct {
	Blocks []database.Block `json:"blocks"`
}

func statusHandler(rw http.ResponseWriter, r *http.Request, n *Node) {
	res := StatusRes{
		Hash:       n.state.LatestBlockHash(),
		Number:     n.state.LatestBlock().Header.Number,
		KnownPeers: n.knownPeers,
	}
	writeRes(rw, res)
}

func txAddHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	req := TxAddReq{}
	err := readReq(r, &req)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	tx := database.NewTx(database.NewAccount(req.From), database.NewAccount(req.To), req.Value, req.Data)

	err = state.AddTx(tx)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	hash, err := state.Persist()
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, TxAddRes{hash})
}

func listBalancesHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	writeRes(w, BalancesRes{state.LatestBlockHash(), state.Balances})
}

func syncHandler(rw http.ResponseWriter, r *http.Request, dataDir string) {
	reqHash := r.URL.Query().Get(endpointSyncQueryFromBlock)
	hash := database.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrRes(rw, err)
		return
	}

	blocks, err := database.GetBlocksAfter(hash, dataDir)
	if err != nil {
		writeErrRes(rw, err)
		return
	}

	writeRes(rw, SyncRes{Blocks: blocks})
}
