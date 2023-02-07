package memo

import (
	"context"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/config"
	"time"
)

type AddrFollow struct {
	Addr       [25]byte
	Seen       time.Time
	TxHash     [32]byte
	Unfollow   bool
	FollowAddr [25]byte
}

func (f *AddrFollow) GetTopic() string {
	return db.TopicMemoAddrFollow
}

func (f *AddrFollow) GetShard() uint {
	return client.GetByteShard(f.Addr[:])
}

func (f *AddrFollow) GetUid() []byte {
	return jutil.CombineBytes(
		f.Addr[:],
		jutil.GetTimeByteBig(f.Seen),
		jutil.ByteReverse(f.TxHash[:]),
	)
}

func (f *AddrFollow) SetUid(uid []byte) {
	if len(uid) != memo.AddressLength+memo.Int8Size+memo.TxHashLength {
		return
	}
	copy(f.Addr[:], uid[:25])
	f.Seen = jutil.GetByteTimeBig(uid[25:33])
	copy(f.TxHash[:], jutil.ByteReverse(uid[33:65]))
}

func (f *AddrFollow) Serialize() []byte {
	var unfollow byte
	if f.Unfollow {
		unfollow = 1
	}
	return jutil.CombineBytes(
		[]byte{unfollow},
		f.FollowAddr[:],
	)
}

func (f *AddrFollow) Deserialize(data []byte) {
	if len(data) < 1+memo.AddressLength {
		return
	}
	f.Unfollow = data[0] == 1
	copy(f.FollowAddr[:], data[1:26])
}

func GetAddrFollows(ctx context.Context, addrs [][25]byte) ([]*AddrFollow, error) {
	var shardPrefixes = make(map[uint32][][]byte)
	for _, addr := range addrs {
		shard := client.GetByteShard32(addr[:])
		shardPrefixes[shard] = append(shardPrefixes[shard], addr[:])
	}
	shardConfigs := config.GetQueueShards()
	var addrFollows []*AddrFollow
	for shard, prefixes := range shardPrefixes {
		shardConfig := config.GetShardConfig(shard, shardConfigs)
		dbClient := client.NewClient(shardConfig.GetHost())
		if err := dbClient.GetWOpts(client.Opts{
			Topic:    db.TopicMemoAddrFollow,
			Prefixes: prefixes,
			Max:      client.ExLargeLimit,
			Context:  ctx,
		}); err != nil {
			return nil, jerr.Get("error getting db addr memo follow by prefix", err)
		}
		for _, msg := range dbClient.Messages {
			var addrFollow = new(AddrFollow)
			db.Set(addrFollow, msg)
			addrFollows = append(addrFollows, addrFollow)
		}
	}
	return addrFollows, nil
}

func GetAddrFollowsSingle(ctx context.Context, addr [25]byte, start time.Time) ([]*AddrFollow, error) {
	shardConfig := config.GetShardConfig(client.GetByteShard32(addr[:]), config.GetQueueShards())
	dbClient := client.NewClient(shardConfig.GetHost())
	var startByte []byte
	if !jutil.IsTimeZero(start) {
		startByte = jutil.CombineBytes(addr[:], jutil.GetTimeByteBig(start))
	} else {
		startByte = addr[:]
	}
	if err := dbClient.GetWOpts(client.Opts{
		Topic:    db.TopicMemoAddrFollow,
		Prefixes: [][]byte{addr[:]},
		Start:    startByte,
		Max:      client.ExLargeLimit,
		Context:  ctx,
	}); err != nil {
		return nil, jerr.Get("error getting db addr memo follow by prefix", err)
	}
	var addrFollows = make([]*AddrFollow, len(dbClient.Messages))
	for i := range dbClient.Messages {
		addrFollows[i] = new(AddrFollow)
		db.Set(addrFollows[i], dbClient.Messages[i])
	}
	return addrFollows, nil
}

func ListenAddrFollows(ctx context.Context, addrs [][25]byte) (chan *AddrFollow, error) {
	if len(addrs) == 0 {
		return nil, nil
	}
	var shardPrefixes = make(map[uint32][][]byte)
	for _, addr := range addrs {
		shard := client.GetByteShard32(addr[:])
		shardPrefixes[shard] = append(shardPrefixes[shard], addr[:])
	}
	shardConfigs := config.GetQueueShards()
	var addrFollowChan = make(chan *AddrFollow)
	cancelCtx := db.NewCancelContext(ctx, func() {
		close(addrFollowChan)
	})
	for shard, prefixes := range shardPrefixes {
		shardConfig := config.GetShardConfig(shard, shardConfigs)
		dbClient := client.NewClient(shardConfig.GetHost())
		chanMessage, err := dbClient.Listen(cancelCtx.Context, db.TopicMemoAddrFollow, prefixes)
		if err != nil {
			return nil, jerr.Get("error listening to db addr memo follows by prefix", err)
		}
		go func() {
			for msg := range chanMessage {
				var addrFollow = new(AddrFollow)
				db.Set(addrFollow, *msg)
				addrFollowChan <- addrFollow
			}
			cancelCtx.Cancel()
		}()
	}
	return addrFollowChan, nil
}
