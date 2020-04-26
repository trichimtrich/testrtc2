package screen

import (
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	runewidth "github.com/mattn/go-runewidth"
)

type TextBox struct {
	screen              tcell.Screen
	x, y, width, height int // no height, only 2 lines
	style               tcell.Style
	title               string
	buffer              string

	snapshotMutex sync.Mutex
	reRender      bool
}

func NewTextBox(s tcell.Screen, x, y, width int, style tcell.Style) *TextBox {
	return &TextBox{s, x, y, width, 2, style, "", "", sync.Mutex{}, true}
}

// relative position
func (tb *TextBox) puts(x, y int, str string, bold bool) {
	s := tb.screen
	style := tb.style.Bold(bold)
	// reposition
	x += tb.x
	y += tb.y
	// overflow
	if len(str) > tb.width {
		str = str[:tb.width]
	}

	i := 0
	var deferred []rune
	dwidth := 0
	zwj := false
	for _, r := range str {
		if r == '\u200d' {
			if len(deferred) == 0 {
				deferred = append(deferred, ' ')
				dwidth = 1
			}
			deferred = append(deferred, r)
			zwj = true
			continue
		}
		if zwj {
			deferred = append(deferred, r)
			zwj = false
			continue
		}
		switch runewidth.RuneWidth(r) {
		case 0:
			if len(deferred) == 0 {
				deferred = append(deferred, ' ')
				dwidth = 1
			}
		case 1:
			if len(deferred) != 0 {
				s.SetContent(x+i, y, deferred[0], deferred[1:], style)
				i += dwidth
			}
			deferred = nil
			dwidth = 1
		case 2:
			if len(deferred) != 0 {
				s.SetContent(x+i, y, deferred[0], deferred[1:], style)
				i += dwidth
			}
			deferred = nil
			dwidth = 2
		}
		deferred = append(deferred, r)
	}
	if len(deferred) != 0 {
		s.SetContent(x+i, y, deferred[0], deferred[1:], style)
		i += dwidth
	}
}

func (tb *TextBox) Render() {
	if !tb.reRender {
		return
	}

	tb.snapshotMutex.Lock()
	title := tb.title
	newBuffer := tb.buffer
	tb.reRender = false
	tb.snapshotMutex.Unlock()

	// clear screen
	for x := 0; x < tb.width; x++ {
		for y := 0; y < tb.height; y++ {
			tb.puts(x, y, " ", false)
		}
	}

	if len(title) > tb.width-20 {
		title = title[:tb.width-20] + "..."
	}
	title = "[ " + title + " ]"
	newTitle := strings.Repeat("-", (tb.width-len(title))/2)
	newTitle += title
	newTitle += strings.Repeat("-", tb.width-len(newTitle))

	if len(newBuffer) > tb.width-1 {
		newBuffer = newBuffer[len(newBuffer)-tb.width+1:]
	}

	tb.puts(0, 0, newTitle, true)
	tb.puts(0, 1, newBuffer, false)

	tb.screen.ShowCursor(tb.x+len(newBuffer), tb.y+1)

	tb.screen.Sync()
}

func (tb *TextBox) SetTitle(str string) {
	tb.snapshotMutex.Lock()
	tb.title = str
	tb.reRender = true
	tb.snapshotMutex.Unlock()
}

func (tb *TextBox) GetBuffer() string {
	return tb.buffer
}

func (tb *TextBox) SetBuffer(str string) {
	tb.snapshotMutex.Lock()
	tb.buffer = str
	tb.reRender = true
	tb.snapshotMutex.Unlock()
}

func (tb *TextBox) Add(str string) {
	tb.SetBuffer(tb.buffer + str)
}

func (tb *TextBox) Backspace() {
	if len(tb.buffer) > 0 {
		tb.SetBuffer(tb.buffer[:len(tb.buffer)-1])
	}
}

func (tb *TextBox) Resize(x, y, width int) {
	tb.x = x
	tb.y = y
	tb.width = width
	tb.reRender = true
}
