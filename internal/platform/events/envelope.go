package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	SpecVersion     = "1.0"
	ContentType     = "application/json"
	MaxPayloadBytes = 256 * 1024
)

type Envelope struct {
	SpecVersion     string          `json:"specversion"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	Subject         string          `json:"subject,omitempty"`
	Time            time.Time       `json:"time"`
	DataContentType string          `json:"datacontenttype"`
	Data            json.RawMessage `json:"data,omitempty"`
}

type Event struct {
	Type    string
	Source  string
	Subject string
	Data    any
}

func New(event Event) (Envelope, error) {
	if err := ValidateType(event.Type); err != nil {
		return Envelope{}, err
	}
	if event.Source == "" {
		return Envelope{}, errors.New("event source is required")
	}

	raw, err := marshalData(event.Data)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		SpecVersion:     SpecVersion,
		ID:              newID(),
		Source:          event.Source,
		Type:            event.Type,
		Subject:         event.Subject,
		Time:            time.Now().UTC(),
		DataContentType: ContentType,
		Data:            raw,
	}, nil
}

func (env Envelope) Validate() error {
	if env.SpecVersion != SpecVersion {
		return fmt.Errorf("unsupported specversion %q", env.SpecVersion)
	}
	if env.ID == "" {
		return errors.New("envelope id is required")
	}
	if env.Source == "" {
		return errors.New("envelope source is required")
	}
	if env.Time.IsZero() {
		return errors.New("envelope time is required")
	}
	if err := ValidateType(env.Type); err != nil {
		return err
	}
	if len(env.Data) > MaxPayloadBytes {
		return fmt.Errorf("envelope payload %d bytes exceeds limit %d", len(env.Data), MaxPayloadBytes)
	}
	if len(env.Data) > 0 && env.DataContentType == "" {
		return errors.New("datacontenttype is required when data is present")
	}
	return nil
}

func (env Envelope) UnmarshalData(target any) error {
	if len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, target); err != nil {
		return fmt.Errorf("unmarshal event data: %w", err)
	}
	return nil
}

func (env Envelope) Marshal() ([]byte, error) {
	if err := env.Validate(); err != nil {
		return nil, fmt.Errorf("validate envelope: %w", err)
	}
	return json.Marshal(env)
}

func Unmarshal(raw []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Envelope{}, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if err := env.Validate(); err != nil {
		return Envelope{}, fmt.Errorf("validate envelope: %w", err)
	}
	return env, nil
}

func marshalData(data any) (json.RawMessage, error) {
	if data == nil {
		return nil, nil
	}
	marshaled, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}
	if len(marshaled) > MaxPayloadBytes {
		return nil, fmt.Errorf("event payload %d bytes exceeds limit %d", len(marshaled), MaxPayloadBytes)
	}
	return marshaled, nil
}

func newID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
