package meowcaller

import (
	"context"
	"errors"
)

// errNotImplemented marks the not-yet-ported engine bodies (the managed surface is the
// locked contract; the orchestration is being lifted from examples/cli).
var errNotImplemented = errors.New("meowcaller: not implemented")

// engine is the internal media + signaling engine behind Client/Call. It owns the
// whatsmeow event wiring (offer / preaccept / accept / relaylatency / mute_v2 / ack /
// terminate), the low-level <ack>/<call> node interception, the relay election and the
// per-frame media loop (encode a Player's frames out, decode the peer's frames into a
// sink). This is where the orchestration currently hand-rolled in examples/cli is being
// lifted to; Client and Call are the public face over it.
//
// The exported surface (Client/Call/Player/AudioSource/AudioSink) is the contract; the
// bodies below are the port target.
type engine struct {
	c *Client
	// TODO(port): per-call map, CallRegistry, whatsmeow glue (resolve LID, encrypt/
	// decrypt callKey), the relay+media loop driven by Call.player/Call.sink, and the
	// <ack>/<call> hook (internalized from examples/cli/callhook.go, or the whatsmeow
	// RawNodeHandler once that lands in the pinned version).
}

// newEngine creates the engine for a Client.
func newEngine(c *Client) *engine {
	return &engine{c: c}
}

// install wires the whatsmeow call event handlers and the <ack>/<call> interception.
// Call before the whatsmeow client connects.
func (e *engine) install() {
	// TODO(port): AddEventHandler for events.CallOffer/CallRelayLatency/CallTransport/
	// CallTerminate, plus the unexported nodeHandlers["ack"]/["call"] hook.
}

// placeCall resolves target to a LID, builds and sends the <offer>, registers the Call,
// and returns it; media starts when the peer answers and the relay endpoint arrives.
func (e *engine) placeCall(ctx context.Context, target string) (*Call, error) {
	// TODO(port): from examples/cli runCall — resolvePeerLID, GetUserDevices,
	// encryptCallKeyForDevice, signaling.BuildOffer, SendNode, coordinator wiring.
	return nil, errNotImplemented
}

// answer sends the deferred preaccept+accept for an inbound call and brings media up.
func (e *engine) answer(c *Call) error {
	// TODO(port): from examples/cli coordinator.onOffer / sendAccept.
	return errNotImplemented
}

// reject declines an inbound call.
func (e *engine) reject(c *Call) error {
	// TODO(port): emit a <reject> stanza (signaling.BuildReject).
	return errNotImplemented
}

// hangup ends a call and tears down its media.
func (e *engine) hangup(c *Call) error {
	// TODO(port): emit <terminate> and cancel the media task.
	return errNotImplemented
}
