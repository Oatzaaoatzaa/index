package queue

import (
	"fmt"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/ref/config"
)

type Get struct {
	Shard uint32
	Items []Item
}

func (r *Get) GetByPrefixes(topic string, prefixes [][]byte) error {
	shardConfig := config.GetShardConfig(r.Shard, config.GetQueueShards())
	db := client.NewClient(fmt.Sprintf("127.0.0.1:%d", shardConfig.Port))
	if err := db.GetWOpts(client.Opts{
		Topic:    topic,
		Prefixes: prefixes,
	}); err != nil {
		return fmt.Errorf("error getting by prefixes using queue client; %w", err)
	}
	r.Items = make([]Item, len(db.Messages))
	for i := range db.Messages {
		r.Items[i] = Item{
			Topic: db.Messages[i].Topic,
			Uid:   db.Messages[i].Uid,
			Data:  db.Messages[i].Message,
		}
	}
	return nil
}

func NewGet(shard uint32) *Get {
	return &Get{
		Shard: shard,
	}
}
