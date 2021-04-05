package node

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/harshrpg/go-blockchain-tut/database"
)

func (n *Node) sync(ctx context.Context) error {
	tickerTimer := 45 * time.Second
	log.Printf("Synching node at every: %x\n", tickerTimer)
	ticker := time.NewTicker(tickerTimer)

	for {
		select {
		case <-ticker.C:
			log.Println("Initializing sync")
			n.doSync()

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync() {
	log.Println("Performing sync for node")
	for i, peer := range n.knownPeers {
		log.Printf("Checking if known peer #%x is same as current node\n", i)
		if n.ip == peer.IP && n.port == peer.Port {
			continue // IMPROVEMENT: Refactor this loop
		}
		log.Printf("Searching for new Peers and their Blocks and Peers: %s\n", peer.TcpAddress())

		status, err := queryPeerStatus(peer)
		if err != nil {
			log.Printf("Error occured: %s\n", err)
			log.Printf("Peer %s was removed from this node's known Peers\n", peer.TcpAddress())
			n.RemovePeer(peer)
			continue
		}

		err = n.joinKnownPeers(peer)
		if err != nil {
			log.Printf("Error after joining peers: %s\n", err)
			continue
		}

		err = n.syncBlocks(peer, status)
		if err != nil {
			log.Printf("Error while synching blocks from peer. Err: %s\n", err)
			continue
		}

		log.Println("Synching known peers with node")
		err = n.syncKnownPeers(peer, status)
		if err != nil {
			log.Printf("Error occurrec while synching known peers with the node. Err: %s\n", err)
			continue
		}
	}
}

func (n *Node) syncBlocks(peer PeerNode, status StatusRes) error {
	log.Println("Syncing peer nodes")
	localBlockNumber := n.state.LatestBlock().Header.Number
	log.Println("Checking if the peer has no blocks")
	if status.Hash.IsEmpty() {
		log.Println("Peer has 0 blocks. Ignoring sync")
		return nil
	}
	log.Println("Checking if the peer has only genesis block")
	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		log.Println("Peer has only genesis block. Ignoring sync")
		return nil
	}
	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 {
		newBlocksCount = 1
	}
	log.Printf("Found %d new blocks from Peer %s\n", newBlocksCount, peer.TcpAddress())
	log.Printf("Fetching the remaining blocks from nodes latest block hash: %s\n", n.state.LatestBlockHash())
	blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash()) // From this block hash fetch the remaining blocks
	if err != nil {
		log.Println("Error while fetching blocks from peer")
		return err
	}

	return n.state.AddBlocks(blocks)
}

func (n *Node) joinKnownPeers(peer PeerNode) error {
	log.Printf("Connecting to peer: %s and adding it to known peers for this node\n", peer.TcpAddress())
	if peer.connected {
		log.Println("Peer already connected. Halting")
		return nil
	}

	url := fmt.Sprintf(
		"http://%s%s?%s=%s&%s=%d",
		peer.TcpAddress(),
		endPointAddPeer,
		endPointAddPeerQueryKeyIP,
		n.ip,
		endpointAddPeerQueryKeyPort,
		n.port,
	)

	log.Printf("Url generated for peer: %s\n", url)
	res, err := http.Get(url)
	if err != nil {
		log.Print("Failed to perform GET from peer")
		return err
	}

	addPeerRes := AddPeerRes{}
	err = readRes(res, &addPeerRes)
	if err != nil {
		log.Print("Failed to read response from peer")
		return err
	}

	if addPeerRes.Error != "" {
		log.Printf("ERROR in peer response: %x", addPeerRes.Error)
		return fmt.Errorf(addPeerRes.Error)
	}

	knownPeer := n.knownPeers[peer.TcpAddress()]
	knownPeer.connected = addPeerRes.Success

	n.AddPeer(knownPeer)
	log.Println("Peer added to node's known Peers")

	if !addPeerRes.Success {
		return fmt.Errorf("unable to join KnownPeers of '%s'", peer.TcpAddress())
	}

	return nil
}

func queryPeerStatus(peer PeerNode) (StatusRes, error) {
	url := fmt.Sprintf("http://%s%s", peer.TcpAddress(), endPointStatus)
	res, err := http.Get(url)
	if err != nil {
		return StatusRes{}, err
	}

	statusRes := StatusRes{}
	err = readRes(res, &statusRes)
	if err != nil {
		return StatusRes{}, err
	}

	return statusRes, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock database.Hash) ([]database.Block, error) {
	log.Printf("Importing blocks from Peer %s...\n", peer.TcpAddress())

	// Make this a common attribute
	url := fmt.Sprintf(
		"http://%s%s?%s=%s",
		peer.TcpAddress(),
		endPointSync,
		endpointSyncQueryFromBlock,
		fromBlock.Hex(),
	)

	log.Printf("Import URL generated for Peer: %s\n", url)

	res, err := http.Get(url)
	if err != nil {
		log.Print("Error while performing GET")
		return nil, err
	}

	syncRes := SyncRes{}
	err = readRes(res, &syncRes)
	if err != nil {
		log.Print("Error while reading response")
		return nil, err
	}

	return syncRes.Blocks, nil
}

func (n *Node) syncKnownPeers(peer PeerNode, status StatusRes) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			log.Printf("Found new Peer %s\n", statusPeer.TcpAddress())
			fmt.Printf("Found new Peer %s\n", statusPeer.TcpAddress())
			n.AddPeer(statusPeer)
		}
	}
	return nil
}
