package script

import (
	"encoding/binary"
	"github.com/jchavannes/btcd/txscript"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/server/ref/bitcoin/memo"
)

type TokenCreate struct {
	Ticker   string
	Name     string
	SlpType  byte
	Decimals int
	DocUrl   string
	Quantity uint64
}

func (t TokenCreate) Get() ([]byte, error) {
	if t.Ticker == "" {
		return nil, jerr.New("ticker not set")
	}
	if t.SlpType == 0 {
		return nil, jerr.New("type not set")
	}
	var quantityBytes = make([]byte, 8)
	binary.BigEndian.PutUint64(quantityBytes, t.Quantity)

	var emptyData = []byte{txscript.OP_PUSHDATA1, 0}

	var script = memo.GetBaseOpReturn().
		AddData(memo.PrefixSlp).
		AddOps([]byte{txscript.OP_DATA_1, t.SlpType}).
		AddData([]byte(memo.SlpTxTypeGenesis)).
		AddData([]byte(t.Ticker))

	if t.Name != "" {
		script = script.AddData([]byte(t.Name))
	} else {
		script = script.AddOps(emptyData)
	}

	if t.DocUrl != "" {
		script = script.AddData([]byte(t.DocUrl))
		// TODO: Support doc hash
	} else {
		script = script.AddOps(emptyData)
	}
	var batonVOut = []byte{txscript.OP_DATA_1, 0x02}
	if t.SlpType == memo.SlpNftChildTokenType {
		batonVOut = emptyData
	}
	pkScript, err := script.
		AddOps(emptyData).
		AddOps([]byte{txscript.OP_DATA_1, byte(t.Decimals % 255)}).
		AddOps(batonVOut).
		AddData(quantityBytes).
		Script()
	if err != nil {
		return nil, jerr.Get("error building script", err)
	}
	return pkScript, nil
}

func (t TokenCreate) Type() memo.OutputType {
	return memo.OutputTypeTokenCreate
}