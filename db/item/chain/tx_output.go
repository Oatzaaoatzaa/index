package chain

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/config"
)

type TxOutput struct {
	TxHash     [32]byte
	Index      uint32
	Value      int64
	LockScript []byte
}

func (t *TxOutput) GetTopic() string {
	return db.TopicChainTxOutput
}

func (t *TxOutput) GetShard() uint {
	return client.GetByteShard(t.TxHash[:])
}

func (t *TxOutput) GetUid() []byte {
	return db.GetTxHashIndexUid(t.TxHash[:], t.Index)
}

func (t *TxOutput) SetUid(uid []byte) {
	if len(uid) != 36 {
		return
	}
	copy(t.TxHash[:], jutil.ByteReverse(uid[:32]))
	t.Index = jutil.GetUint32Big(uid[32:36])
}

func (t *TxOutput) Serialize() []byte {
	return jutil.CombineBytes(
		jutil.GetInt64Data(t.Value),
		t.LockScript,
	)
}

func (t *TxOutput) Deserialize(data []byte) {
	if len(data) < 8 {
		return
	}
	t.Value = jutil.GetInt64(data[:8])
	t.LockScript = data[8:]
}

func GetAllTxOutputs(shard uint32, startUid []byte) ([]*TxOutput, error) {
	shardConfig := config.GetShardConfig(shard, config.GetQueueShards())
	dbClient := client.NewClient(shardConfig.GetHost())
	if err := dbClient.GetWOpts(client.Opts{
		Topic: db.TopicChainTxOutput,
		Start: startUid,
		Max:   client.HugeLimit,
	}); err != nil {
		return nil, jerr.Get("error getting db message chain tx outputs for all", err)
	}
	var txOutputs = make([]*TxOutput, len(dbClient.Messages))
	for i := range dbClient.Messages {
		txOutputs[i] = new(TxOutput)
		db.Set(txOutputs[i], dbClient.Messages[i])
	}
	return txOutputs, nil
}

func GetTxOutputsByHashes(txHashes [][32]byte) ([]*TxOutput, error) {
	var shardPrefixes = make(map[uint32][][]byte)
	for i := range txHashes {
		shard := uint32(db.GetShardByte(txHashes[i][:]))
		shardPrefixes[shard] = append(shardPrefixes[shard], jutil.ByteReverse(txHashes[i][:]))
	}
	messages, err := db.GetByPrefixes(db.TopicChainTxOutput, shardPrefixes)
	if err != nil {
		return nil, jerr.Get("error getting client message chain tx output", err)
	}
	var txOutputs []*TxOutput
	for _, msg := range messages {
		var txOutput = new(TxOutput)
		db.Set(txOutput, msg)
		txOutputs = append(txOutputs, txOutput)
	}
	return txOutputs, nil
}

func GetTxOutput(out memo.Out) (*TxOutput, error) {
	txOutputs, err := GetTxOutputs([]memo.Out{out})
	if err != nil {
		return nil, jerr.Get("error getting tx outputs for single", err)
	}
	if len(txOutputs) == 0 {
		return nil, nil
	}
	return txOutputs[0], nil
}

func GetTxOutputs(outs []memo.Out) ([]*TxOutput, error) {
	var shardUids = make(map[uint32][][]byte)
	for _, out := range outs {
		shard := db.GetShardByte32(out.TxHash)
		shardUids[shard] = append(shardUids[shard], db.GetTxHashIndexUid(out.TxHash, out.Index))
	}
	messages, err := db.GetSpecific(db.TopicChainTxOutput, shardUids)
	if err != nil {
		return nil, jerr.Get("error getting client message chain tx output", err)
	}
	var txOutputs []*TxOutput
	for i := range messages {
		var txOutput = new(TxOutput)
		db.Set(txOutput, messages[i])
		txOutputs = append(txOutputs, txOutput)
	}
	return txOutputs, nil
}
