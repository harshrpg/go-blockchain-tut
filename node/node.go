package node

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/harshrpg/go-blockchain-tut/database"
)

const DefaultIP = "127.0.0.1"
const DefaultHTTPPort = 8080
const endPointStatus = "/node/status"
const endPointSync = "/node/sync"
const endpointSyncQueryFromBlock = "fromBlock" // /node/sync?fromBloc=0x913223...

const endPointAddPeer = "/node/peer"
const endPointAddPeerQueryKeyIP = "ip"
const endpointAddPeerQueryKeyPort = "port"

type PeerNode struct {
	IP          string `json:"ip"`
	Port        uint64 `json:"port"`
	IsBootstrap bool   `json:"is_bootstrap"`
	// Whenever current node established connection, initiate sync
	connected bool
}

type Node struct {
	dataDir string
	ip      string
	port    uint64

	// To inject the state into HTTP Handlers
	state *database.State

	knownPeers map[string]PeerNode
}

func (pn PeerNode) TcpAddress() string {
	log.Printf("Fetching Peer Node's TCP Address: '%s:%d'", pn.IP, pn.Port)
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func New(dataDir string, ip string, port uint64, bootstrap PeerNode) *Node {
	log.Println("Crearing a new node")
	knownPeers := make(map[string]PeerNode)
	knownPeers[bootstrap.TcpAddress()] = bootstrap
	return &Node{
		dataDir:    dataDir,
		ip:         ip,
		port:       port,
		knownPeers: knownPeers,
	}
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, connected bool) PeerNode {
	log.Println("Creating a new Peer node")
	return PeerNode{ip, port, isBootstrap, connected}
}

func (n *Node) Run() error {
	ctx := context.Background()
	log.Println(fmt.Sprintf("Listening on %s:%d", n.ip, n.port))

	log.Println("Fetching new state from the disk")
	state, err := database.NewStateFromDisk(n.dataDir)
	if err != nil {
		return err
	}
	defer state.Close()
	log.Println("State fetched and safely closed")
	n.state = state

	go n.sync(ctx)

	// listing all the balances
	http.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, r, state)
	})

	// Adding a new transaction
	http.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		txAddHandler(w, r, state)
	})

	// Exposing current node's state
	http.HandleFunc(endPointStatus, func(rw http.ResponseWriter, r *http.Request) {
		statusHandler(rw, r, n)
	})

	// Sync with peers
	http.HandleFunc(endPointSync, func(rw http.ResponseWriter, r *http.Request) {
		syncHandler(rw, r, n)
	})

	// Adding a new peer
	http.HandleFunc(endPointAddPeer, func(rw http.ResponseWriter, r *http.Request) {
		log.Println("Received request to add a new peer")
		addPeerHandler(rw, r, n)
	})

	return http.ListenAndServe(fmt.Sprintf("%s:%d", n.ip, n.port), nil)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	log.Printf("Removing Peer: %s\n", peer.TcpAddress())
	delete(n.knownPeers, peer.TcpAddress())
	log.Println("Peer removed")
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	log.Println("Checking if the peer is known")
	if peer.IP == n.ip && peer.Port == n.port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]
	return isKnownPeer
}
