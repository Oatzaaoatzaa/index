package db

import (
	"context"
	"crypto/rand"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/ref/config"
	"sort"
	"sync"
	"time"
)

const (
	TopicBlockTxRaw            = "block_tx_raw"
	TopicDoubleSpendInput      = "double_spend_input"
	TopicDoubleSpendOutput     = "double_spend_output"
	TopicDoubleSpendSeen       = "double_spend_seen"
	TopicFoundPeer             = "found_peer"
	TopicHeightBlockShard      = "height_block_shard"
	TopicHeightProcessed       = "height_processed"
	TopicLockAddress           = "lock_address"
	TopicLockBalance           = "lock_balance"
	TopicLockOutput            = "lock_output"
	TopicLockHeightOutput      = "lock_height_output"
	TopicLockHeightOutputInput = "lock_height_output_input"
	TopicLockUtxo              = "lock_utxo"
	TopicLockUtxoLost          = "lock_utxo_lost"
	TopicMempoolTxRaw          = "mempool_tx_raw"
	TopicMessage               = "message"
	TopicPeer                  = "peer"
	TopicPeerConnection        = "peer_connection"
	TopicPeerFound             = "peer_found"
	TopicProcessError          = "process_error"
	TopicProcessStatus         = "process_status"
	TopicSyncStatus            = "sync_status"
	TopicTx                    = "tx"
	TopicTxLost                = "tx_lost"
	TopicTxProcessed           = "tx_processed"
	TopicTxSeen                = "tx_seen"
	TopicTxSuspect             = "tx_suspect"

	TopicMemoLikeTip              = "memo_like_tip"
	TopicMemoLockHeightFollow     = "memo_lock_height_follow"
	TopicMemoLockHeightFollowed   = "memo_lock_height_followed"
	TopicMemoLockHeightLike       = "memo_lock_height_like"
	TopicMemoLockHeightName       = "memo_lock_height_name"
	TopicMemoLockHeightPost       = "memo_lock_height_post"
	TopicMemoLockHeightProfile    = "memo_lock_height_profile"
	TopicMemoLockHeightProfilePic = "memo_lock_height_profile_pic"
	TopicMemoLockHeightRoomFollow = "memo_lock_height_room_follow"
	TopicMemoPost                 = "memo_post"
	TopicMemoPostChild            = "memo_post_child"
	TopicMemoPostHeightLike       = "memo_post_height_like"
	TopicMemoPostParent           = "memo_post_parent"
	TopicMemoPostRoom             = "memo_post_room"
	TopicMemoRoomHeightFollow     = "memo_room_follow"
	TopicMemoRoomHeightPost       = "memo_room_height_post"

	TopicChainBlock           = "chain_block"
	TopicChainBlockHeight     = "chain_block_height"
	TopicChainHeightBlock     = "chain_height_block"
	TopicChainHeightDuplicate = "chain_height_duplicate"
	TopicChainBlockInfo       = "chain_block_info"
	TopicChainBlockTx         = "chain_block_tx"
	TopicChainOutputInput     = "chain_output_input"
	TopicChainTx              = "chain_tx"
	TopicChainTxBlock         = "chain_tx_block"
	TopicChainTxInput         = "chain_tx_input"
	TopicChainTxOutput        = "chain_tx_output"

	TopicAddrP2pkhHeightOutput = "addr_p2pkh_height_output"
	TopicAddrP2pkhHeightInput  = "addr_p2pkh_height_input"
)

type Object interface {
	GetUid() []byte
	GetTopic() string
	GetShard() uint
	SetUid(uid []byte)
	Serialize() []byte
	Deserialize(data []byte)
}

func GetShardByte(b []byte) uint {
	return GetShard(client.GetByteShard(b))
}

func GetShardByte32(b []byte) uint32 {
	return uint32(GetShardByte(b))
}

func GetShard(shard uint) uint {
	return shard % uint(GetShardCount())
}

func GetShard32(shard uint) uint32 {
	return uint32(GetShard(shard))
}

var _shardCount uint32

func GetShardCount() uint32 {
	if _shardCount == 0 {
		configs := config.GetQueueShards()
		if len(configs) > 0 {
			_shardCount = configs[0].Total
		}
	}
	return _shardCount
}

func Save(objects []Object) error {
	var shardMessages = make(map[uint][]*client.Message)
	for i := 0; len(objects) > 0; i++ {
		var object Object
		object, objects = objects[0], objects[1:]
		uid := object.GetUid()
		if len(uid) == 0 {
			uid = make([]byte, 32)
			_, err := rand.Read(uid)
			if err != nil {
				return jerr.Get("error getting uid", err)
			}
			object.SetUid(uid)
		}
		shard := GetShard(object.GetShard())
		shardMessages[shard] = append(shardMessages[shard], &client.Message{
			Uid:     uid,
			Message: object.Serialize(),
			Topic:   object.GetTopic(),
		})
	}
	configs := config.GetQueueShards()
	var wg sync.WaitGroup
	wg.Add(len(shardMessages))
	var errs []error
	for shardT, messagesT := range shardMessages {
		go func(shard uint, messages []*client.Message) {
			defer wg.Done()
			sort.Slice(messages, func(i, j int) bool {
				return jutil.ByteLT(messages[i].Uid, messages[j].Uid)
			})
			shardConfig := config.GetShardConfig(uint32(shard), configs)
			queueClient := client.NewClient(shardConfig.GetHost())
			err := queueClient.Save(messages, time.Now())
			if err != nil {
				errs = append(errs, jerr.Get("error saving client message", err))
			}
		}(shardT, messagesT)
	}
	wg.Wait()
	if len(errs) > 0 {
		return jerr.Get("error saving messages", jerr.Combine(errs...))
	}
	return nil
}

func Remove(objects []Object) error {
	var shardTopicUids = make(map[uint]map[string][][]byte)
	for _, obj := range objects {
		if shardTopicUids[obj.GetShard()] == nil {
			shardTopicUids[obj.GetShard()] = make(map[string][][]byte)
		}
		shardTopicUids[obj.GetShard()][obj.GetTopic()] =
			append(shardTopicUids[obj.GetShard()][obj.GetTopic()], obj.GetUid())
	}
	queueShards := config.GetQueueShards()
	for shard, topicObjects := range shardTopicUids {
		shardConfig := config.GetShardConfig(GetShard32(shard), queueShards)
		db := client.NewClient(shardConfig.GetHost())
		for topic, uids := range topicObjects {
			if err := db.DeleteMessages(topic, uids); err != nil {
				return jerr.Getf(err, "error deleting shard topic items: %d %s", shard, topic)
			}
		}
	}
	return nil
}

func GetTxHashIndexUid(txHash []byte, index uint32) []byte {
	return jutil.CombineBytes(jutil.ByteReverse(txHash), jutil.GetUint32Data(index))
}

func Set(obj Object, msg client.Message) {
	obj.SetUid(msg.Uid)
	obj.Deserialize(msg.Message)
}

type Wait struct {
	Group sync.WaitGroup
	Lock  sync.RWMutex
	Errs  []error
}

func (w *Wait) AddError(err error) {
	w.Errs = append(w.Errs, err)
}

func NewWait(size int) *Wait {
	var wait = new(Wait)
	wait.Group.Add(size)
	return wait
}

type CancelContext struct {
	Context context.Context
	Cancel  func()
}

func NewCancelContext(ctx context.Context, done func()) *CancelContext {
	var c = new(CancelContext)
	c.Context, c.Cancel = context.WithCancel(ctx)
	if done != nil {
		go func() {
			<-c.Context.Done()
			done()
		}()
	}
	return c
}
