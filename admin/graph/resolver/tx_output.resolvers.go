package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/admin/graph/dataloader"
	"github.com/memocash/index/admin/graph/generated"
	"github.com/memocash/index/admin/graph/model"
	"github.com/memocash/index/node/obj/get"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

// Tx is the resolver for the tx field.
func (r *txOutputResolver) Tx(ctx context.Context, obj *model.TxOutput) (*model.Tx, error) {
	preloads := GetPreloads(ctx)
	var tx = &model.Tx{
		Hash: obj.Hash,
	}
	if jutil.StringsInSlice([]string{"outputs", "inputs", "raw"}, preloads) {
		txRaw, err := dataloader.NewTxRawLoader(txRawLoaderConfig).Load(obj.Hash)
		if err != nil {
			return nil, jerr.Get("error getting tx raw for output from loader", err)
		}
		tx.Raw = txRaw
	}
	return tx, nil
}

// Spends is the resolver for the spends field.
func (r *txOutputResolver) Spends(ctx context.Context, obj *model.TxOutput) ([]*model.TxInput, error) {
	txInputs, err := dataloader.NewTxOutputSpendLoader(txOutputSpendLoaderConfig).Load(model.HashIndex{
		Hash:  obj.Hash,
		Index: obj.Index,
	})
	if err != nil {
		return nil, jerr.Get("error getting tx inputs for spends from loader", err)
	}
	return txInputs, nil
}

// DoubleSpend is the resolver for the double_spend field.
func (r *txOutputResolver) DoubleSpend(ctx context.Context, obj *model.TxOutput) (*model.DoubleSpend, error) {
	panic(fmt.Errorf("not implemented"))
}

// Lock is the resolver for the lock field.
func (r *txOutputResolver) Lock(ctx context.Context, obj *model.TxOutput) (*model.Lock, error) {
	if len(obj.Script) == 0 {
		return nil, nil
	}
	lockScript, err := hex.DecodeString(obj.Script)
	if err != nil {
		return nil, jerr.Get("error parsing lock script for tx output lock resolver", err)
	}
	balance := get.NewBalance(lockScript)
	if err := balance.GetBalance(); err != nil {
		return nil, jerr.Get("error getting lock balance for tx output resolver", err)
	}
	return &model.Lock{
		Hash:    hex.EncodeToString(script.GetLockHash(lockScript)),
		Address: wallet.GetAddressStringFromPkScript(lockScript),
		Balance: balance.Balance,
	}, nil
}

// TxOutput returns generated.TxOutputResolver implementation.
func (r *Resolver) TxOutput() generated.TxOutputResolver { return &txOutputResolver{r} }

type txOutputResolver struct{ *Resolver }
