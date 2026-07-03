package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/maksim-mshp/nornickel-hackathon/internal/platform/events"
	natsclient "github.com/nats-io/nats.go"
)

const (
	defaultReconnectWait = 2 * time.Second
	defaultConnectWait   = 10 * time.Second
	defaultStreamMaxAge  = 72 * time.Hour
)

type StreamSpec struct {
	Name     string
	Subjects []string
}

func DefaultStreams() []StreamSpec {
	return []StreamSpec{
		{Name: "KMAP_DOCS", Subjects: []string{"kmap.doc.v1.>"}},
		{Name: "KMAP_FACTS", Subjects: []string{"kmap.facts.v1.>"}},
		{Name: "KMAP_EPI", Subjects: []string{"kmap.epistemic.v1.>"}},
		{Name: "KMAP_AUDIT", Subjects: []string{"kmap.audit.v1.>"}},
		{Name: "KMAP_DLQ", Subjects: []string{"kmap.dlq.>"}},
	}
}

type Config struct {
	URL     string
	Name    string
	Streams []StreamSpec
}

type Bus struct {
	conn *natsclient.Conn
	js   natsclient.JetStreamContext
}

func New(_ context.Context, cfg Config) (*Bus, error) {
	if cfg.URL == "" {
		return nil, errors.New("nats url is required")
	}
	name := cfg.Name
	if name == "" {
		name = "kmap"
	}

	conn, err := natsclient.Connect(cfg.URL,
		natsclient.Name(name),
		natsclient.MaxReconnects(-1),
		natsclient.ReconnectWait(defaultReconnectWait),
		natsclient.Timeout(defaultConnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("init jetstream: %w", err)
	}

	bus := &Bus{conn: conn, js: js}
	if err := bus.ensureStreams(cfg.Streams); err != nil {
		conn.Close()
		return nil, err
	}
	return bus, nil
}

func (bus *Bus) ensureStreams(specs []StreamSpec) error {
	if len(specs) == 0 {
		specs = DefaultStreams()
	}
	for _, spec := range specs {
		if spec.Name == "" || len(spec.Subjects) == 0 {
			return fmt.Errorf("invalid stream spec: %+v", spec)
		}
		config := &natsclient.StreamConfig{
			Name:      spec.Name,
			Subjects:  spec.Subjects,
			Storage:   natsclient.FileStorage,
			Retention: natsclient.LimitsPolicy,
			MaxAge:    defaultStreamMaxAge,
		}
		if _, err := bus.js.AddStream(config); err != nil {
			if _, updateErr := bus.js.UpdateStream(config); updateErr != nil {
				return fmt.Errorf("ensure stream %s: %w", spec.Name, errors.Join(err, updateErr))
			}
		}
	}
	return nil
}

func (bus *Bus) Publish(ctx context.Context, env events.Envelope) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := env.Marshal()
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	msg := &natsclient.Msg{Subject: env.Type, Data: data, Header: natsclient.Header{}}
	msg.Header.Set(natsclient.MsgIdHdr, env.ID)
	if _, err := bus.js.PublishMsg(msg); err != nil {
		return fmt.Errorf("publish %s: %w", env.Type, err)
	}
	return nil
}

func (bus *Bus) Subscribe(ctx context.Context, sub events.Subscription) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if sub.Subject == "" {
		return errors.New("subscription subject is required")
	}
	if sub.Handler == nil {
		return errors.New("subscription handler is required")
	}

	opts := []natsclient.SubOpt{
		natsclient.Durable(durableName(sub)),
		natsclient.ManualAck(),
		natsclient.DeliverAll(),
		natsclient.AckWait(15 * time.Minute),
		natsclient.MaxDeliver(-1),
	}

	handler := func(msg *natsclient.Msg) {
		bus.handle(ctx, sub.Handler, msg)
	}
	if sub.Queue != "" {
		nsub, err := bus.js.QueueSubscribe(sub.Subject, sub.Queue, handler, opts...)
		if err != nil {
			return fmt.Errorf("subscribe %s: %w", sub.Subject, err)
		}
		go func() {
			<-ctx.Done()
			_ = nsub.Unsubscribe()
		}()
		return nil
	}

	nsub, err := bus.js.Subscribe(sub.Subject, handler, opts...)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", sub.Subject, err)
	}

	go func() {
		<-ctx.Done()
		_ = nsub.Unsubscribe()
	}()
	return nil
}

func (bus *Bus) handle(ctx context.Context, handler events.Handler, msg *natsclient.Msg) {
	env, err := events.Unmarshal(msg.Data)
	if err != nil {
		_ = msg.Term()
		return
	}
	message := events.Message{
		Envelope:    env,
		NatsMsgID:   msg.Header.Get(natsclient.MsgIdHdr),
		TraceParent: msg.Header.Get("traceparent"),
	}
	if meta, err := msg.Metadata(); err == nil {
		message.Delivered = meta.NumDelivered
	}
	switch handler(ctx, message) {
	case events.Ack:
		_ = msg.Ack()
	case events.Nack:
		_ = msg.Nak()
	case events.Term:
		_ = msg.Term()
	}
}

func (bus *Bus) Close() error {
	if bus.conn != nil {
		if err := bus.conn.Drain(); err != nil {
			return fmt.Errorf("drain nats: %w", err)
		}
	}
	return nil
}

func durableName(sub events.Subscription) string {
	if sub.Durable != "" {
		return sub.Durable
	}
	return "kmap-" + sanitize(sub.Subject)
}

func sanitize(value string) string {
	out := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}
