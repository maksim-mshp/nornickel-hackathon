package events

import "context"

type Publisher interface {
	Publish(ctx context.Context, env Envelope) error
}

type Consumer interface {
	Subscribe(ctx context.Context, sub Subscription) error
}

type Bus interface {
	Publisher
	Consumer
}

type AckAction int

const (
	Ack AckAction = iota
	Nack
	Term
)

type Message struct {
	Envelope    Envelope
	NatsMsgID   string
	TraceParent string
	Delivered   uint64
}

type Handler func(ctx context.Context, msg Message) AckAction

type Subscription struct {
	Subject string
	Queue   string
	Durable string
	Handler Handler
}
