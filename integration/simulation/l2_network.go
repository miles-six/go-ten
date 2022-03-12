package simulation

import (
	"time"

	common3 "github.com/obscuronet/obscuro-playground/go/common"
	obscuro_node "github.com/obscuronet/obscuro-playground/go/obscuronode"
	"github.com/obscuronet/obscuro-playground/go/obscuronode/common"
)

// L2NetworkCfg - models a full network including artificial random latencies
type L2NetworkCfg struct {
	nodes            []*obscuro_node.Node
	avgLatency       uint64
	avgBlockDuration uint64
}

// NewL2Network returns an instance of a configured L2 Network (no nodes)
func NewL2Network(avgBlockDuration uint64, avgLatency uint64) *L2NetworkCfg {
	return &L2NetworkCfg{
		avgLatency:       avgLatency,
		avgBlockDuration: avgBlockDuration,
	}
}

// BroadcastRollup Broadcasts the rollup to all L2 peers
func (cfg *L2NetworkCfg) BroadcastRollup(r common3.EncodedRollup) {
	for _, a := range cfg.nodes {
		rol := common.DecodeRollup(r)
		if a.ID != rol.Header.Agg {
			t := a
			common3.Schedule(cfg.delay(), func() { t.P2PGossipRollup(r) })
		}
	}
}

func (cfg *L2NetworkCfg) BroadcastTx(tx common.EncryptedTx) {
	for _, a := range cfg.nodes {
		t := a
		common3.Schedule(cfg.delay()/2, func() { t.P2PReceiveTx(tx) })
	}
}

// Start kicks off the l2 nodes waiting a delay between each node
func (cfg *L2NetworkCfg) Start(delay time.Duration) {
	// Start l1 nodes
	for _, m := range cfg.nodes {
		t := m
		go t.Start()
		time.Sleep(delay)
	}
}

func (cfg *L2NetworkCfg) Stop() {
	for _, m := range cfg.nodes {
		m.Stop()
		// fmt.Printf("Stopped L2 node: %d\n", m.ID)
	}
}

// delay returns an expected delay on the l2
func (cfg *L2NetworkCfg) delay() uint64 {
	return common3.RndBtw(cfg.avgLatency/10, 2*cfg.avgLatency)
}