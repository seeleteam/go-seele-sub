/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import (
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/merkle"
)

// PublicSubchainAPI provides an API to access subchain node information.
type PublicSubchainAPI struct {
	s Backend
}

// NewPublicSubchainAPI creates a new PublicSubchainAPI object for rpc service.
func NewPublicSubchainAPI(s Backend) *PublicSubchainAPI {
	return &PublicSubchainAPI{s}
}

// get block creator
func (api *PublicSubchainAPI) GetBlockCreator(height int64) (string, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height) // return subblock
	if err != nil {
		return "", err
	}

	return block.Header.Creator.Hex(), nil
}

// get the root hash of the balance/state tree of a block
func (api *PublicSubchainAPI) GetBalanceTreeRoot(height int64) (string, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height) // return subblock
	if err != nil {
		return "", err
	}

	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return "", err
	}
	return swExtra.StateHashStem.Hex(), nil
}

// get the root hash of the tx tree of a block
func (api *PublicSubchainAPI) GetTxTreeRoot(height int64) (string, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height) // return subblock
	if err != nil {
		return "", err
	}

	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return "", err
	}
	return swExtra.TxHashStem.Hex(), nil
}

// get the block signature
func (api *PublicSubchainAPI) GetBlockSignature(height int64) (interface{}, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height) // return subblock
	if err != nil {
		return nil, err
	}

	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return "", err
	}
	return swExtra.BlockSig.Sig, nil
}

// get block information needed by stem contract
func (api *PublicSubchainAPI) GetBlockInfoForStem(height int64) (interface{}, error) {
	block, err := api.s.GetBlock(common.EmptyHash, height) // return subblock
	if err != nil {
		return nil, err
	}

	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return "", err
	}

	val := []interface{}{
		block.Header.Creator,
		uint64(height),
		swExtra.TxHashStem,
		swExtra.StateHashStem,
	}

	data, err := rlp.EncodeToBytes(val)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// get merkle index and merkle proof of a tx
func (api *PublicSubchainAPI) GetTxMerkleInfo(hashHex string) (map[string]interface{}, error) {

	hashByte, err := hexutil.HexToBytes(hashHex)
	if err != nil {
		return nil, err
	}
	txHash := common.BytesToHash(hashByte)

	// Try to find transaction in blockchain.
	bcStore := api.s.ChainBackend().GetStore()
	txIdx, err := bcStore.GetTxIndex(txHash)
	if err != nil {
		return nil, err
	}

	if txIdx == nil {
		return nil, nil
	}
	// fmt.Printf("txIdx blockHash: %v, index: %d", txIdx.BlockHash, txIdx.Index)
	block, err := bcStore.GetBlock(txIdx.BlockHash)
	if err != nil {
		return nil, err
	}

	var level []common.Hash
	var txPayload *types.PayloadExtra
	for i, tx := range block.Transactions {
		if i == 0 {
			continue
		}
		txPayload, err = types.ExtractTxPayload(tx.Data.Payload)
		if err != nil {
			return nil, err
		}
		// fmt.Printf("tx hashForStem: %v", txPayload.HashForStem)
		level = append(level, txPayload.HashForStem)
	}
	var proofs []common.Hash
	proofs = merkle.GetMerkleProof(level, int(txIdx.Index))

	proofData, err := rlp.EncodeToBytes(proofs)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"merkle index": txIdx.Index - 1,
		"merkle proof": proofData,
	}

	return info, nil
}

// get merkle index and merkle proof of an account at a given height
func (api *PublicSubchainAPI) GetBalanceMerkleInfo(account common.Address, height int64) (map[string]interface{}, error) {

	var statedb *state.Statedb
	var err error

	if height < 0 {
		if statedb, err = api.s.ChainBackend().GetCurrentState(); err != nil {
			return nil, err
		}
	} else {
		if statedb, err = api.GetStatedbByHeight(uint64(height)); err != nil {
			return nil, err
		}
	}
	nonce := statedb.GetNonce(account)
	balance := statedb.GetBalance(account)

	var index uint64
	var indexBytes []byte
	if indexBytes, err = api.s.GetAccountIndexDB().Get(account.Bytes()); err != nil {
		return nil, err
	}

	if err = rlp.DecodeBytes(indexBytes, &index); err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(common.EmptyHash, height) // return subblock
	if err != nil {
		return nil, err
	}

	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return nil, err
	}

	if index >= swExtra.AccountCount {
		return nil, err
	}

	var level []common.Hash
	var accountBytes []byte
	for i := uint64(0); i < swExtra.AccountCount; i++ {
		indexBytes, _ = rlp.EncodeToBytes(uint(i))
		if accountBytes, err = api.s.GetIndexAccountDB().Get(indexBytes); err != nil {
			return nil, err
		}
		tempAccount := common.BytesToAddress(accountBytes)
		tempNonce := statedb.GetNonce(tempAccount)
		tempBalance := statedb.GetBalance(tempAccount)
		tempState := []interface{}{
			tempAccount.Bytes(),
			tempBalance,
			tempNonce,
		}

		stateBytes, _ := rlp.EncodeToBytes(tempState)
		level = append(level, crypto.MustHash(stateBytes))
	}

	var proofs []common.Hash
	proofs = merkle.GetMerkleProof(level, int(index))

	proofData, err := rlp.EncodeToBytes(proofs)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"account":      account,
		"nonce":        nonce,
		"balance":      balance,
		"merkle index": index,
		"merkle proof": proofData,
	}
	return info, nil
}

// get the root hash of the recentTxTree of a block
func (api *PublicSubchainAPI) GetRecentTxTreeRoot(height uint64) (string, error) {
	block, err := api.s.GetBlock(common.EmptyHash, int64(height)) // return subblock
	if err != nil {
		return "", err
	}

	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return "", err
	}

	return swExtra.RecentTxHashStem.Hex(), nil
}

// get the merkle index and proof of the recent txs of an account
func (api *PublicSubchainAPI) GetRecentTxMerkleInfo(account common.Address, height uint64) (map[string]interface{}, error) {
	accountToIndexMap := make(map[common.Address]uint)
	var accTxs []*types.AccountTxs
	start := height - common.RelayInterval + 1
	end := height
	for i := start; i <= end; i++ {
		prevBlock, err := api.s.ChainBackend().GetStore().GetBlockByHeight(uint64(i))
		if err != nil {
			return nil, err
		}
		// Obtain a SubTransaction tx
		for txIndex, prevTx := range prevBlock.Transactions {
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
				return nil, err
			}
			for j := 0; j < 2; j++ {
				var acc common.Address
				if j == 0 {
					acc = prevTx.Data.From
				} else if prevTx.Data.From != prevTx.Data.To {
					acc = prevTx.Data.To
				} else {
					break
				}
				if index, ok := accountToIndexMap[acc]; ok {
					accTxs[index].Txs = append(accTxs[index].Txs, dataForStem)
				} else {
					// new account
					var curlen = len(accountToIndexMap)
					var newTxs [][]byte
					// fmt.Printf("account: %v, curlen: %d", acc, curlen)
					accountToIndexMap[acc] = uint(curlen)
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
	var level []common.Hash
	for _, value := range accTxs {
		valueBytes, err := rlp.EncodeToBytes(value)
		if err != nil {
			return nil, err
		}
		level = append(level, crypto.MustHash(valueBytes))
	}
	var index uint
	index, ok := accountToIndexMap[account]
	if !ok {
		// TODO: return error not found
		return nil, errors.New("Not Found")
	}
	var proofs []common.Hash
	proofs = merkle.GetMerkleProof(level, int(index))

	proofData, err := rlp.EncodeToBytes(proofs)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"account":      account,
		"merkle index": index,
		"merkle proof": proofData,
	}
	return info, nil
}

// get txs and signatures of an account between two block heights
func (api *PublicSubchainAPI) GetAccountTx(account common.Address, start uint64, end uint64) (map[string]interface{}, error) {
	var txs [][]byte
	var sigs [][]byte
	for i := start; i <= end; i++ {
		prevBlock, err := api.s.ChainBackend().GetStore().GetBlockByHeight(uint64(i))
		if err != nil {
			return nil, err
		}

		// Obtain a SubTransaction
		for txIdx, prevTx := range prevBlock.Transactions {
			if txIdx == 0 {
				continue
			}
			if account == prevTx.Data.From || account == prevTx.Data.To {
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
					return nil, err
				}
				txs = append(txs, dataForStem)

				payloadExtra, err := types.ExtractTxPayload(prevTx.Data.Payload)
				if err != nil {
					return nil, err
				}
				sigs = append(sigs, []byte(payloadExtra.SignStringForStem))
			}
		}
	}
	txsData, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return nil, err
	}
	sigsData, err := rlp.EncodeToBytes(sigs)
	if err != nil {
		return nil, err
	}
	info := map[string]interface{}{
		"account":    account,
		"txs":        txsData,
		"signatures": sigsData,
	}
	return info, nil
}

// get the updated accounts during the last relayInterval (traced back from given height)
func (api *PublicSubchainAPI) GetUpdatedAccountInfo(height uint64) (map[string]interface{}, error) {
	var updatedAccounts []common.Address
	var balances []*big.Int
	var curStatedb *state.Statedb
	var prevStatedb *state.Statedb
	var err error
	if height < common.RelayInterval {
		return nil, errors.New("Height too low")
	}
	prevHeight := height - common.RelayInterval

	if curStatedb, err = api.GetStatedbByHeight(height); err != nil {
		return nil, err
	}

	if prevStatedb, err = api.GetStatedbByHeight(prevHeight); err != nil {
		return nil, err
	}

	block, err := api.s.GetBlock(common.EmptyHash, int64(height)) // return subblock
	if err != nil {
		return nil, err
	}
	swExtra, err := types.ExtractSecondWitnessInfo(block.Header)
	if err != nil {
		return nil, err
	}
	for i := uint64(0); i < swExtra.AccountCount; i++ {
		indexBytes, _ := rlp.EncodeToBytes(uint(i))
		accountBytes, err := api.s.GetIndexAccountDB().Get(indexBytes)
		if err != nil {
			return nil, err
		}
		account := common.BytesToAddress(accountBytes)
		curBalance := curStatedb.GetBalance(account)
		prevBalance := prevStatedb.GetBalance(account)
		if curBalance != prevBalance {
			updatedAccounts = append(updatedAccounts, account)
			balances = append(balances, curBalance)
		}
	}
	updatedAccountsBytes, _ := rlp.EncodeToBytes(updatedAccounts)
	balancesBytes, _ := rlp.EncodeToBytes(balances)
	info := map[string]interface{}{
		"updated accounts": updatedAccountsBytes,
		"balances":         balancesBytes,
	}
	return info, nil
}

func (api *PublicSubchainAPI) GetFee(height uint64) (map[string]interface{}, error) {
	var curStatedb *state.Statedb
	var prevStatedb *state.Statedb
	var err error
	if height < common.RelayInterval || height%common.RelayInterval != 0 {
		return nil, errors.New("Must be a relay block")
	}
	prevHeight := height - common.RelayInterval

	if curStatedb, err = api.GetStatedbByHeight(height); err != nil {
		return nil, err
	}

	if prevStatedb, err = api.GetStatedbByHeight(prevHeight); err != nil {
		return nil, err
	}
	account := common.SubchainFeeAccount
	curBalance := curStatedb.GetBalance(account)
	prevBalance := prevStatedb.GetBalance(account)
	fee := big.NewInt(int64(0)).Sub(curBalance, prevBalance)

	block, err := api.s.GetBlock(common.EmptyHash, int64(height)) // return subblock
	if err != nil {
		return nil, err
	}
	bftExt, err := types.ExtractBftExtra(block.Header)
	if err != nil {
		return nil, err
	}
	verNum := big.NewInt(int64(len(bftExt.Verifiers)))
	fee.Div(fee, verNum)
	info := map[string]interface{}{
		"fee":    fee,
		"verNum": verNum,
	}
	return info, nil
}

// get block relay interval
func (api *PublicSubchainAPI) GetRelayInterval() uint64 {
	return common.RelayInterval
}

func (api *PublicSubchainAPI) GetStatedbByHeight(height uint64) (*state.Statedb, error) {
	// current statedb
	blockHash, err := api.s.ChainBackend().GetStore().GetBlockHash(height)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block hash by height %v", height)
	}
	header, err := api.s.ChainBackend().GetStore().GetBlockHeader(blockHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get block header by hash %v", blockHash)
	}
	statedb, err := api.s.ChainBackend().GetState(header.StateHash)
	if err != nil {
		return nil, err
	}
	return statedb, nil
}
