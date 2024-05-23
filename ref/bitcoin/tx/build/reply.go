package build

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/script"
)

type ReplyRequest struct {
	Wallet  Wallet
	TxHash  []byte
	Message string
}

func Reply(request ReplyRequest) ([]*memo.Tx, error) {
	txs, err := Simple(request.Wallet, []*memo.Output{{
		Script: &script.Reply{
			TxHash:  request.TxHash,
			Message: request.Message,
		},
	}})
	if err != nil {
		return nil, fmt.Errorf("error building reply tx; %w", err)
	}
	return txs, nil
}
