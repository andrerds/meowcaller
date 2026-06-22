package meowcaller

import (
	"go.mau.fi/whatsmeow/types"

	"github.com/purpshell/meowcaller/rtp"
	"github.com/purpshell/meowcaller/srtp"
)

// Call state machine and the media-pipeline composition (Opus payload → RTP WARP
// header → E2E-SRTP protect, and the reverse). The byte-level crypto/framing lives
// in the rtp/srtp packages; this stitches it together.

// CallDirection is the originating direction of a call.
type CallDirection int

const (
	CallDirectionOutgoing CallDirection = iota
	CallDirectionIncoming
)

// CallPhase is the lifecycle phase of a call.
type CallPhase int

const (
	CallPhaseIdle CallPhase = iota
	CallPhaseCalling
	CallPhaseRinging
	CallPhaseConnecting
	CallPhaseActive
	CallPhaseEnded
)

// CallSession is the per-call signaling state with validated phase transitions.
type CallSession struct {
	CallID      string
	PeerJID     types.JID
	CallCreator types.JID
	Direction   CallDirection
	IsVideo     bool
	phase       CallPhase
}

// NewOutgoingSession starts an outgoing call session in the Idle phase.
func NewOutgoingSession(callID string, peerJID, callCreator types.JID) *CallSession {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L45-L54
	return &CallSession{
		CallID:      callID,
		PeerJID:     peerJID,
		CallCreator: callCreator,
		Direction:   CallDirectionOutgoing,
		phase:       CallPhaseIdle,
	}
}

// NewIncomingSession starts an incoming call session in the Ringing phase.
func NewIncomingSession(callID string, peerJID, callCreator types.JID) *CallSession {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L56-L65
	return &CallSession{
		CallID:      callID,
		PeerJID:     peerJID,
		CallCreator: callCreator,
		Direction:   CallDirectionIncoming,
		phase:       CallPhaseRinging,
	}
}

// Phase returns the current lifecycle phase.
func (s *CallSession) Phase() CallPhase {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L67-L69
	return s.phase
}

// IsActive reports whether the call is in the Active phase.
func (s *CallSession) IsActive() bool {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L71-L73
	return s.phase == CallPhaseActive
}

// IsEnded reports whether the call has ended.
func (s *CallSession) IsEnded() bool {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L75-L77
	return s.phase == CallPhaseEnded
}

// TransitionTo attempts a phase transition, returning false (no-op) if illegal.
// Ended is reachable from anything except Ended.
func (s *CallSession) TransitionTo(next CallPhase) bool {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L81-L97
	var ok bool
	switch {
	case s.phase == CallPhaseEnded:
		ok = false
	case next == CallPhaseEnded:
		ok = true
	case s.phase == CallPhaseIdle && next == CallPhaseCalling:
		ok = s.Direction == CallDirectionOutgoing
	case s.phase == CallPhaseCalling && next == CallPhaseRinging:
		ok = true
	case s.phase == CallPhaseRinging && next == CallPhaseConnecting:
		ok = true
	case s.phase == CallPhaseConnecting && next == CallPhaseActive:
		ok = true
	case s.phase == next:
		ok = true
	default:
		ok = false
	}
	if ok {
		s.phase = next
	}
	return ok
}

// MediaPipeline composes the outbound (protect) and inbound (unprotect) E2E 1:1
// media path. SFrame is omitted (plain Opus inside WAHKDF SRTP).
type MediaPipeline struct {
	sendKeys     srtp.E2eSrtpKeys
	recvKeys     srtp.E2eSrtpKeys
	warpMITagLen int
	stream       *rtp.RtpStream
	sendRoc      srtp.RocTracker
	recvRoc      srtp.RecvRocTracker
}

// NewMediaPipeline derives both directions from the 32-byte callKey: send keys from
// the self LID, recv keys from the peer LID (an interop-load-bearing convention).
func NewMediaPipeline(callKey []byte, selfJID, peerJID string, ssrc, samplesPerPacket uint32) (*MediaPipeline, error) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L118-L133
	sendKeys, err := srtp.DeriveE2eKeys(callKey, rtp.FormatE2ESrtpParticipantID(selfJID))
	if err != nil {
		return nil, err
	}
	recvKeys, err := srtp.DeriveE2eKeys(callKey, rtp.FormatE2ESrtpParticipantID(peerJID))
	if err != nil {
		return nil, err
	}
	return &MediaPipeline{
		sendKeys:     sendKeys,
		recvKeys:     recvKeys,
		warpMITagLen: srtp.WarpMITagLen,
		stream:       rtp.NewRtpStream(ssrc, samplesPerPacket, false),
	}, nil
}

// ProtectAudio wraps an Opus payload in an RTP WARP header, E2E-SRTP encrypts, and
// appends the WARP MI tag.
func (p *MediaPipeline) ProtectAudio(opusPayload []byte) ([]byte, error) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L136-L150
	header := p.stream.NextPacket(opusPayload, false)
	roc := p.sendRoc.Advance(header.SequenceNumber)
	packet := rtp.EncodeRtpHeader(&header)
	encrypted, err := srtp.CryptPayload(&p.sendKeys, header.Ssrc, header.SequenceNumber, roc, opusPayload)
	if err != nil {
		return nil, err
	}
	packet = append(packet, encrypted...)
	return srtp.AppendWarpMITag(p.sendKeys.AuthKey[:], packet, roc, p.warpMITagLen), nil
}

// UnprotectAudio strips the WARP MI tag (not verified), parses the header, and
// decrypts the payload, guessing the ROC from the recv tracker. ok=false on a
// malformed packet.
func (p *MediaPipeline) UnprotectAudio(packet []byte) (rtp.RtpHeader, []byte, bool) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/src/voip/session.rs#L155-L175
	if len(packet) < 12+p.warpMITagLen {
		return rtp.RtpHeader{}, nil, false
	}
	withoutTag := packet[:len(packet)-p.warpMITagLen]
	header, ok := rtp.ParseRtpHeader(withoutTag)
	if !ok {
		return rtp.RtpHeader{}, nil, false
	}
	headerLen, ok := rtp.RtpHeaderByteLength(withoutTag)
	if !ok || len(withoutTag) <= headerLen {
		return rtp.RtpHeader{}, nil, false
	}
	roc := p.recvRoc.GuessRoc(header.SequenceNumber)
	plain, err := srtp.CryptPayload(&p.recvKeys, header.Ssrc, header.SequenceNumber, roc, withoutTag[headerLen:])
	if err != nil {
		return rtp.RtpHeader{}, nil, false
	}
	return header, plain, true
}
