# Terminal

`giode.Terminal()` is a terminal. It runs a shell or any program in a pseudo-terminal, or
shows one the app connects itself, such as an SSH session or a serial port. It draws the
program's screen and takes the keyboard for it. Create it once and lay it out every frame.
It fills the space it's given and resizes the program's terminal to match.

```go
term := giode.Terminal().
	OnTitle(func(title string) { tab.SetTitle(title) }).
	OnExit(func(code int, err error) { closeTab() })

if err := term.Start(nil); err != nil { // nil runs the user's shell
	log.Print(err)
}

app.Run(func() giode.Element { return term })
```

`Start(cmd)` runs an `*exec.Cmd` instead: `exec.Command("htop")`, or a shell with
arguments, environment and a working directory of your choosing. The terminal sets `TERM`
to `xterm-256color`, and sets a UTF-8 locale when the app has none (apps opened from the
desktop often don't).

**Your own connection.** `Attach(w)` sends what's typed to `w`. `Write(p)` shows what the
other end sends. `OnResize` reports the new size to pass on:

```go
term := giode.Terminal().Attach(sshStdin).OnResize(func(cols, rows int) {
	session.WindowChange(rows, cols)
})
go io.Copy(term, sshStdout)
```

## What it supports

- **Emulation**: VT100/VT220 and the xterm extensions programs use. This covers cursor
  movement, erasing, insert and delete, scroll regions, the alternate screen (vim, less,
  htop), 16, 256 and 24-bit colors, bold, italic, underline, strikethrough, inverse, dim,
  and the DEC line-drawing set. Wide CJK characters and emoji take two cells, and combining
  accents join their letter. It answers cursor-position and device-attribute queries, and
  takes the window title (`OnTitle`) and the shell's working directory (OSC 7,
  `Emulator().WorkingDir()`).
- **Box drawing and blocks**: box-drawing lines (light, heavy, double, rounded) and block
  elements are drawn as shapes, so borders join without gaps.
- **Keyboard**: typed text comes in any layout, with dead keys and input methods. The
  candidate window opens at the cursor. Arrows, Home, End, Page Up and Down, F1-F12,
  Ctrl+letters, Alt as Meta, Tab and Shift+Tab, Backspace and Delete send what programs
  expect, including the cursor keys' application mode. Keys it doesn't send, such as Cmd
  shortcuts on macOS, go on to the app's shortcuts. On phones it shows the on-screen
  keyboard.
- **Copy and paste**: copy with Ctrl+Shift+C and paste with Ctrl+Shift+V (Cmd+C and
  Cmd+V on Apple platforms, where Ctrl+C stays an interrupt). Paste is bracketed when the
  program asks for it, so a shell doesn't run pasted lines one at a time. Dragging selects,
  a double-click selects a word and a triple-click a line. `Selection()`, `SelectAll()` and
  `Paste(text)` do the same from code.
- **History**: 10,000 lines. The wheel scrolls through it, as do Shift+Page Up, Shift+Page
  Down, Shift+Home and Shift+End. `ScrollBy`, `ScrollToBottom` and `Clear` do the same from
  code. Typing returns to the live screen. In full-screen programs that don't take the
  mouse, the wheel sends arrow keys.
- **Mouse**: programs that ask for the mouse (htop, `vim` with `mouse=a`, tmux) get
  clicks, drags, moves and the wheel. Shift+drag still selects.

## Options

| Method | |
|---|---|
| `FontSize(sp)` | Text size; 13sp by default. |
| `FontFamily(name)` | A monospace font; Go Mono by default. Characters it lacks come from the fallback fonts. |
| `Padding(dp)` | Space around the cells; 6dp by default. |
| `Palette(giode.TerminalPalette{...})` | Foreground, background, cursor, selection and the 16 ANSI colors, as any colors or `theme.*` tokens. By default `terminal.DefaultDarkPalette` or `DefaultLightPalette`, following the app's mode. |
| `Label(s)` | The name screen readers announce; "Terminal" by default. |
| `OnResize`, `OnTitle`, `OnExit`, `OnBell` | Callbacks, run on the UI goroutine. |
| `Focus()`, `Blur()`, `Focused()` | Keyboard focus; a click focuses it. |
| `Send(b)` | Sends bytes to the program as if typed. |
| `Close()` | Hangs up the program's terminal, as closing a terminal window does. |
| `Emulator()` | The emulator, for reading the screen (`Snapshot`, `Text`) or its modes. |

## Platforms

Programs run in Unix pseudo-terminals on Linux and macOS, and in ConPTY on Windows 10 1809
and later. Android can run programs in the app's own sandbox, such as `/system/bin/sh`. iOS
and the web can't run programs. There `Start` returns `pty.ErrUnsupported`, but a terminal
fed by `Attach` and `Write` (an SSH client) works everywhere.

The emulator (`terminal.NewEmulator`) and the key, paste and mouse encodings
(`terminal.EncodeKey`, `EncodePaste`, `EncodeMouse`) can be used without the element. The
`pty` package runs programs in a pseudo-terminal without one.
