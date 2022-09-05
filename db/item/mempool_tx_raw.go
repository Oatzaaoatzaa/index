package item

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/config"
	"sync"
)

type MempoolTxRaw struct {
	TxHash []byte
	Raw    []byte
}

func (t MempoolTxRaw) GetUid() []byte {
	return jutil.ByteReverse(t.TxHash)
}

func (t MempoolTxRaw) GetShard() uint {
	return client.GetByteShard(t.TxHash)
}

func (t MempoolTxRaw) GetTopic() string {
	return db.TopicMempoolTxRaw
}

func (t MempoolTxRaw) Serialize() []byte {
	return t.Raw
}

func (t *MempoolTxRaw) SetUid(uid []byte) {
	if len(uid) != 32 {
		return
	}
	t.TxHash = jutil.ByteReverse(uid[:32])
}

func (t *MempoolTxRaw) Deserialize(data []byte) {
	t.Raw = data
}

func GetMempoolTxRawByHash(txHash []byte) (*MempoolTxRaw, error) {
	shardConfig := config.GetShardConfig(db.GetShardByte32(txHash), config.GetQueueShards())
	dbClient := client.NewClient(shardConfig.GetHost())
	if err := dbClient.GetSingle(db.TopicMempoolTxRaw, jutil.ByteReverse(txHash)); err != nil {
		return nil, jerr.Get("error getting client message raw tx by hash", err)
	}
	if len(dbClient.Messages) != 1 {
		return nil, jerr.Newf("error unexpected number of client messages raw tx by hash returned (%d)",
			len(dbClient.Messages))
	}
	var tx = new(MempoolTxRaw)
	db.Set(tx, dbClient.Messages[0])
	return tx, nil
}

func GetMempoolTxRawByHashes(txHashes [][]byte) ([]*MempoolTxRaw, error) {
	var shardUids = make(map[uint32][][]byte)
	for _, txHash := range txHashes {
		shard := db.GetShardByte32(txHash)
		shardUids[shard] = append(shardUids[shard], jutil.ByteReverse(txHash))
	}
	var shardMempoolTxRaw = make(map[uint32][]*MempoolTxRaw)
	var wg sync.WaitGroup
	var lock sync.RWMutex
	wg.Add(len(shardUids))
	var errs []error
	for shardT, uidsT := range shardUids {
		go func(shard uint32, uids [][]byte) {
			defer wg.Done()
			shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
			dbClient := client.NewClient(shardConfig.GetHost())
			if err := dbClient.GetSpecific(db.TopicMempoolTxRaw, uids); err != nil {
				errs = append(errs, jerr.Get("error getting client raw tx message", err))
				return
			}
			for _, msg := range dbClient.Messages {
				var mempoolTxRaw = new(MempoolTxRaw)
				db.Set(mempoolTxRaw, msg)
				lock.Lock()
				shardMempoolTxRaw[shard] = append(shardMempoolTxRaw[shard], mempoolTxRaw)
				lock.Unlock()
			}
		}(shardT, uidsT)
	}
	wg.Wait()
	if len(errs) > 0 {
		return nil, jerr.Get("error getting mempool raw tx messages", jerr.Combine(errs...))
	}
	var allTxs []*MempoolTxRaw
	for _, txs := range shardMempoolTxRaw {
		allTxs = append(allTxs, txs...)
	}
	return allTxs, nil
}

// GetMempoolTxs begins on shard 0 if no start tx specified.
// If the limit is not reached it will move onto the next shard.
// If the start tx is specified, results will begin with the shard of the start tx.
func GetMempoolTxs(startTx []byte, limit uint32) ([]*MempoolTxRaw, error) {
	var startShard uint32
	if len(startTx) > 0 {
		startShard = client.GetByteShard32(startTx)
	}
	configQueueShards := config.GetQueueShards()
	startShardConfig := config.GetShardConfig(startShard, configQueueShards)
	if limit == 0 {
		limit = client.LargeLimit
	}
	var txs []*MempoolTxRaw
	for shard := startShardConfig.Shard; shard < startShardConfig.Total; shard++ {
		shardConfig := config.GetShardConfig(shard, configQueueShards)
		dbClient := client.NewClient(shardConfig.GetHost())
		if err := dbClient.GetWOpts(client.Opts{
			Topic: db.TopicMempoolTxRaw,
			Start: startTx,
			Max:   limit,
		}); err != nil {
			return nil, jerr.Get("error getting client message for mempool tx raw", err)
		}
		for _, msg := range dbClient.Messages {
			tx := new(MempoolTxRaw)
			db.Set(tx, msg)
			txs = append(txs, tx)
		}
		limit -= uint32(len(dbClient.Messages))
		if limit <= 0 {
			break
		}
	}
	return txs, nil
}

func RemoveMempoolTxRaws(mempoolTxRaws []*MempoolTxRaw) error {
	var shardUidsMap = make(map[uint32][][]byte)
	for _, mempoolTxRaw := range mempoolTxRaws {
		shard := db.GetShard32(mempoolTxRaw.GetShard())
		shardUidsMap[shard] = append(shardUidsMap[shard], mempoolTxRaw.GetUid())
	}
	for shard, shardUids := range shardUidsMap {
		shardUids = jutil.RemoveDupesAndEmpties(shardUids)
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		dbClient := client.NewClient(shardConfig.GetHost())
		if err := dbClient.DeleteMessages(db.TopicMempoolTxRaw, shardUids); err != nil {
			return jerr.Get("error deleting items topic mempool tx raw", err)
		}
	}
	return nil
}
