package node

import (
	"fmt"
	"net/http"

	"github.com/harshrpg/go-blockchain-tut/database"
)

const DefaultHTTPPort = 8080

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	IsActive    bool   `json:"is_active"`
}

type Node struct {
	dataDir string
	port    uint64

	// To inject the state into HTTP Handlers
	state *database.State

	knownPeers []PeerNode
}

func New(dataDir string, port uint64, bootstrap PeerNode) *Node {
	return &Node{
		dataDir:    dataDir,
		port:       port,
		knownPeers: []PeerNode{bootstrap},
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, isActive bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, isActive}
}

func (n *Node) Run() error {
	fmt.Println(fmt.Sprintf("Listening on HTTP port: %d", n.port))

	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()

	n.state = state

	// listing all the balances
	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})

	// Adding a new transaction
	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, state)
	})

	// Exposing current node's state
	http.HandleFunc("/node/status", func(rw http.ResponseWriter, r *http.Request) {
		statusHandler(rw, r, n)
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", DefaultHTTPPort), nil)
}
