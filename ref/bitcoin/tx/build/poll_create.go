package build

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/script"
)

type PollCreateRequest struct {
	Wallet      Wallet
	PollType    memo.PollType
	Question    string
	OptionCount int
}

func PollCreate(request PollCreateRequest) (*memo.Tx, error) {
	tx, err := SimpleSingle(request.Wallet, []*memo.Output{{
		Script: &script.PollCreate{
			PollType:    request.PollType,
			Question:    request.Question,
			OptionCount: request.OptionCount,
		},
	}})
	if err != nil {
		return nil, fmt.Errorf("error building poll create tx; %w", err)
	}
	return tx, nil
}
