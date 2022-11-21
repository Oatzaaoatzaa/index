package dbi

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/btcd/wire"
)

type TxSave interface {
	SaveTxs(*Block) error
}

type BlockSave interface {
	SaveBlock(wire.BlockHeader) error
	GetBlock(int64) ([]byte, error)
}

type BlockHeightSave interface {
	SaveHeights([]*BlockHeight) error
}

type TxBlockSave interface {
	SaveTxBlocks([]*TxBlock) error
}

type InputSave interface {
	SaveInputs([]*Input) error
}

type OutputSave interface {
	SaveOutputs([]*Output) error
}

type Block struct {
	Header       wire.BlockHeader
	Transactions []Tx
}

func (b *Block) IsNil() bool {
	return b == nil || (b.Header.Timestamp.IsZero() && len(b.Transactions) == 0)
}

func (b *Block) ToWireBlock() *wire.MsgBlock {
	return BlockToWireBlock(b)
}

type Tx struct {
	BlockIndex uint32
	MsgTx      *wire.MsgTx
}

func (t *Tx) Hash() chainhash.Hash {
	return t.MsgTx.TxHash()
}

type Input struct {
	Hash         []byte
	Index        uint32
	Sequence     uint32
	PrevHash     []byte
	PrevIndex    uint32
	UnlockScript []byte
}

func (i *Input) GetPrevHash() *chainhash.Hash {
	h, _ := chainhash.NewHash(i.PrevHash)
	return h
}

type Output struct {
	Hash       []byte
	Index      uint32
	Value      int64
	LockScript []byte
}

type TxBlock struct {
	TxHash    []byte
	BlockHash []byte
}

type BlockHeight struct {
	BlockHash []byte
	Height    int64
}

func WireBlockToBlock(wireBlock *wire.MsgBlock) *Block {
	if wireBlock == nil {
		return &Block{}
	}
	block := &Block{Header: wireBlock.Header}
	for i, wireTx := range wireBlock.Transactions {
		tx := WireTxToTx(wireTx, uint32(i))
		block.Transactions = append(block.Transactions, *tx)
	}
	return block
}

func WireTxToTx(wireTx *wire.MsgTx, index uint32) *Tx {
	tx := &Tx{
		MsgTx:      wireTx,
		BlockIndex: index,
	}
	return tx
}

func BlockToWireBlock(block *Block) *wire.MsgBlock {
	if block == nil {
		return nil
	}
	wireBlock := wire.NewMsgBlock(&block.Header)
	for _, tx := range block.Transactions {
		wireBlock.AddTransaction(TxToWireTx(&tx))
	}
	return wireBlock
}

func TxToWireTx(tx *Tx) *wire.MsgTx {
	return tx.MsgTx
}
