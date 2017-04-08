package main

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestSaturateTime(t *testing.T) {
	log.SetFlags(log.Lmicroseconds)
	pChan := make(chan (Packet))
	edge := Edge{
		Throughput:    1000 * 1000, // 1 megabit
		PacketChannel: &pChan,
	}

	hundredKb := Packet{
		Payload: make([]byte, 1000*100),
	}

	go edge.SendPacket(hundredKb)
	<-pChan
	timer := time.Now()
L:
	for {
		go edge.SendPacket(Packet{})
		select {
		default:
		case <-pChan:
			fmt.Println(time.Since(timer))
			break L
		}
	}
}

func TestSaturateBasic(t *testing.T) {
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
	case <-time.After(10 * time.Millisecond):
		t.Fatalf("Packet should have succeeded.")
	case <-pChan:
	}

	go edge.SendPacket(hundredKb)
	select {
	case <-time.After(10 * time.Millisecond):
	case <-pChan:
		t.Fatalf("Packet should have been dropped.")
	}

	time.Sleep(time.Millisecond * 905)

	go edge.SendPacket(hundredKb)
	select {
	case <-time.After(10 * time.Millisecond):
		t.Fatalf("Packet should have succeeded.")
	case <-pChan:
	}
}

func TestSaturateProbability(t *testing.T) {
	successChan := make(chan (bool))
	numTries := 100

	for i := 0; i < numTries; i++ {
		go func() {
			pChan := make(chan (Packet))
			edge := Edge{
				Throughput:    1000 * 1000, // .5 megabit
				PacketChannel: &pChan,
			}

			hundredKb := Packet{
				Payload: make([]byte, 1000*100),
			}

			go edge.SendPacket(hundredKb)
			<-pChan
			time.Sleep(time.Millisecond * 800)

			go edge.SendPacket(hundredKb)
			select {
			case <-time.After(1 * time.Millisecond):
				successChan <- false
			case <-pChan:
				successChan <- true
			}
		}()
	}

	var numSucceeded int
	for i := 0; i < numTries; i++ {
		if <-successChan {
			numSucceeded++
		}
	}

	fmt.Println(numSucceeded)
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
