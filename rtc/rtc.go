// Package rtc makes audio and video calls and data connections over
// WebRTC, on pion. It doesn't do signaling: the app carries the offer,
// the answer and the candidates between the peers, over its own server.
//
//	c, _ := rtc.New(ctx, app, rtc.Config{
//		ICEServers: []rtc.ICEServer{{URLs: []string{"stun:stun.example.com:3478"}}},
//		Microphone: true, Speaker: true,
//		OnCandidate: func(c string) { signal.Send("candidate", c) },
//	})
//	offer, _ := c.Offer(ctx)   // send it; the other side calls Answer
//	...
//	c.Accept(answer)           // their answer
//	c.AddCandidate(candidate)  // each of theirs
//
// Audio is G.711 (PCMU, PCMA) out of the box, which every WebRTC peer
// speaks; Config.AudioCodecs adds others, such as Opus. Video needs a
// VideoCodec: frames go out through SendVideo and arrive on
// RemoteVideo.
package rtc

import (
	"image"
	"time"
)

// AudioCodec is an audio codec a call can negotiate.
type AudioCodec interface {
	// MimeType is "audio/opus", "audio/PCMU"...
	MimeType() string
	ClockRate() int
	Channels() int
	// Fmtp is the SDP format parameters line, or "".
	Fmtp() string
	PayloadType() uint8
	// SampleRate is the rate of the PCM it encodes and decodes.
	SampleRate() int
	NewEncoder() (AudioEncoder, error)
	NewDecoder() (AudioDecoder, error)
}

// AudioEncoder encodes 20 ms of 16-bit PCM at a time.
type AudioEncoder interface {
	Encode(pcm []int16) ([]byte, error)
	Close()
}

// AudioDecoder decodes a packet's payload.
type AudioDecoder interface {
	Decode(payload []byte) ([]int16, error)
	Close()
}

// VideoCodec is a video codec a call can negotiate: VP8, VP9, H.264 or
// AV1, whose packets the call splits and joins.
type VideoCodec interface {
	// MimeType is "video/VP8", "video/H264"...
	MimeType() string
	Fmtp() string
	PayloadType() uint8
	NewEncoder(w, h, fps int) (VideoEncoder, error)
	NewDecoder() (VideoDecoder, error)
}

// VideoEncoder encodes frames into the codec's bitstream (Annex B for
// H.264).
type VideoEncoder interface {
	// Encode encodes a frame; keyframe asks for a key frame, which a
	// peer that lost packets needs.
	Encode(frame *image.RGBA, keyframe bool) ([]byte, error)
	Close()
}

// VideoDecoder decodes a frame's bitstream.
type VideoDecoder interface {
	// Decode decodes a frame, or returns nil while it waits for a key
	// frame.
	Decode(frame []byte) (image.Image, error)
	Close()
}

// ICEServer is a STUN or TURN server.
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

// State is a call's connection state.
type State int

const (
	StateNew State = iota
	StateConnecting
	StateConnected
	// StateDisconnected is a lost connection that may come back.
	StateDisconnected
	StateFailed
	StateClosed
)

func (s State) String() string {
	return [...]string{"new", "connecting", "connected", "disconnected", "failed", "closed"}[s]
}

// Config configures New.
type Config struct {
	ICEServers []ICEServer
	// RelayOnly sends everything through TURN, hiding the peers'
	// addresses from each other.
	RelayOnly bool

	// Microphone sends the microphone, with the platform's echo
	// cancellation, noise suppression and gain control.
	Microphone bool
	// MicrophoneDevice is an audio.Inputs ID; the default when empty.
	MicrophoneDevice string
	// Speaker plays the other side; SpeakerPhone uses the loudspeaker
	// on phones.
	Speaker, SpeakerPhone bool
	// SendAudio offers to send audio without the microphone: write it
	// with WriteAudio.
	SendAudio bool
	// OnRemoteAudio receives the other side's audio, decoded, as it
	// arrives (as well as playing it with Speaker).
	OnRemoteAudio func(pcm []int16, sampleRate int)
	// AudioCodecs are preferred over the built-in G.711, in order.
	AudioCodecs []AudioCodec

	// VideoCodecs are the video codecs to offer, in order; without
	// one, there's no video.
	VideoCodecs []VideoCodec
	// SendVideo offers to send video, written with SendVideo.
	SendVideo bool
	// VideoFPS is the rate SendVideo frames are timed at: 30 by
	// default.
	VideoFPS int

	// OnCandidate gets each local ICE candidate, to send to the other
	// side as it's found (trickle ICE). Without it, Offer and Answer
	// wait until all candidates are found and include them.
	OnCandidate func(candidate string)
	// OnState is called as the connection changes.
	OnState func(State)
	// OnDataChannel is called when the other side opens a data channel.
	OnDataChannel func(*DataChannel)

	// test hooks
	loopback bool
}

// Stats are a call's counters.
type Stats struct {
	AudioSent, AudioReceived int // packets
	VideoSent, VideoReceived int // frames
	// RemoteLevel is the other side's loudness, 0 to 1, for a meter.
	RemoteLevel float64
	// Since is when the call connected.
	Since time.Time
}
