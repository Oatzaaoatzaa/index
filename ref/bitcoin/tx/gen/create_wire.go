package gen

import (
	"fmt"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/btcd/wire"
	"github.com/memocash/index/ref/bitcoin/memo"
)

func (c Create) getWireTx() (*wire.MsgTx, error) {
	tx, err := CreateWireTx(c.InputsToUse, c.Outputs)
	if err != nil {
		return nil, fmt.Errorf("error getting wire tx; %w", err)
	}
	return tx, nil
}

func CreateWireTx(inputs []memo.UTXO, outputs []*memo.Output) (*wire.MsgTx, error) {
	var msg = wire.NewMsgTx(wire.TxVersion)
	for _, input := range inputs {
		hash, err := chainhash.NewHash(input.Input.PrevOutHash)
		if err != nil {
			return nil, fmt.Errorf("error getting chain hash; %w", err)
		}
		msg.TxIn = append(msg.TxIn, wire.NewTxIn(wire.NewOutPoint(hash, input.Input.PrevOutIndex), nil))
	}
	for _, output := range outputs {
		pkScript, err := output.GetPkScript()
		if err != nil {
			return nil, fmt.Errorf("error getting pk script for output; %w", err)
		}
		msg.TxOut = append(msg.TxOut, wire.NewTxOut(output.Amount, pkScript))
	}
	return msg, nil
}
