package saver

import (
	"github.com/jchavannes/btcd/chaincfg/chainhash"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/chain"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/config"
	"github.com/memocash/index/ref/dbi"
)

type Block struct {
	Verbose         bool
	BlockHash       chainhash.Hash
	PrevBlockHash   chainhash.Hash
	PrevBlockHeight int64
}

func (b *Block) SaveBlock(info dbi.BlockInfo) error {
	b.BlockHash = info.Header.BlockHash()
	if err := b.saveBlockObjects(info); err != nil {
		return jerr.Get("error saving block objects", err)
	}
	return nil
}

func (b *Block) saveBlockObjects(info dbi.BlockInfo) error {
	var objects = make([]db.Object, 1)
	if b.Verbose {
		jlog.Logf("saving block: %s\n", b.BlockHash.String())
	}
	headerRaw := memo.GetRawBlockHeader(info.Header)
	objects[0] = &chain.Block{
		Hash: b.BlockHash,
		Raw:  headerRaw,
	}
	var parentHeight int64
	var hasParent bool
	if info.Header.PrevBlock == b.PrevBlockHash {
		parentHeight = b.PrevBlockHeight
		hasParent = true
	} else {
		parentBlockHeight, err := chain.GetBlockHeight(info.Header.PrevBlock[:])
		if err != nil && !client.IsEntryNotFoundError(err) {
			return jerr.Get("error getting parent block height for potential orphan", err)
		}
		if parentBlockHeight != nil {
			parentHeight = parentBlockHeight.Height
			hasParent = true
			if !jutil.AllZeros(b.PrevBlockHash[:]) {
				objects = append(objects, &chain.HeightDuplicate{
					Height:    parentHeight + 1,
					BlockHash: b.BlockHash,
				})
			}
		}
	}
	var skipHeight bool
	var newBlockHeight int64
	if hasParent {
		newBlockHeight = parentHeight + 1
	} else {
		initBlockParent, err := chainhash.NewHashFromStr(config.GetInitBlockParent())
		if err != nil {
			return jerr.Get("error parsing init block hash", err)
		}
		if *initBlockParent == info.Header.PrevBlock {
			newBlockHeight = int64(config.GetInitBlockHeight())
		} else {
			skipHeight = true
			// block does not match parent or config init block
		}
	}
	var heightBlock *chain.HeightBlock
	if !skipHeight {
		heightBlock = &chain.HeightBlock{
			Height:    newBlockHeight,
			BlockHash: b.BlockHash,
		}
		var blockHeight = &chain.BlockHeight{
			Height:    newBlockHeight,
			BlockHash: b.BlockHash,
		}
		objects = append(objects, blockHeight)
		b.PrevBlockHeight = newBlockHeight
		b.PrevBlockHash = b.BlockHash
	}
	if info.Size > 0 {
		objects = append(objects, &chain.BlockInfo{
			BlockHash: b.BlockHash,
			Size:      info.Size,
			TxCount:   info.TxCount,
		})
	}
	if err := db.Save(objects); err != nil {
		return jerr.Get("error saving new db block objects", err)
	}
	if heightBlock != nil {
		// Save height block afterward to avoid race conditions with listeners not being able to find block info
		if err := db.Save([]db.Object{heightBlock}); err != nil {
			return jerr.Get("error saving height block", err)
		}
	}
	return nil
}

func (b *Block) GetBlock(heightBack int64) (*chainhash.Hash, error) {
	heightBlock, err := chain.GetRecentHeightBlock()
	if err != nil {
		return nil, jerr.Get("error getting recent height block from queue", err)
	}
	if heightBlock == nil {
		return nil, nil
	}
	if heightBack > 0 {
		height := heightBlock.Height - heightBack
		heightBlock, err = chain.GetHeightBlockSingle(height)
		if err != nil {
			return nil, jerr.Getf(err, "error getting height back height block (height: %d, back: %d)",
				height, heightBack)
		}
	}
	b.PrevBlockHash = heightBlock.BlockHash
	b.PrevBlockHeight = heightBlock.Height
	blockHash := chainhash.Hash(heightBlock.BlockHash)
	return &blockHash, nil
}

func NewBlock(verbose bool) *Block {
	return &Block{
		Verbose: verbose,
	}
}
