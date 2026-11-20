//go:build js

package rtc

import (
	"context"
	"errors"
	"image"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/video"
)

// ErrUnsupported is returned in the browser, where pion's WebRTC is a
// different API.
var ErrUnsupported = errors.New("rtc: not supported in the browser yet")

type Call struct{}

func New(context.Context, *giode.App, Config) (*Call, error)       { return nil, ErrUnsupported }
func (c *Call) Offer(context.Context) (string, error)              { return "", ErrUnsupported }
func (c *Call) Answer(context.Context, string) (string, error)     { return "", ErrUnsupported }
func (c *Call) Accept(string) error                                { return ErrUnsupported }
func (c *Call) AddCandidate(string) error                          { return ErrUnsupported }
func (c *Call) State() State                                       { return StateClosed }
func (c *Call) Stats() Stats                                       { return Stats{} }
func (c *Call) SetMuted(bool)                                      {}
func (c *Call) Muted() bool                                        { return false }
func (c *Call) RemoteVideo() *video.Surface                        { return video.NewSurface(nil) }
func (c *Call) AudioCodec() string                                 { return "" }
func (c *Call) AudioSampleRate() int                               { return 0 }
func (c *Call) WriteAudio([]int16) error                           { return ErrUnsupported }
func (c *Call) SendVideo(*image.RGBA) error                        { return ErrUnsupported }
func (c *Call) OpenDataChannel(string, bool) (*DataChannel, error) { return nil, ErrUnsupported }
func (c *Call) Close()                                             {}

type DataChannel struct{}

func (d *DataChannel) Label() string                          { return "" }
func (d *DataChannel) Send([]byte) error                      { return ErrUnsupported }
func (d *DataChannel) SendText(string) error                  { return ErrUnsupported }
func (d *DataChannel) OnOpen(func())                          {}
func (d *DataChannel) OnClose(func())                         {}
func (d *DataChannel) OnMessage(func(data []byte, text bool)) {}
func (d *DataChannel) Close() error                           { return nil }
