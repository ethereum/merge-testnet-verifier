package main

import (
	"fmt"
	"strings"
)

type Client interface {
	// Get a specific data for a specific slot/block number
	GetDataPoint(dataName MetricName, blockSlotNumber uint64) (interface{}, error)

	// Get the latest block/slot number for this client
	GetLatestBlockSlotNumber() (uint64, error)

	// Update the TTD Block information:
	//   Execution clients will ask for the totalDifficulty and find the TTD block
	//   Consensus clients this is a no-op
	UpdateGetTTDBlockSlot() (*uint64, error)

	// Get the client type: Execution or Beacon
	ClientLayer() ClientLayer

	// Get the client type: Execution or Beacon
	ClientType() ClientType

	// Get the client version if available
	ClientVersion() (string, error)

	// Get the client ID
	ClientID() int

	// Get the client version if available
	String() string

	// Close the clietn
	Close() error
}

type Clients []*Client

func (cs *Clients) ContainsID(clientType ClientType, id int) bool {
	for _, c := range *cs {
		if (*c).ClientType() == clientType && (*c).ClientID() == id {
			return true
		}
	}
	return false
}

func (cs *Clients) Set(typeUrl string) error {
	splitUrl := strings.Split(typeUrl, ",")
	if len(splitUrl) != 2 {
		return fmt.Errorf("Invalid format")
	}
	clientTypeStr := splitUrl[0]
	url := splitUrl[1]

	clientType, ok := ParseClientTypeString(clientTypeStr)
	if !ok {
		return fmt.Errorf("Invalid client type: %s", clientTypeStr)
	}

	ct, ok := ClientTypeToLayer[clientType]
	if !ok {
		return fmt.Errorf("Unknown client type")
	}

	clientID := 0
	for {
		if !cs.ContainsID(clientType, clientID) {
			break
		}
		clientID += 1
	}

	switch ct {
	case Execution:
		el, err := NewExecutionClient(clientType, clientID, url)
		if err != nil {
			return err
		}
		var c Client
		c = el
		*cs = append(*cs, &c)
		return nil
	case Beacon:
		bc, err := NewBeaconClient(clientType, clientID, url)
		if err != nil {
			return err
		}
		var c Client
		c = bc
		*cs = append(*cs, &c)
		return nil
	}

	return fmt.Errorf("Unable to instantiate client %s", clientType)
}

func (cs *Clients) String() string {
	str := make([]string, 0)
	for _, c := range *cs {
		str = append(str, (*c).String())
	}
	return strings.Join(str, ",")
}
