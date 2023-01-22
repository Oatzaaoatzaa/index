package saver

import (
	"bytes"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/chain"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/index/ref/dbi"
	"sort"
)

type Utxo struct {
	Verbose     bool
	InitialSync bool
}

func (u *Utxo) SaveTxs(b *dbi.Block) error {
	if b.IsNil() {
		return jerr.Newf("error nil block")
	}
	var lockUtxos []*item.LockUtxo
	var txOutputs []*chain.TxOutput
	var txInputs []*chain.TxInput
	var lockAddresses []*item.LockAddress
	var ins []memo.Out
	var lockHashes [][]byte
	for _, tx := range b.Transactions {
		var msgTx = tx.MsgTx
		txHash := msgTx.TxHash()
		txHashBytes := txHash.CloneBytes()
		meta := parse.GetMeta(msgTx)
		if meta.Multi {
			jlog.Logf("FOUND meta with multi OP_RETURNS! %s\n", txHash)
		}
		if u.Verbose {
			jlog.Logf("Utxo tx: %s\n", txHash.String())
		}
		specialIndexes := meta.GetSpecialIndexes()
		for g := range msgTx.TxIn {
			ins = append(ins, memo.Out{
				TxHash: msgTx.TxIn[g].PreviousOutPoint.Hash.CloneBytes(),
				Index:  msgTx.TxIn[g].PreviousOutPoint.Index,
			})
			txInputs = append(txInputs, &chain.TxInput{
				TxHash:    txHash,
				Index:     uint32(g),
				PrevHash:  msgTx.TxIn[g].PreviousOutPoint.Hash,
				PrevIndex: msgTx.TxIn[g].PreviousOutPoint.Index,
			})
		}
		for g, txOut := range msgTx.TxOut {
			var lockUtxo = &item.LockUtxo{
				Hash:     txHashBytes,
				Index:    uint32(g),
				Value:    txOut.Value,
				LockHash: script.GetLockHash(txOut.PkScript),
			}
			if jutil.InUintSlice(uint(g), specialIndexes) {
				lockUtxo.Special = true
			}
			lockUtxos = append(lockUtxos, lockUtxo)
			lockHashes = append(lockHashes, lockUtxo.LockHash)
			var txOutput = &chain.TxOutput{
				TxHash:     txHash,
				Index:      uint32(g),
				Value:      txOut.Value,
				LockScript: txOut.PkScript,
			}
			txOutputs = append(txOutputs, txOutput)
			address, _ := wallet.GetAddressFromPkScript(txOut.PkScript)
			if address != nil {
				lockAddresses = append(lockAddresses, &item.LockAddress{
					LockHash: lockUtxo.LockHash,
					Address:  address.GetEncoded(),
				})
			}
		}
	}
	sort.Slice(lockUtxos, func(i, j int) bool {
		switch bytes.Compare(lockUtxos[i].Hash, lockUtxos[j].Hash) {
		case -1:
			return true
		case 1:
			return false
		default:
			return lockUtxos[i].Index < lockUtxos[j].Index
		}
	})
	sort.Slice(ins, func(i, j int) bool {
		switch bytes.Compare(ins[i].TxHash, ins[j].TxHash) {
		case -1:
			return true
		case 1:
			return false
		default:
			return ins[i].Index < ins[j].Index
		}
	})
	var outs []memo.Out
	var g = 0
LockUtxoLoop:
	for i := 0; i < len(lockUtxos); i++ {
		lockUtxo := lockUtxos[i]
		for ; g < len(ins); g++ {
			if bytes.Equal(ins[g].TxHash, lockUtxo.Hash) && ins[g].Index == lockUtxo.Index {
				lockUtxos = append(lockUtxos[:i], lockUtxos[i+1:]...)
				i--
				continue LockUtxoLoop
			}
			switch bytes.Compare(ins[g].TxHash, lockUtxo.Hash) {
			case 1:
				break
			case 0:
				if ins[g].Index > lockUtxo.Index {
					break
				}
			}
		}
		outs = append(outs, memo.Out{
			TxHash: lockUtxos[i].Hash,
			Index:  lockUtxos[i].Index,
		})
	}
	outputInputs, err := chain.GetOutputInputs(outs)
	if err != nil {
		return jerr.Get("error getting utxo output inputs", err)
	}
	for i := 0; i < len(lockUtxos); i++ {
		lockUtxo := lockUtxos[i]
		for g, outputInput := range outputInputs {
			if bytes.Equal(outputInput.PrevHash[:], lockUtxo.Hash) &&
				outputInput.PrevIndex == lockUtxo.Index {
				lockUtxos = append(lockUtxos[:i], lockUtxos[i+1:]...)
				i--
				outputInputs = append(outputInputs[:g], outputInputs[g+1:]...)
				break
			}
		}
	}
	var numLockUtxos = len(lockUtxos)
	var numLockUtxosAndTxOutputs = numLockUtxos + len(txOutputs)
	var objects = make([]db.Object, numLockUtxosAndTxOutputs+len(txInputs))
	for i := range lockUtxos {
		objects[i] = lockUtxos[i]
	}
	for i := range txOutputs {
		objects[numLockUtxos+i] = txOutputs[i]
	}
	for i := range txInputs {
		objects[numLockUtxosAndTxOutputs+i] = txInputs[i]
	}
	for _, lockAddress := range lockAddresses {
		objects = append(objects, lockAddress)
	}
	if err = db.Save(objects); err != nil {
		return jerr.Get("error saving new utxo objects", err)
	}
	if u.InitialSync {
		return nil
	}
	matchingTxOutputs, err := chain.GetTxOutputs(ins)
	if err != nil {
		return jerr.Get("error getting matching tx outputs for inputs", err)
	}
	var spentOuts = make([]*item.LockUtxo, len(matchingTxOutputs))
	for i := range matchingTxOutputs {
		spentOuts[i] = &item.LockUtxo{
			LockHash: script.GetLockHash(matchingTxOutputs[i].LockScript),
			Hash:     matchingTxOutputs[i].TxHash[:],
			Index:    matchingTxOutputs[i].Index,
		}
		lockHashes = append(lockHashes, spentOuts[i].LockHash)
	}
	if err = item.RemoveLockUtxos(spentOuts); err != nil {
		return jerr.Get("error removing lock utxos", err)
	}
	if err = item.RemoveLockBalances(lockHashes); err != nil {
		return jerr.Get("error removing lock balances", err)
	}
	return nil
}

func NewUtxo(verbose bool) *Utxo {
	return &Utxo{
		Verbose: verbose,
	}
}
