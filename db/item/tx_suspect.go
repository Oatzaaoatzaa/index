package item

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/config"
)

type TxSuspect struct {
	TxHash []byte
}

func (s TxSuspect) GetUid() []byte {
	return jutil.ByteReverse(s.TxHash)
}

func (s TxSuspect) GetShard() uint {
	return client.GetByteShard(s.TxHash)
}

func (s TxSuspect) GetTopic() string {
	return db.TopicTxSuspect
}

func (s TxSuspect) Serialize() []byte {
	return nil
}

func (s *TxSuspect) SetUid(uid []byte) {
	if len(uid) != 32 {
		return
	}
	s.TxHash = jutil.ByteReverse(uid)
}

func (s *TxSuspect) Deserialize([]byte) {}

func GetTxSuspects(txHashes [][]byte) ([]*TxSuspect, error) {
	var shardTxHashGroups = make(map[uint32][][]byte)
	for _, txHash := range txHashes {
		shard := db.GetShardByte32(txHash)
		shardTxHashGroups[shard] = append(shardTxHashGroups[shard], txHash)
	}
	var txSuspects []*TxSuspect
	for shard, outGroup := range shardTxHashGroups {
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		dbClient := client.NewClient(shardConfig.GetHost())
		var uids = make([][]byte, len(outGroup))
		for i := range outGroup {
			uids[i] = jutil.ByteReverse(outGroup[i])
		}
		if err := dbClient.GetSpecific(db.TopicTxSuspect, uids); err != nil {
			return nil, jerr.Get("error getting by uids for tx suspects", err)
		}
		for i := range dbClient.Messages {
			var txSuspect = new(TxSuspect)
			db.Set(txSuspect, dbClient.Messages[i])
			txSuspects = append(txSuspects, txSuspect)
		}
	}
	return txSuspects, nil
}

func RemoveTxSuspects(txHashes [][]byte) error {
	var shardUidsMap = make(map[uint32][][]byte)
	for _, txHash := range txHashes {
		shard := uint32(db.GetShard(client.GetByteShard(txHash)))
		shardUidsMap[shard] = append(shardUidsMap[shard], jutil.ByteReverse(txHash))
	}
	for shard, shardUids := range shardUidsMap {
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		dbClient := client.NewClient(shardConfig.GetHost())
		if err := dbClient.DeleteMessages(db.TopicTxSuspect, shardUids); err != nil {
			return jerr.Get("error deleting topic tx suspects", err)
		}
	}
	return nil
}
