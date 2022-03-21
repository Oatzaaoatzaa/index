package saver

import (
	"github.com/jchavannes/btcd/wire"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/db/item"
)

type ClearSuspect struct {
	Verbose bool
}

func (s *ClearSuspect) SaveTxs(block *wire.MsgBlock) error {
	var txHashes = make([][]byte, len(block.Transactions))
	for i := range block.Transactions {
		txHash := block.Transactions[i].TxHash()
		txHashes[i] = txHash.CloneBytes()
	}
	doubleSpendInputs, err := item.GetDoubleSpendInputsByTxHashes(txHashes)
	if err != nil {
		return jerr.Get("error getting double spend inputs by tx hashes", err)
	}
	var inputTxsToClear = make([][]byte, len(doubleSpendInputs))
	for i := range doubleSpendInputs {
		inputTxsToClear[i] = doubleSpendInputs[i].TxHash
	}
	if err := s.ClearSuspectAndDescendants(inputTxsToClear, true); err != nil {
		return jerr.Get("error clearing suspect and descendants", err)
	}
	return nil
}

func (s *ClearSuspect) ClearSuspectAndDescendants(txHashes [][]byte, checkHasSuspect bool) error {
	for i := 0; len(txHashes) > 0; i++ {
		var processTxHashes = txHashes
		txHashes = nil
		var removeSuspectTxHashes [][]byte
		if checkHasSuspect {
			txSuspects, err := item.GetTxSuspects(processTxHashes)
			if err != nil {
				return jerr.Getf(err, "error getting tx suspects for process clear suspect txs (loop: %d)", i)
			}
			removeSuspectTxHashes = make([][]byte, len(txSuspects))
			for i := range txSuspects {
				removeSuspectTxHashes[i] = txSuspects[i].TxHash
			}
		} else {
			removeSuspectTxHashes = processTxHashes
		}
		if err := item.RemoveTxSuspects(removeSuspectTxHashes); err != nil {
			return jerr.Get("error removing suspect txs", err)
		}
		outputInputs, err := item.GetOutputInputsForTxHashes(removeSuspectTxHashes)
		if err != nil {
			return jerr.Get("error getting output inputs for clear suspect tx hash descendants", err)
		}
		for _, outputInput := range outputInputs {
			txHashes = append(txHashes, outputInput.Hash)
		}
	}
	return nil
}

func (s *ClearSuspect) GetBlock(int64) ([]byte, error) {
	return nil, nil
}

func NewClearSuspect(verbose bool) *ClearSuspect {
	return &ClearSuspect{
		Verbose: verbose,
	}
}
