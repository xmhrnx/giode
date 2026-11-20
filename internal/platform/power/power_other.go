//go:build !(linux && !android) && !windows && !(darwin && !ios)

package power

import "github.com/xmhrnx/giode/internal/platform"

func Watch(h platform.Host, done <-chan struct{}, event func(suspend bool), hold bool) {}
