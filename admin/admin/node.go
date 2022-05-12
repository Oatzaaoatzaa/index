package admin

import (
	"github.com/memocash/index/db/item"
	"time"
)

type NodeDisconnectRequest struct {
	NodeId string
}

type NodeConnectRequest struct {
	Ip   []byte
	Port uint16
}

type NodeHistoryRequest struct {
	SuccessOnly bool
	Ip          string
	Port        uint32 `json:",string"`
}

type Connection struct {
	Ip     string
	Port   uint16
	Time   time.Time
	Status item.PeerConnectionStatus
}

type NodeHistoryResponse struct {
	Connections []Connection
}

type NodeFoundPeersRequest struct {
	Ip   []byte
	Port uint16
}

type NodeFoundPeersResponse struct {
	FoundPeers []*item.FoundPeer
}

type NodePeersRequest struct {
	Page   uint
	Filter string
}

type NodePeersResponse struct {
	Peers []*Peer
}

type Peer struct {
	Ip       string
	Port     uint16
	Services uint64
	Time     time.Time
	Status   item.PeerConnectionStatus
}

type NodePeerReportResponse struct {
	TotalPeers     uint64
	PeersAttempted uint64
	TotalAttempts  uint64
	PeersConnected uint64
	PeersFailed    uint64
}

type Topic struct {
	Name string
}

type TopicListResponse struct {
	Topics []Topic
}

type TopicViewRequest struct {
	Topic string
}

type TopicItem struct {
	Topic   string
	Uid     string
	Content string
	Shard   uint
}

type TopicViewResponse struct {
	Name  string
	Items []TopicItem
}
