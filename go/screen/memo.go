package screen

import (
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	runewidth "github.com/mattn/go-runewidth"
)

type Row struct {
	id     int
	start  int
	length int
}

type Memo struct {
	screen              tcell.Screen
	x, y, width, height int
	style               tcell.Style
	rawBuf              []string // actual data buffer
	rowArr              []Row    // screen buffer
	rowCur              int
	autoScroll          bool

	snapshotMutex sync.Mutex
	reRender      bool
}

func NewMemo(s tcell.Screen, x, y, width, height int, style tcell.Style) *Memo {
	return &Memo{s, x, y, width, height, style, nil, nil, 0, true, sync.Mutex{}, true}
}

// relative position
func (m *Memo) puts(x, y int, str string) {
	s := m.screen
	style := m.style
	// reposition
	x += m.x
	y += m.y
	// overflow
	if len(str) > m.width {
		str = str[:m.width]
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

func (m *Memo) addLineToRow(line string, bufIdx int) (newRow int, numRows int) {
	newRow = len(m.rowArr)
	numRows = 0
	lineIdx := 0
	lineLen := len(line)
	for {
		numRows++
		if lineIdx+m.width >= lineLen {
			m.rowArr = append(m.rowArr, Row{bufIdx, lineIdx, lineLen - lineIdx})
			break
		} else {
			m.rowArr = append(m.rowArr, Row{bufIdx, lineIdx, m.width})
			lineIdx += m.width
		}
	}

	return
}

func (m *Memo) Clear() {
	m.snapshotMutex.Lock()
	m.rawBuf = nil
	m.rowArr = nil
	m.rowCur = 0
	m.reRender = true
	m.snapshotMutex.Unlock()
}

func (m *Memo) Resize(x, y, width, height int) {
	m.x = x
	m.y = y
	m.width = width
	m.height = height
	// TODO: reparse
	m.snapshotMutex.Lock()
	m.rowArr = nil
	m.snapshotMutex.Unlock()
	for idx := 0; idx < len(m.rawBuf); idx++ {
		m.addLineToRow(m.rawBuf[idx], idx)
	}
	m.ScrollEnd()
}

func (m *Memo) Render() {
	if !m.reRender {
		return
	}

	m.snapshotMutex.Lock()
	rowArr := m.rowArr
	rawBuf := m.rawBuf
	rowIdx := m.rowCur
	m.reRender = false
	m.snapshotMutex.Unlock()

	// clear screen
	for x := 0; x < m.width; x++ {
		for y := 0; y < m.height; y++ {
			m.puts(x, y, " ")
		}
	}

	// print now
	for y := 0; y < Min(m.height, len(rowArr)-rowIdx); y++ {
		r := rowArr[y+rowIdx]
		str := rawBuf[r.id][r.start : r.start+r.length]
		m.puts(0, y, str)
	}

	m.screen.Sync()
}

func (m *Memo) setCursor(rowIdx int) {
	m.snapshotMutex.Lock()
	m.rowCur = rowIdx
	m.reRender = true
	m.snapshotMutex.Unlock()
}

func (m *Memo) Println(rline string) {
	for _, line := range strings.Split(rline, "\r\n") {
		bufIdx := len(m.rawBuf)
		m.rawBuf = append(m.rawBuf, line)
		m.addLineToRow(line, bufIdx)
	}

	if m.autoScroll {
		m.setCursor(Max(0, len(m.rowArr)-m.height))
	} else {
		m.setCursor(m.rowCur)
	}
}

func (m *Memo) ScrollHome() {
	m.autoScroll = false
	m.setCursor(0)
}

func (m *Memo) ScrollEnd() {
	m.autoScroll = true
	m.setCursor(Max(0, len(m.rowArr)-m.height))
}

func (m *Memo) ScrollUp() {
	if m.rowCur == 0 {
		return
	}
	m.autoScroll = false
	m.setCursor(m.rowCur - 1)
}

func (m *Memo) ScrollDown() {
	if m.rowCur >= len(m.rowArr)-m.height {
		m.autoScroll = true
		return
	}
	m.setCursor(m.rowCur + 1)
}
