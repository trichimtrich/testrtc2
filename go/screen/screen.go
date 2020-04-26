package screen

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gdamore/tcell"
)

type Screen struct {
	screen   tcell.Screen
	txtLog   *Memo
	txtInput *TextBox
	txtHelp  *Memo
	callback ScreenCB
}

type ScreenCB func(string)

func NewScreen() *Screen {
	return &Screen{nil, nil, nil, nil, nil}
}

func (s *Screen) Log(rline string) {
	// fmt.Println(rline)
	s.txtLog.Println(rline)
}

func (s *Screen) SetTitle(title string) {
	s.txtInput.SetTitle(title)
}

func (s *Screen) SetBuffer(buffer string) {
	s.txtInput.SetBuffer(buffer)
}

func (s *Screen) RegisterCallback(f ScreenCB) {
	s.callback = f
}

func (s *Screen) Clear() {
	s.screen.Clear()
}

func (s *Screen) Show() {
	s.screen.Show()
}

func (s *Screen) Fini() {
	s.screen.Fini()
}

func (s *Screen) Init() {
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err = screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	//// some terminal supports scroll
	//// disable this for mouse selection
	// screen.EnableMouse()
	screen.Clear()

	w, h := screen.Size()
	s.txtLog = NewMemo(screen, 0, 0, w-26, h-2, tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack))
	s.txtInput = NewTextBox(screen, 0, h-2, w, tcell.StyleDefault.Background(tcell.ColorIndianRed).Foreground(tcell.ColorBlue))
	s.txtHelp = NewMemo(screen, w-26, 0, 26, h-2, tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack))

	s.txtHelp.Println("Keyboard shortcut")
	s.txtHelp.Println("  Escape     : Quit")
	s.txtHelp.Println("  Arrow Up   : Scroll Up")
	s.txtHelp.Println("  Arrow Down : Scroll Down")
	s.txtHelp.Println("  Arrow Left : First page")
	s.txtHelp.Println("  Arrow Right: Last page")
	s.txtHelp.Println("")
	s.txtHelp.Println("")
	s.txtHelp.Println("Text command")
	s.txtHelp.Println(" /new    : new rtc")
	s.txtHelp.Println(" /peer id: set peer")
	s.txtHelp.Println(" /offer  : send offer")
	s.txtHelp.Println(" /media  : add media")
	s.txtHelp.Println(" /data   : add channel")

	s.screen = screen
}

func (s *Screen) RenderLoop(quit chan struct{}) {
	ticker := time.NewTicker(time.Second / 20) // 30 fps
	defer ticker.Stop()
	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			// render
			s.txtLog.Render()
			s.txtHelp.Render()
			s.txtInput.Render()
		}
	}
}

func (s *Screen) EventLoop(quit chan struct{}) {
	for {
		ev := s.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				close(quit)
				return

			case tcell.KeyUp:
				s.txtLog.ScrollUp()
			case tcell.KeyDown:
				s.txtLog.ScrollDown()
			case tcell.KeyLeft:
				s.txtLog.ScrollHome()
			case tcell.KeyRight:
				s.txtLog.ScrollEnd()

			case tcell.KeyBackspace, tcell.KeyBackspace2:
				s.txtInput.Backspace()

			case tcell.KeyEnter:
				data := s.txtInput.GetBuffer()
				if s.callback != nil {
					s.callback(data)
				}

			default:
				r := ev.Rune()
				if strconv.IsPrint(r) {
					s.txtInput.Add(string(r))
				}

			}

		case *tcell.EventResize:
			s.Clear()
			w, h := s.screen.Size()
			s.txtLog.Resize(0, 0, w-26, h-2)
			s.txtInput.Resize(0, h-2, w)
			s.txtHelp.Resize(w-26, 0, 26, h-2)
			s.Show()

		case *tcell.EventMouse:
			btn := ev.Buttons()
			if btn&tcell.WheelUp != 0 {
				s.txtLog.ScrollUp()
			}
			if btn&tcell.WheelDown != 0 {
				s.txtLog.ScrollDown()
			}
		}
	}
}
