package node

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/jchavannes/bchutil"
	"github.com/jchavannes/btcd/chaincfg"
	"github.com/jchavannes/btcd/peer"
	"github.com/jchavannes/btcd/wire"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/db"
	log2 "log"
	"net"
	"time"
)

const (
	DefaultPort = 8333
)

func GetLocalhost() net.IP {
	var localhostByte, _ = hex.DecodeString("00000000000000000000ffff7f000001")
	return localhostByte
}

type Server struct {
	Peer *peer.Peer
	Ip   []byte
	Port uint16
}

func (s *Server) GetAddr() error {
	if s.Peer == nil {
		return fmt.Errorf("error peer not set")
	}
	s.Peer.QueueMessage(wire.NewMsgGetAddr(), nil)
	return nil
}

func (s *Server) SaveConnectionResult(success bool) error {
	var peerConnection = &item.PeerConnection{
		Ip:   s.Ip,
		Port: s.Port,
		Time: time.Now(),
	}
	if success {
		peerConnection.Status = item.PeerConnectionStatusSuccess
	} else {
		peerConnection.Status = item.PeerConnectionStatusFail
	}
	var objects = []db.Object{
		peerConnection,
	}
	if err := db.Save(objects); err != nil {
		return fmt.Errorf("error saving connection result object; %w", err)
	}
	return nil
}

func (s *Server) Run() error {
	params := &chaincfg.MainNetParams
	params.Net = bchutil.MainnetMagic
	var err error
	connectionAddress := fmt.Sprintf("[%s]:%d", net.IP(s.Ip), s.Port)
	log := func(msg string, params ...interface{}) {
		log2.Printf(connectionAddress+": "+msg, params...)
	}
	s.Peer, err = peer.NewOutboundPeer(&peer.Config{
		UserAgentName:    "memo-node",
		UserAgentVersion: "0.3.0",
		ChainParams:      params,
		Listeners: peer.MessageListeners{
			OnAddr: func(p *peer.Peer, msg *wire.MsgAddr) {
				log("on addr: %d\n", len(msg.AddrList))
				var objects = make([]db.Object, len(msg.AddrList)*3)
				for i := range msg.AddrList {
					objects[i*3] = &item.Peer{
						Ip:       msg.AddrList[i].IP,
						Port:     msg.AddrList[i].Port,
						Services: uint64(msg.AddrList[i].Services),
					}
					objects[i*3+1] = &item.PeerFound{
						Ip:         msg.AddrList[i].IP,
						Port:       msg.AddrList[i].Port,
						FinderIp:   s.Ip,
						FinderPort: s.Port,
					}
					objects[i*3+2] = &item.FoundPeer{
						Ip:        s.Ip,
						Port:      s.Port,
						FoundIp:   msg.AddrList[i].IP,
						FoundPort: msg.AddrList[i].Port,
					}
				}
				if err := db.Save(objects); err != nil {
					log2.Printf("error saving peers; %v", err)
				}
			},
			OnVerAck: func(p *peer.Peer, msg *wire.MsgVerAck) {
				log("ver ack from peer\n")
				if err := s.GetAddr(); err != nil {
					log2.Printf("error peer get addr; %v", err)
				}
			},
			OnHeaders: func(p *peer.Peer, msg *wire.MsgHeaders) {
				log("on headers from peer\n")
				for _, header := range msg.Headers {
					log("header: %s %s\n", header.BlockHash(), header.Timestamp.Format("2006-01-02 15:04:05"))
				}
			},
			/*OnInv: func(p *peer.Peer, msg *wire.MsgInv) {
				log.Println("on inv from peer", msg.InvList)
				for _, inv := range msg.InvList {
					log.Printf("inv type: %s, hash: %s\n", inv.Type, inv.Hash)
				}
			},*/
			OnBlock: func(p *peer.Peer, msg *wire.MsgBlock, buf []byte) {
				log("on block from peer\n")
				log("block: %s, txs: %d\n", msg.BlockHash(), len(msg.Transactions))
			},
			OnTx: func(p *peer.Peer, msg *wire.MsgTx) {
				log("on tx from peer\n")
				log("tx: %s\n", msg.TxHash())
			},
			OnReject: func(p *peer.Peer, msg *wire.MsgReject) {
				log("on reject from peer\n")
				log("msg: %v\n", msg)
			},
			OnPing: func(p *peer.Peer, msg *wire.MsgPing) {
				log("on ping from peer\n")
				log("Nonce: %d\n", msg.Nonce)
			},
			OnMerkleBlock: func(p *peer.Peer, msg *wire.MsgMerkleBlock) {
				log("on merkle from peer\n")
				log("merkle block: %s, txs: %d, hashes: %d\n",
					msg.Header.BlockHash(), msg.Transactions, len(msg.Hashes))
			},
			OnVersion: func(p *peer.Peer, msg *wire.MsgVersion) {
				log("on version from peer\n")
				log("version: %d, user agent: %s\n", msg.ProtocolVersion, msg.UserAgent)
			},
		},
	}, connectionAddress)
	if err != nil {
		return fmt.Errorf("error getting new outbound peer; %w", err)
	}
	log("Starting node\n")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", connectionAddress)
	if err != nil {
		if err2 := s.SaveConnectionResult(false); err2 != nil {
			return fmt.Errorf("error saving connection result on fail; %w", err2)
		}
		return fmt.Errorf("error getting network connection; %w", err)
	}
	if err := s.SaveConnectionResult(true); err != nil {
		return fmt.Errorf("error saving connection result on success; %w", err)
	}
	log("Associating connection\n")
	s.Peer.AssociateConnection(conn)
	log("wait for disconnect\n")
	disconnected := make(chan interface{})
	go func() {
		s.Peer.WaitForDisconnect()
		<-disconnected
	}()
	select {
	case <-time.NewTimer(1 * time.Minute).C:
		s.Disconnect()
		return nil
	case <-disconnected:
		return fmt.Errorf("error node disconnected")
	}
}

func (s *Server) Disconnect() {
	if s.Peer != nil {
		s.Peer.Disconnect()
	}
}

func NewServer(ip []byte, port uint16) *Server {
	return &Server{
		Ip:   ip,
		Port: port,
	}
}
