package main

type Client interface {
	// Get a specific data for a specific slot/block number
	GetDataPoint(dataName string, blockSlotNumber uint64) (interface{}, error)

	// Get the latest block/slot number for this client
	GetLatestBlockSlotNumber() (uint64, error)

	// Update the TTD Block information:
	//   Execution clients will ask for the totalDifficulty and find the TTD block
	//   Consensus clients this is a no-op
	UpdateGetTTDBlockSlot() (*uint64, error)

	// Get the client type: Execution or Beacon
	ClientType() string
}
