package item

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/config"
	"sort"
)

type TxOutput struct {
	TxHash   []byte
	Index    uint32
	Value    int64
	LockHash []byte
}

func (t TxOutput) GetUid() []byte {
	return GetTxOutputUid(t.TxHash, t.Index)
}

func (t TxOutput) GetShard() uint {
	return client.GetByteShard(t.TxHash)
}

func (t TxOutput) GetTopic() string {
	return db.TopicTxOutput
}

func (t TxOutput) Serialize() []byte {
	return jutil.CombineBytes(
		jutil.GetInt64Data(t.Value),
		t.LockHash,
	)
}

func (t *TxOutput) SetUid(uid []byte) {
	if len(uid) != 36 {
		return
	}
	t.TxHash = jutil.ByteReverse(uid[:32])
	t.Index = jutil.GetUint32(uid[32:36])
}

func (t *TxOutput) Deserialize(data []byte) {
	if len(data) != 40 {
		return
	}
	t.Value = jutil.GetInt64(data[:8])
	t.LockHash = data[8:40]
}

func GetTxOutputUid(txHash []byte, index uint32) []byte {
	return db.GetTxHashIndexUid(txHash, index)
}

func GetTxOutputsByHashes(txHashes [][]byte) ([]*TxOutput, error) {
	var shardTxHashes = make(map[uint32][][]byte)
	for _, txHash := range txHashes {
		shard := uint32(db.GetShardByte(txHash))
		shardTxHashes[shard] = append(shardTxHashes[shard], jutil.ByteReverse(txHash))
	}
	var txOutputs []*TxOutput
	for shard, txHashes := range shardTxHashes {
		shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
		dbClient := client.NewClient(shardConfig.GetHost())
		err := dbClient.GetByPrefixes(db.TopicTxOutput, txHashes)
		if err != nil {
			return nil, jerr.Get("error getting db message tx outputs", err)
		}
		for _, msg := range dbClient.Messages {
			var txOutput = new(TxOutput)
			db.Set(txOutput, msg)
			txOutputs = append(txOutputs, txOutput)
		}
	}
	return txOutputs, nil
}

func GetTxOutputs(outs []memo.Out) ([]*TxOutput, error) {
	var shardOutGroups = make(map[uint32][]memo.Out)
	for _, out := range outs {
		shard := db.GetShardByte32(out.TxHash)
		shardOutGroups[shard] = append(shardOutGroups[shard], out)
	}
	wait := db.NewWait(len(shardOutGroups))
	var txOutputs []*TxOutput
	for shardT, outGroupT := range shardOutGroups {
		go func(shard uint32, outGroup []memo.Out) {
			defer wait.Group.Done()
			shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
			dbClient := client.NewClient(shardConfig.GetHost())
			var uids = make([][]byte, len(outGroup))
			for i := range outGroup {
				uids[i] = GetTxOutputUid(outGroup[i].TxHash, outGroup[i].Index)
			}
			sort.Slice(uids, func(i, j int) bool {
				return jutil.ByteLT(uids[i], uids[j])
			})
			if err := dbClient.GetSpecific(db.TopicTxOutput, uids); err != nil {
				wait.AddError(jerr.Get("error getting db", err))
				return
			}
			wait.Lock.Lock()
			for i := range dbClient.Messages {
				var txOutput = new(TxOutput)
				db.Set(txOutput, dbClient.Messages[i])
				txOutputs = append(txOutputs, txOutput)
			}
			wait.Lock.Unlock()
		}(shardT, outGroupT)
	}
	wait.Group.Wait()
	if len(wait.Errs) > 0 {
		return nil, jerr.Get("error getting tx outputs", jerr.Combine(wait.Errs...))
	}
	return txOutputs, nil
}

func GetTxOutput(hash []byte, index uint32) (*TxOutput, error) {
	shard := db.GetShardByte32(hash)
	shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
	dbClient := client.NewClient(shardConfig.GetHost())
	uid := GetTxOutputUid(hash, index)
	if err := dbClient.GetSingle(db.TopicTxOutput, uid); err != nil {
		return nil, jerr.Get("error getting db", err)
	}
	if len(dbClient.Messages) != 1 {
		return nil, nil
	}
	var txOutput = new(TxOutput)
	db.Set(txOutput, dbClient.Messages[0])
	return txOutput, nil
}
