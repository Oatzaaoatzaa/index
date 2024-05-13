package op_return

import (
	"context"
	"fmt"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/db"
	dbMemo "github.com/memocash/index/db/item/memo"
	"github.com/memocash/index/node/obj/op_return/save"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/bitcoin/tx/parse"
)

var memoRoomPostHandler = &Handler{
	prefix: memo.PrefixTopicMessage,
	handle: func(ctx context.Context, info parse.OpReturn) error {
		if len(info.PushData) != 3 {
			if err := item.LogProcessError(&item.ProcessError{
				TxHash: info.TxHash,
				Error:  fmt.Sprintf("invalid chat room post, incorrect push data (%d)", len(info.PushData)),
			}); err != nil {
				return jerr.Get("error saving process error for memo chat room post incorrect push data", err)
			}
			return nil
		}
		var room = jutil.GetUtf8String(info.PushData[1])
		var post = jutil.GetUtf8String(info.PushData[2])
		if err := save.MemoPost(ctx, info, post); err != nil {
			return jerr.Get("error saving memo post for memo chat room post handler", err)
		}
		var memoPostRoom = &dbMemo.PostRoom{
			TxHash: info.TxHash,
			Room:   room,
		}
		// Save first to prevent race condition
		if err := db.Save([]db.Object{memoPostRoom}); err != nil {
			return jerr.Get("error saving db memo room post object", err)
		}
		var memoRoomHeightPost = &dbMemo.RoomPost{
			RoomHash: dbMemo.GetRoomHash(room),
			Seen:     info.Seen,
			TxHash:   info.TxHash,
		}
		if err := db.Save([]db.Object{memoRoomHeightPost}); err != nil {
			return jerr.Get("error saving db memo room height post object", err)
		}
		return nil
	},
}
