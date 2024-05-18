package load

import (
	"context"
	"fmt"
	"github.com/memocash/index/graph/model"
	"github.com/memocash/index/db/item/slp"
	"github.com/memocash/index/ref/bitcoin/memo"
)

type SlpGeneses struct {
	baseA
	SlpGeneses []*model.SlpGenesis
}

func AttachToSlpGeneses(ctx context.Context, fields []Field, slpGeneses []*model.SlpGenesis) error {
	if len(slpGeneses) == 0 {
		return nil
	}
	o := SlpGeneses{
		baseA:      baseA{Ctx: ctx, Fields: fields},
		SlpGeneses: slpGeneses,
	}
	o.Wait.Add(2)
	go o.AttachSlpOutputs()
	go o.AttachTxs()
	o.Wait.Wait()
	if len(o.Errors) > 0 {
		return fmt.Errorf("error attaching to slp geneses; %w", o.Errors[0])
	}
	return nil
}

func (o *SlpGeneses) GetTokenOuts() []memo.Out {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()
	var txOuts []memo.Out
	for i := range o.SlpGeneses {
		txOuts = append(txOuts, memo.Out{
			TxHash: o.SlpGeneses[i].Hash[:],
			Index:  memo.SlpMintTokenIndex,
		})
	}
	return txOuts
}

func (o *SlpGeneses) AttachSlpOutputs() {
	defer o.Wait.Done()
	if !o.HasField([]string{"output"}) {
		return
	}
	slpOutputs, err := slp.GetOutputs(o.Ctx, o.GetTokenOuts())
	if err != nil {
		o.AddError(fmt.Errorf("error getting tx outputs for model slp geneses; %w", err))
		return
	}
	var allSlpOutputs []*model.SlpOutput
	o.Mutex.Lock()
	for i := range o.SlpGeneses {
		for j := range slpOutputs {
			if o.SlpGeneses[i].Hash != slpOutputs[j].TxHash || memo.SlpMintTokenIndex != slpOutputs[j].Index {
				continue
			}
			o.SlpGeneses[i].Output = &model.SlpOutput{
				Hash:      slpOutputs[j].TxHash,
				Index:     slpOutputs[j].Index,
				TokenHash: slpOutputs[j].TokenHash,
				Amount:    slpOutputs[j].Quantity,
			}
			allSlpOutputs = append(allSlpOutputs, o.SlpGeneses[i].Output)
			break
		}
	}
	o.Mutex.Unlock()
	if err := AttachToSlpOutputs(o.Ctx, GetPrefixFields(o.Fields, "output."), allSlpOutputs); err != nil {
		o.AddError(fmt.Errorf("error attaching to slp outputs for slp geneses; %w", err))
		return
	}
}

func (o *SlpGeneses) AttachTxs() {
	defer o.Wait.Done()
	if !o.HasField([]string{"tx"}) {
		return
	}
	var allTxs []*model.Tx
	o.Mutex.Lock()
	for j := range o.SlpGeneses {
		o.SlpGeneses[j].Tx = &model.Tx{Hash: o.SlpGeneses[j].Hash}
		allTxs = append(allTxs, o.SlpGeneses[j].Tx)
	}
	o.Mutex.Unlock()
	if err := AttachToTxs(o.Ctx, GetPrefixFields(o.Fields, "tx."), allTxs); err != nil {
		o.AddError(fmt.Errorf("error attaching to txs for model slp geneses; %w", err))
		return
	}
}
