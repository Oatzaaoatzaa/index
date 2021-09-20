package server

import (
	"context"
	"github.com/jchavannes/jgo/jerr"
	"github.com/memocash/server/db/client"
	"github.com/memocash/server/db/proto/queue_pb"
	"github.com/memocash/server/db/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"time"
)

type Server struct {
	Port        uint
	Shard       uint
	Stopped     bool
	MsgDoneChan chan *MsgDone
	Timeout     time.Duration
	Grpc        *grpc.Server
	queue_pb.UnimplementedQueueServer
}

func (s *Server) SaveMessages(_ context.Context, messages *queue_pb.Messages) (*queue_pb.ErrorReply, error) {
	var msgs []*Msg
	for _, message := range messages.Messages {
		msgs = append(msgs, &Msg{
			Uid:     message.Uid,
			Topic:   message.Topic,
			Message: message.Message,
		})
	}
	err := s.queueSaveMessage(msgs)
	var errMsg string
	if err != nil {
		errMsg = jerr.Get("error queueing message", err).Error()
	}
	return &queue_pb.ErrorReply{
		Error: errMsg,
	}, nil
}

func (s *Server) StartMessageChan() {
	s.MsgDoneChan = make(chan *MsgDone)
	for {
		msgDone := <-s.MsgDoneChan
		msgDone.Done <- s.execSaveMessage(msgDone)
	}
}

func (s *Server) execSaveMessage(msgDone *MsgDone) error {
	err := s.SaveMsgs(msgDone.Msgs)
	if err != nil {
		return jerr.Get("error setting message", err)
	}
	return nil
}

func (s *Server) SaveMsgs(msgs []*Msg) error {
	var topicMessagesToSave = make(map[string][]*store.Message)
	for _, msg := range msgs {
		topicMessagesToSave[msg.Topic] = append(topicMessagesToSave[msg.Topic], &store.Message{
			Uid:     msg.Uid,
			Message: msg.Message,
		})
	}
	for topic, messagesToSave := range topicMessagesToSave {
		err := store.SaveMessages(topic, s.Shard, messagesToSave)
		if err != nil {
			return jerr.Getf(err, "error saving messages for topic: %s", topic)
		}
		for _, message := range messagesToSave {
			ReceiveNew(topic, message.Uid)
		}
	}
	return nil
}

func (s *Server) queueSaveMessage(msgs []*Msg) error {
	var timeout = s.Timeout
	if timeout == 0 {
		timeout = client.DefaultGetTimeout
	}
	msgDone := NewMsgDone(msgs)
	select {
	case s.MsgDoneChan <- msgDone:
		err := <-msgDone.Done
		if err != nil {
			return jerr.Get("error queueing message", err)
		}
		return nil
	case <-time.NewTimer(timeout).C:
		return jerr.Newf("error queue message timeout (%s)", timeout)
	}
}

func (s *Server) GetMessage(_ context.Context, request *queue_pb.RequestSingle) (*queue_pb.Message, error) {
	message, err := store.GetMessage(request.Topic, s.Shard, request.Uid)
	if err != nil && !store.IsNotFoundError(err) {
		return nil, jerr.Getf(err, "error getting message for topic: %s, uid: %x", request.Topic, request.Uid)
	}
	if message == nil {
		return &queue_pb.Message{}, nil
	}
	return &queue_pb.Message{
		Topic:   request.Topic,
		Uid:     message.Uid,
		Message: message.Message,
	}, nil
}

func (s *Server) Run() error {
	lis, err := net.Listen("tcp", GetHost(s.Port))
	if err != nil {
		return jerr.Get("failed to listen", err)
	}
	go s.StartMessageChan()
	s.Grpc = grpc.NewServer(grpc.MaxRecvMsgSize(32*10e6), grpc.MaxSendMsgSize(32*10e6))
	queue_pb.RegisterQueueServer(s.Grpc, s)
	reflection.Register(s.Grpc)
	if err = s.Grpc.Serve(lis); err != nil {
		return jerr.Get("failed to serve", err)
	}
	return jerr.New("queue server disconnected")
}

func (s *Server) Stop() {
	if s.Grpc != nil {
		s.Grpc.Stop()
		s.Stopped = true
	}
}

func NewServer(port uint, shard uint) *Server {
	return &Server{
		Port:  port,
		Shard: shard,
	}
}
