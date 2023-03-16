package load

import (
	"context"
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/admin/graph/model"
)

func Tx(ctx context.Context, txHash [32]byte) (*model.Tx, error) {
	var tx = &model.Tx{Hash: txHash}
	if err := AttachAllToTxs(GetPreloads(ctx), []*model.Tx{tx}); err != nil {
		return nil, jerr.Get("error attaching all to single tx", err)
	}
	return tx, nil
}

func TxString(ctx context.Context, txHash string) (*model.Tx, error) {
	txHashBytes, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return nil, jerr.Get("error decoding tx hash from string for graph load", err)
	}
	tx, err := Tx(ctx, *txHashBytes)
	if err != nil {
		return nil, jerr.Get("error getting tx for graph load from string", err)
	}
	return tx, nil
}
