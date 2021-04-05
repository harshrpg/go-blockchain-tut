package node

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

type AddPeerRes struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
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

	block := database.NewBlock(
		state.LatestBlockHash(),
		state.NextBlockNumber(),
		uint64(time.Now().Unix()),
		[]database.Tx{tx},
	)

	hash, err := state.AddBlock(block)
	if err != nil {
		writeErrRes(w, err)
		return
	}

	writeRes(w, TxAddRes{hash})
}

func listBalancesHandler(w http.ResponseWriter, r *http.Request, state *database.State) {
	writeRes(w, BalancesRes{state.LatestBlockHash(), state.Balances})
}

func syncHandler(rw http.ResponseWriter, r *http.Request, node *Node) {
	log.Println("Handling sync request for node")
	reqHash := r.URL.Query().Get(endpointSyncQueryFromBlock)
	hash := database.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrRes(rw, err)
		return
	}

	blocks, err := database.GetBlocksAfter(hash, node.dataDir)
	if err != nil {
		writeErrRes(rw, err)
		return
	}

	writeRes(rw, SyncRes{Blocks: blocks})
}

func addPeerHandler(rw http.ResponseWriter, r *http.Request, n *Node) {
	log.Println("Fetching peer's IP")
	peerIp := r.URL.Query().Get(endPointAddPeerQueryKeyIP)
	log.Printf("Peer Ip Found")
	log.Println("Fetching peer's Connection port")
	peerPortRaw := r.URL.Query().Get(endpointAddPeerQueryKeyPort)

	peerPort, err := strconv.ParseInt(peerPortRaw, 10, 32)
	if err != nil {
		log.Print("Error occurred while parsing peer's port to integer")
		writeRes(rw, AddPeerRes{false, err.Error()})
		return
	}
	log.Println("Peer Port found")
	log.Println("Creating a new Peer Node")
	peer := NewPeerNode(peerIp, uint64(peerPort), false, true) // IMPROVEMENT: can fetch peer's activity from peer itself
	log.Println("Adding peer to node")
	n.AddPeer(peer)
	fmt.Printf("Peer %s was addedd successfully to Known Peer's of this node", peer.TcpAddress())
	writeRes(rw, AddPeerRes{true, ""})
}
