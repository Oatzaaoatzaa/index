package serve

import (
	"github.com/jchavannes/jgo/jutil"
	db "github.com/memocash/index/db/server"
	"github.com/memocash/index/ref/config"
	"github.com/spf13/cobra"
	"log"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "db [shard]",
	Run: func(c *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalf("fatal error must specify a shard")
		}
		shard := jutil.GetIntFromString(args[0])
		shards := config.GetQueueShards()
		if len(shards) < shard {
			log.Fatalf("fatal error shard specified greater than num shards: %d %d", shard, len(shards))
		}
		go config.SetProfileSignalListener()
		server := db.NewServer(shards[shard].Port, uint(shard))
		log.Printf("Starting queue db server shard %d on port %d...\n", server.Shard, server.Port)
		log.Fatalf("fatal error running queue db server; %v", server.Run())
	},
}
