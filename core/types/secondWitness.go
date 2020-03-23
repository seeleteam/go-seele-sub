package types

import (
	"bytes"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

// type SecondWitnessExtra struct {
// 	ChallengedTxs    []*Transaction
// 	DepositVers      []common.Address
// 	ExitVers         []common.Address
// 	AccountCount     uint64
// 	TxHashStem       common.Hash
// 	StateHashStem    common.Hash
// 	RecentTxHashStem common.Hash
// 	BlockSig         crypto.Signature
// }

type SecondWitnessInfo struct {
	ChallengedTxs    []*Transaction
	DepositVers      []common.Address
	ExitVers         []common.Address
	AccountCount     uint64
	TxHashStem       common.Hash
	StateHashStem    common.Hash
	RecentTxHashStem common.Hash
	BlockSig         crypto.Signature
}

// func (swExtra *SecondWitnessExtra) EncodeRLP(w io.Writer) error {
// 	return rlp.Encode(w, []interface{}{
// 		swExtra.DepositVers,
// 		swExtra.ExitVers,
// 	})
// }

// func (swExtra *SecondWitnessExtra) DecodeRLP(s *rlp.Stream) error {
// 	var secondWitnessExtra struct {
// 		DepositVers []common.Address
// 		ExitVers    []common.Address
// 	}
// 	if err := s.Decode(&secondWitnessExtra); err != nil {
// 		return err
// 	}
// 	swExtra.DepositVers, swExtra.ExitVers = secondWitnessExtra.DepositVers, secondWitnessExtra.ExitVers
// 	return nil
// }

// // ExtractSWExtra extract verifiers from SecondWitness
// func (swExtra *SecondWitnessExtra) ExtractSWExtra(h *BlockHeader) error {
// 	// if len(h.ExtraData) < BftExtraVanity {
// 	// 	fmt.Printf("header extra data len %d is smaller than BftExtraVanity %d\n", len(h.ExtraData), BftExtraVanity)
// 	// 	return nil, ErrInvalidBftHeaderExtra
// 	// }

// 	// var swExtra SecondWitnessExtra
// 	err := rlp.DecodeBytes(h.SecondWitness[:], &swExtra)
// 	if err != nil {
// 		fmt.Println("DecodeBytes err, ", err)
// 		return err
// 	}
// 	return nil
// }

// func ExtractSWExtra(h *BlockHeader) (*SecondWitnessExtra, error) {
// 	// if len(h.ExtraData) < BftExtraVanity {
// 	// 	fmt.Printf("header extra data len %d is smaller than BftExtraVanity %d\n", len(h.ExtraData), BftExtraVanity)
// 	// 	return nil, ErrInvalidBftHeaderExtra
// 	// }
// 	fmt.Println("decode swextra", h.SecondWitness)
// 	var bftExtra *SecondWitnessExtra
// 	err := rlp.DecodeBytes(h.ExtraData, &bftExtra)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return bftExtra, nil
// }

// func ExtractSWExtra(h *BlockHeader, val interface{}) error {
// 	return rlp.DecodeBytes(h.SecondWitness[:], val)
// }

func ExtractSecondWitnessInfo(h *BlockHeader) (*SecondWitnessInfo, error) {
	if len(h.ExtraData) < BftExtraVanity {
		return nil, ErrInvalidBftHeaderExtra
	}
	var swInfo *SecondWitnessInfo
	err := rlp.DecodeBytes(h.SecondWitness[BftExtraVanity:], &swInfo)
	if err != nil {
		return nil, err
	}
	return swInfo, nil
}

func PrepareSecondWitness(chTxs []*Transaction, depositVers []common.Address, exitVers []common.Address, accountCount uint64, txHashStem common.Hash, stateHashStem common.Hash, recentTxHashStem common.Hash, blockSig crypto.Signature) ([]byte, error) {
	var buf bytes.Buffer
	// compensate the lack bytes if header.Extra is not enough BftExtraVanity bytes.
	var temp []byte
	temp = append(temp, bytes.Repeat([]byte{0x00}, BftExtraVanity)...)
	buf.Write(temp[:BftExtraVanity])

	swInfo := &SecondWitnessInfo{ // we share the BftExtra struct
		ChallengedTxs:    chTxs,
		DepositVers:      depositVers,
		ExitVers:         exitVers,
		AccountCount:     accountCount,
		TxHashStem:       txHashStem,
		StateHashStem:    stateHashStem,
		RecentTxHashStem: recentTxHashStem,
		BlockSig:         blockSig,
	}

	payload, err := rlp.EncodeToBytes(&swInfo)
	if err != nil {
		return nil, err
	}
	return append(buf.Bytes(), payload...), nil
}
