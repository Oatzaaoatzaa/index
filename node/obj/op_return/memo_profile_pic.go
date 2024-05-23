package op_return

import (
	"context"
	"fmt"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/db"
	dbMemo "github.com/memocash/index/db/item/memo"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
)

var memoProfilePicHandler = &Handler{
	prefix: memo.PrefixSetProfilePic,
	handle: func(ctx context.Context, info parse.OpReturn) error {
		if len(info.PushData) != 2 {
			if err := item.LogProcessError(&item.ProcessError{
				TxHash: info.TxHash,
				Error:  fmt.Sprintf("invalid set profile pic, incorrect push data (%d)", len(info.PushData)),
			}); err != nil {
				return fmt.Errorf("error saving process error; %w", err)
			}
			return nil
		}
		var pic = jutil.GetUtf8String(info.PushData[1])
		var addrMemoProfilePic = &dbMemo.AddrProfilePic{
			Addr:   info.Addr,
			Seen:   info.Seen,
			TxHash: info.TxHash,
			Pic:    pic,
		}
		if err := db.Save([]db.Object{addrMemoProfilePic}); err != nil {
			return fmt.Errorf("error saving db addr memo profile pic object; %w", err)
		}
		return nil
	},
}
