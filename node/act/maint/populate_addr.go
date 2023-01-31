package maint

import (
	"github.com/jchavannes/jgo/jerr"
	"github.com/jchavannes/jgo/jlog"
	"github.com/jchavannes/jgo/jutil"
	"github.com/memocash/index/db/client"
	"github.com/memocash/index/db/item"
	"github.com/memocash/index/db/item/addr"
	"github.com/memocash/index/db/item/chain"
	"github.com/memocash/index/db/item/db"
	"github.com/memocash/index/ref/bitcoin/wallet"
	"github.com/memocash/index/ref/config"
	"sync"
	"time"
)

type PopulateAddr struct {
	status   map[uint]*item.ProcessStatus
	hasError bool
	mu       sync.Mutex
	Checked  int64
	Saved    int64
}

func NewPopulateAddr() *PopulateAddr {
	return &PopulateAddr{
		status: make(map[uint]*item.ProcessStatus),
	}
}

func (p *PopulateAddr) SetShardStatus(shard uint32, status []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status[uint(shard)] = item.NewProcessStatus(uint(shard), item.ProcessStatusPopulateAddr)
	p.status[uint(shard)].Status = status
}

func (p *PopulateAddr) GetShardStatus(shard uint32) *item.ProcessStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	if shardStatus, ok := p.status[uint(shard)]; !ok {
		return nil
	} else {
		return shardStatus
	}
}

func (p *PopulateAddr) SetHasError(hasError bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hasError = hasError
}

func (p *PopulateAddr) GetHasError() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hasError
}

func (p *PopulateAddr) Populate(newRun bool) error {
	shardConfigs := config.GetQueueShards()
	if !newRun {
		for _, shardConfig := range shardConfigs {
			syncStatus, err := item.GetProcessStatus(uint(shardConfig.Shard), item.ProcessStatusPopulateAddr)
			if err != nil && !client.IsMessageNotSetError(err) {
				return jerr.Get("error getting sync status", err)
			} else if syncStatus != nil {
				p.SetShardStatus(shardConfig.Shard, syncStatus.Status)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(len(shardConfigs))
	var errChan = make(chan error)
	for _, shardConfig := range shardConfigs {
		go func(config config.Shard) {
			for {
				if done, err := p.populateShardSingle(config.Shard); done {
					jlog.Logf("Completed populating addr for shard: %d\n", config.Shard)
					wg.Done()
				} else if err != nil {
					errChan <- jerr.Getf(err, "error populating addr for shard: %d", config.Shard)
				} else {
					continue
				}
				return
			}
		}(shardConfig)
	}
	var success = make(chan bool)
	go func() {
		wg.Wait()
		success <- true
	}()
	for {
		select {
		case err := <-errChan:
			p.SetHasError(true)
			return jerr.Get("error populating addr direct", err)
		case <-success:
			return nil
		case <-time.NewTimer(time.Second * 10).C:
			p.mu.Lock()
			jlog.Logf("Populating addr direct: %d checked, %d saved\n", p.Checked, p.Saved)
			for shard, status := range p.status {
				jlog.Logf("Shard %d status: %x\n", shard, status.Status)
			}
			p.mu.Unlock()
		}
	}
}

func (p *PopulateAddr) populateShardSingle(shard uint32) (bool, error) {
	shardStatus := p.GetShardStatus(shard)
	if shardStatus == nil {
		shardStatus = item.NewProcessStatus(uint(shard), item.ProcessStatusPopulateAddr)
	}
	txOutputs, err := chain.GetAllTxOutputs(shard, shardStatus.Status)
	if err != nil {
		return false, jerr.Getf(err, "error getting tx outputs for populate addr shard: %d", shard)
	}
	seenTime := time.Now()
	var objMap = make(map[[57]byte]*addr.SeenTx)
	for _, txOutput := range txOutputs {
		uid := txOutput.GetUid()
		if jutil.ByteGT(uid, shardStatus.Status) {
			shardStatus.Status = uid
		}
		address, err := wallet.GetAddrFromLockScript(txOutput.LockScript)
		if err != nil {
			continue
		}
		objMap[getAddrTxHashId(*address, txOutput.TxHash)] = &addr.SeenTx{
			Addr:   *address,
			TxHash: txOutput.TxHash,
			Seen:   seenTime,
		}
	}
	var objectsToSave = make([]db.Object, 0, len(objMap))
	for _, obj := range objMap {
		objectsToSave = append(objectsToSave, obj)
	}
	if err := db.Save(objectsToSave); err != nil {
		return false, jerr.Get("error saving objects for addr populate single", err)
	}
	p.mu.Lock()
	p.Saved += int64(len(objectsToSave))
	p.Checked += int64(len(txOutputs))
	p.mu.Unlock()
	if err := shardStatus.Save(); err != nil {
		return false, jerr.Get("error saving process status", err)
	}
	p.SetShardStatus(shard, shardStatus.Status)
	return len(txOutputs) < client.HugeLimit, nil
}
