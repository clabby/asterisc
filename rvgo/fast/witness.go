package fast

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-preimage"
)

type StepWitness struct {
	// encoded state witness
	State []byte

	MemProof []byte

	PreimageKey    [32]byte // zeroed when no pre-image is accessed
	PreimageValue  []byte   // including the 8-byte length prefix
	PreimageOffset uint64
}

func uint64ToBytes32(v uint64) []byte {
	var out [32]byte
	binary.BigEndian.PutUint64(out[32-8:], v)
	return out[:]
}

func (wit *StepWitness) EncodeStepInput() []byte {
	abiStatePadding := (32 - (uint64(len(wit.State)) % 32)) % 32

	var input []byte
	input = append(input, StepBytes4...)
	input = append(input, uint64ToBytes32(32*2)...)                                           // state data offset in bytes
	input = append(input, uint64ToBytes32(32*2+32+uint64(len(wit.State))+abiStatePadding)...) // proof data offset in bytes
	// TODO pad state data to multiple of 32 bytes
	// TODO also pad proof data

	input = append(input, uint64ToBytes32(uint64(len(wit.State)))...) // state data length in bytes
	input = append(input, wit.State[:]...)
	input = append(input, make([]byte, abiStatePadding)...)
	input = append(input, uint64ToBytes32(uint64(len(wit.MemProof)))...) // proof data length in bytes
	input = append(input, wit.MemProof[:]...)
	return input
}

func (wit *StepWitness) HasPreimage() bool {
	return wit.PreimageKey != ([32]byte{})
}

func (wit *StepWitness) EncodePreimageOracleInput() ([]byte, error) {
	if wit.PreimageKey == ([32]byte{}) {
		return nil, errors.New("cannot encode pre-image oracle input, witness has no pre-image to proof")
	}

	switch preimage.KeyType(wit.PreimageKey[0]) {
	case preimage.LocalKeyType:
		// We have no on-chain form of preparing the bootstrap pre-images onchain yet.
		// So instead we cheat them in.
		// In production usage there should be an on-chain contract that exposes this,
		// rather than going through the global keccak256 oracle.
		var input []byte
		input = append(input, CheatBytes4...)
		input = append(input, uint64ToBytes32(wit.PreimageOffset)...)
		input = append(input, wit.PreimageKey[:]...)
		var tmp [32]byte
		copy(tmp[:], wit.PreimageValue[wit.PreimageOffset:])
		input = append(input, tmp[:]...)
		input = append(input, uint64ToBytes32(uint64(len(wit.PreimageValue))-8)...)
		// Note: we can pad calldata to 32 byte multiple, but don't strictly have to
		return input, nil
	case preimage.Keccak256KeyType:
		var input []byte
		input = append(input, LoadKeccak256PreimagePartBytes4...)
		input = append(input, uint64ToBytes32(wit.PreimageOffset)...)
		input = append(input, uint64ToBytes32(32+32)...) // partOffset, calldata offset
		input = append(input, uint64ToBytes32(uint64(len(wit.PreimageValue))-8)...)
		input = append(input, wit.PreimageValue[8:]...)
		// Note: we can pad calldata to 32 byte multiple, but don't strictly have to
		return input, nil
	default:
		return nil, fmt.Errorf("unsupported pre-image type %d, cannot prepare preimage with key %x offset %d for oracle",
			wit.PreimageKey[0], wit.PreimageKey, wit.PreimageOffset)
	}
}
