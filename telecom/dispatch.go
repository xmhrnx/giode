package telecom

// dispatch hands a platform event to the handler.
func dispatch(kind, a, b string) {
	switch kind {
	case "answer":
		deliver(func(h Handler) { call(h.OnAnswer, a) })
	case "end":
		deliver(func(h Handler) { call(h.OnEnd, a) })
	case "start":
		deliver(func(h Handler) {
			if h.OnStart != nil {
				h.OnStart(Call{ID: a, Handle: b, Name: b})
			}
		})
	case "mute", "unmute":
		deliver(func(h Handler) {
			if h.OnMute != nil {
				h.OnMute(a, kind == "mute")
			}
		})
	case "hold", "unhold":
		deliver(func(h Handler) {
			if h.OnHold != nil {
				h.OnHold(a, kind == "hold")
			}
		})
	case "audio_on", "audio_off":
		deliver(func(h Handler) {
			if h.OnAudio != nil {
				h.OnAudio(kind == "audio_on")
			}
		})
	case "voip_token":
		deliver(func(h Handler) { call(h.OnVoIPToken, a) })
	}
}

func call(fn func(string), arg string) {
	if fn != nil {
		fn(arg)
	}
}
