package parse_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/util/testing/test_tx"
	"log"
	"testing"
)

type SlpSendTest struct {
	PkScript   string
	SlpType    uint16
	TokenHash  []byte
	Quantities []uint64
}

func (tst SlpSendTest) Test(t *testing.T) {
	tokenSend := script.TokenSend{
		TokenHash:  tst.TokenHash,
		SlpType:    byte(tst.SlpType),
		Quantities: tst.Quantities,
	}
	scr, err := tokenSend.Get()
	if err != nil {
		t.Error(fmt.Errorf("error creating token send script; %w", err))
	}
	if hex.EncodeToString(scr) != tst.PkScript {
		t.Error(fmt.Errorf("error scr %x does not match expected %s", scr, tst.PkScript))
	} else if testing.Verbose() {
		log.Printf("scr %x, expected %s\n", scr, tst.PkScript)
	}
	slpSend := parse.NewSlpSend()
	if err := slpSend.Parse(scr); err != nil {
		t.Error(fmt.Errorf("error parsing slp create pk script; %w", err))
	}
	if slpSend.TokenType != tst.SlpType {
		t.Error(fmt.Errorf("slpSend.SlpType %s does not match expected %s",
			memo.SlpTypeString(slpSend.TokenType), memo.SlpTypeString(tst.SlpType)))
	} else if testing.Verbose() {
		log.Printf("slpSend.SlpType %s, expected %s\n",
			memo.SlpTypeString(slpSend.TokenType), memo.SlpTypeString(tst.SlpType))
	}
	if !bytes.Equal(slpSend.TokenHash, tst.TokenHash) {
		t.Error(fmt.Errorf("slpSend.TokenHash %x does not match expected %x", slpSend.TokenHash, tst.TokenHash))
	} else if testing.Verbose() {
		log.Printf("slpSend.TokenHash %x, expected %x\n", slpSend.TokenHash, tst.TokenHash)
	}
	if len(slpSend.Quantities) != len(tst.Quantities) {
		t.Error(fmt.Errorf("len(slpSend.Quantities) %d does not match expected %d",
			len(slpSend.Quantities), len(tst.Quantities)))
	} else {
		for i := range tst.Quantities {
			if slpSend.Quantities[i] != tst.Quantities[i] {
				t.Error(fmt.Errorf("slpSend.Quantities[%d] %d does not match expected %d",
					i, slpSend.Quantities[i], tst.Quantities[i]))
			} else if testing.Verbose() {
				log.Printf("slpSend.Quantities[%d] %d, expected %d\n", i, slpSend.Quantities[i], tst.Quantities[i])
			}
		}
	}
}

const (
	SlpSendDefaultScript  = "6a04534c500001010453454e4420b158efa8e85ef8283481e000f9fb13b12599a8fa58fce12633093762ebd1cb75080000000000002710"
	SlpSendNftGroupScript = "6a04534c500001810453454e4420ad8b36425e100db1b0bb4677dd447cf08babb493afa0fecced1e9f4d13544ad0080000000000000000080000000000000032"
	SlpSendNftChildScript = "6a04534c500001410453454e4420e0a9936a36780efa0e50e30cb466e8077c70623cba95a28e3b2125754c98aab70800000000000000000800000000000005dc"
)

func TestSlpSendDefault(t *testing.T) {
	SlpSendTest{
		PkScript:   SlpSendDefaultScript,
		SlpType:    memo.SlpDefaultTokenType,
		TokenHash:  test_tx.GenericTxHash0,
		Quantities: []uint64{10000},
	}.Test(t)
}

func TestSlpSendNftGroup(t *testing.T) {
	SlpSendTest{
		PkScript:   SlpSendNftGroupScript,
		SlpType:    memo.SlpNftGroupTokenType,
		TokenHash:  test_tx.GenericTxHash1,
		Quantities: []uint64{0, 50},
	}.Test(t)
}

func TestSlpSendNftChild(t *testing.T) {
	SlpSendTest{
		PkScript:   SlpSendNftChildScript,
		SlpType:    memo.SlpNftChildTokenType,
		TokenHash:  test_tx.GenericTxHash2,
		Quantities: []uint64{0, 1500},
	}.Test(t)
}
