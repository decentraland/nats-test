package main

import "log"
import "fmt"
import "time"
import "sync"
import "flag"
import "sync/atomic"
import "github.com/nats-io/nats.go"
import "github.com/golang/protobuf/proto"
import "github.com/decentraland/nats-test/protocol"

func sendMessageEvery(wg *sync.WaitGroup, conn *nats.Conn, topic string, data []byte, freq time.Duration) {
	defer wg.Done()
	for {
		conn.Publish(topic, data)
		time.Sleep(freq)
	}
}

func buildPacket(src string, message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	msg := &protocol.Packet{
		Src: src,
		Data: data,
	}
	return proto.Marshal(msg)
}

func simulatePeer(wg *sync.WaitGroup, conn *nats.Conn, peerID string, islandID string) error {
	positionTopic := fmt.Sprintf("messageBus.%s.position", islandID)
	positionData, err := buildPacket(peerID, &protocol.PositionData{})
	if err != nil {
		return err
	}

	profileTopic := fmt.Sprintf("messageBus.%s.profile", islandID)
	profileData, err := buildPacket(peerID, &protocol.ProfileData{})
	if err != nil {
		return err
	}

	islandTopic := fmt.Sprintf("messageBus.%s.>", islandID)

	wg.Add(3)
	go sendMessageEvery(wg, conn, positionTopic, positionData, 100 * time.Millisecond)
	go sendMessageEvery(wg, conn, profileTopic, profileData, 1000 * time.Millisecond)

	var messages uint32
	conn.Subscribe(islandTopic, func(m *nats.Msg) {
		messages = atomic.AddUint32(&messages, 1)
	})

	go func() {
		for {
			m := atomic.LoadUint32(&messages)
			if m > 0 {
				fmt.Printf("%s (from %s), %d messages received\n", peerID, islandID, m)
			}
			time.Sleep(60 * time.Second)
		}
	}()

	return nil
}

func main() {
	var url string
	var start int
	var totalIslands int
	var totalPeers int
	var debug bool

	flag.StringVar(&url, "url", nats.DefaultURL, "Nats cluster URL")
	flag.IntVar(&start, "start", 0, "Start from this peer number")
	flag.IntVar(&totalIslands, "islands", 10, "Total islands to distribute peers between")
	flag.IntVar(&totalPeers, "peers", 10000, "Total peers to simulate")
	flag.BoolVar(&debug, "debug", false, "Print debug info")

	flag.Parse()

	conn, err := nats.Connect(url)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Drain()

	peerCountByIsland := make(map[string]int, totalIslands)
	var wg sync.WaitGroup
	for i := start; i < (start + totalPeers); i++ {
		peerID := fmt.Sprintf("peer%d", i)
		islandID := fmt.Sprintf("island%d", i % totalIslands)

		count, _ := peerCountByIsland[islandID]
		peerCountByIsland[islandID] = count + 1

		if err := simulatePeer(&wg, conn, peerID, islandID); err != nil {
			log.Fatal(err)
		}
	}

	if debug {
		fmt.Println("Peers by islands in this SuperNode:")
		for island, count := range peerCountByIsland {
			fmt.Printf("island %s: %d peers\n", island, count)
		}
	}


	wg.Wait()
}
