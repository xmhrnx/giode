//go:build !js

package rtc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"strings"
	"sync"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	pmedia "github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/audio"
	"github.com/xmhrnx/giode/video"
)

const frameDur = 20 * time.Millisecond

// Call is one peer connection.
type Call struct {
	app *giode.App
	cfg Config
	pc  *webrtc.PeerConnection

	audioTrack *webrtc.TrackLocalStaticSample
	videoTrack *webrtc.TrackLocalStaticSample
	remote     *video.Surface

	mu         sync.Mutex
	state      State
	muted      bool
	pending    []webrtc.ICECandidateInit
	haveRemote bool
	enc        AudioEncoder
	encCodec   AudioCodec
	partial    []int16
	vEnc       VideoEncoder
	vEncSize   image.Point
	keyframe   bool
	mic        *audio.Input
	speaker    *audio.Output
	stats      Stats
	closed     bool
	stop       chan struct{}
}

// New makes a call. Then one side calls Offer and the other Answer.
func New(ctx context.Context, app *giode.App, cfg Config) (*Call, error) {
	if cfg.VideoFPS <= 0 {
		cfg.VideoFPS = 30
	}
	m := &webrtc.MediaEngine{}
	audioCodecs := append(append([]AudioCodec(nil), cfg.AudioCodecs...), PCMU, PCMA)
	cfg.AudioCodecs = audioCodecs
	for _, c := range audioCodecs {
		if err := m.RegisterCodec(webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: c.MimeType(), ClockRate: uint32(c.ClockRate()), Channels: uint16(c.Channels()), SDPFmtpLine: c.Fmtp()},
			PayloadType:        webrtc.PayloadType(c.PayloadType()),
		}, webrtc.RTPCodecTypeAudio); err != nil {
			return nil, err
		}
	}
	for _, c := range cfg.VideoCodecs {
		if depacketizer(c.MimeType()) == nil {
			return nil, fmt.Errorf("rtc: no packetization for %s", c.MimeType())
		}
		if err := m.RegisterCodec(webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: c.MimeType(), ClockRate: 90000, SDPFmtpLine: c.Fmtp(),
				RTCPFeedback: []webrtc.RTCPFeedback{{Type: "nack"}, {Type: "nack", Parameter: "pli"}, {Type: "ccm", Parameter: "fir"}}},
			PayloadType: webrtc.PayloadType(c.PayloadType()),
		}, webrtc.RTPCodecTypeVideo); err != nil {
			return nil, err
		}
	}
	reg := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, reg); err != nil {
		return nil, err
	}
	se := webrtc.SettingEngine{}
	if cfg.loopback {
		se.SetIncludeLoopbackCandidate(true)
		se.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
		se.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(reg), webrtc.WithSettingEngine(se))
	pcfg := webrtc.Configuration{}
	for _, s := range cfg.ICEServers {
		pcfg.ICEServers = append(pcfg.ICEServers, webrtc.ICEServer{URLs: s.URLs, Username: s.Username, Credential: s.Credential})
	}
	if cfg.RelayOnly {
		pcfg.ICETransportPolicy = webrtc.ICETransportPolicyRelay
	}
	pc, err := api.NewPeerConnection(pcfg)
	if err != nil {
		return nil, err
	}
	c := &Call{app: app, cfg: cfg, pc: pc, remote: video.NewSurface(app), stop: make(chan struct{})}

	sendAudio := cfg.Microphone || cfg.SendAudio
	recvAudio := cfg.Speaker || cfg.OnRemoteAudio != nil
	if sendAudio {
		first := audioCodecs[0]
		c.audioTrack, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: first.MimeType(), ClockRate: uint32(first.ClockRate()), Channels: uint16(first.Channels())}, "audio", "giode")
		if err != nil {
			pc.Close()
			return nil, err
		}
		dir := webrtc.RTPTransceiverDirectionSendrecv
		if !recvAudio {
			dir = webrtc.RTPTransceiverDirectionSendonly
		}
		tr, err := pc.AddTransceiverFromTrack(c.audioTrack, webrtc.RTPTransceiverInit{Direction: dir})
		if err != nil {
			pc.Close()
			return nil, err
		}
		go drainRTCP(tr.Sender(), nil)
	} else if recvAudio {
		if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
			pc.Close()
			return nil, err
		}
	}
	if len(cfg.VideoCodecs) > 0 {
		if cfg.SendVideo {
			first := cfg.VideoCodecs[0]
			c.videoTrack, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: first.MimeType(), ClockRate: 90000}, "video", "giode")
			if err != nil {
				pc.Close()
				return nil, err
			}
			sender, err := pc.AddTrack(c.videoTrack)
			if err != nil {
				pc.Close()
				return nil, err
			}
			go drainRTCP(sender, func() {
				c.mu.Lock()
				c.keyframe = true
				c.mu.Unlock()
			})
		} else {
			if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
				pc.Close()
				return nil, err
			}
		}
	}

	pc.OnICECandidate(func(ic *webrtc.ICECandidate) {
		if ic == nil || cfg.OnCandidate == nil {
			return
		}
		b, _ := json.Marshal(ic.ToJSON())
		cfg.OnCandidate(string(b))
	})
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		st := map[webrtc.PeerConnectionState]State{
			webrtc.PeerConnectionStateNew: StateNew, webrtc.PeerConnectionStateConnecting: StateConnecting,
			webrtc.PeerConnectionStateConnected: StateConnected, webrtc.PeerConnectionStateDisconnected: StateDisconnected,
			webrtc.PeerConnectionStateFailed: StateFailed, webrtc.PeerConnectionStateClosed: StateClosed,
		}[s]
		c.mu.Lock()
		c.state = st
		if st == StateConnected && c.stats.Since.IsZero() {
			c.stats.Since = time.Now()
		}
		c.mu.Unlock()
		if st == StateConnected && cfg.Microphone {
			go c.openMicrophone()
		}
		if cfg.OnState != nil {
			cfg.OnState(st)
		}
		c.invalidate()
	})
	pc.OnTrack(func(t *webrtc.TrackRemote, r *webrtc.RTPReceiver) {
		go drainRTCP(r, nil)
		if t.Kind() == webrtc.RTPCodecTypeAudio {
			go c.receiveAudio(t)
		} else {
			go c.receiveVideo(t)
		}
	})
	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		if cfg.OnDataChannel != nil {
			cfg.OnDataChannel(&DataChannel{dc: d})
		}
	})
	return c, nil
}

// rtcpReader is a sender's or receiver's RTCP.
type rtcpReader interface {
	ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error)
}

// drainRTCP reads RTCP, which the interceptors need, calling onPLI on
// key frame requests.
func drainRTCP(r rtcpReader, onPLI func()) {
	for {
		pkts, _, err := r.ReadRTCP()
		if err != nil {
			return
		}
		if onPLI == nil {
			continue
		}
		for _, p := range pkts {
			switch p.(type) {
			case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
				onPLI()
			}
		}
	}
}

// Offer makes the offer to send to the other side.
func (c *Call) Offer(ctx context.Context) (string, error) {
	o, err := c.pc.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	return c.setLocal(ctx, o)
}

// Answer takes the other side's offer and makes the answer to send
// back.
func (c *Call) Answer(ctx context.Context, offer string) (string, error) {
	if err := c.setRemote(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: offer}); err != nil {
		return "", err
	}
	a, err := c.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	return c.setLocal(ctx, a)
}

// Accept takes the other side's answer to this side's offer.
func (c *Call) Accept(answer string) error {
	return c.setRemote(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answer})
}

func (c *Call) setLocal(ctx context.Context, d webrtc.SessionDescription) (string, error) {
	var gathered <-chan struct{}
	if c.cfg.OnCandidate == nil {
		gathered = webrtc.GatheringCompletePromise(c.pc)
	}
	if err := c.pc.SetLocalDescription(d); err != nil {
		return "", err
	}
	if gathered != nil {
		select {
		case <-gathered:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return c.pc.LocalDescription().SDP, nil
}

func (c *Call) setRemote(d webrtc.SessionDescription) error {
	if err := c.pc.SetRemoteDescription(d); err != nil {
		return err
	}
	c.mu.Lock()
	c.haveRemote = true
	pending := c.pending
	c.pending = nil
	c.mu.Unlock()
	for _, p := range pending {
		if err := c.pc.AddICECandidate(p); err != nil {
			return err
		}
	}
	return nil
}

// AddCandidate adds one of the other side's ICE candidates, as its
// OnCandidate gave it.
func (c *Call) AddCandidate(candidate string) error {
	var init webrtc.ICECandidateInit
	if strings.HasPrefix(strings.TrimSpace(candidate), "{") {
		if err := json.Unmarshal([]byte(candidate), &init); err != nil {
			return err
		}
	} else {
		init.Candidate = candidate
	}
	c.mu.Lock()
	if !c.haveRemote {
		c.pending = append(c.pending, init)
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	return c.pc.AddICECandidate(init)
}

// State is the connection's state.
func (c *Call) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Stats are the call's counters.
func (c *Call) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}

// SetMuted stops or resumes sending audio; silence goes out instead.
func (c *Call) SetMuted(m bool) {
	c.mu.Lock()
	c.muted = m
	c.mu.Unlock()
	c.invalidate()
}

// Muted reports whether audio is muted.
func (c *Call) Muted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.muted
}

// RemoteVideo is where the other side's video shows.
func (c *Call) RemoteVideo() *video.Surface { return c.remote }

// AudioCodec is the codec audio is sent with, once negotiated.
func (c *Call) AudioCodec() string {
	if c.audioTrack == nil {
		return ""
	}
	for _, t := range c.pc.GetTransceivers() {
		if t.Sender() != nil && t.Sender().Track() == c.audioTrack {
			if p := t.Sender().GetParameters(); len(p.Codecs) > 0 {
				return p.Codecs[0].MimeType
			}
		}
	}
	return ""
}

func (c *Call) codecFor(mime string) AudioCodec {
	for _, a := range c.cfg.AudioCodecs {
		if strings.EqualFold(a.MimeType(), mime) {
			return a
		}
	}
	return nil
}

// WriteAudio sends PCM at the negotiated codec's sample rate (see
// AudioSampleRate), from a source other than the microphone.
func (c *Call) WriteAudio(pcm []int16) error {
	if c.audioTrack == nil {
		return errors.New("rtc: the call doesn't send audio")
	}
	codec := c.codecFor(c.AudioCodec())
	if codec == nil {
		return errors.New("rtc: audio isn't negotiated yet")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.encCodec != codec {
		if c.enc != nil {
			c.enc.Close()
		}
		enc, err := codec.NewEncoder()
		if err != nil {
			return err
		}
		c.enc, c.encCodec, c.partial = enc, codec, nil
	}
	n := codec.SampleRate() * codec.Channels() / 50 // 20 ms
	c.partial = append(c.partial, pcm...)
	for len(c.partial) >= n {
		frame := c.partial[:n]
		if c.muted {
			frame = make([]int16, n)
		}
		b, err := c.enc.Encode(frame)
		c.partial = c.partial[n:]
		if err != nil {
			return err
		}
		if err := c.audioTrack.WriteSample(pmedia.Sample{Data: b, Duration: frameDur}); err != nil {
			return err
		}
		c.stats.AudioSent++
	}
	if len(c.partial) == 0 {
		c.partial = nil
	}
	return nil
}

// AudioSampleRate is the rate WriteAudio takes, once negotiated.
func (c *Call) AudioSampleRate() int {
	if a := c.codecFor(c.AudioCodec()); a != nil {
		return a.SampleRate()
	}
	return 0
}

func (c *Call) openMicrophone() {
	c.mu.Lock()
	if c.mic != nil || c.closed {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	rate := c.AudioSampleRate()
	if rate == 0 {
		return
	}
	in, err := audio.OpenInput(context.Background(), c.app, audio.Options{
		Device: c.cfg.MicrophoneDevice, SampleRate: rate, Voice: true, Frame: frameDur,
	}, func(pcm []int16) { c.WriteAudio(pcm) })
	if err != nil {
		return
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		in.Close()
		return
	}
	c.mic = in
	c.mu.Unlock()
}

func (c *Call) receiveAudio(t *webrtc.TrackRemote) {
	codec := c.codecFor(t.Codec().MimeType)
	if codec == nil {
		return
	}
	dec, err := codec.NewDecoder()
	if err != nil {
		return
	}
	defer dec.Close()
	var out *audio.Output
	if c.cfg.Speaker {
		out, err = audio.OpenOutput(context.Background(), c.app, audio.Options{
			SampleRate: codec.SampleRate(), Channels: codec.Channels(), Voice: true, SpeakerPhone: c.cfg.SpeakerPhone,
		})
		if err == nil {
			c.mu.Lock()
			c.speaker = out
			c.mu.Unlock()
		}
	}
	var last uint16
	started := false
	for {
		pkt, _, err := t.ReadRTP()
		if err != nil {
			return
		}
		// Drop late and repeated packets.
		if started && int16(pkt.SequenceNumber-last) <= 0 {
			continue
		}
		started, last = true, pkt.SequenceNumber
		pcm, err := dec.Decode(pkt.Payload)
		if err != nil || len(pcm) == 0 {
			continue
		}
		c.mu.Lock()
		c.stats.AudioReceived++
		c.stats.RemoteLevel = audio.Level(pcm)
		c.mu.Unlock()
		if c.cfg.OnRemoteAudio != nil {
			c.cfg.OnRemoteAudio(pcm, codec.SampleRate())
		}
		if out != nil {
			// Keep latency down: skip a frame when too much is queued.
			if out.Buffered() < 150*time.Millisecond {
				out.Write(context.Background(), pcm)
			}
		}
	}
}

// depacketizer joins a video codec's RTP packets into frames.
func depacketizer(mime string) rtp.Depacketizer {
	switch strings.ToLower(mime) {
	case "video/vp8":
		return &codecs.VP8Packet{}
	case "video/vp9":
		return &codecs.VP9Packet{}
	case "video/h264":
		return &codecs.H264Packet{}
	case "video/av1":
		return &codecs.AV1Depacketizer{}
	}
	return nil
}

func (c *Call) receiveVideo(t *webrtc.TrackRemote) {
	var codec VideoCodec
	for _, v := range c.cfg.VideoCodecs {
		if strings.EqualFold(v.MimeType(), t.Codec().MimeType) {
			codec = v
		}
	}
	if codec == nil {
		return
	}
	dec, err := codec.NewDecoder()
	if err != nil {
		return
	}
	defer dec.Close()
	sb := samplebuilder.New(128, depacketizer(codec.MimeType()), 90000)
	lastPLI := time.Time{}
	pli := func() {
		if time.Since(lastPLI) > time.Second {
			lastPLI = time.Now()
			c.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(t.SSRC())}})
		}
	}
	pli()
	for {
		pkt, _, err := t.ReadRTP()
		if err != nil {
			return
		}
		sb.Push(pkt)
		for s := sb.Pop(); s != nil; s = sb.Pop() {
			if s.PrevDroppedPackets > 0 {
				pli()
			}
			img, err := dec.Decode(s.Data)
			if err != nil || img == nil {
				pli()
				continue
			}
			c.remote.Push(img)
			c.mu.Lock()
			c.stats.VideoReceived++
			c.mu.Unlock()
		}
	}
}

// SendVideo sends a frame, timed at Config.VideoFPS.
func (c *Call) SendVideo(frame *image.RGBA) error {
	if c.videoTrack == nil {
		return errors.New("rtc: the call doesn't send video")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	size := frame.Rect.Size()
	if c.vEnc == nil || c.vEncSize != size {
		if c.vEnc != nil {
			c.vEnc.Close()
		}
		enc, err := c.cfg.VideoCodecs[0].NewEncoder(size.X, size.Y, c.cfg.VideoFPS)
		if err != nil {
			return err
		}
		c.vEnc, c.vEncSize, c.keyframe = enc, size, true
	}
	key := c.keyframe
	c.keyframe = false
	b, err := c.vEnc.Encode(frame, key)
	if err != nil || len(b) == 0 {
		return err
	}
	c.stats.VideoSent++
	return c.videoTrack.WriteSample(pmedia.Sample{Data: b, Duration: time.Second / time.Duration(c.cfg.VideoFPS)})
}

// OpenDataChannel opens a data channel; ordered delivers messages in
// order, retransmitting lost ones.
func (c *Call) OpenDataChannel(label string, ordered bool) (*DataChannel, error) {
	d, err := c.pc.CreateDataChannel(label, &webrtc.DataChannelInit{Ordered: &ordered})
	if err != nil {
		return nil, err
	}
	return &DataChannel{dc: d}, nil
}

// Close hangs up.
func (c *Call) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	mic, spk, enc, venc := c.mic, c.speaker, c.enc, c.vEnc
	c.mu.Unlock()
	close(c.stop)
	if mic != nil {
		mic.Close()
	}
	c.pc.Close()
	if spk != nil {
		spk.Close()
	}
	if enc != nil {
		enc.Close()
	}
	if venc != nil {
		venc.Close()
	}
}

func (c *Call) invalidate() {
	if c.app != nil {
		c.app.Invalidate()
	}
}

// DataChannel carries messages between the peers.
type DataChannel struct{ dc *webrtc.DataChannel }

// Label is the name it was opened with.
func (d *DataChannel) Label() string { return d.dc.Label() }

// Send sends binary data.
func (d *DataChannel) Send(b []byte) error { return d.dc.Send(b) }

// SendText sends a text message.
func (d *DataChannel) SendText(s string) error { return d.dc.SendText(s) }

// OnOpen is called when it can send.
func (d *DataChannel) OnOpen(fn func()) { d.dc.OnOpen(fn) }

// OnClose is called when it closes.
func (d *DataChannel) OnClose(fn func()) { d.dc.OnClose(fn) }

// OnMessage is called with each message; text tells text from binary.
func (d *DataChannel) OnMessage(fn func(data []byte, text bool)) {
	d.dc.OnMessage(func(m webrtc.DataChannelMessage) { fn(m.Data, m.IsString) })
}

// Close closes it.
func (d *DataChannel) Close() error { return d.dc.Close() }
