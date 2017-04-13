package main

/*
When you start receiving packets from a certain source, start an interval timer.

On that interval, send the source an ack message containing a start time, an end time,
and the number of bytes you have received from that source during the time period.

When you receive such an ack from a destination, discard it if you have switched next hops to
that destination after the ack's start time.
(This is because the ack will not be relevant to the new next hop)

Next, compare the number of bytes reported by the ack to the number of bytes you have sent to
that destination during the ack's time period.

Calculate packet success percentage using these numbers. If the packet success percentage computed
from the ack is over some threshold lower than the packet success percentage reported by the next hop,
adjust the packet success percentage through that next hop.
The details of this adjustment are covered elsewhere.

Forward the ack to the next hop for that destination.

When you receive a forwarded ack, follow the same procedure.
*/

import (
	"log"
	"time"
)

const (
	timeMultiplier = 1000
	hopMultiplier  = 0.94
)

type Packet struct {
	Type        string
	Source      string
	Destination string
	Payload     []byte
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	a := Node{
		Address:       "A",
		PacketChannel: make(chan (Packet)),
		Sources:       map[string]Source{},
	}

	b := Node{
		Address:       "B",
		PacketChannel: make(chan (Packet)),
		Sources:       map[string]Source{},
	}

	aToB := Edge{
		PacketChannel: &b.PacketChannel,
		Throughput:    1000 * 100,
	}

	bToA := Edge{
		PacketChannel: &a.PacketChannel,
		Throughput:    1000 * 100,
	}

	a.Neighbors = map[string]Neighbor{
		"B": Neighbor{
			Edge: &aToB,
		},
	}

	a.Destinations = map[string]Destination{
		"B": {
			Address: "B",
			NextHop: NextHop{
				Address: "B",
			},
			PacketsSent: PacketRecords{},
		},
	}

	b.Neighbors = map[string]Neighbor{
		"A": Neighbor{
			Edge: &bToA,
		},
	}

	b.Destinations = map[string]Destination{
		"A": {
			Address: "A",
			NextHop: NextHop{
				Address: "A",
			},
			PacketsSent: PacketRecords{},
		},
	}

	go a.Listen()
	go b.Listen()

	a.SendSpeedTest("B", time.Millisecond*300, 5*1000)
	// time.Sleep(time.Minute)
}
