package maint

import (
	"bytes"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/node/obj/saver"
	"github.com/memocash/index/ref/config"
	"github.com/spf13/cobra"
)

var txLostCleanupCmd = &cobra.Command{
	Use: "tx-lost",
	Run: func(cmd *cobra.Command, args []string) {
		var totalTxLosts int
		var txLostsRemoved int
		currentHeightBlock, err := item.GetRecentHeightBlock()
		if err != nil {
			jerr.Get("fatal error getting recent height block", err).Fatal()
		}
		confirmsRequired := config.GetBlocksToConfirm()
		for _, shard := range config.GetQueueShards() {
			var lastTxHash []byte
			for {
				txLosts, err := item.GetAllTxLosts(shard.Min, lastTxHash)
				if err != nil {
					jerr.Get("fatal error getting all tx losts for cleanup", err).Fatal()
				}
				if len(lastTxHash) > 0 && len(txLosts) > 0 && bytes.Equal(txLosts[0].TxHash, lastTxHash) {
					txLosts = txLosts[1:]
				}
				var txHashes = make([][]byte, len(txLosts))
				for i := range txLosts {
					if len(txLosts[i].DoubleSpend) > 0 {
						txHashes[i] = txLosts[i].DoubleSpend
					} else {
						txHashes[i] = txLosts[i].TxHash
					}
				}
				txBlocks, err := item.GetTxBlocks(txHashes)
				if err != nil {
					jerr.Get("fatal error getting tx blocks for tx losts maint", err).Fatal()
				}
				var blockHashes = make([][]byte, len(txBlocks))
				for i := range txBlocks {
					blockHashes[i] = txBlocks[i].BlockHash
				}
				blockHeights, err := item.GetBlockHeights(blockHashes)
				if err != nil {
					jerr.Get("fatal error getting block heights", err).Fatal()
				}
				var txLostsToRemove = make([][]byte, len(txBlocks))
			TxBlocksLoop:
				for i := range txBlocks {
					var blockHeightFound *item.BlockHeight
					for _, blockHeight := range blockHeights {
						if bytes.Equal(blockHeight.BlockHash, txBlocks[i].BlockHash) {
							blockHeightFound = blockHeight
							break
						}
					}
					if blockHeightFound == nil || currentHeightBlock.Height-blockHeightFound.Height < int64(confirmsRequired) {
						// Block found but not considered confirmed yet
						continue
					}
					for _, txLost := range txLosts {
						var txLostTxHash []byte
						if len(txLost.DoubleSpend) > 0 {
							txLostTxHash = txLost.DoubleSpend
						} else {
							txLostTxHash = txLost.TxHash
						}
						if bytes.Equal(txLostTxHash, txBlocks[i].TxHash) {
							txLostsToRemove[i] = txLost.TxHash
							continue TxBlocksLoop
						}
					}
				}
				lockHashBalancesToRemove, err := saver.GetTxLockHashes(txLostsToRemove)
				if err != nil {
					jerr.Get("fatal error getting tx lock hashes", err).Fatal()
				}
				if err := item.RemoveLockBalances(lockHashBalancesToRemove); err != nil {
					jerr.Get("fatal error removing lock balances", err)
				}
				if err := item.RemoveTxLosts(txLostsToRemove); err != nil {
					jerr.Get("fatal error removing tx losts for maint", err).Fatal()
				}
				totalTxLosts += len(txLosts)
				txLostsRemoved += len(txLostsToRemove)
				jlog.Logf("len(txLosts): %d, len(txLostsToRemove): %d, len(lockHashBalancesToRemove): %d\n",
					len(txLosts), len(txLostsToRemove), len(lockHashBalancesToRemove))
				if len(txLosts) < client.DefaultLimit-1 {
					break
				}
				lastTxHash = txLosts[len(txLosts)-1].TxHash
			}
		}
		jlog.Logf("TotalTxLosts: %d, TxLostsRemoved: %d\n", totalTxLosts, txLostsRemoved)
	},
}
