package main

import (
	"log"
	"sync"
	"time"
)

func main() {
	ch := make(chan int)
	go listen(ch)
	go send(ch, 4)
	time.Sleep(time.Second)
	go send(ch, 1)
	time.Sleep(time.Second)
	go send(ch, 1)
	time.Sleep(time.Second)
	go send(ch, 1)
	time.Sleep(time.Second)
	go send(ch, 1)
	time.Sleep(time.Second)
	go send(ch, 1)

	time.Sleep(time.Minute)
}

func send(ch chan int, i int) {
	ch <- i
}

type Link struct {
	sat bool
	mut *sync.Mutex
}

func MakeLink() Link {
	return Link{false, &sync.Mutex{}}
}

func (l *Link) Saturate(i int) {
	l.mut.Lock()
	l.sat = true
	l.mut.Unlock()

	time.Sleep(time.Duration(i) * time.Second)

	l.mut.Lock()
	l.sat = false
	l.mut.Unlock()
}

func (l *Link) IsSaturated() bool {
	l.mut.Lock()
	sat := l.sat
	l.mut.Unlock()

	return sat
}

func listen(ch chan int) {
	l := MakeLink()
	for {
		i := <-ch
		if !l.IsSaturated() {
			go l.Saturate(i)
			log.Println("keeping", i)
		} else {
			log.Println("discarding", i)
		}
	}
}

// package main

// import (
// 	"encoding/json"
// 	"errors"
// 	"log"
// 	"time"
// )

// const (
// 	timeMultiplier = 1000
// 	hopPenalty     = 15 / 255
// )

// type Network struct {
// 	Nodes map[string]*Node
// 	Edges map[string]*Edge
// }

// type Node struct {
// 	*Network
// 	Address       string
// 	Originators   map[string]*Originator
// 	Neighbors     map[string]*Neighbor
// 	OgmSequence   int
// 	PacketChannel chan (Packet)
// }

// type Neighbor struct {
// 	Address    string
// 	Throughput int
// }

// type Originator struct {
// 	Address string
// 	NextHop struct {
// 		Address    string
// 		Throughput int
// 	}
// 	OgmSequence int
// }

// type Edge struct {
// 	Throughput     int
// 	Bucket         int
// 	PacketChannel  chan (Packet)
// 	TicksPerSecond int
// 	Destination    *Node
// }

// type OGM struct {
// 	Sequence          int
// 	OriginatorAddress string
// 	SenderAddress     string
// 	Throughput        int
// 	Timestamp         int
// }

// type Packet struct {
// 	Type    string
// 	Payload []byte
// }

// // func (router *Node) BroadcastOgm() {
// // 	router.OgmSequence++
// // 	for address, _ := range router {
// // 		router.
// // 	}
// // }

// func (edge *Edge) SendBits(num int) bool {
// 	newBucket := edge.Bucket + num
// 	if newBucket > edge.Throughput {
// 		edge.Bucket = edge.Throughput
// 		return false
// 	}
// 	edge.Bucket = newBucket
// 	return true
// }

// func (edge *Edge) RunTicker() {
// 	for range time.Tick(time.Second / time.Duration(edge.TicksPerSecond)) {
// 		if edge.Bucket > 0 {
// 			edge.Bucket -= edge.Throughput / edge.TicksPerSecond
// 		}
// 	}
// }

// func (edge *Edge) Listen() {
// 	for {
// 		packet := <-edge.PacketChannel
// 		if edge.SendBits(len(packet.Payload)) {
// 			edge.Destination.PacketChannel <- packet
// 		}
// 	}
// }

// func (node *Node) SendPacket(destAddress string, packetType string, payload []byte) error {
// 	edge := node.Network.Edges[node.Address+"->"+destAddress]
// 	if edge == nil {
// 		return errors.New("edge not found")
// 	}

// 	if edge.SendBits(len(payload)) {
// 		edge.PacketChannel <- Packet{packetType, payload}
// 	}

// 	return nil
// }

// func (node *Node) Listen() {
// 	for {
// 		packet := <-node.PacketChannel
// 		var err error
// 		switch packet.Type {
// 		case "OGM":
// 			err = node.HandleOGM(packet.Payload)
// 		}

// 		if err != nil {
// 			log.Println(node.Address, err)
// 		}
// 	}
// }

// func (node *Node) HandleOGM(payload []byte) error {
// 	ogm := OGM{}
// 	err := json.Unmarshal(payload, ogm)
// 	if err != nil {
// 		return err
// 	}

// 	if ogm.OriginatorAddress == node.Address {
// 		return nil
// 	}

// 	adjustedOGM, err := node.AdjustOGM(ogm)
// 	if err != nil {
// 		return err
// 	}

// 	err = node.UpdateOriginator(*adjustedOGM)
// 	if err != nil {
// 		return err
// 	}

// 	node.RebroadcastOGM(*adjustedOGM)
// 	return nil
// }

// func (node *Node) AdjustOGM(ogm OGM) (*OGM, error) {
// 	/* Update the received throughput metric to match the link
// 	 * characteristic:
// 	 *  - If this OGM traveled one hop so far (emitted by single hop
// 	 *    neighbor) the path throughput metric equals the link throughput.
// 	 *  - For OGMs traversing more than one hop the path throughput metric is
// 	 *    the smaller of the path throughput and the link throughput.
// 	 */
// 	neighbor := node.Neighbors[ogm.SenderAddress]
// 	if neighbor == nil {
// 		return nil, errors.New("OGM not sent from neighbor")
// 	}

// 	if ogm.OriginatorAddress == ogm.SenderAddress {
// 		ogm.Throughput = neighbor.Throughput
// 	} else {
// 		if neighbor.Throughput < ogm.Throughput {
// 			ogm.Throughput = neighbor.Throughput
// 		}
// 	}

// 	ogm.Throughput = ogm.Throughput - (ogm.Throughput * hopPenalty)
// 	return &ogm, nil
// }

// func (node *Node) UpdateOriginator(ogm OGM) error {
// 	originator := node.Originators[ogm.OriginatorAddress]

// 	if originator == nil {
// 		originator := Originator{
// 			OgmSequence: ogm.Sequence,
// 			Address:     ogm.OriginatorAddress,
// 		}
// 		originator.NextHop.Address = ogm.SenderAddress
// 		originator.NextHop.Throughput = ogm.Throughput

// 		node.Originators[ogm.OriginatorAddress] = &originator
// 	} else if ogm.Sequence > originator.OgmSequence {
// 		if originator.NextHop.Throughput < ogm.Throughput {
// 			originator.NextHop.Address = ogm.SenderAddress
// 			originator.NextHop.Throughput = ogm.Throughput
// 		}
// 	} else {
// 		return errors.New("ogm sequence too low")
// 	}
// 	return nil
// }

// func (node *Node) RebroadcastOGM(ogm OGM) error {
// 	ogm.SenderAddress = node.Address
// 	payload, err := json.Marshal(ogm)
// 	if err != nil {
// 		return err
// 	}
// 	for _, neighbor := range node.Neighbors {
// 		err = node.SendPacket(neighbor.Address, "OGM", payload)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }
