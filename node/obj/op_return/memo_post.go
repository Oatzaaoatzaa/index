package op_return

import (
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/node/obj/op_return/save"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
)

var memoPostHandler = &Handler{
	prefix: memo.PrefixPost,
	handle: func(info parse.OpReturn) error {
		if len(info.PushData) != 2 {
			if err := item.LogProcessError(&item.ProcessError{
				TxHash: info.TxHash,
				Error:  fmt.Sprintf("invalid post, incorrect push data (%d)", len(info.PushData)),
			}); err != nil {
				return jerr.Get("error saving process error for memo post incorrect push data", err)
			}
			return nil
		}
		var post = jutil.GetUtf8String(info.PushData[1])
		if err := save.MemoPost(info, post); err != nil {
			return jerr.Get("error saving memo post for memo post handler", err)
		}
		return nil
	},
}
