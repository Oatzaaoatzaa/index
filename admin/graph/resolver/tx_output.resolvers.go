package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/admin/graph/generated"
	"github.com/memocash/index/admin/graph/load"
	"github.com/memocash/index/admin/graph/model"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

// Tx is the resolver for the tx field.
func (r *txOutputResolver) Tx(ctx context.Context, obj *model.TxOutput) (*model.Tx, error) {
	var tx = &model.Tx{Hash: obj.Hash}
	if err := load.AttachToTxs(load.GetPreloads(ctx), []*model.Tx{tx}); err != nil {
		return nil, jerr.Get("error attaching all to output tx", err)
	}
	return tx, nil
}

// Lock is the resolver for the lock field.
func (r *txOutputResolver) Lock(ctx context.Context, obj *model.TxOutput) (*model.Lock, error) {
	if len(obj.Script) == 0 {
		return nil, nil
	}
	var modelLock = &model.Lock{
		Address: wallet.GetAddressStringFromPkScript(obj.Script),
	}
	if load.HasField(ctx, "balance") {
		// TODO: Reimplement if needed
		return nil, jerr.New("error balance no longer implemented")
	}
	return modelLock, nil
}

// TxOutput returns generated.TxOutputResolver implementation.
func (r *Resolver) TxOutput() generated.TxOutputResolver { return &txOutputResolver{r} }

type txOutputResolver struct{ *Resolver }
