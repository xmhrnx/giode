package systray

import (
	"bytes"
	"encoding/binary"
	"image/png"

	"fyne.io/systray"
)

// setIcon sets the tray icon from PNG bytes, wrapped in an ICO
// container: the Windows tray loads .ico files.
func setIcon(icon []byte, _ bool) {
	if ico := pngToICO(icon); ico != nil {
		systray.SetIcon(ico)
	}
}

// pngToICO wraps PNG bytes in a single-image ICO file, which Windows
// Vista and later read.
func pngToICO(p []byte) []byte {
	cfg, err := png.DecodeConfig(bytes.NewReader(p))
	if err != nil || cfg.Width > 256 || cfg.Height > 256 {
		return nil
	}
	dim := func(v int) byte {
		if v == 256 {
			return 0
		}
		return byte(v)
	}
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, [3]uint16{0, 1, 1})
	b.Write([]byte{dim(cfg.Width), dim(cfg.Height), 0, 0})
	binary.Write(&b, binary.LittleEndian, [2]uint16{1, 32})
	binary.Write(&b, binary.LittleEndian, [2]uint32{uint32(len(p)), 22})
	b.Write(p)
	return b.Bytes()
}
