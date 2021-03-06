package proto

import (
	"context"
	"encoding/json"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

// Metadata holds metadata for the request.
type Metadata = frame.Metadata

// Payload is the request payload with metadata and data.
type Payload struct {
	HasMetadata bool
	Metadata    Metadata
	Data        []byte
}

// Bytes creates a Payload without metadata.
func Bytes(data []byte) *Payload {
	return &Payload{false, nil, data}
}

// Text creates a plain/text Payload without metadata.
func Text(s string) *Payload {
	return &Payload{false, nil, []byte(s)}
}

// JSON creates a application/json Payload without metadata.
func JSON(v interface{}) (*Payload, error) {
	data, err := json.Marshal(v)

	if err != nil {
		return nil, err
	}

	return &Payload{false, nil, []byte(data)}, nil
}

// Text returnes the data as plain/text.
func (payload *Payload) Text() string {
	return string(payload.Data)
}

// WithMetadata returns a Payload with metadata.
func (payload *Payload) WithMetadata(metadata Metadata) *Payload {
	payload.HasMetadata = true
	payload.Metadata = metadata

	return payload
}

// Result of Payload or error
type Result struct {
	Payload *Payload

	Err error
}

// Ok returns a Result with Payload
func Ok(payload *Payload) *Result {
	return &Result{payload, nil}
}

// Err returns a Result with error
func Err(err error) *Result {
	return &Result{nil, err}
}

// PayloadStream returns the payload or error for the stream or channel.
type PayloadStream struct {
	C <-chan *Result
}

// Recv the payload or error for the stream or channel.
func (s *PayloadStream) Recv(ctx context.Context) (*Payload, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case result, ok := <-s.C:
		if ok && result != nil {
			return result.Payload, result.Err
		}

		return nil, nil
	}
}

// TryRecv returns the payload or error for the stream or channel when ready.
func (s *PayloadStream) TryRecv(ctx context.Context) (*Result, bool) {
	select {
	case <-ctx.Done():
		return Err(ctx.Err()), true

	case result, ok := <-s.C:
		if ok && result != nil {
			return result, true
		}

		return nil, true
	default:
		return nil, false
	}
}

// PayloadSink send the payload or erro to the stream or channel.
type PayloadSink struct {
	C chan<- *Result
}

// Close the stream
func (s *PayloadSink) Close() error {
	close(s.C)
	return nil
}

// Send the payload or erro to the stream or channel.
func (s *PayloadSink) Send(ctx context.Context, result *Result) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.C <- result:
		return nil
	}
}

func (payload *Payload) buildRequestResponseFrame(streamID StreamID) *frame.RequestResponseFrame {
	return frame.NewRequestResponseFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildRequestFireAndForgetFrame(streamID StreamID) *frame.RequestFireAndForgetFrame {
	return frame.NewRequestFireAndForgetFrame(streamID, false, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildRequestStreamFrame(streamID StreamID, initReqs uint32) *frame.RequestStreamFrame {
	return frame.NewRequestStreamFrame(streamID, false, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildRequestChannelFrame(streamID StreamID, complete bool, initReqs uint32) *frame.RequestChannelFrame {
	if payload == nil {
		return frame.NewRequestChannelFrame(streamID, false, complete, initReqs, false, nil, nil)
	}

	return frame.NewRequestChannelFrame(streamID, false, complete, initReqs, payload.HasMetadata, payload.Metadata, payload.Data)
}

func (payload *Payload) buildPayloadFrame(streamID StreamID, complete bool) *frame.PayloadFrame {
	return frame.NewPayloadFrame(streamID, false, complete, true, payload.HasMetadata, payload.Metadata, payload.Data)
}

func buildCompleteFrame(streamID StreamID) *frame.PayloadFrame {
	return frame.NewPayloadFrame(streamID, false, true, false, false, nil, nil)
}
