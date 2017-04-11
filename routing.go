package main

import (
	"encoding/json"
	"errors"
	"time"
)

type OGM struct {
	Sequence           int
	DestinationAddress string
	SenderAddress      string
	PacketSuccess      float64
	Timestamp          int
}

func (node *Node) HandleOGM(payload []byte) error {
	ogm := OGM{}
	err := json.Unmarshal(payload, ogm)
	if err != nil {
		return err
	}

	if ogm.DestinationAddress == node.Address {
		return nil
	}

	adjustedOGM, err := node.AdjustOGM(ogm)
	if err != nil {
		return err
	}

	err = node.UpdateDestination(*adjustedOGM)
	if err != nil {
		return err
	}

	node.RebroadcastOGM(*adjustedOGM)
	return nil
}

func (node *Node) AdjustOGM(ogm OGM) (*OGM, error) {
	neighbor, exists := node.Neighbors[ogm.SenderAddress]
	if !exists {
		return nil, errors.New("OGM not sent from neighbor")
	}

	ogm.PacketSuccess = ogm.PacketSuccess * neighbor.PacketSuccess * hopMultiplier
	return &ogm, nil
}

func (node *Node) UpdateDestination(ogm OGM) error {
	dest, exists := node.Destinations[ogm.DestinationAddress]

	if !exists {
		dest := Destination{
			OgmSequence: ogm.Sequence,
			Address:     ogm.DestinationAddress,
		}
		dest.NextHop.Address = ogm.SenderAddress
		dest.NextHop.PacketSuccess = ogm.PacketSuccess

		node.Destinations[ogm.DestinationAddress] = dest
		return nil
	}

	if ogm.Sequence <= dest.OgmSequence {
		return errors.New("ogm sequence too low")
	}

	if dest.NextHop.PacketSuccess < ogm.PacketSuccess {
		dest.NextHop.PacketSuccess = ogm.PacketSuccess

		if dest.NextHop.Address != ogm.SenderAddress {
			dest.NextHop.Address = ogm.SenderAddress
			dest.NextHop.TimeSwitched = time.Now()
		}
	}

	return nil
}

func (node *Node) RebroadcastOGM(ogm OGM) error {
	ogm.SenderAddress = node.Address
	payload, err := json.Marshal(ogm)
	if err != nil {
		return err
	}
	for _, neighbor := range node.Neighbors {
		node.SendPacket(Packet{
			Type:        "OGM",
			Source:      node.Address,
			Destination: neighbor.Address,
			Payload:     payload,
		})
	}
	return nil
}
