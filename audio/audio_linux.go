//go:build linux && !android

package audio

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"

	"github.com/xmhrnx/giode"
)

func appName() string {
	exe, err := os.Executable()
	if err != nil {
		return "Giode"
	}
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
}

func client() (*pulse.Client, error) {
	return pulse.NewClient(pulse.ClientApplicationName(appName()))
}

func devices(ctx context.Context, input bool) ([]Device, error) {
	c, err := client()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	var out []Device
	if input {
		srcs, err := c.ListSources()
		if err != nil {
			return nil, err
		}
		def := ""
		if d, err := c.DefaultSource(); err == nil {
			def = d.ID()
		}
		for _, s := range srcs {
			if strings.HasSuffix(s.ID(), ".monitor") {
				continue // what a speaker plays, not a microphone
			}
			out = append(out, Device{ID: s.ID(), Name: s.Name(), Default: s.ID() == def})
		}
		return out, nil
	}
	sinks, err := c.ListSinks()
	if err != nil {
		return nil, err
	}
	def := ""
	if d, err := c.DefaultSink(); err == nil {
		def = d.ID()
	}
	for _, s := range sinks {
		out = append(out, Device{ID: s.ID(), Name: s.Name(), Default: s.ID() == def})
	}
	return out, nil
}

func requestPermission(context.Context, *giode.App) (bool, error) { return true, nil }

func channelMap(n int) proto.ChannelMap {
	if n == 2 {
		return proto.ChannelMap{proto.ChannelLeft, proto.ChannelRight}
	}
	return proto.ChannelMap{proto.ChannelMono}
}

// voiceProps asks the sound server to treat the stream as a call: its
// echo-cancel filter where it has one.
func voiceProps(p *proto.PropList) {
	if *p == nil {
		*p = proto.PropList{}
	}
	(*p)["media.role"] = proto.PropListString("phone")
	(*p)["filter.want"] = proto.PropListString("echo-cancel")
}

type platformInput struct {
	c *pulse.Client
	s *pulse.RecordStream
}

func (in *Input) open(ctx context.Context, _ *giode.App) error {
	c, err := client()
	if err != nil {
		return err
	}
	opts := []pulse.RecordOption{
		pulse.RecordSampleRate(in.opts.SampleRate),
		pulse.RecordChannels(channelMap(in.opts.Channels)),
		pulse.RecordLatency(in.opts.Frame.Seconds()),
		pulse.RecordMediaName("Microphone"),
	}
	if in.opts.Device != "" {
		src, err := c.SourceByID(in.opts.Device)
		if err != nil {
			c.Close()
			return err
		}
		opts = append(opts, pulse.RecordSource(src))
	}
	if in.opts.Voice {
		opts = append(opts, pulse.RecordRawOption(func(s *proto.CreateRecordStream) { voiceProps(&s.Properties) }))
	}
	w := pulse.Int16Writer(func(buf []int16) (int, error) {
		in.deliver(buf)
		return len(buf), nil
	})
	s, err := c.NewRecord(w, opts...)
	if err != nil {
		c.Close()
		return err
	}
	s.Start()
	in.plat = platformInput{c: c, s: s}
	return nil
}

func (p platformInput) close() error {
	if p.s != nil {
		p.s.Stop()
		p.s.Close()
	}
	if p.c != nil {
		p.c.Close()
	}
	return nil
}

type platformOutput struct {
	c *pulse.Client
	s *pulse.PlaybackStream
}

func (o *Output) open(ctx context.Context, _ *giode.App) error {
	c, err := client()
	if err != nil {
		return err
	}
	opts := []pulse.PlaybackOption{
		pulse.PlaybackSampleRate(o.opts.SampleRate),
		pulse.PlaybackChannels(channelMap(o.opts.Channels)),
		pulse.PlaybackLatency(0.06),
		pulse.PlaybackMediaName("Audio"),
	}
	if o.opts.Device != "" {
		sink, err := c.SinkByID(o.opts.Device)
		if err != nil {
			c.Close()
			return err
		}
		opts = append(opts, pulse.PlaybackSink(sink))
	}
	if o.opts.Voice {
		opts = append(opts, pulse.PlaybackRawOption(func(s *proto.CreatePlaybackStream) { voiceProps(&s.Properties) }))
	}
	r := pulse.Int16Reader(func(buf []int16) (int, error) {
		o.pull(buf)
		return len(buf), nil
	})
	s, err := c.NewPlayback(r, opts...)
	if err != nil {
		c.Close()
		return err
	}
	s.Start()
	o.plat = platformOutput{c: c, s: s}
	return nil
}

func (p platformOutput) close() error {
	if p.s != nil {
		p.s.Stop()
		p.s.Close()
	}
	if p.c != nil {
		p.c.Close()
	}
	return nil
}
