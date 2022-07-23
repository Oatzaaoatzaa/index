package resolver

import (
	"context"
	"encoding/hex"
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/admin/graph/dataloader"
	"github.com/memocash/index/admin/graph/model"
	"github.com/memocash/index/ref/bitcoin/tx/script"
)

func LockLoader(ctx context.Context, lockHash string) (*model.Lock, error) {
	address, err := dataloader.NewLockAddressLoader(lockAddressLoaderConfig).Load(lockHash)
	if err != nil {
		return nil, jerr.Get("error getting address from lock dataloader for profile resolver", err)
	}
	var lock = &model.Lock{
		Hash:    hex.EncodeToString(script.GetLockHashForAddress(*address)),
		Address: address.GetEncoded(),
	}
	if jutil.StringInSlice("balance", GetPreloads(ctx)) {
		balance, err := dataloader.NewAddressBalanceLoader(addressBalanceLoaderConfig).Load(address.GetEncoded())
		if err != nil {
			return nil, jerr.Get("error getting address from lock dataloader for profile resolver", err)
		}
		lock.Balance = balance
	}
	return lock, nil
}
