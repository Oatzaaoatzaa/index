package attach

import (
	"context"
	"fmt"
	"github.com/memocash/index/db/item/memo"
	"github.com/memocash/index/graph/model"
)

type MemoRoom struct {
	base
	Rooms []*model.Room
}

func ToMemoRooms(ctx context.Context, fields []Field, rooms []*model.Room) error {
	if len(rooms) == 0 {
		return nil
	}
	o := MemoRoom{
		base:  base{Ctx: ctx, Fields: fields},
		Rooms: rooms,
	}
	o.Wait.Add(2)
	go o.AttachPosts()
	go o.AttachFollowers()
	o.Wait.Wait()
	if len(o.Errors) > 0 {
		return fmt.Errorf("error attaching to memo rooms; %w", o.Errors[0])
	}
	return nil
}

func (o *MemoRoom) GetRoomNames() []string {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()
	var roomNames = make([]string, len(o.Rooms))
	for i := range o.Rooms {
		roomNames[i] = o.Rooms[i].Name
	}
	return roomNames
}

func (o *MemoRoom) AttachPosts() {
	defer o.Wait.Done()
	if !o.HasField([]string{"posts"}) {
		return
	}
	// TODO: Implement "start" field support
	var allPosts []*model.Post
	for _, roomName := range o.GetRoomNames() {
		roomPosts, err := memo.GetRoomPosts(o.Ctx, roomName)
		if err != nil {
			o.AddError(fmt.Errorf("error getting room height posts for room resolver; %w", err))
			return
		}
		var posts = make([]*model.Post, len(roomPosts))
		for i := range roomPosts {
			posts[i] = &model.Post{TxHash: roomPosts[i].TxHash}
			allPosts = append(allPosts, posts[i])
		}
		o.Mutex.Lock()
		for i := range o.Rooms {
			if o.Rooms[i].Name != roomName {
				continue
			}
			o.Rooms[i].Posts = posts
			break
		}
		o.Mutex.Unlock()
	}
	/*if err := AttachToPosts(o.Ctx, GetPrefixFields(o.Fields, "posts."), allOutputs); err != nil {
		o.AddError(fmt.Errorf("error attaching to posts for memo rooms; %w", err))
		return
	}*/
}

func (o *MemoRoom) AttachFollowers() {
	defer o.Wait.Done()
	if !o.HasField([]string{"followers"}) {
		return
	}
	// TODO: Implement "start" field support
	var allRoomFollows []*model.RoomFollow
	for _, roomName := range o.GetRoomNames() {
		dbRoomFollows, err := memo.GetRoomFollows(o.Ctx, roomName)
		if err != nil {
			o.AddError(fmt.Errorf("error getting room follows for room resolver; %w", err))
			return
		}
		var modelRoomFollows = make([]*model.RoomFollow, len(dbRoomFollows))
		for i := range modelRoomFollows {
			modelRoomFollows[i] = &model.RoomFollow{
				Name:     roomName,
				Address:  dbRoomFollows[i].Addr,
				Unfollow: dbRoomFollows[i].Unfollow,
				TxHash:   dbRoomFollows[i].TxHash,
			}
			allRoomFollows = append(allRoomFollows, modelRoomFollows[i])
		}
		o.Mutex.Lock()
		for i := range o.Rooms {
			if o.Rooms[i].Name != roomName {
				continue
			}
			o.Rooms[i].Followers = modelRoomFollows
			break
		}
		o.Mutex.Unlock()
	}
	if err := ToMemoRoomFollows(o.Ctx, GetPrefixFields(o.Fields, "followers."), allRoomFollows); err != nil {
		o.AddError(fmt.Errorf("error attaching to followers for memo rooms; %w", err))
		return
	}
}
