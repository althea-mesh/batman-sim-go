package main

import (
	"testing"
	"time"
)

func TestSaturate(t *testing.T) {
	pChan := make(chan (Packet))
	edge := Edge{
		Throughput:    1000 * 1000, // 1 megabit
		PacketChannel: &pChan,
	}

	hundredKb := Packet{
		Payload: make([]byte, 1000*100),
	}

	go edge.SendPacket(hundredKb)
	select {
	case <-time.After(time.Millisecond * 1):
		t.Fatalf("Packet should have succeeded.")
	case <-pChan:
	}

	go edge.SendPacket(hundredKb)
	select {
	case <-time.After(time.Millisecond * 1):
	case <-pChan:
		t.Fatalf("Packet should have been dropped.")
	}

	time.Sleep(time.Millisecond * 1805)

	go edge.SendPacket(hundredKb)
	select {
	case <-time.After(time.Millisecond * 10):
		t.Fatalf("Packet should have succeeded.")
	case <-pChan:
	}

	go edge.SendPacket(hundredKb)
	select {
	case <-time.After(time.Millisecond * 10):
	case <-pChan:
		t.Fatalf("Packet should have been dropped.")
	}
}

// 	timer := time.Now()
// L:
// 	for {
// 		go edge.SendPacket(Packet{})
// 		select {
// 		default:
// 		case <-pChan:
// 			fmt.Println(time.Since(timer))
// 			// if time.Since(timer) > time.Millisecond*805 ||
// 			// 	time.Since(timer) < time.Millisecond*795 {
// 			// 	t.Fatalf("Not in time range 1.5s - 1.7s %v", time.Since(timer))
// 			// }
// 			break L
// 		}
// 	}
