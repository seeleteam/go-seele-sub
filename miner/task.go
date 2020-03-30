/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/memory"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/txs"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/merkle"
)

// Task is a mining work for engine, containing block header, transactions, and transaction receipts.
type Task struct {
	header   *types.BlockHeader
	txs      []*types.Transaction
	receipts []*types.Receipt
	debts    []*types.Debt

	coinbase     common.Address
	debtVerifier types.DebtVerifier
	// verifierTxs  []*types.Transaction
	// exitTxs      []*types.Transaction
	accountCount  uint64
	challengedTxs []*types.Transaction
	depositVers   []common.Address
	exitVers      []common.Address
}

// NewTask return Task object
func NewTask(header *types.BlockHeader, coinbase common.Address, verifier types.DebtVerifier) *Task {
	return &Task{
		header:       header,
		coinbase:     coinbase,
		debtVerifier: verifier,
	}
}

// applyTransactionsAndDebts TODO need to check more about the transactions, such as gas limit
func (task *Task) applyTransactionsAndDebts(seele SeeleBackend, statedb *state.Statedb, accountStateDB database.Database, parent *types.Block, engine consensus.Engine, log *log.SeeleLog) error {
	now := time.Now()
	// entrance
	memory.Print(log, "task applyTransactionsAndDebts entrance", now, false)

	// choose transactions from the given txs
	var size int
	if task.header.Consensus != types.BftConsensus { // subchain doese not support debts.
		size = task.chooseDebts(seele, statedb, log)
	} else {
		size = core.BlockByteLimit
	}

	// the reward tx will always be at the first of the block's transactions
	reward, err := task.handleMinerRewardTx(statedb)
	if err != nil {
		return err
	}

	task.chooseTransactions(seele, statedb, log, size)

	log.Info("mining block height:%d, reward:%s, transaction number:%d, debt number: %d",
		task.header.Height, reward, len(task.txs), len(task.debts))

	batch := accountStateDB.NewBatch()
	root, err := statedb.Commit(batch)
	if err != nil {
		return err
	}

	task.header.StateHash = root

	if task.header.Consensus == types.BftConsensus {
		swExtra, err := types.ExtractSecondWitnessInfo(parent.Header)
		if err != nil {
			return err
		}
		task.accountCount = swExtra.AccountCount

		// log.Error("Account Count: %d", task.accountCount)

		// update txHash and stateHash
		txHashStem, stateHashStem, err := task.getStemHashes(seele, statedb, log)
		if err != nil {
			return err
		}

		// update recentTxHashStem
		recentTxHashStem, err := task.getRecentTxHashStem(seele, log)
		if err != nil {
			return err
		}

		// sign the block
		blockInfo := []interface{}{
			task.header.Creator,
			task.header.Height,
			txHashStem,
			stateHashStem,
		}

		blockInfoBytes, err := rlp.EncodeToBytes(blockInfo)
		if err != nil {
			return err
		}
		blockInfoHash := crypto.MustHash(blockInfoBytes)
		blockSig := *crypto.MustSign(engine.GetPrivateKey(), blockInfoHash.Bytes())

		// log.Error("fee account: %v", common.SubchainFeeAccount)
		// log.Error("blockSig: %v", blockSig.Sig)
		// update secondWitness
		log.Info("[%d]deposit verifiers, [%d]exit verifiers, [%d]challenge txs", len(task.depositVers), len(task.exitVers), len(task.challengedTxs))
		extraSecondWitnessInfo, err := types.PrepareSecondWitness(task.challengedTxs, task.depositVers, task.exitVers, task.accountCount, txHashStem, stateHashStem, recentTxHashStem, blockSig)
		if err != nil {
			log.Error("failed to prepare deposit or exit tx into secondwitness")
		}
		task.header.SecondWitness = extraSecondWitnessInfo
		log.Debug("apply new verifiers info witness, %+v", task.header.SecondWitness)
	}

	// exit
	memory.Print(log, "task applyTransactionsAndDebts exit", now, true)

	return nil
}

func (task *Task) getStemHashes(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog) (common.Hash, common.Hash, error) {
	// update hashes for stem
	var level []common.Hash
	for txIndex, tx := range task.txs {
		if txIndex == 0 {
			continue
		}
		// log.Error("handling tx")
		txPayload, err := types.ExtractTxPayload(tx.Data.Payload)
		if err != nil {
			return common.EmptyHash, common.EmptyHash, err
		}
		level = append(level, txPayload.HashForStem)
		// log.Error("hashForStem: %v, level: %v", txPayload.HashForStem, level)
		// log.Error("packHeighgt: %d", txPayload.LargestPackHeight)
		// get the current balance and nonce of tx.from or tx.to
		for i := 0; i < 2; i++ {
			var account common.Address
			if i == 0 {
				account = tx.Data.From
			} else {
				account = tx.Data.To
			}

			//  update accountIndexDB
			if isExisted, err := seele.GetAccountIndexDB().Has(account.Bytes()); err != nil {
				return common.EmptyHash, common.EmptyHash, err
			} else if !isExisted {
				task.accountCount++
				index := uint(task.accountCount - 1)
				indexBytes, _ := rlp.EncodeToBytes(index)
				// log.Error("accountCount: %d, account: %v, index: %d", task.accountCount, account.Hex(), index)
				if err = seele.GetAccountIndexDB().Put(account.Bytes(), indexBytes); err != nil {
					return common.EmptyHash, common.EmptyHash, err
				}

				if err = seele.GetIndexAccountDB().Put(indexBytes, account.Bytes()); err != nil {
					return common.EmptyHash, common.EmptyHash, err
				}
			}
		}
	}

	// update txHashStem
	txHashStem := merkle.GetBinaryMerkleRoot(level)
	// log.Error("txHashStem: %v", txHashStem)

	level = make([]common.Hash, 0)
	// update stateHashStem
	for i := uint64(0); i < task.accountCount; i++ {
		indexBytes, _ := rlp.EncodeToBytes(uint(i))
		accountBytes, err := seele.GetIndexAccountDB().Get(indexBytes)
		if err != nil {
			return common.EmptyHash, common.EmptyHash, err
		}
		account := common.BytesToAddress(accountBytes)
		nonce := statedb.GetNonce(account)
		balance := statedb.GetBalance(account)
		// log.Error("account: %v, index: %d, nonce: %d, balance: %d", account.Hex(), i, nonce, balance)
		state := []interface{}{
			account.Bytes(),
			balance,
			nonce,
		}

		stateBytes, _ := rlp.EncodeToBytes(state)
		level = append(level, crypto.MustHash(stateBytes))
	}
	stateHashStem := merkle.GetBinaryMerkleRoot(level)
	// log.Error("stateHashStem: %v", stateHashStem)
	return txHashStem, stateHashStem, nil
}

func (task *Task) getRecentTxHashStem(seele SeeleBackend, log *log.SeeleLog) (common.Hash, error) {
	var recentTxHashStem common.Hash
	if int(task.header.Height)%int(common.RelayInterval) > 0 {
		recentTxHashStem = common.EmptyHash
	} else {
		accountToIndexMap := make(map[common.Address]uint)
		var accTxs []*types.AccountTxs
		start := task.header.Height - common.RelayInterval + 1
		end := task.header.Height
		for i := start; i <= end; i++ {
			var transactions []*types.Transaction
			if i != end {
				prevBlock, err := seele.BlockChain().GetStore().GetBlockByHeight(uint64(i))
				if err != nil {
					return common.EmptyHash, err
				}
				transactions = prevBlock.Transactions
			} else {
				transactions = task.txs
			}
			// Obtain a SubTransaction tx
			for txIndex, prevTx := range transactions {
				if txIndex == 0 {
					continue
				}
				val := []interface{}{
					prevTx.Data.From,
					prevTx.Data.To,
					prevTx.Data.Amount,
					prevTx.Data.AccountNonce,
					prevTx.Data.GasPrice,
					prevTx.Data.GasLimit,
				}

				dataForStem, err := rlp.EncodeToBytes(val)
				if err != nil {
					return common.EmptyHash, err
				}
				var account common.Address
				for i := 0; i < 2; i++ {
					if i == 0 {
						account = prevTx.Data.From
					} else if prevTx.Data.From != prevTx.Data.To {
						account = prevTx.Data.To
					} else {
						break
					}
					if index, ok := accountToIndexMap[account]; ok {
						accTxs[index].Txs = append(accTxs[index].Txs, dataForStem)
					} else {
						// new account
						var curlen = len(accountToIndexMap)
						var newTxs [][]byte
						accountToIndexMap[account] = uint(curlen)
						newTxs = append(newTxs, dataForStem)
						newAccountTxs := &types.AccountTxs{
							Txs: newTxs,
						}
						accTxs = append(accTxs, newAccountTxs)
					}
				}
			}
		}
		// after traversing all the txs
		level := make([]common.Hash, 0)
		for _, value := range accountToIndexMap {
			txsBytes, err := rlp.EncodeToBytes(accTxs[value])
			if err != nil {
				return common.EmptyHash, err
			}
			level = append(level, crypto.MustHash(txsBytes))
		}
		recentTxHashStem = merkle.GetBinaryMerkleRoot(level)
	}
	// log.Error("recentTxHashStem: %v", recentTxHashStem)
	return recentTxHashStem, nil
}

func (task *Task) chooseDebts(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog) int {
	now := time.Now()
	// entrance
	memory.Print(log, "task chooseDebts entrance", now, false)

	size := core.BlockByteLimit

	for size > 0 {
		debts, _ := seele.DebtPool().GetProcessableDebts(size)
		if len(debts) == 0 {
			break
		}

		for _, d := range debts {
			err := seele.BlockChain().ApplyDebtWithoutVerify(statedb, d, task.coinbase)
			if err != nil {
				log.Warn("apply debt error %s", err)
				seele.DebtPool().RemoveDebtByHash(d.Hash)
				continue
			}

			size = size - d.Size()
			task.debts = append(task.debts, d)
		}
	}

	// exit
	memory.Print(log, "task chooseDebts exit", now, true)

	return size
}

// handleMinerRewardTx handles the miner reward transaction.
func (task *Task) handleMinerRewardTx(statedb *state.Statedb) (*big.Int, error) {
	reward := consensus.GetReward(task.header.Height)
	if task.header.Consensus == types.BftConsensus {
		reward = big.NewInt(int64(0))
	}
	rewardTx, err := txs.NewRewardTx(task.coinbase, reward, task.header.CreateTimestamp.Uint64())
	if err != nil {
		return nil, err
	}

	rewardTxReceipt, err := txs.ApplyRewardTx(rewardTx, statedb)
	if err != nil {
		return nil, err
	}

	task.txs = append(task.txs, rewardTx)

	// add the receipt of the reward tx
	task.receipts = append(task.receipts, rewardTxReceipt)

	return reward, nil
}

func (task *Task) chooseTransactions(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog, size int) {
	now := time.Now()
	// entrance
	memory.Print(log, "task chooseTransactions entrance", now, false)

	// TEST the event listner and fire function!
	// curHeight := task.header.Height
	// if curHeight%50 == 0 {
	// 	event.ChallengedTxEventManager.Fire(event.ChallengedTxEvent)
	// }

	//this code section for test the verifier is correctly added into secondwitness

	// task.depositVers = append(task.depositVers, common.BytesToAddress(hexutil.MustHexToBytes("0x1b9412d61a25f5f5decbf489fe5ed595d8b610a1")))
	// task.exitVers = append(task.exitVers, common.BytesToAddress(hexutil.MustHexToBytes("0x1b9412d61a25f5f5decbf489fe5ed595d8b610a1")))

	// if len(task.depositVers) > 0 || len(task.exitVers) > 0 {
	// 	log.Warn("deposit verifiers", task.depositVers)
	// 	log.Warn("exit verifiers", task.exitVers)
	// 	var err error
	// 	task.header.SecondWitness, err = task.prepareWitness(task.header, task.challengedTxs, task.depositVers, task.exitVers)
	// 	if err != nil {
	// 		log.Error("failed to prepare deposit or exit tx into secondwitness")
	// 	}
	// 	log.Debug("apply new verifiers into witness, %s", task.header.SecondWitness)

	// }
	// test code end here

	txIndex := 1 // the first tx is miner reward

	for size > 0 {
		txs, txsSize := seele.TxPool().GetProcessableTransactions(size)
		log.Info("tx size %d", len(txs))
		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			if err := tx.Validate(statedb, task.header.Height); err != nil {
				seele.TxPool().RemoveTransaction(tx.Hash)
				log.Error("failed to validate tx %s, for %s", tx.Hash.Hex(), err)
				txsSize = txsSize - tx.Size()
				continue
			}

			receipt, err := seele.BlockChain().ApplyTransaction(tx, txIndex, task.coinbase, statedb, task.header)
			if err != nil {
				seele.TxPool().RemoveTransaction(tx.Hash)
				log.Error("failed to apply tx %s, %s", tx.Hash.Hex(), err)
				txsSize = txsSize - tx.Size()
				continue
			}
			if task.header.Consensus == types.BftConsensus { // for bft, the secondwitness will be used as deposit&exit address holder.
				rootAccounts := seele.GenesisInfo().Rootaccounts
				fmt.Printf("rootAccounts %+v", rootAccounts)
				// if there is any successful challenge tx, need to revert blockchain first to specific point!
				if tx.IsChallengedTx(rootAccounts) {
					// will revert the block and db here, so the
					task.challengedTxs = append(task.challengedTxs, tx)
					// event.ChallengedTxEventManager.Fire(event.ChallengedTxEvent)
					return
				}

				if tx.IsVerifierTx(rootAccounts) {
					task.depositVers = append(task.depositVers, tx.ToAccount())
					// task.depositVers = append(task.depositVers, tx.FromAccount())
				}

				if tx.IsExitTx(rootAccounts) {
					task.exitVers = append(task.exitVers, tx.ToAccount())
				}
			}

			task.txs = append(task.txs, tx)
			task.receipts = append(task.receipts, receipt)
			txIndex++
		}
		size -= txsSize
	}
	// if task.header.Consensus == types.BftConsensus {
	// 	log.Info("[%d]deposit verifiers, [%d]exit verifiers, [%d]challenge txs", len(task.depositVers), len(task.exitVers), len(task.challengedTxs))
	// 	var err error
	// 	task.header.SecondWitness, err = task.prepareWitness(task.header, task.challengedTxs, task.depositVers, task.exitVers)
	// 	if err != nil {
	// 		log.Error("failed to prepare deposit or exit tx into secondwitness")
	// 	}
	// 	log.Debug("apply new verifiers into witness, %+v", task.header.SecondWitness)
	// }

	// exit
	memory.Print(log, "task chooseTransactions exit", now, true)
}

// generateBlock builds a block from task
func (task *Task) generateBlock() *types.Block {
	return types.NewBlock(task.header, task.txs, task.receipts, task.debts)
}

// Result is the result mined by engine. It contains the raw task and mined block.
type Result struct {
	task  *Task
	block *types.Block // mined block, with good nonce
}
