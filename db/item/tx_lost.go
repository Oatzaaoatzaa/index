package item

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/ref/config"
)

type TxLost struct {
	TxHash      []byte
	DoubleSpend []byte
}

func (l TxLost) GetUid() []byte {
	return jutil.CombineBytes(jutil.ByteReverse(l.TxHash), jutil.ByteReverse(l.DoubleSpend))
}

func (l TxLost) GetShard() uint {
	return client.GetByteShard(l.TxHash)
}

func (l TxLost) GetTopic() string {
	return TopicTxLost
}

func (l TxLost) Serialize() []byte {
	return nil
}

func (l *TxLost) SetUid(uid []byte) {
	if len(uid) != 32 && len(uid) != 64 {
		return
	}
	l.TxHash = jutil.ByteReverse(uid[:32])
	if len(uid) != 64 {
		return
	}
	l.DoubleSpend = jutil.ByteReverse(uid[32:])
}

func (l *TxLost) Deserialize([]byte) {}

func GetTxLosts(txHashes [][]byte) ([]*TxLost, error) {
	var shardTxHashGroups = make(map[uint32][][]byte)
	for _, txHash := range txHashes {
		shard := GetShardByte32(txHash)
		shardTxHashGroups[shard] = append(shardTxHashGroups[shard], txHash)
	}
	var txLosts []*TxLost
	for shard, outGroup := range shardTxHashGroups {
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		db := client.NewClient(shardConfig.GetHost())
		var uids = make([][]byte, len(outGroup))
		for i := range outGroup {
			uids[i] = jutil.ByteReverse(outGroup[i])
		}
		if err := db.GetSpecific(TopicTxLost, uids); err != nil {
			return nil, jerr.Get("error getting by uids for tx losts", err)
		}
		for i := range db.Messages {
			var txLost = new(TxLost)
			txLost.SetUid(db.Messages[i].Uid)
			txLosts = append(txLosts, txLost)
		}
	}
	return txLosts, nil
}

func GetAllTxLosts(shard uint32, startTxLost []byte) ([]*TxLost, error) {
	shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
	db := client.NewClient(shardConfig.GetHost())
	var txLosts []*TxLost
	if err := db.GetWOpts(client.Opts{
		Topic: TopicTxLost,
		Start: jutil.ByteReverse(startTxLost),
		Max:   client.DefaultLimit,
	}); err != nil {
		return nil, jerr.Get("error getting all tx losts", err)
	}
	for i := range db.Messages {
		var txLost = new(TxLost)
		txLost.SetUid(db.Messages[i].Uid)
		txLosts = append(txLosts, txLost)
	}
	return txLosts, nil
}

func RemoveTxLosts(txHashes [][]byte) error {
	var shardUidsMap = make(map[uint32][][]byte)
	for _, txHash := range txHashes {
		shard := uint32(GetShard(client.GetByteShard(txHash)))
		shardUidsMap[shard] = append(shardUidsMap[shard], jutil.ByteReverse(txHash))
	}
	for shard, shardUids := range shardUidsMap {
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		db := client.NewClient(shardConfig.GetHost())
		if err := db.DeleteMessages(TopicTxLost, shardUids); err != nil {
			return jerr.Get("error deleting topic tx losts", err)
		}
	}
	return nil
}
