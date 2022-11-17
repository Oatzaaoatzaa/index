package item

import (
	"context"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/config"
)

type LockHeightOutputInput struct {
	LockHash  []byte
	Height    int64
	PrevHash  []byte
	PrevIndex uint32
	Hash      []byte
	Index     uint32
}

func (t LockHeightOutputInput) GetUid() []byte {
	return jutil.CombineBytes(
		t.LockHash,
		jutil.GetInt64DataBig(t.Height),
		jutil.ByteReverse(t.PrevHash),
		jutil.GetUint32Data(t.PrevIndex),
		jutil.ByteReverse(t.Hash),
		jutil.GetUint32Data(t.Index),
	)
}

func (t *LockHeightOutputInput) SetUid(uid []byte) {
	if len(uid) != 112 {
		return
	}
	t.LockHash = uid[:32]
	t.Height = jutil.GetInt64Big(uid[32:40])
	t.PrevHash = jutil.ByteReverse(uid[40:72])
	t.PrevIndex = jutil.GetUint32(uid[72:76])
	t.Hash = jutil.ByteReverse(uid[76:108])
	t.Index = jutil.GetUint32(uid[108:112])
}

func (t LockHeightOutputInput) GetShard() uint {
	return client.GetByteShard(t.PrevHash)
}

func (t LockHeightOutputInput) GetTopic() string {
	return db.TopicLockHeightOutputInput
}

func (t LockHeightOutputInput) Serialize() []byte {
	return nil
}

func (t *LockHeightOutputInput) Deserialize([]byte) {}
func ListenMempoolLockHeightOutputInputs(ctx context.Context, lockHash []byte) (chan *LockHeightOutputInput, error) {
	lockHeightChan, err := ListenMempoolLockHeightOutputInputsMultiple(ctx, [][]byte{lockHash})
	if err != nil {
		return nil, jerr.Get("error getting lock height output listen message chan", err)
	}
	if len(lockHeightChan) != 1 {
		return nil, jerr.Newf("invalid lock height input listen message chan length: %d", len(lockHeightChan))
	}
	return lockHeightChan[0], nil
}
func ListenMempoolLockHeightOutputInputsMultiple(ctx context.Context, lockHashes [][]byte) ([]chan *LockHeightOutputInput, error) {
	var shardLockHashGroups = make(map[uint32][][]byte)
	for _, lockHash := range lockHashes {
		shard := db.GetShardByte32(lockHash)
		shardLockHashGroups[shard] = append(shardLockHashGroups[shard], lockHash)
	}
	var chanLockHeightOutputInputs []chan *LockHeightOutputInput
	for shard, lockHashGroup := range shardLockHashGroups {
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		dbClient := client.NewClient(shardConfig.GetHost())
		chanMessage, err := dbClient.Listen(ctx, db.TopicLockHeightOutputInput, lockHashGroup)
		if err != nil {
			return nil, jerr.Get("error getting lock height output input listen message chan", err)
		}
		var chanLockHeightOutputInput = make(chan *LockHeightOutputInput)
		go func() {
			for {
				msg, ok := <-chanMessage
				if !ok {
					close(chanLockHeightOutputInput)
					return
				}
				var lockHeightOutputInput = new(LockHeightOutputInput)
				db.Set(lockHeightOutputInput, *msg)
				chanLockHeightOutputInput <- lockHeightOutputInput
			}
		}()
		chanLockHeightOutputInputs = append(chanLockHeightOutputInputs, chanLockHeightOutputInput)
	}
	return chanLockHeightOutputInputs, nil
}

func RemoveLockHeightOutputInputs(lockHeightOutputInputs []*LockHeightOutputInput) error {
	var shardUidsMap = make(map[uint32][][]byte)
	for _, lockHeightOutputInput := range lockHeightOutputInputs {
		shard := db.GetShard32(lockHeightOutputInput.GetShard())
		shardUidsMap[shard] = append(shardUidsMap[shard], lockHeightOutputInput.GetUid())
	}
	for shard, shardUids := range shardUidsMap {
		shardUids = jutil.RemoveDupesAndEmpties(shardUids)
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		dbClient := client.NewClient(shardConfig.GetHost())
		if err := dbClient.DeleteMessages(db.TopicLockHeightOutputInput, shardUids); err != nil {
			return jerr.Get("error deleting items topic lock height output input", err)
		}
	}
	return nil
}
