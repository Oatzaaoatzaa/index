package op_return

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/script"
	"github.com/memocash/index/ref/bitcoin/wallet"
)

var memoFollowHandler = &Handler{
	prefix: memo.PrefixFollow,
	handle: func(info Info) error {
		if len(info.PushData) != 2 {
			return jerr.Newf("invalid set follow, incorrect push data (%d)", len(info.PushData))
		}
		followLockHash := script.GetLockHashForAddress(wallet.GetAddressFromPkHash(info.PushData[1]))
		var memoFollow = &item.MemoFollow{
			LockHash: info.LockHash,
			Height:   info.Height,
			TxHash:   info.TxHash,
			Follow:   followLockHash,
		}
		if err := item.Save([]item.Object{memoFollow}); err != nil {
			return jerr.Get("error saving db memo follow object", err)
		}
		if info.Height != item.HeightMempool {
			memoFollow.Height = item.HeightMempool
			if err := item.RemoveMemoFollow(memoFollow); err != nil {
				return jerr.Get("error removing db memo follow", err)
			}
		}
		return nil
	},
}
