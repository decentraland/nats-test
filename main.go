package main

import "log"
import "fmt"
import "time"
import "flag"
import "os"
import "os/signal"
import "sync/atomic"
import "syscall"
import "github.com/nats-io/nats.go"
import "github.com/golang/protobuf/proto"
import "github.com/decentraland/nats-test/protocol"

type Context struct {
	Conn  *nats.Conn
	Debug bool
}

func sendMessageEvery(ctx *Context, topic string, data []byte, freq time.Duration) {
	for {
		ctx.Conn.Publish(topic, data)
		time.Sleep(freq)
	}
}

func buildPacket(src string, message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	msg := &protocol.Packet{
		Src:  src,
		Data: data,
	}
	return proto.Marshal(msg)
}

func simulatePeer(ctx *Context, peerID string, islandID string) error {
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

	go sendMessageEvery(ctx, positionTopic, positionData, 100*time.Millisecond)
	go sendMessageEvery(ctx, profileTopic, profileData, 1000*time.Millisecond)

	var messages uint32
	ctx.Conn.Subscribe(fmt.Sprintf("messageBus.%s.>", islandID), func(m *nats.Msg) {
		messages = atomic.AddUint32(&messages, 1)
	})

	go func() {
		for {
			m := atomic.LoadUint32(&messages)
			atomic.StoreUint32(&messages, 0)
			if ctx.Debug {
				fmt.Printf("%s (from %s), %d messages received last minute\n", peerID, islandID, m)
			}
			time.Sleep(60 * time.Second)
		}
	}()

	return nil
}

func natsErrHandler(nc *nats.Conn, sub *nats.Subscription, natsErr error) {
	fmt.Printf("error: %v\n", natsErr)
	if natsErr == nats.ErrSlowConsumer {
		pendingMsgs, _, err := sub.Pending()
		if err != nil {
			fmt.Printf("couldn't get pending messages: %v", err)
			return
		}
		fmt.Printf("Falling behind with %d pending messages on subject %q.\n", pendingMsgs, sub.Subject)
	}
}

func main() {
	var url string
	var totalIslands int
	var totalPeers int
	var ctx Context

	flag.StringVar(&url, "url", nats.DefaultURL, "Nats cluster URL")
	flag.IntVar(&totalIslands, "islands", 10, "Total islands to distribute peers between")
	flag.IntVar(&totalPeers, "peers", 10000, "Total peers to simulate")
	flag.BoolVar(&ctx.Debug, "debug", false, "Print debug info")

	flag.Parse()

	conn, err := nats.Connect(url, nats.ErrorHandler(natsErrHandler))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Drain()
	ctx.Conn = conn

	nodeID := time.Now().UTC().UnixMilli()
	peerCountByIsland := make(map[string]int, totalIslands)
	for i := 0; i < totalPeers; i++ {
		peerID := fmt.Sprintf("%d|peer%d", nodeID, i)
		islandID := fmt.Sprintf("island%d", i%totalIslands)

		count, _ := peerCountByIsland[islandID]
		peerCountByIsland[islandID] = count + 1

		if err := simulatePeer(&ctx, peerID, islandID); err != nil {
			log.Fatal(err)
		}
	}

	if ctx.Debug {
		fmt.Println("Peers by islands in this SuperNode:")
		for island, count := range peerCountByIsland {
			fmt.Printf("island %s: %d peers\n", island, count)
		}
	}

	sigChannel := make(chan os.Signal, 1)
  signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

	<-sigChannel
	fmt.Println("Exiting")
	os.Exit(0)
}
