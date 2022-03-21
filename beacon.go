package main

import (
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	GET_REQUEST = "GET"

	// Beacon Endpoints
	V1_CONFIG_SPEC_ENDPOINT                       = "/eth/v1/config/spec"
	V1_BEACON_GENESIS_ENDPOINT                    = "/eth/v1/beacon/genesis"
	V1_BEACON_HEADERS_ENDPOINT                    = "/eth/v1/beacon/headers/%d"
	V1_BEACON_STATE_FINALITY_CHECKPOINTS_ENDPOINT = "/eth/v1/beacon/states/%d/finality_checkpoints"
	V1_BEACON_STATE_COMMITTEES_ENDPOINT           = "/eth/v1/beacon/states/%d/committees"
	V1_BEACON_BLOCKS_ATTESTATIONS_ENDPOINT        = "/eth/v1/beacon/blocks/%d/attestations"
)

type Spec struct {
	SecondsPerSlot uint64 `json:"SECONDS_PER_SLOT,string"`
	SlotsPerEpoch  uint64 `json:"SLOTS_PER_EPOCH,string"`
}

type GenesisResponse struct {
	GenesisTime uint64 `json:"genesis_time,string"`
}

type BeaconBlockMessage struct {
	Slot          uint64 `json:"slot,string"`
	ProposerIndex uint64 `json:"proposer_index,string"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

type BeaconBlockHeader struct {
	Message   BeaconBlockMessage `json:"message"`
	Signature string             `json:"signature"`
}

type BeaconBlockResponse struct {
	Root      string            `json:"root"`
	Canonical bool              `json:"canonical"`
	Header    BeaconBlockHeader `json:"header"`
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

//go:generate go run github.com/fjl/gencodec -type Attestation -field-override AttestationMarshaling -out gen_att.go
type Attestation struct {
	AggregationBits uint64          `json:"aggregation_bits"`
	Data            AttestationData `json:"data"`
	Signature       string          `json:"signature"`
}
type AttestationMarshaling struct {
	AggregationBits hexutil.Uint64  `json:"aggregation_bits"`
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
