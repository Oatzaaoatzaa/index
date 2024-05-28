package resolver

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"time"

	"github.com/memocash/index/db/item/memo"
	"github.com/memocash/index/graph/generated"
	"github.com/memocash/index/graph/load"
	"github.com/memocash/index/graph/model"
)

// Lock is the resolver for the lock field.
func (r *profileResolver) Lock(ctx context.Context, obj *model.Profile) (*model.Lock, error) {
	lock, err := load.GetLock(ctx, obj.Address)
	if err != nil {
		return nil, fmt.Errorf("error getting addr from loader for profile resolver: %s; %w", obj.Address, err)
	}
	return lock, nil
}

// Following is the resolver for the following field.
func (r *profileResolver) Following(ctx context.Context, obj *model.Profile, start *model.Date) ([]*model.Follow, error) {
	var startTime time.Time
	if start != nil {
		startTime = time.Time(*start)
	}
	addrMemoFollows, err := memo.GetAddrFollowsSingle(ctx, obj.Address, startTime)
	if err != nil {
		return nil, fmt.Errorf("error getting address memo follows for address; %w", err)
	}
	var follows []*model.Follow
	for _, addrMemoFollow := range addrMemoFollows {
		follows = append(follows, &model.Follow{
			TxHash:        addrMemoFollow.TxHash,
			Address:       addrMemoFollow.Addr,
			FollowAddress: addrMemoFollow.FollowAddr,
			Unfollow:      addrMemoFollow.Unfollow,
		})
	}
	if err := load.AttachToMemoFollows(ctx, load.GetFields(ctx), follows); err != nil {
		return nil, fmt.Errorf("error attaching to memo following for profile resolver: %s; %w", obj.Address, err)
	}
	return follows, nil
}

// Followers is the resolver for the followers field.
func (r *profileResolver) Followers(ctx context.Context, obj *model.Profile, start *model.Date) ([]*model.Follow, error) {
	var startTime time.Time
	if start != nil {
		startTime = time.Time(*start)
	}
	addrMemoFolloweds, err := memo.GetAddrFollowedsSingle(ctx, obj.Address, startTime)
	if err != nil {
		return nil, fmt.Errorf("error getting addr memo follows for address: %s; %w", obj.Address, err)
	}
	var follows []*model.Follow
	for _, addrMemoFollowed := range addrMemoFolloweds {
		follows = append(follows, &model.Follow{
			TxHash:        addrMemoFollowed.TxHash,
			Address:       addrMemoFollowed.Addr,
			FollowAddress: addrMemoFollowed.FollowAddr,
			Unfollow:      addrMemoFollowed.Unfollow,
		})
	}
	if err := load.AttachToMemoFollows(ctx, load.GetFields(ctx), follows); err != nil {
		return nil, fmt.Errorf("error attaching to memo followers for profile resolver: %s; %w", obj.Address, err)
	}
	return follows, nil
}

// Posts is the resolver for the posts field.
func (r *profileResolver) Posts(ctx context.Context, obj *model.Profile, start *model.Date, newest *bool) ([]*model.Post, error) {
	var startTime time.Time
	if start != nil {
		startTime = time.Time(*start)
	}
	addrMemoPosts, err := memo.GetSingleAddrPosts(ctx, obj.Address, newest != nil && *newest, startTime)
	if err != nil {
		return nil, fmt.Errorf("error getting addr memo posts for profile resolver: %s; %w", obj.Address, err)
	}
	var posts []*model.Post
	for _, addrMemoPost := range addrMemoPosts {
		posts = append(posts, &model.Post{TxHash: addrMemoPost.TxHash})
	}
	if err := load.AttachToMemoPosts(ctx, load.GetFields(ctx), posts); err != nil {
		return nil, InternalError{fmt.Errorf("error attaching to posts for query resolver posts; %w", err)}
	}
	return posts, nil
}

// Rooms is the resolver for the rooms field.
func (r *profileResolver) Rooms(ctx context.Context, obj *model.Profile, start *model.Date) ([]*model.RoomFollow, error) {
	lockRoomFollows, err := memo.GetAddrRoomFollows(ctx, [][25]byte{obj.Address})
	if err != nil {
		return nil, fmt.Errorf("error getting addr room follows for profile resolver: %s; %w", obj.Address, err)
	}
	var roomFollows = make([]*model.RoomFollow, len(lockRoomFollows))
	for i := range lockRoomFollows {
		roomFollows[i] = &model.RoomFollow{
			Name:     lockRoomFollows[i].Room,
			Address:  lockRoomFollows[i].Addr,
			Unfollow: lockRoomFollows[i].Unfollow,
			TxHash:   lockRoomFollows[i].TxHash,
		}
	}
	return roomFollows, nil
}

// Tx is the resolver for the tx field.
func (r *setNameResolver) Tx(ctx context.Context, obj *model.SetName) (*model.Tx, error) {
	tx, err := load.GetTx(ctx, obj.TxHash)
	if err != nil {
		return nil, fmt.Errorf("error getting tx from loader for set name resolver: %s; %w", obj.TxHash, err)
	}
	return tx, nil
}

// Lock is the resolver for the lock field.
func (r *setNameResolver) Lock(ctx context.Context, obj *model.SetName) (*model.Lock, error) {
	lock, err := load.GetLock(ctx, obj.Address)
	if err != nil {
		return nil, fmt.Errorf("error getting lock from loader for set name resolver: %s %x; %w", obj.TxHash, obj.Address, err)
	}
	return lock, nil
}

// Tx is the resolver for the tx field.
func (r *setPicResolver) Tx(ctx context.Context, obj *model.SetPic) (*model.Tx, error) {
	tx, err := load.GetTx(ctx, obj.TxHash)
	if err != nil {
		return nil, fmt.Errorf("error getting tx from loader for set pic resolver: %s; %w", obj.TxHash, err)
	}
	return tx, nil
}

// Lock is the resolver for the lock field.
func (r *setPicResolver) Lock(ctx context.Context, obj *model.SetPic) (*model.Lock, error) {
	lock, err := load.GetLock(ctx, obj.Address)
	if err != nil {
		return nil, fmt.Errorf("error getting lock from loader for set pic resolver: %s %x; %w", obj.TxHash, obj.Address, err)
	}
	return lock, nil
}

// Tx is the resolver for the tx field.
func (r *setProfileResolver) Tx(ctx context.Context, obj *model.SetProfile) (*model.Tx, error) {
	tx, err := load.GetTx(ctx, obj.TxHash)
	if err != nil {
		return nil, fmt.Errorf("error getting tx from loader for set profile resolver: %s; %w", obj.TxHash, err)
	}
	return tx, nil
}

// Lock is the resolver for the lock field.
func (r *setProfileResolver) Lock(ctx context.Context, obj *model.SetProfile) (*model.Lock, error) {
	lock, err := load.GetLock(ctx, obj.Address)
	if err != nil {
		return nil, fmt.Errorf("error getting lock from loader for set profile resolver: %s; %w", obj.TxHash, err)
	}
	return lock, nil
}

// Profile returns generated.ProfileResolver implementation.
func (r *Resolver) Profile() generated.ProfileResolver { return &profileResolver{r} }

// SetName returns generated.SetNameResolver implementation.
func (r *Resolver) SetName() generated.SetNameResolver { return &setNameResolver{r} }

// SetPic returns generated.SetPicResolver implementation.
func (r *Resolver) SetPic() generated.SetPicResolver { return &setPicResolver{r} }

// SetProfile returns generated.SetProfileResolver implementation.
func (r *Resolver) SetProfile() generated.SetProfileResolver { return &setProfileResolver{r} }

type profileResolver struct{ *Resolver }
type setNameResolver struct{ *Resolver }
type setPicResolver struct{ *Resolver }
type setProfileResolver struct{ *Resolver }
