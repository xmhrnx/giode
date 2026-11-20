//go:build darwin

package audio

/*
#cgo CFLAGS: -fmodules -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AudioToolbox -framework AVFoundation -framework Foundation
#cgo !ios LDFLAGS: -framework CoreAudio

#include <stdlib.h>
#include "audio_darwin.h"
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/xmhrnx/giode"
)

func requestPermission(context.Context, *giode.App) (bool, error) {
	return C.giodeAudioPermission() == 1, nil
}

func devices(ctx context.Context, input bool) ([]Device, error) {
	in := C.int(0)
	if input {
		in = 1
	}
	cs := C.giodeAudioDevices(in)
	defer C.free(unsafe.Pointer(cs))
	var out []Device
	for _, line := range strings.Split(C.GoString(cs), "\n") {
		f := strings.Split(line, "\t")
		if len(f) == 3 {
			out = append(out, Device{ID: f[0], Name: f[1], Default: f[2] == "1"})
		}
	}
	return out, nil
}

// unit is an open audio unit.
type unit struct {
	ref unsafe.Pointer
}

func openUnit(opts Options, capture, render bool) (*unit, error) {
	b := func(v bool) C.int {
		if v {
			return 1
		}
		return 0
	}
	if capture && opts.Device != "" {
		dev := C.CString(opts.Device)
		C.giodeAudioPreferInput(dev)
		C.free(unsafe.Pointer(dev))
	}
	dev := C.CString(opts.Device)
	defer C.free(unsafe.Pointer(dev))
	var cerr *C.char
	ref := C.giodeAudioOpen(b(capture), b(render), b(opts.Voice), b(opts.SpeakerPhone), C.double(opts.SampleRate), C.int(opts.Channels), dev, &cerr)
	if ref == nil {
		msg := "audio: couldn't open the audio unit"
		if cerr != nil {
			msg = C.GoString(cerr)
			C.free(unsafe.Pointer(cerr))
		}
		return nil, errors.New(msg)
	}
	return &unit{ref: ref}, nil
}

// The voice unit is shared: the platform cancels the echo of what it
// plays from what it captures, so a call's input and output must be
// one unit.
var voice struct {
	sync.Mutex
	u          *unit
	users      int
	input      *Input
	output     *Output
	stop, done chan struct{}
}

func voiceAttach(opts Options, in *Input, out *Output) error {
	voice.Lock()
	defer voice.Unlock()
	if voice.u == nil {
		u, err := openUnit(opts, true, true)
		if err != nil {
			return err
		}
		voice.u = u
		voice.stop, voice.done = make(chan struct{}), make(chan struct{})
		go pumpVoice(u, voice.stop, voice.done, opts)
	}
	voice.users++
	if in != nil {
		voice.input = in
	}
	if out != nil {
		voice.output = out
	}
	return nil
}

func voiceDetach(in *Input, out *Output) {
	voice.Lock()
	if in != nil && voice.input == in {
		voice.input = nil
	}
	if out != nil && voice.output == out {
		voice.output = nil
	}
	voice.users--
	var u *unit
	var stop, done chan struct{}
	if voice.users <= 0 {
		u, stop, done = voice.u, voice.stop, voice.done
		voice.u, voice.users = nil, 0
	}
	voice.Unlock()
	if u != nil {
		close(stop)
		<-done
		C.giodeAudioClose(u.ref)
	}
}

// pumpVoice moves samples between the voice unit and whichever input
// and output use it.
func pumpVoice(u *unit, stop, done chan struct{}, opts Options) {
	defer close(done)
	buf := make([]int16, opts.SampleRate/50*opts.Channels)
	play := make([]int16, len(buf))
	for {
		select {
		case <-stop:
			return
		default:
		}
		voice.Lock()
		in, out := voice.input, voice.output
		voice.Unlock()
		if out != nil {
			n := out.ring.read(play)
			for k := 0; k < n; {
				w := int(C.giodeAudioWrite(u.ref, (*C.int16_t)(unsafe.Pointer(&play[k])), C.int(n-k)))
				k += w
				if w == 0 {
					time.Sleep(5 * time.Millisecond)
				}
			}
		}
		n := int(C.giodeAudioRead(u.ref, (*C.int16_t)(unsafe.Pointer(&buf[0])), C.int(len(buf)), 10))
		if n < 0 {
			return
		}
		if n > 0 && in != nil {
			in.deliver(append([]int16(nil), buf[:n]...))
		}
	}
}

type platformInput struct {
	in   *Input
	u    *unit
	stop chan struct{}
	done chan struct{}
}

func (in *Input) open(ctx context.Context, _ *giode.App) error {
	if in.opts.Voice {
		if err := voiceAttach(in.opts, in, nil); err != nil {
			return err
		}
		in.plat = platformInput{in: in}
		return nil
	}
	u, err := openUnit(in.opts, true, false)
	if err != nil {
		return err
	}
	p := platformInput{in: in, u: u, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		buf := make([]int16, in.opts.SampleRate/50*in.opts.Channels)
		for {
			select {
			case <-p.stop:
				return
			default:
			}
			n := int(C.giodeAudioRead(u.ref, (*C.int16_t)(unsafe.Pointer(&buf[0])), C.int(len(buf)), 50))
			if n < 0 {
				return
			}
			if n > 0 {
				in.deliver(append([]int16(nil), buf[:n]...))
			}
		}
	}()
	in.plat = p
	return nil
}

func (p platformInput) close() error {
	if p.u == nil {
		voiceDetach(p.in, nil)
		return nil
	}
	close(p.stop)
	<-p.done
	C.giodeAudioClose(p.u.ref)
	return nil
}

type platformOutput struct {
	out  *Output
	u    *unit
	stop chan struct{}
	done chan struct{}
}

func (o *Output) open(ctx context.Context, _ *giode.App) error {
	if o.opts.Voice {
		if err := voiceAttach(o.opts, nil, o); err != nil {
			return err
		}
		o.plat = platformOutput{out: o}
		return nil
	}
	u, err := openUnit(o.opts, false, true)
	if err != nil {
		return err
	}
	p := platformOutput{out: o, u: u, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		buf := make([]int16, o.opts.SampleRate/100*o.opts.Channels)
		pending := 0
		for {
			select {
			case <-p.stop:
				return
			default:
			}
			if pending == 0 {
				pending = o.ring.read(buf)
			}
			if pending == 0 {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			w := int(C.giodeAudioWrite(u.ref, (*C.int16_t)(unsafe.Pointer(&buf[0])), C.int(pending)))
			copy(buf, buf[w:pending])
			pending -= w
			if w == 0 {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()
	o.plat = p
	return nil
}

func (p platformOutput) close() error {
	if p.u == nil {
		voiceDetach(nil, p.out)
		return nil
	}
	close(p.stop)
	<-p.done
	C.giodeAudioClose(p.u.ref)
	return nil
}
