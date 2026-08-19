package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	stream, err := js.AddStream(&nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"orders.created"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("stream created: %s, messages: %d", stream.Config.Name, stream.State.Msgs)
}
