package op_return

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/ref/bitcoin/memo"
)

var memoNameHandler = &Handler{
	prefix: memo.PrefixSetName,
	handle: func(info Info) error {
		if len(info.PushData) != 2 {
			return jerr.Newf("invalid set name, incorrect push data (%d)", len(info.PushData))
		}
		var name = jutil.GetUtf8String(info.PushData[1])
		var setName = &item.MemoName{
			LockHash: info.LockHash,
			Height:   info.Height,
			TxHash:   info.TxHash,
			Name:     name,
		}
		if err := item.Save([]item.Object{setName}); err != nil {
			return jerr.Get("error saving db memo name object", err)
		}
		return nil
	},
}
