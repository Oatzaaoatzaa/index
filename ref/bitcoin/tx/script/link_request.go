package script

import (
	"fmt"
	"github.com/memocash/index/ref/bitcoin/memo"
)

type LinkRequest struct {
	ParentPkHash []byte
	Message      string
}

func (l LinkRequest) Get() ([]byte, error) {
	if len(l.ParentPkHash) != memo.PkHashLength {
		return nil, fmt.Errorf("incorrect parent hash size")
	}
	var msgByte = []byte(l.Message)
	if len(msgByte) > memo.OldMaxSendSize {
		return nil, fmt.Errorf("error message too big %d, max %d", len(msgByte), memo.OldMaxSendSize)
	}
	script := memo.GetBaseOpReturn().
		AddData(memo.PrefixLinkRequest).
		AddData(l.ParentPkHash)
	if len(l.Message) > 0 {
		script = script.AddData(msgByte)
	}
	pkScript, err := script.Script()
	if err != nil {
		return nil, fmt.Errorf("error creating link request script; %w", err)
	}
	return pkScript, nil
}

func (l LinkRequest) Type() memo.OutputType {
	return memo.OutputTypeLinkRequest
}
