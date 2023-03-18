package model

type Tx struct {
	Hash     Hash        `json:"hash"`
	Raw      Bytes       `json:"raw"`
	Seen     Date        `json:"seen"`
	Version  int32       `json:"version"`
	LockTime uint32      `json:"locktime"`
	Inputs   []*TxInput  `json:"inputs"`
	Outputs  []*TxOutput `json:"outputs"`
	Blocks   []*TxBlock  `json:"blocks"`
}

type TxOutput struct {
	Hash   Hash       `json:"hash"`
	Index  uint32     `json:"index"`
	Amount int64      `json:"amount"`
	Script Bytes      `json:"script"`
	Spends []*TxInput `json:"spends"`
	Slp    *SlpOutput `json:"slp"`
}

type TxInput struct {
	Hash      Hash   `json:"hash"`
	Index     uint32 `json:"index"`
	PrevHash  Hash   `json:"prev_hash"`
	PrevIndex uint32 `json:"prev_index"`
	Script    Bytes  `json:"script"`
	Sequence  uint32 `json:"sequence"`
	Tx        *Tx    `json:"tx"`
}

type Lock struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
}

type TxBlock struct {
	TxHash    Hash   `json:"tx_hash"`
	BlockHash Hash   `json:"block_hash"`
	Tx        *Tx    `json:"tx"`
	Block     *Block `json:"block"`
	Index     uint32 `json:"index"`
}

type Block struct {
	Hash      Hash  `json:"hash"`
	Raw       Bytes `json:"raw"`
	Timestamp Date  `json:"timestamp"`
	Height    *int  `json:"height"`
	Size      int64 `json:"size"`
	TxCount   int   `json:"tx_count"`
}

type Profile struct {
	Address string      `json:"address"`
	Name    *SetName    `json:"name"`
	Profile *SetProfile `json:"profile"`
	Pic     *SetPic     `json:"pic"`
}

type Follow struct {
	TxHash        string `json:"tx_hash"`
	Address       string `json:"address"`
	FollowAddress string `json:"follow_address"`
	Unfollow      bool   `json:"unfollow"`
}

type SetName struct {
	TxHash  string `json:"tx_hash"`
	Address string `json:"address"`
	Name    string `json:"name"`
}

type SetPic struct {
	TxHash  string `json:"tx_hash"`
	Address string `json:"address"`
	Pic     string `json:"pic"`
}

type SetProfile struct {
	TxHash  string `json:"tx_hash"`
	Address string `json:"address"`
	Text    string `json:"text"`
}

type Post struct {
	TxHash  string `json:"tx_hash"`
	Address string `json:"address"`
	Text    string `json:"text"`
}

type Like struct {
	TxHash     string `json:"tx_hash"`
	Address    string `json:"address"`
	PostTxHash string `json:"post_tx_hash"`
	Tip        int64  `json:"tip"`
}

type Room struct {
	Name string `json:"name"`
}

type RoomFollow struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Unfollow bool   `json:"unfollow"`
	TxHash   string `json:"tx_hash"`
}

type SlpBaton struct {
	Hash      string `json:"hash"`
	Index     uint32 `json:"index"`
	TokenHash string `json:"token_hash"`
}

type SlpGenesis struct {
	Hash       string `json:"hash"`
	TokenType  Uint8  `json:"token_type"`
	Decimals   Uint8  `json:"decimals"`
	BatonIndex uint32 `json:"baton_index"`
	Ticker     string `json:"ticker"`
	Name       string `json:"name"`
	DocURL     string `json:"doc_url"`
	DocHash    string `json:"doc_hash"`
}

type SlpOutput struct {
	Hash      Hash   `json:"hash"`
	Index     uint32 `json:"index"`
	TokenHash Hash   `json:"token_hash"`
	Amount    uint64 `json:"amount"`
}
