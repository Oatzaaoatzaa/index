package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/admin/graph/generated"
	"github.com/memocash/index/admin/graph/load"
	"github.com/memocash/index/admin/graph/model"
	"github.com/memocash/index/ref/bitcoin/memo"
)

// Tx is the resolver for the tx field.
func (r *slpGenesisResolver) Tx(ctx context.Context, obj *model.SlpGenesis) (*model.Tx, error) {
	tx, err := load.GetTx(ctx, obj.Hash)
	if err != nil {
		return nil, jerr.Get("error getting tx for slp genesis resolver", err)
	}
	return tx, nil
}

// Output is the resolver for the output field.
func (r *slpGenesisResolver) Output(ctx context.Context, obj *model.SlpGenesis) (*model.SlpOutput, error) {
	var slpOutput = &model.SlpOutput{
		Hash:  obj.Hash,
		Index: memo.SlpMintTokenIndex,
	}
	if err := load.AttachToSlpOutputs(ctx, load.GetFields(ctx), []*model.SlpOutput{slpOutput}); err != nil {
		return nil, jerr.Get("error attaching slp output for slp genesis from loader", err)
	}
	return slpOutput, nil
}

// Baton is the resolver for the baton field.
func (r *slpGenesisResolver) Baton(ctx context.Context, obj *model.SlpGenesis) (*model.SlpBaton, error) {
	var slpBaton = &model.SlpBaton{
		Hash:  obj.Hash,
		Index: obj.BatonIndex,
	}
	if err := load.AttachToSlpBatons(ctx, load.GetFields(ctx), []*model.SlpBaton{slpBaton}); err != nil {
		return nil, jerr.Get("error attaching slp baton for slp genesis from loader", err)
	}
	return slpBaton, nil
}

// Output is the resolver for the output field.
func (r *slpOutputResolver) Output(ctx context.Context, obj *model.SlpOutput) (*model.TxOutput, error) {
	txOutput, err := load.GetTxOutput(ctx, obj.Hash, obj.Index)
	if err != nil {
		return nil, jerr.Get("error getting tx output for slp output from loader", err)
	}
	return txOutput, nil
}

// SlpGenesis returns generated.SlpGenesisResolver implementation.
func (r *Resolver) SlpGenesis() generated.SlpGenesisResolver { return &slpGenesisResolver{r} }

// SlpOutput returns generated.SlpOutputResolver implementation.
func (r *Resolver) SlpOutput() generated.SlpOutputResolver { return &slpOutputResolver{r} }

type slpGenesisResolver struct{ *Resolver }
type slpOutputResolver struct{ *Resolver }
