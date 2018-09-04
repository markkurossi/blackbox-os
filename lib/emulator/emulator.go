//
// emulator.go
//
// Copyright (c) 2018 Markku Rossi
//
// All rights reserved.
//

package emulator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/markkurossi/blackbox-os/kernel/kmsg"
)

type RGBA uint32

const (
	White = RGBA(0xffffffff)
	Black = RGBA(0x000000ff)
)

var (
	blank = Char{
		Code:       0xa0,
		Foreground: Black,
		Background: White,
	}
)

type Char struct {
	Code       int
	Foreground RGBA
	Background RGBA
}

type Action func(e *Emulator, state *State, ch int)

func actError(e *Emulator, state *State, ch int) {
	kmsg.Print(fmt.Sprintf("Emulator error: state=%s, ch=0x%x\n", state, ch))
	e.state = stStart
}

func actInsertChar(e *Emulator, state *State, ch int) {
	e.InsertChar(ch)
}

func actC0Control(e *Emulator, state *State, ch int) {
	switch ch {
	case 0x09: // Horizontal Tabulation.
		var x = e.Col + 1
		for ; x%8 != 0; x++ {
		}
		e.MoveTo(e.Row, x)
	case 0x0a: // Linefeed
		e.MoveTo(e.Row+1, e.Col)
	case 0x0d: // Carriage Return
		e.MoveTo(e.Row, 0)
	case 0x08: // BS
		e.MoveTo(e.Row, e.Col-1)

	default:
		kmsg.Print(fmt.Sprintf("actC0Control: %s: %x\n", state, ch))
	}
}

func actCSIParam(e *Emulator, state *State, ch int) {
	e.params = append(e.params, rune(ch))
}

func actCSI(e *Emulator, state *State, ch int) {
	switch ch {
	case 'C':
		e.MoveTo(e.Row, e.Col+1)
	case 'K':
		// XXX intermediate 0, 1, 2
		e.ClearLine(e.Row, e.Col, e.Width)
	case 'P':
		// XXX intermediate: how many characters to delete
		e.DeleteChars(e.Row, e.Col, 1)
	case 'H': // Cursor position.
		row, col := e.csiParams(1, 1)
		e.MoveTo(row-1, col-1)
	case 'J': // Erase in Display (cursor does not move)
		switch e.csiParam(0) {
		case 0: // Erase from current position to end (inclusive)
			// XXX
			e.Clear()
		case 1: // Erase from beginning ot current position (inclusive)
			// XXX
		case 2: // Erase entire display
			e.Clear()
		}
	default:
		kmsg.Print(fmt.Sprintf("actCSI: %s: 0x%x\n", state, ch))
	}
}

var reParam = regexp.MustCompilePOSIX("^([^0-9;:]*)([0-9;:]*)$")

func (e *Emulator) parseCSIParam(defaults []int) (string, []int) {
	matches := reParam.FindStringSubmatch(string(e.params))
	e.params = nil
	if matches == nil {
		return "", nil
	}
	for idx, param := range strings.Split(matches[2], ";") {
		i, err := strconv.Atoi(param)
		if err != nil {
			if idx < len(defaults) {
				i = defaults[idx]
			}
		}
		if idx < len(defaults) {
			defaults[idx] = i
		} else {
			defaults = append(defaults, i)
		}
	}

	return matches[1], defaults
}

func (e *Emulator) csiParam(a int) int {
	_, values := e.parseCSIParam([]int{a})
	return values[0]
}

func (e *Emulator) csiParams(a, b int) (int, int) {
	_, values := e.parseCSIParam([]int{a, b})
	return values[0], values[1]
}

type Transition struct {
	Action Action
	Next   *State
}

type State struct {
	Name        string
	Default     Action
	Transitions map[int]*Transition
}

func (s *State) String() string {
	return s.Name
}

func (s *State) AddActions(from, to int, act Action, next *State) {
	transition := &Transition{
		Action: act,
		Next:   next,
	}

	for ; from <= to; from++ {
		s.Transitions[from] = transition
	}
}

func (s *State) Input(e *Emulator, code int) *State {
	var act Action
	var next *State

	transition, ok := s.Transitions[code]
	if ok {
		act = transition.Action
		next = transition.Next
	} else {
		act = s.Default
	}

	if act != nil {
		act(e, s, code)
	}

	return next
}

func NewState(name string, def Action) *State {
	return &State{
		Name:        name,
		Default:     def,
		Transitions: make(map[int]*Transition),
	}
}

var (
	stStart  = NewState("start", actInsertChar)
	stESC    = NewState("ESC", actError)
	stCSI    = NewState("CSI", actError)
	stESCSeq = NewState("ESCSeq", actError)
	stOSC    *State
)

func init() {
	stStart.AddActions(0x00, 0x1f, actC0Control, nil)
	stStart.AddActions(0x9b, 0x9b, nil, stCSI)
	stStart.AddActions(0x1b, 0x1b, nil, stESC)

	stESC.AddActions('[', '[', nil, stCSI)

	stCSI.AddActions(0x30, 0x3f, actCSIParam, nil)
	stCSI.AddActions(0x40, 0x7e, actCSI, stStart)
}

type Emulator struct {
	Width  int
	Height int
	Col    int
	Row    int
	Lines  [][]Char
	state  *State
	params []rune
}

func (e *Emulator) Resize(width, height int) {
	e.Width = width
	e.Height = height

	lines := make([][]Char, e.Height)
	for i := 0; i < e.Height; i++ {
		lines[i] = make([]Char, e.Width)
		for j := 0; j < e.Width; j++ {
			lines[i][j] = blank
		}
	}

	e.Lines = lines
}

func (e *Emulator) ClearLine(line, from, to int) {
	if line < 0 || line >= e.Height {
		return
	}
	for i := from; i < to; i++ {
		e.Lines[line][i] = blank
	}
}

func (e *Emulator) Clear() {
	for i := 0; i < e.Height; i++ {
		e.ClearLine(i, 0, e.Width)
	}
}

func (e *Emulator) MoveTo(row, col int) {
	if col < 0 {
		col = 0
	}
	if col > e.Width {
		col = e.Width
	}
	e.Col = col

	if row < 0 {
		row = 0
	}
	if row >= e.Height {
		e.ScrollUp(e.Height - row + 1)
		row = e.Height - 1
	}
	e.Row = row
}

func (e *Emulator) ScrollUp(count int) {
	if count >= e.Height {
		e.Clear()
		return
	}

	for i := 0; i < count; i++ {
		saved := e.Lines[0]
		e.Lines = append(e.Lines[1:], saved)
	}
	for i := 0; i < count; i++ {
		e.ClearLine(e.Height-1-i, 0, e.Width)
	}
}

func (e *Emulator) InsertChar(code int) {
	if e.Col >= e.Width {
		if e.Row+1 >= e.Height {
			e.ScrollUp(1)
			e.MoveTo(e.Row, 0)
		} else {
			e.MoveTo(e.Row+1, 0)
		}
	}
	e.Lines[e.Row][e.Col] = Char{
		Code:       code,
		Foreground: Black,
		Background: White,
	}
	e.MoveTo(e.Row, e.Col+1)
}

func (e *Emulator) DeleteChars(row, col, count int) {
	r := e.Lines[e.Row]

	for x := col; x < e.Width; x++ {
		if x+count < e.Width {
			r[x] = r[x+count]
		} else {
			r[x] = blank
		}
	}
}

func (e *Emulator) Input(code int) {
	// kmsg.Print(fmt.Sprintf("Emulator.Input: %s<-0x%x", e.state, code))
	next := e.state.Input(e, code)
	if next != nil {
		// kmsg.Print(fmt.Sprintf("Emulator: %s->%s", e.state, next))
		e.state = next
	}
}

func NewEmulator() *Emulator {
	return &Emulator{
		state: stStart,
	}
}
