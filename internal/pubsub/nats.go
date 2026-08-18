package pubsub

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

const (
	OrderCreatedSubject = "orders.created"
)

type NATS struct {
	conn *nats.Conn
}

func NewNATS(url string) (*NATS, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS: %w", err)
	}

	return &NATS{
		conn: conn,
	}, nil
}

func (n *NATS) Close() {
	n.conn.Close()
}

func (n *NATS) PublishOrderCreated(orderID string) error {
	if err := n.conn.Publish(OrderCreatedSubject, []byte(orderID)); err != nil {
		return fmt.Errorf("publishing order created event: %w", err)
	}

	return n.conn.Flush()
}
