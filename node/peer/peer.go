package peer

import (
	"context"
	"fmt"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/btcd/peer"
	"github.com/jchavannes/btcd/wire"
	"github.com/jchavannes/btclog"
	"github.com/jchavannes/jgo/jfmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/index/ref/config"
	"github.com/memocash/index/ref/dbi"
	"log"
	"net"
	"os"
	"time"
)

const (
	MaxHeightBack = 20
)

type Peer struct {
	peer        *peer.Peer
	HandleError func(error)
	BlockSave   dbi.BlockSave
	TxSave      dbi.TxSave
	LastBlock   *chainhash.Hash
	HasExisting bool
	HeightBack  int64
	SyncDone    bool
	Mempool     bool
}

func (p *Peer) Error(err error) {
	if p.HandleError != nil {
		p.HandleError(err)
	} else {
		log.Fatalf("fatal peer node error; %v", err)
	}
}

func (p *Peer) Connect() error {
	SetBtcdLogLevel()
	connectionString := config.GetNodeHost()
	newPeer, err := peer.NewOutboundPeer(&peer.Config{
		UserAgentName:    "memo-index",
		UserAgentVersion: "0.3.0",
		ChainParams:      wallet.GetMainNetParams(),
		Listeners: peer.MessageListeners{
			OnVerAck:      p.OnVerAck,
			OnHeaders:     p.OnHeaders,
			OnInv:         p.OnInv,
			OnBlock:       p.OnBlock,
			OnTx:          p.OnTx,
			OnReject:      p.OnReject,
			OnPing:        p.OnPing,
			OnMerkleBlock: p.OnMerkleBlock,
			OnVersion:     p.OnVersion,
		},
	}, connectionString)
	if err != nil {
		return fmt.Errorf("error getting new outbound peer; %w", err)
	}
	p.peer = newPeer
	log.Printf("Starting node listener: %s\n", connectionString)
	conn, err := net.Dial("tcp", connectionString)
	if err != nil {
		return fmt.Errorf("error getting network connection; %w", err)
	}
	newPeer.AssociateConnection(conn)
	newPeer.WaitForDisconnect()
	return nil
}

func (p *Peer) Disconnect() {
	if p != nil && p.peer != nil {
		p.peer.Disconnect()
	}
}

func (p *Peer) OnVerAck(_ *peer.Peer, _ *wire.MsgVerAck) {
	if p.Mempool {
		p.peer.QueueMessage(wire.NewMsgMemPool(), nil)
		return
	}
	msgGetHeaders := wire.NewMsgGetHeaders()
	if jutil.IsNil(p.BlockSave) {
		return
	}
	blockHash, err := p.BlockSave.GetBlock(0)
	if err != nil {
		p.Error(fmt.Errorf("error getting node block; %w", err))
		return
	}
	if blockHash != nil && blockHash != wallet.GetGenesisBlock().Hash {
		p.HasExisting = true
		msgGetHeaders.BlockLocatorHashes = append(msgGetHeaders.BlockLocatorHashes, blockHash)
	}
	if len(msgGetHeaders.BlockLocatorHashes) == 0 {
		initBlockParent := config.GetInitBlockParent()
		if len(initBlockParent) == 0 {
			initBlock := config.GetInitBlock()
			if initBlock == "" {
				p.Error(fmt.Errorf("error init block not set"))
				return
			}
			p.LastBlock, err = chainhash.NewHashFromStr(initBlock)
			if err != nil {
				p.Error(fmt.Errorf("error getting init block; %w", err))
				return
			}
			msgGetData := wire.NewMsgGetData()
			err := msgGetData.AddInvVect(&wire.InvVect{
				Type: wire.InvTypeBlock,
				Hash: *p.LastBlock,
			})
			if err != nil {
				p.Error(fmt.Errorf("error adding init block inventory vector; %w", err))
				return
			}
			p.peer.QueueMessage(msgGetData, nil)
			return
		}
		blockHash, err := chainhash.NewHashFromStr(initBlockParent)
		if err != nil {
			p.Error(fmt.Errorf("error getting block hash for init block parent; %w", err))
			return
		}
		msgGetHeaders.BlockLocatorHashes = append(msgGetHeaders.BlockLocatorHashes, blockHash)
	}
	p.peer.QueueMessage(msgGetHeaders, nil)
}

func (p *Peer) OnHeaders(_ *peer.Peer, msg *wire.MsgHeaders) {
	if jutil.IsNil(p.BlockSave) {
		return
	}
	if len(msg.Headers) == 0 {
		if !p.SyncDone {
			log.Printf("No headers received, disconnecting, sync done: %t\n", p.SyncDone)
			p.SyncDone = true
			p.Disconnect()
		}
		return
	}
	msgGetData := wire.NewMsgGetData()
	for _, blockHeader := range msg.Headers {
		blockHash := blockHeader.BlockHash()
		if p.HasExisting && blockHash == *wallet.GetFirstBlock().Hash {
			go func() {
				time.Sleep(5 * time.Second)
				p.HeightBack++
				if p.HeightBack > MaxHeightBack {
					p.Error(fmt.Errorf("error beginning of block loop, potential orphan and height back (%d) "+
						"over max height back (%d)", p.HeightBack, MaxHeightBack))
					return
				}
				blockHash, err := p.BlockSave.GetBlock(p.HeightBack)
				if err != nil {
					p.Error(fmt.Errorf("error getting node block after orphan; %w", err))
					return
				}
				msgGetHeaders := wire.NewMsgGetHeaders()
				msgGetHeaders.BlockLocatorHashes = append(msgGetHeaders.BlockLocatorHashes, blockHash)
				p.peer.QueueMessage(msgGetHeaders, nil)
				return
			}()
			//p.Error(fmt.Errorf("error beginning of block loop, potentially due to orphan?"))
			return
		}
		p.HeightBack = 0
		err := msgGetData.AddInvVect(&wire.InvVect{
			Type: wire.InvTypeBlock,
			Hash: blockHeader.BlockHash(),
		})
		if err != nil {
			p.Error(fmt.Errorf("error adding block inventory vector from header; %w", err))
		}
	}
	if len(msgGetData.InvList) > 0 {
		p.LastBlock = &msgGetData.InvList[len(msgGetData.InvList)-1].Hash
		p.peer.QueueMessage(msgGetData, nil)
	}
}

func (p *Peer) OnInv(_ *peer.Peer, msg *wire.MsgInv) {
	msgGetData := wire.NewMsgGetData()
	for _, invItem := range msg.InvList {
		switch invItem.Type {
		case wire.InvTypeTx:
			if !p.Mempool {
				// Don't save mempool items on block node
				continue
			}
			err := msgGetData.AddInvVect(&wire.InvVect{
				Type: wire.InvTypeTx,
				Hash: invItem.Hash,
			})
			if err != nil {
				p.Error(fmt.Errorf("error adding tx inventory vector; %w", err))
			}
		case wire.InvTypeBlock:
			if jutil.IsNil(p.BlockSave) {
				return
			}
			err := msgGetData.AddInvVect(&wire.InvVect{
				Type: wire.InvTypeBlock,
				Hash: invItem.Hash,
			})
			if err != nil {
				p.Error(fmt.Errorf("error adding block inventory vector; %w", err))
			}
		}
	}
	if len(msgGetData.InvList) > 0 {
		p.peer.QueueMessage(msgGetData, nil)
	}
}

func (p *Peer) OnBlock(_ *peer.Peer, msg *wire.MsgBlock, _ []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	if p.TxSave != nil {
		err := p.TxSave.SaveTxs(ctx, dbi.WireBlockToBlock(msg))
		if err != nil {
			p.Error(fmt.Errorf("error saving txs; %w", err))
		}
	}
	// Save block second in case exit/failure during saving transactions will requeue block again
	if !jutil.IsNil(p.BlockSave) {
		err := p.BlockSave.SaveBlock(dbi.BlockInfo{
			Header:  msg.Header,
			Size:    int64(msg.SerializeSize()),
			TxCount: len(msg.Transactions),
		})
		if err != nil {
			p.Error(fmt.Errorf("error saving block; %w", err))
		}
	}
	blockHash := msg.BlockHash()
	if blockHash.IsEqual(p.LastBlock) {
		msgGetHeaders := wire.NewMsgGetHeaders()
		msgGetHeaders.BlockLocatorHashes = append(msgGetHeaders.BlockLocatorHashes, &blockHash)
		p.peer.QueueMessage(msgGetHeaders, nil)
	}
}

func (p *Peer) OnTx(_ *peer.Peer, msg *wire.MsgTx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if p.TxSave != nil {
		log.Printf("OnTx: %s, in: %s, out: %s, size: %s\n", msg.TxHash().String(), jfmt.AddCommasInt(len(msg.TxIn)),
			jfmt.AddCommasInt(len(msg.TxOut)), jfmt.AddCommasInt(msg.SerializeSize()))
		err := p.TxSave.SaveTxs(ctx, dbi.WireBlockToBlock(memo.GetBlockFromTxs([]*wire.MsgTx{msg}, nil)))
		if err != nil {
			p.Error(fmt.Errorf("error saving new tx; %w", err))
		}
		return
	}
}

func (p *Peer) OnReject(_ *peer.Peer, msg *wire.MsgReject) {
	log.Printf("OnReject: %#v\n", msg)
}

func (p *Peer) OnPing(_ *peer.Peer, msg *wire.MsgPing) {
	log.Printf("OnPing: %d\n", msg.Nonce)
	pong := wire.NewMsgPong(msg.Nonce + 1)
	p.peer.QueueMessage(pong, nil)
}

func (p *Peer) OnMerkleBlock(_ *peer.Peer, msg *wire.MsgMerkleBlock) {
	log.Printf("OnMerkleBlock: %#v\n", msg)
}

func (p *Peer) OnVersion(_ *peer.Peer, msg *wire.MsgVersion) {
	log.Printf("OnVersion: %s (last: %d)\n", msg.UserAgent, msg.LastBlock)
}

func (p *Peer) BroadcastTx(ctx context.Context, msgTx *wire.MsgTx) error {
	var done = make(chan struct{})
	p.peer.QueueMessage(msgTx, done)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("error context timeout")
	}
}

func NewConnection(txSave dbi.TxSave, blockSave dbi.BlockSave) *Peer {
	return &Peer{
		BlockSave: blockSave,
		TxSave:    txSave,
	}
}

func SetBtcdLogLevel() {
	logger := btclog.NewBackend(os.Stdout).Logger("MEMO")
	logger.SetLevel(btclog.LevelError)
	peer.UseLogger(logger)
}
