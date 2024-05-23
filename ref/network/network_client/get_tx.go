package network_client

import (
	"context"
	"fmt"
	"github.com/jchavannes/btcd/wire"
	"github.com/memocash/index/ref/bitcoin/memo"
	"github.com/memocash/index/ref/network/gen/network_pb"
	"google.golang.org/grpc"
	"time"
)

type GetTx struct {
	Raw       []byte
	BlockHash []byte
	Msg       *wire.MsgTx
}

func (t *GetTx) Get(hash []byte) error {
	rpcConfig := GetConfig()
	if !rpcConfig.IsSet() {
		return fmt.Errorf("error config not set")
	}
	conn, err := grpc.Dial(rpcConfig.String(), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("error dial grpc did not connect network; %w", err)
	}
	defer conn.Close()
	c := network_pb.NewNetworkClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	txReply, err := c.GetTx(ctx, &network_pb.TxRequest{
		Hash: hash,
	})
	if err != nil {
		return fmt.Errorf("error getting rpc network tx by hash; %w", err)
	}
	tx := txReply.GetTx()
	t.Raw = tx.GetRaw()
	t.BlockHash = tx.GetBlock()
	t.Msg, err = memo.GetMsgFromRaw(t.Raw)
	if err != nil {
		return fmt.Errorf("error getting wire tx from raw; %w", err)
	}
	return nil
}

func NewGetTx() *GetTx {
	return &GetTx{
	}
}
