package op_return

import (
	"context"
	"fmt"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/db"
	dbMemo "github.com/memocash/index/db/item/memo"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

var memoLikeHandler = &Handler{
	prefix: memo.PrefixLike,
	handle: func(ctx context.Context, info parse.OpReturn) error {
		if len(info.PushData) != 2 {
			if err := item.LogProcessError(&item.ProcessError{
				TxHash: info.TxHash,
				Error:  fmt.Sprintf("invalid set like, incorrect push data (%d)", len(info.PushData)),
			}); err != nil {
				return fmt.Errorf("error saving process error memo like incorrect push data; %w", err)
			}
			return nil
		}
		if len(info.PushData[1]) != memo.TxHashLength {
			if err := item.LogProcessError(&item.ProcessError{
				TxHash: info.TxHash,
				Error:  fmt.Sprintf("error like tx hash not correct size: %d", len(info.PushData[1])),
			}); err != nil {
				return fmt.Errorf("error saving process error memo like post tx hash; %w", err)
			}
			return nil
		}
		var postTxHash [32]byte
		copy(postTxHash[:], info.PushData[1])
		var memoLike = &dbMemo.AddrLike{
			Addr:       info.Addr,
			Seen:       info.Seen,
			LikeTxHash: info.TxHash,
			PostTxHash: postTxHash,
		}
		var memoLiked = &dbMemo.PostLike{
			PostTxHash: postTxHash,
			Seen:       info.Seen,
			LikeTxHash: info.TxHash,
			Addr:       info.Addr,
		}
		memoPost, err := dbMemo.GetPost(ctx, postTxHash)
		if err != nil {
			return fmt.Errorf("error getting memo post for like op return handler; %w", err)
		}
		var objects = []db.Object{memoLike, memoLiked}
		if memoPost != nil && memoLike.Addr != memoPost.Addr {
			var tip int64
			for _, txOut := range info.Outputs {
				outputAddress, _ := wallet.GetAddrFromLockScript(txOut.PkScript)
				if outputAddress != nil && *outputAddress == memoPost.Addr {
					tip += txOut.Value
				}
			}
			if tip > 0 {
				objects = append(objects, &dbMemo.LikeTip{
					LikeTxHash: info.TxHash,
					Tip:        tip,
				})
			}
		}
		if err := db.Save(objects); err != nil {
			return fmt.Errorf("error saving db memo like object; %w", err)
		}
		return nil
	},
}
