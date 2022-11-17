package db

import (
	"bytes"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/obscuronet/go-obscuro/go/common"
	"github.com/obscuronet/go-obscuro/go/common/log"
)

// TODO - #718 - Remove any rollup-based getters that are superseded by the batch-based getters.

// DB methods relating to rollups.

// GetHeadRollupHeader returns the header of the node's current head rollup, or (nil, false) if no such header is found.
func (db *DB) GetHeadRollupHeader() (*common.Header, bool) {
	headRollupHash, found := db.readHeadRollupHash()
	if !found {
		return nil, false
	}
	return db.readRollupHeader(*headRollupHash)
}

// GetRollupHeader returns the rollup header given the Hash, or (nil, false) if no such header is found.
func (db *DB) GetRollupHeader(hash gethcommon.Hash) (*common.Header, bool) {
	return db.readRollupHeader(hash)
}

// AddRollupHeader adds a rollup's header to the known headers
func (db *DB) AddRollupHeader(header *common.Header, txHashes []common.TxHash) {
	b := db.kvStore.NewBatch()
	db.writeRollupHeader(b, header)
	db.writeRollupTxHashes(b, header.Hash(), txHashes) // Required by ObscuroScan, to display a list of recent transactions.
	db.writeRollupHash(b, header)
	for _, txHash := range txHashes {
		db.writeRollupNumber(b, header, txHash)
	}

	// There's a potential race here, but absolute accuracy of the number of transactions is not required.
	currentTotal := db.readTotalTransactions()
	newTotal := big.NewInt(0).Add(currentTotal, big.NewInt(int64(len(txHashes))))
	db.writeTotalTransactions(b, newTotal)

	// update the head if the new height is greater than the existing one
	headRollupHeader, found := db.GetHeadRollupHeader()
	if !found || headRollupHeader.Number.Int64() <= header.Number.Int64() {
		db.writeHeadRollupHash(b, header.Hash())
	}

	if err := b.Write(); err != nil {
		db.logger.Crit("Could not write rollup.", log.ErrKey, err)
	}
}

// GetRollupHash returns the hash of a rollup given its number, or (nil, false) if no such rollup is found.
func (db *DB) GetRollupHash(number *big.Int) (*gethcommon.Hash, bool) {
	return db.readRollupHash(number)
}

// GetRollupNumber returns the number of the rollup containing the given transaction hash, or (nil, false) if no such rollup is found.
func (db *DB) GetRollupNumber(txHash gethcommon.Hash) (*big.Int, bool) {
	return db.readRollupNumber(txHash)
}

// GetRollupTxs returns the transaction hashes of the rollup with the given hash, or (nil, false) if no such rollup is found.
func (db *DB) GetRollupTxs(rollupHash gethcommon.Hash) ([]gethcommon.Hash, bool) {
	return db.readRollupTxHashes(rollupHash)
}

// GetTotalTransactions returns the total number of rolled-up transactions.
// TODO - #718 - Return number of batched transactions, instead.
func (db *DB) GetTotalTransactions() *big.Int {
	return db.readTotalTransactions()
}

// headerKey = rollupHeaderPrefix  + hash
func rollupHeaderKey(hash gethcommon.Hash) []byte {
	return append(rollupHeaderPrefix, hash.Bytes()...)
}

// headerKey = rollupTxHashesPrefix + rollup hash
func rollupTxHashesKey(hash gethcommon.Hash) []byte {
	return append(rollupTxHashesPrefix, hash.Bytes()...)
}

// headerKey = rollupHashPrefix + number
func rollupHashKey(num *big.Int) []byte {
	return append(rollupHashPrefix, []byte(num.String())...)
}

// headerKey = rollupNumberPrefix + hash
func rollupNumberKey(txHash gethcommon.Hash) []byte {
	return append(rollupNumberPrefix, txHash.Bytes()...)
}

// Stores a rollup header into the database
func (db *DB) writeRollupHeader(w ethdb.KeyValueWriter, header *common.Header) {
	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		db.logger.Crit("could not encode rollup header.", log.ErrKey, err)
	}
	key := rollupHeaderKey(header.Hash())
	if err := w.Put(key, data); err != nil {
		db.logger.Crit("could not put header in DB.", log.ErrKey, err)
	}
}

// Retrieves the rollup header corresponding to the hash, or (nil, false) if no such header is found.
func (db *DB) readRollupHeader(hash gethcommon.Hash) (*common.Header, bool) {
	f, err := db.kvStore.Has(rollupHeaderKey(hash))
	if err != nil {
		db.logger.Crit("could not retrieve rollup header.", log.ErrKey, err)
	}
	if !f {
		return nil, false
	}
	data, err := db.kvStore.Get(rollupHeaderKey(hash))
	if err != nil {
		db.logger.Crit("could not retrieve rollup header.", log.ErrKey, err)
	}
	if len(data) == 0 {
		return nil, false
	}
	header := new(common.Header)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		db.logger.Crit("could not decode rollup header.", log.ErrKey, err)
	}
	return header, true
}

// Writes the transaction hashes against the rollup containing them.
func (db *DB) writeRollupTxHashes(w ethdb.KeyValueWriter, rollupHash common.L2RootHash, txHashes []gethcommon.Hash) {
	data, err := rlp.EncodeToBytes(txHashes)
	if err != nil {
		db.logger.Crit("could not encode rollup transaction hashes.", log.ErrKey, err)
	}
	key := rollupTxHashesKey(rollupHash)
	if err := w.Put(key, data); err != nil {
		db.logger.Crit("could not put rollup transaction hashes in DB.", log.ErrKey, err)
	}
}

// Returns the transaction hashes in the rollup with the given hash, or (nil, false) if no such header is found.
func (db *DB) readRollupTxHashes(hash gethcommon.Hash) ([]gethcommon.Hash, bool) {
	f, err := db.kvStore.Has(rollupTxHashesKey(hash))
	if err != nil {
		db.logger.Crit("could not retrieve rollup tx hashes.", log.ErrKey, err)
	}
	if !f {
		return nil, false
	}
	data, err := db.kvStore.Get(rollupTxHashesKey(hash))
	if err != nil {
		db.logger.Crit("could not retrieve rollup tx hashes.", log.ErrKey, err)
	}
	if len(data) == 0 {
		return nil, false
	}
	txHashes := []gethcommon.Hash{}
	if err := rlp.Decode(bytes.NewReader(data), &txHashes); err != nil {
		db.logger.Crit("could not decode tx hashes.", log.ErrKey, err)
	}
	return txHashes, true
}

// Returns the head rollup's hash, or (nil, false) is no such hash is found.
func (db *DB) readHeadRollupHash() (*gethcommon.Hash, bool) {
	f, err := db.kvStore.Has(headRollup)
	if err != nil {
		db.logger.Crit("could not retrieve head rollup.", log.ErrKey, err)
	}
	if !f {
		return nil, false
	}
	value, err := db.kvStore.Get(headRollup)
	if err != nil {
		db.logger.Crit("could not retrieve head rollup.", log.ErrKey, err)
	}
	h := gethcommon.BytesToHash(value)
	return &h, true
}

// Stores the head rollup's hash in the database.
func (db *DB) writeHeadRollupHash(w ethdb.KeyValueWriter, val gethcommon.Hash) {
	err := w.Put(headRollup, val.Bytes())
	if err != nil {
		db.logger.Crit("could not write head rollup.", log.ErrKey, err)
	}
}

// Stores a rollup's hash in the database, keyed by the rollup's number.
func (db *DB) writeRollupHash(w ethdb.KeyValueWriter, header *common.Header) {
	key := rollupHashKey(header.Number)
	if err := w.Put(key, header.Hash().Bytes()); err != nil {
		db.logger.Crit("could not put header in DB.", log.ErrKey, err)
	}
}

// Stores a rollup's number in the database, keyed by the hash of a transaction in that rollup.
func (db *DB) writeRollupNumber(w ethdb.KeyValueWriter, header *common.Header, txHash gethcommon.Hash) {
	key := rollupNumberKey(txHash)
	// TODO - Investigate this off-by-one issue. The tx hashes that are in the `BlockSubmissionResponse` for rollup #1
	//  are actually the transactions for rollup #2.
	number := big.NewInt(0).Add(header.Number, big.NewInt(1))
	if err := w.Put(key, number.Bytes()); err != nil {
		db.logger.Crit("could not put rollup number in DB.", log.ErrKey, err)
	}
}

// Stores the total number of transactions in the database.
func (db *DB) writeTotalTransactions(w ethdb.KeyValueWriter, newTotal *big.Int) {
	err := w.Put(totalTransactionsKey, newTotal.Bytes())
	if err != nil {
		db.logger.Crit("Could not save total transactions.", log.ErrKey, err)
	}
}

// Retrieves the hash for the rollup with the given number, or (nil, false) if no such rollup is found.
func (db *DB) readRollupHash(number *big.Int) (*gethcommon.Hash, bool) {
	f, err := db.kvStore.Has(rollupHashKey(number))
	if err != nil {
		db.logger.Crit("could not retrieve rollup hash.", log.ErrKey, err)
	}
	if !f {
		return nil, false
	}
	data, err := db.kvStore.Get(rollupHashKey(number))
	if err != nil {
		db.logger.Crit("could not retrieve rollup hash.", log.ErrKey, err)
	}
	if len(data) == 0 {
		return nil, false
	}
	hash := gethcommon.BytesToHash(data)
	return &hash, true
}

// Retrieves the number of the rollup containing the transaction with the given hash, or (nil, false) if no such rollup is found.
func (db *DB) readRollupNumber(txHash gethcommon.Hash) (*big.Int, bool) {
	f, err := db.kvStore.Has(rollupNumberKey(txHash))
	if err != nil {
		db.logger.Crit("could not retrieve rollup number.", log.ErrKey, err)
	}
	if !f {
		return nil, false
	}
	data, err := db.kvStore.Get(rollupNumberKey(txHash))
	if err != nil {
		db.logger.Crit("could not retrieve rollup number.", log.ErrKey, err)
	}
	if len(data) == 0 {
		return nil, false
	}
	return big.NewInt(0).SetBytes(data), true
}

// Retrieves the total number of rolled-up transactions.
func (db *DB) readTotalTransactions() *big.Int {
	f, err := db.kvStore.Has(totalTransactionsKey)
	if err != nil {
		db.logger.Crit("could not retrieve total transactions.", log.ErrKey, err)
	}
	if !f {
		return big.NewInt(0)
	}
	data, err := db.kvStore.Get(totalTransactionsKey)
	if err != nil {
		db.logger.Crit("could not retrieve total transactions.", log.ErrKey, err)
	}
	if len(data) == 0 {
		return big.NewInt(0)
	}
	return big.NewInt(0).SetBytes(data)
}