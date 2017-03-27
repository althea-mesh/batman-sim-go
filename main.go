package main

// func main() {
// 	ch := make(chan int)
// 	go listen(ch)
// 	go send(ch, 4)
// 	time.Sleep(time.Second)
// 	go send(ch, 1)
// 	time.Sleep(time.Second)
// 	go send(ch, 1)
// 	time.Sleep(time.Second)
// 	go send(ch, 1)
// 	time.Sleep(time.Second)
// 	go send(ch, 1)
// 	time.Sleep(time.Second)
// 	go send(ch, 1)

// 	time.Sleep(time.Minute)
// }

// func send(ch chan int, i int) {
// 	ch <- i
// }

// func listen(ch chan int) {
// 	l := MakeLink()
// 	for {
// 		i := <-ch
// 		if !l.IsSaturated() {
// 			go l.Saturate(i)
// 			log.Println("keeping", i)
// 		} else {
// 			log.Println("discarding", i)
// 		}
// 	}
// }

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

const (
	timeMultiplier = 1000
	hopPenalty     = 15 / 255
)

// type Network struct {
// 	Nodes map[string]*Node
// 	Edges map[string]*Edge
// }

type Node struct {
	Address       string
	Originators   map[string]Originator
	Neighbors     map[string]Neighbor
	OgmSequence   int
	PacketChannel chan (Packet)
}

type Neighbor struct {
	Address    string
	Throughput int
	Edge       *Edge
}

type Originator struct {
	Address string
	NextHop struct {
		Address    string
		Throughput int
	}
	OgmSequence int
}

type Edge struct {
	Throughput  int
	Destination *Node
	sat         bool
	mut         sync.Mutex
}

type OGM struct {
	Sequence          int
	OriginatorAddress string
	SenderAddress     string
	Throughput        int
	Timestamp         int
}

type Packet struct {
	Type    string
	Payload []byte
}

// func InitNetwork() Network {
// 	return Network{
// 		Nodes: map[string]*Node{
// 			"A": &Node{
// 				Address
// 			},
// 			"B": &Node{},
// 		},
// 		Edges: map[string]*Edge{
// 			"A->B": &Edge{},
// 		},
// 	}
// }

func main() {
	a := Node{
		Address:       "A",
		PacketChannel: make(chan (Packet)),
	}

	b := Node{
		Address:       "B",
		PacketChannel: make(chan (Packet)),
	}

	aToB := Edge{
		Destination: &b,
		Throughput:  10,
	}

	bToA := Edge{
		Destination: &a,
		Throughput:  10,
	}

	a.Neighbors = map[string]Neighbor{
		"B": Neighbor{
			Edge: &aToB,
		},
	}

	b.Neighbors = map[string]Neighbor{
		"A": Neighbor{
			Edge: &bToA,
		},
	}

	go a.Listen()
	go b.Listen()

	a.Neighbors["B"].Edge.SendPacket(Packet{"DATA", []byte("shibby")})
	a.Neighbors["B"].Edge.SendPacket(Packet{"DATA", []byte("shibby")}) // second packet is dropped because edge is saturated
	time.Sleep(time.Minute)
}

func (edge *Edge) Saturate(bits int) {
	edge.mut.Lock()
	edge.sat = true
	edge.mut.Unlock()

	satDuration := time.Duration(bits*(1000000/edge.Throughput)) * time.Microsecond

	log.Println("saturated for", satDuration)
	time.Sleep(satDuration)
	log.Println("unsaturated")

	edge.mut.Lock()
	edge.sat = false
	edge.mut.Unlock()
}

func (edge *Edge) IsSaturated() bool {
	edge.mut.Lock()
	sat := edge.sat
	edge.mut.Unlock()

	return sat
}

func (edge *Edge) SendPacket(packet Packet) {
	if !edge.IsSaturated() {
		go edge.Saturate(len(packet.Payload))
		edge.Destination.PacketChannel <- packet
	}
}

func (node *Node) Listen() {
	for {
		packet := <-node.PacketChannel
		var err error
		switch packet.Type {
		case "OGM":
			err = node.HandleOGM(packet.Payload)
		case "DATA":
			log.Println(string(packet.Payload))
		}

		if err != nil {
			log.Println(node.Address, err)
		}
	}
}

func (node *Node) HandleOGM(payload []byte) error {
	ogm := OGM{}
	err := json.Unmarshal(payload, ogm)
	if err != nil {
		return err
	}

	if ogm.OriginatorAddress == node.Address {
		return nil
	}

	adjustedOGM, err := node.AdjustOGM(ogm)
	if err != nil {
		return err
	}

	err = node.UpdateOriginator(*adjustedOGM)
	if err != nil {
		return err
	}

	node.RebroadcastOGM(*adjustedOGM)
	return nil
}

func (node *Node) AdjustOGM(ogm OGM) (*OGM, error) {
	/* Update the received throughput metric to match the link
	 * characteristic:
	 *  - If this OGM traveled one hop so far (emitted by single hop
	 *    neighbor) the path throughput metric equals the link throughput.
	 *  - For OGMs traversing more than one hop the path throughput metric is
	 *    the smaller of the path throughput and the link throughput.
	 */
	neighbor, exists := node.Neighbors[ogm.SenderAddress]
	if !exists {
		return nil, errors.New("OGM not sent from neighbor")
	}

	if ogm.OriginatorAddress == ogm.SenderAddress {
		ogm.Throughput = neighbor.Throughput
	} else {
		if neighbor.Throughput < ogm.Throughput {
			ogm.Throughput = neighbor.Throughput
		}
	}

	ogm.Throughput = ogm.Throughput - (ogm.Throughput * hopPenalty)
	return &ogm, nil
}

func (node *Node) UpdateOriginator(ogm OGM) error {
	originator, exists := node.Originators[ogm.OriginatorAddress]

	if !exists {
		originator := Originator{
			OgmSequence: ogm.Sequence,
			Address:     ogm.OriginatorAddress,
		}
		originator.NextHop.Address = ogm.SenderAddress
		originator.NextHop.Throughput = ogm.Throughput

		node.Originators[ogm.OriginatorAddress] = originator
	} else if ogm.Sequence > originator.OgmSequence {
		if originator.NextHop.Throughput < ogm.Throughput {
			originator.NextHop.Address = ogm.SenderAddress
			originator.NextHop.Throughput = ogm.Throughput
		}
	} else {
		return errors.New("ogm sequence too low")
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
		neighbor.Edge.SendPacket(Packet{"OGM", payload})
	}
	return nil
}
