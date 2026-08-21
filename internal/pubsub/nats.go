package pubsub

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	OrderCreatedSubject = "orders.created"
	OrderStream         = "ORDERS"
	MatchingConsumer    = "matching"
)

type NATS struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

func NewNATS(url string) (*NATS, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("creating JetStream: %w", err)
	}

	return &NATS{
		conn: conn,
		js:   js,
	}, nil
}

func (n *NATS) Close() {
	n.conn.Close()
}

func (n *NATS) PublishOrderCreated(ctx context.Context, orderID string) error {
	_, err := n.js.Publish(ctx, OrderCreatedSubject, []byte(orderID))
	if err != nil {
		return fmt.Errorf("publishing order created event: %w", err)
	}

	return nil
}

func (n *NATS) Publish(ctx context.Context, subject string, payload []byte) (*jetstream.PubAck, error) {
	return n.js.Publish(ctx, subject, payload)
}

func (n *NATS) Subscribe(
	subject string,
	handler nats.MsgHandler,
) (*nats.Subscription, error) {
	sub, err := n.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribing to %s: %w", subject, err)
	}

	return sub, nil
}

func (n *NATS) Conn() *nats.Conn {
	return n.conn
}
