package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/bits"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

var (
	GET_REQUEST = "GET"

	// Beacon Endpoints
	V1_CONFIG_SPEC_ENDPOINT                       = "/eth/v1/config/spec"
	V1_BEACON_GENESIS_ENDPOINT                    = "/eth/v1/beacon/genesis"
	V2_BEACON_BLOCKS_ENDPOINT                     = "/eth/v2/beacon/blocks/%d"
	V1_BEACON_HEADERS_ENDPOINT                    = "/eth/v1/beacon/headers/%d"
	V1_BEACON_STATE_FINALITY_CHECKPOINTS_ENDPOINT = "/eth/v1/beacon/states/%d/finality_checkpoints"
	V1_BEACON_STATE_COMMITTEES_ENDPOINT           = "/eth/v1/beacon/states/%d/committees"
	V1_BEACON_BLOCKS_ATTESTATIONS_ENDPOINT        = "/eth/v1/beacon/blocks/%d/attestations"
)

type Spec struct {
	SecondsPerSlot    uint64 `json:"SECONDS_PER_SLOT,string"`
	SlotsPerEpoch     uint64 `json:"SLOTS_PER_EPOCH,string"`
	SyncCommitteeSize uint64 `json:"SYNC_COMMITTEE_SIZE,string"`
}

type GenesisResponse struct {
	GenesisTime uint64 `json:"genesis_time,string"`
}

type AggregationBits []byte

func (ab *AggregationBits) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}
	if str[0:2] != "0x" {
		return fmt.Errorf("SyncCommitteeBits: Not an hex string")
	}
	decodeByteArray, err := hex.DecodeString(str[2:])
	if err != nil {
		return err
	}
	*ab = decodeByteArray
	return nil
}

func (ab AggregationBits) CountSetBits() uint64 {
	count := uint64(0)
	for _, b := range ab[:] {
		count += uint64(bits.OnesCount8(uint8(b)))
	}
	return count
}

type SyncCommitteeSignature [96]byte

func (scs *SyncCommitteeSignature) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}
	if str[0:2] != "0x" {
		return fmt.Errorf("SyncCommitteeSignature: Not an hex string")
	}
	decodeByteArray, err := hex.DecodeString(str[2:])
	if err != nil {
		return err
	}
	if len(SyncCommitteeSignature{}) != len(decodeByteArray) {
		return fmt.Errorf("SyncCommitteeSignature: Invalid length")
	}
	copy((*scs)[:], decodeByteArray)
	return nil
}

type BeaconSyncAggregate struct {
	SyncCommitteeBits      AggregationBits        `json:"sync_committee_bits"`
	SyncCommitteeSignature SyncCommitteeSignature `json:"sync_committee_signature"`
}

type BeaconEth1Data struct {
	DepositRoot  common.Hash `json:"deposit_root"`
	DepositCount uint64      `json:"deposit_count,string"`
	BlockHash    common.Hash `json:"block_hash"`
}

type BeaconBlockBody struct {
	BeaconEth1Data BeaconEth1Data      `json:"eth1_data"`
	Attestations   []Attestation       `json:"attestations"`
	SyncAggregate  BeaconSyncAggregate `json:"sync_aggregate"`
}

type BeaconBlockMessage struct {
	Slot          uint64          `json:"slot,string"`
	ProposerIndex uint64          `json:"proposer_index,string"`
	ParentRoot    common.Hash     `json:"parent_root"`
	StateRoot     common.Hash     `json:"state_root"`
	Body          BeaconBlockBody `json:"body"`
}

type BeaconBlock struct {
	BlockMessage BeaconBlockMessage `json:"message"`
	Signature    string             `json:"signature"`
}

type BeaconHeaderMessage struct {
	Slot          uint64      `json:"slot,string"`
	ProposerIndex uint64      `json:"proposer_index,string"`
	ParentRoot    common.Hash `json:"parent_root"`
	StateRoot     common.Hash `json:"state_root"`
	BodyRoot      common.Hash `json:"body_root"`
}

type BeaconHeader struct {
	Message   BeaconHeaderMessage `json:"message"`
	Signature string              `json:"signature"`
}

type BeaconHeaderResponse struct {
	Root      string       `json:"root"`
	Canonical bool         `json:"canonical"`
	Header    BeaconHeader `json:"header"`
}

type FinalityCheckpoint struct {
	Epoch uint64      `json:"epoch,string"`
	Root  common.Hash `json:"root"`
}

type StateFinalityCheckpoints struct {
	PreviousJustified FinalityCheckpoint `json:"previous_justified"`
	Justified         FinalityCheckpoint `json:"current_justified"`
	Finalized         FinalityCheckpoint `json:"finalized"`
}
type Validators []uint64
type Committee struct {
	Slot       uint64     `json:"slot,string"`
	Index      uint64     `json:"index,string"`
	Validators Validators `json:"validators"`
}
type CommitteeMarshaling struct {
	Slot       uint64   `json:"slot,string"`
	Index      uint64   `json:"index,string"`
	Validators []string `json:"validators"`
}

type AttestationData struct {
	Slot            uint64             `json:"slot,string"`
	Index           uint64             `json:"index,string"`
	BeaconBlockRoot string             `json:"beacon_block_root"`
	Source          FinalityCheckpoint `json:"source"`
	Target          FinalityCheckpoint `json:"target"`
}

type Attestation struct {
	AggregationBits AggregationBits `json:"aggregation_bits"`
	Data            AttestationData `json:"data"`
	Signature       string          `json:"signature"`
}

func (v *Validators) UnmarshalJSON(b []byte) error {
	var sA []string
	err := json.Unmarshal(b, &sA)
	if err != nil {
		return err
	}
	*v = make(Validators, len(sA))
	for i, s := range sA {
		(*v)[i], err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
	}
	return nil
}
