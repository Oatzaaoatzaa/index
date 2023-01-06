package lead

import (
	"context"
	"github.com/jchavannes/btcd/wire"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jfmt"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/chain"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/node/obj/saver"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/cluster/proto/cluster_pb"
	"github.com/memocash/index/ref/dbi"
	"sync"
)

type Processor struct {
	On          bool
	StopChan    chan struct{}
	Clients     map[int]*Client
	ErrorChan   chan ShardError
	BlockNode   *Node
	MemPoolNode *Node
	Verbose     bool
	Synced      bool
}

func (p *Processor) Start() error {
	if p.On {
		return nil
	}
	p.On = true
	syncStatusTxs, err := item.GetSyncStatus(item.SyncStatusSaveTxs)
	if err != nil && !client.IsEntryNotFoundError(err) {
		return jerr.Get("error getting sync status txs", err)
	}
	if syncStatusTxs != nil {
		p.Synced = true
		go func() {
			p.MemPoolNode = NewNode()
			p.MemPoolNode.Start(true)
			jlog.Logf("Started mempool node...\n")
			for p.Process(<-p.MemPoolNode.NewBlock) {
			}
		}()
	}
	p.StopChan = make(chan struct{})
	p.BlockNode = NewNode()
	p.BlockNode.Start(false)
	go func() {
		jlog.Logf("Started block node...\n")
		for {
			select {
			case block := <-p.BlockNode.NewBlock:
				if p.Process(block) {
					continue
				}
			case <-p.BlockNode.SyncDone:
				jlog.Logf("Node sync done\n")
				p.Synced = true
				recentBlock, err := chain.GetRecentHeightBlock()
				if err != nil {
					jerr.Get("error getting recent height block", err).Fatal()
				}
				if err := db.Save([]db.Object{&item.SyncStatus{
					Name:   item.SyncStatusSaveTxs,
					Height: recentBlock.Height,
				}}); err != nil {
					jerr.Get("error setting sync status txs", err).Fatal()
				}
				p.On = false
				if err := p.Start(); err != nil {
					jerr.Get("error starting lead processor after block sync complete", err).Fatal()
				}
			case <-p.StopChan:
			}
			jlog.Log("Stopping node listener")
			return
		}
	}()
	return nil
}

func (p *Processor) Process(block *wire.MsgBlock) bool {
	if !p.On {
		return false
	}
	var shardBlocks = make(map[uint32]*cluster_pb.Block)
	for i, tx := range block.Transactions {
		txHash := tx.TxHash()
		shard := db.GetShardByte32(txHash[:])
		if _, ok := shardBlocks[shard]; !ok {
			shardBlocks[shard] = &cluster_pb.Block{
				Header: memo.GetRawBlockHeader(block.Header),
			}
		}
		shardBlocks[shard].Txs = append(shardBlocks[shard].Txs, &cluster_pb.Tx{
			Index: uint32(i),
			Raw:   memo.GetRaw(tx),
		})
	}
	blockHash := block.BlockHash()
	blockInfo := dbi.BlockInfo{
		Header:  block.Header,
		Size:    int64(block.SerializeSize()),
		TxCount: len(block.Transactions),
	}
	blockSaver := saver.NewBlock(p.Verbose)
	if err := blockSaver.SaveBlock(blockInfo); err != nil {
		jerr.Get("error saving block for lead node", err).Print()
		return false
	}
	if blockSaver.NewHeight == 0 {
		// A block without a height can happen if you receive a new block while syncing, ignore it.
		return true
	}
	if !p.WaitForProcess(blockHash[:], blockSaver.NewHeight, shardBlocks, ProcessTypeTx) {
		return false
	}
	if !block.Header.Timestamp.IsZero() {
		jlog.Logf("Saved block: %s %s, %7s txs, size: %14s\n",
			blockHash, block.Header.Timestamp.Format("2006-01-02 15:04:05"), jfmt.AddCommasInt(blockInfo.TxCount),
			jfmt.AddCommasInt(int(blockInfo.Size)))
	}
	return true
}

type ProcessType string

const (
	ProcessTypeTx             ProcessType = "tx"
	ProcessTypeProcessInitial ProcessType = "process-initial"
	ProcessTypeProcess        ProcessType = "process"
)

func (p *Processor) WaitForProcess(blockHash []byte, height int64, shardBlocks map[uint32]*cluster_pb.Block, processType ProcessType) bool {
	var wg sync.WaitGroup
	var hadError bool
	for _, c := range p.Clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			if _, ok := shardBlocks[c.Config.Shard]; !ok && processType == ProcessTypeTx {
				return
			}
			var err error
			switch processType {
			case ProcessTypeTx:
				_, err = c.Client.SaveTxs(context.Background(), &cluster_pb.SaveReq{
					Block:     shardBlocks[c.Config.Shard],
					IsInitial: !p.Synced,
					Height:    height,
				})
			case ProcessTypeProcessInitial:
				_, err = c.Client.ProcessInitial(context.Background(), &cluster_pb.ProcessReq{BlockHash: blockHash[:]})
			case ProcessTypeProcess:
				_, err = c.Client.Process(context.Background(), &cluster_pb.ProcessReq{BlockHash: blockHash[:]})
			}
			if err != nil {
				hadError = true
				p.ErrorChan <- ShardError{
					Shard: c.Config.Int(),
					Error: jerr.Getf(err, "error cluster shard process: %s - %d", processType, c.Config.Shard),
				}
			}
		}(c)
	}
	wg.Wait()
	return !hadError
}

func (p *Processor) Stop() {
	if p.On {
		p.On = false
		close(p.StopChan)
		p.BlockNode.Stop()
	}
}

func NewProcessor(verbose bool, clients map[int]*Client, errorChan chan ShardError) *Processor {
	return &Processor{
		Verbose:   verbose,
		Clients:   clients,
		ErrorChan: errorChan,
	}
}
