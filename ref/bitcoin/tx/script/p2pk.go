package script

import (
	"fmt"
	"github.com/jchavannes/btcd/txscript"
	"github.com/memocash/index/ref/bitcoin/memo"
)

type P2pk struct {
	PubKey []byte
}

func (p P2pk) Get() ([]byte, error) {
	if len(p.PubKey) != memo.PubKeyLength {
		return nil, fmt.Errorf("invalid pub key length: %d (expected %d)", len(p.PubKey), memo.PubKeyLength)
	}
	pkScript, err := txscript.NewScriptBuilder().
		AddData(p.PubKey).
		AddOp(txscript.OP_CHECKSIG).
		Script()
	if err != nil {
		return nil, fmt.Errorf("error building p2pk script; %w", err)
	}
	return pkScript, nil
}

func (p P2pk) Type() memo.OutputType {
	return memo.OutputTypeP2PK
}
