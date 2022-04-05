package nodecommon

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/obscuronet/obscuro-playground/go/obscurocommon"
)

// Enclave is the interface for interacting with the node's enclave
type Enclave interface {
	// IsReady checks whether the enclave is ready to process requests
	IsReady() error

	// Attestation - Produces an attestation report which will be used to request the shared secret from another enclave.
	Attestation() obscurocommon.AttestationReport

	// GenerateSecret - the genesis enclave is responsible with generating the secret entropy
	GenerateSecret() obscurocommon.EncryptedSharedEnclaveSecret

	// FetchSecret - return the shared secret encrypted with the key from the attestation
	FetchSecret(report obscurocommon.AttestationReport) obscurocommon.EncryptedSharedEnclaveSecret

	// InitEnclave - initialise an enclave with a seed received by another enclave
	InitEnclave(secret obscurocommon.EncryptedSharedEnclaveSecret)

	// IsInitialised - true if the shared secret is available
	IsInitialised() bool

	// ProduceGenesis - the genesis enclave produces the genesis rollup
	ProduceGenesis(blkHash common.Hash) BlockSubmissionResponse

	// IngestBlocks - feed L1 blocks into the enclave to catch up
	IngestBlocks(blocks []*types.Block) []BlockSubmissionResponse

	// Start - start speculative execution
	Start(block types.Block)

	// SubmitBlock - When a new POBI round starts, the host submits a block to the enclave, which responds with a rollup
	// it is the responsibility of the host to gossip the returned rollup
	// For good functioning the caller should always submit blocks ordered by height
	// submitting a block before receiving a parent of it, will result in it being ignored
	SubmitBlock(block types.Block) BlockSubmissionResponse

	// SubmitRollup - receive gossiped rollups
	SubmitRollup(rollup ExtRollup)

	// SubmitTx - user transactions
	SubmitTx(tx EncryptedTx) error

	// Balance - returns the balance of an address with a block delay
	Balance(address common.Address) uint64

	// RoundWinner - calculates and returns the winner for a round, and whether this node is the winner
	RoundWinner(parent obscurocommon.L2RootHash) (ExtRollup, bool)

	// Stop gracefully stops the enclave
	Stop()

	// GetTransaction returns a transaction given its signed hash, or nil if the transaction is unknown
	GetTransaction(txHash common.Hash) *L2Tx

	// StopClient stops the enclave client if one exists
	StopClient()
}

// BlockSubmissionResponse is the response sent from the enclave back to the node after ingesting a block
type BlockSubmissionResponse struct {
	L1Hash      obscurocommon.L1RootHash // The Header Hash of the ingested Block
	L1Height    uint64                   // The L1 Height of the ingested Block
	L1Parent    obscurocommon.L2RootHash // The L1 Parent of the ingested Block
	L2Hash      obscurocommon.L2RootHash // The Rollup Hash in the ingested Block
	L2Height    uint64                   // The Rollup Height in the ingested Block
	L2Parent    obscurocommon.L2RootHash // The Rollup Hash Parent inside the ingested Block
	Withdrawals []Withdrawal             // The Withdrawals available in Rollup of the ingested Block

	ProducedRollup    ExtRollup // The new Rollup when ingesting the block produces a new Rollup
	IngestedBlock     bool      // Whether the Block was ingested or discarded
	IngestedNewRollup bool      // Whether the Block had a new Rollup and the enclave has ingested it
}
