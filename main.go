package main

import "log"
import "fmt"
import "time"
import "sync"
import "flag"
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

	wg.Add(2)
	go sendMessageEvery(wg, conn, positionTopic, positionData, 100 * time.Millisecond)
	go sendMessageEvery(wg, conn, profileTopic, profileData, 1000 * time.Millisecond)
	return nil
}

func main() {
	var url string
	var start int

	flag.StringVar(&url, "url", nats.DefaultURL, "Nats cluster URL")
	flag.IntVar(&start, "start", 0, "Start from peer")

	flag.Parse()

	conn, err := nats.Connect(url)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Drain()

	var wg sync.WaitGroup
	for i := start; i < (start + 1000); i++ {
		peerID := fmt.Sprintf("peer%d", i)
		islandID := fmt.Sprintf("island%d", i % 100)
		if err := simulatePeer(&wg, conn, peerID, islandID); err != nil {
			log.Fatal(err)
		}
	}

	wg.Wait()
}
