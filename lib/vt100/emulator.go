//
// emulator.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type RGBA uint32

const (
	White = RGBA(0xffffffff)
	Black = RGBA(0x000000ff)
	debug = false
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
	e.Debug("Emulator error: state=%s, ch=0x%x,%d,%s\n", state, ch, ch,
		string(ch))
	e.SetState(stStart)
}

func actInsertChar(e *Emulator, state *State, ch int) {
	e.InsertChar(ch)
}

func actInsertSpace(e *Emulator, state *State, ch int) {
	e.InsertChar(' ')
}

func actC0Control(e *Emulator, state *State, ch int) {
	switch ch {
	case 0x08: // BS
		e.MoveTo(e.Row, e.Col-1)
	case 0x09: // Horizontal Tabulation.
		var x = e.Col + 1
		for ; x%8 != 0; x++ {
		}
		e.MoveTo(e.Row, x)
	case 0x0a: // Linefeed
		e.MoveTo(e.Row+1, e.Col)
	case 0x0d: // Carriage Return
		e.MoveTo(e.Row, 0)

	default:
		e.Debug("actC0Control: %s: %s0x%x\n", state, string(state.params), ch)
	}
}

func actC1Control(e *Emulator, state *State, ch int) {
	switch 0x40 + ch {
	case 0x84: // Index, moves down one line same column regardless of NL
		e.MoveTo(e.Row+1, e.Col)
	case 0x85: // NEw Line, moves done one line and to first column (CR+LF)
		e.MoveTo(e.Row+1, 0)
	case 0x8d: // Reverse Index, go up one line, reverse scroll if necessary
		e.MoveTo(e.Row-1, e.Col)
	default:
		e.Debug("actC1Control: %s: %s0x%x\n", state, string(state.params), ch)
	}
}

func actAppendParam(e *Emulator, state *State, ch int) {
	state.params = append(state.params, rune(ch))
}

func actPrivateFunction(e *Emulator, state *State, ch int) {
	switch ch {
	case '8':
		switch string(state.params) {
		case "#": // DECALN - Alignment display, fill screen with "E"
			for row := 0; row < e.Height; row++ {
				for col := 0; col < e.Width; col++ {
					e.Lines[row][col] = Char{
						Code:       'E',
						Foreground: Black,
						Background: White,
					}
				}
			}

		default:
			e.Debug("actPrivateFunction: %s%c", string(state.params), ch)
		}

	default:
		e.Debug("actPrivateFunction: %s%c", string(state.params), ch)
	}
}

func actOSC(e *Emulator, state *State, ch int) {
	params := state.Params()
	if len(params) != 2 {
		e.Debug("OSC: invalid parameters: %v")
		return
	}
	switch params[0] {
	case "0":
		e.SetIconName(params[1])
		e.SetWindowTitle(params[1])

	case "1":
		e.SetIconName(params[1])

	case "2":
		e.SetWindowTitle(params[1])

	default:
		e.Debug("OSC: unsupported control: %v", params)
	}
}

func actCSI(e *Emulator, state *State, ch int) {
	if debug {
		e.Debug("actCSI: ESC[%s%c (0x%x)", string(state.params), ch, ch)
	}
	switch ch {
	case '@': // ICH - Insert CHaracter
		e.InsertChars(e.Row, e.Col, state.CSIParam(1))

	case 'A': // CUU - CUrsor Up
		e.MoveTo(e.Row-state.CSIParam(1), e.Col)

	case 'B': // CUD - CUrsor Down
		row := e.Row + state.CSIParam(1)
		if row >= e.Height {
			row = e.Height - 1
		}
		e.MoveTo(row, e.Col)

	case 'C': // CUF - CUrsor Forward
		e.MoveTo(e.Row, e.Col+state.CSIParam(1))

	case 'D': // CUB - CUrsor Backward
		e.MoveTo(e.Row, e.Col-state.CSIParam(1))

	case 'K': // EL  - Erase in Line (cursor does not move)
		switch state.CSIParam(0) {
		case 0:
			e.ClearLine(e.Row, e.Col, e.Width)
		case 1:
			e.ClearLine(e.Row, 0, e.Col+1)
		case 2:
			e.ClearLine(e.Row, 0, e.Width)
		}

	case 'P':
		e.DeleteChars(e.Row, e.Col, state.CSIParam(1))

	case 'H': // CUP - CUrsor Position
		_, row, col := state.CSIParams(1, 1)
		e.MoveTo(row-1, col-1)

	case 'J': // Erase in Display (cursor does not move)
		switch state.CSIParam(0) {
		case 0: // Erase from current position to end (inclusive)
			// XXX
			e.Clear()
		case 1: // Erase from beginning ot current position (inclusive)
			// XXX
		case 2: // Erase entire display
			e.Clear()
		}

	case 'c':
		e.Output("\x1b[?62;1;2;7;8;9;15;18;21;44;45;46c")

	case 'f': // HVP - Horizontal and Vertical Position (depends on PUM)
		_, row, col := state.CSIParams(1, 1)
		e.MoveTo(row-1, col-1)

	case 'h':
		prefix, mode := state.CSIPrefixParam(0)
		switch prefix {
		case "": // Set Mode (SM)
			switch mode {
			case 2: // Keyboard Action Mode (AM)
			case 4: // Insert Mode (IRM)
			case 12: // Send/receive (SRM)
			case 20: // Automatic Newline (LNM)

			default:
				e.Debug("Set Mode (SM): unknown mode %d", mode)
			}

		case "?":
			switch mode {
			case 1034: // Interpret "meta" key, sets eight bit (eightBitInput)

			default:
				e.Debug("Unsupported ESC[%sh", string(state.params))
			}
		}

	case 'l':
		prefix, mode := state.CSIPrefixParam(0)
		switch prefix {
		case "":
			e.Debug("Unsupported ESC[%sl", string(state.params))

		case "?": // DEC*
			switch mode {
			case 3: // DECCOLM - 80 characters per line (erases screen)
				e.Clear()
				e.MoveTo(0, 0)

			default:
				e.Debug("DEC*: unknown mode %d", mode)
			}
		}

	default:
		e.Debug("actCSI: unsupported: ESC[%s%c (0x%x)\n", string(state.params),
			ch, ch)
	}
}

type Transition struct {
	Action Action
	Next   *State
}

type State struct {
	Name        string
	Default     Action
	params      []rune
	Transitions map[int]*Transition
}

func (s *State) String() string {
	return s.Name
}

func (s *State) Reset() {
	s.params = nil
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

func (s *State) Params() []string {
	return strings.Split(string(s.params), ";")
}

func (s *State) CSIParam(a int) int {
	_, values := s.parseCSIParam([]int{a})
	return values[0]
}

func (s *State) CSIPrefixParam(a int) (string, int) {
	prefix, values := s.parseCSIParam([]int{a})
	return prefix, values[0]
}

func (s *State) CSIParams(a, b int) (string, int, int) {
	prefix, values := s.parseCSIParam([]int{a, b})
	return prefix, values[0], values[1]
}

var reParam = regexp.MustCompilePOSIX("^([^0-9;:]*)([0-9;:]*)$")

func (s *State) parseCSIParam(defaults []int) (string, []int) {
	matches := reParam.FindStringSubmatch(string(s.params))
	if matches == nil {
		return "", defaults
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
	stOSC    = NewState("OSC", actError)
)

func init() {
	stStart.AddActions(0x00, 0x1f, actC0Control, nil)
	stStart.AddActions(0x9b, 0x9b, nil, stCSI)
	stStart.AddActions(0x1b, 0x1b, nil, stESC)

	stESC.AddActions(0x20, 0x2f, actAppendParam, nil)
	stESC.AddActions(0x30, 0x3f, actPrivateFunction, stStart)
	stESC.AddActions(0x40, 0x5f, actC1Control, stStart)
	stESC.AddActions(0x20, 0x20, actInsertSpace, nil) // Always space
	stESC.AddActions(0xa0, 0xa0, actInsertSpace, nil) // Always space
	stESC.AddActions(0x7f, 0x7f, nil, nil)            // Delete always ignored
	stESC.AddActions('[', '[', nil, stCSI)
	stESC.AddActions(']', ']', nil, stOSC)

	stOSC.AddActions(0x20, 0x7e, actAppendParam, nil)
	stOSC.AddActions(0x07, 0x07, actOSC, stStart)
	stOSC.AddActions(0x9c, 0x9c, actOSC, stStart)

	stCSI.AddActions(0x30, 0x3f, actAppendParam, nil)
	stCSI.AddActions(0x40, 0x7e, actCSI, stStart)
}

type Emulator struct {
	Width  int
	Height int
	Col    int
	Row    int
	Lines  [][]Char
	state  *State
	output io.Writer
	debug  io.Writer
}

func NewEmulator(output, debug io.Writer) *Emulator {
	return &Emulator{
		state:  stStart,
		output: output,
		debug:  debug,
	}
}

func (e *Emulator) SetState(state *State) {
	e.state = state
	e.state.Reset()
}

func (e *Emulator) Output(format string, a ...interface{}) {
	if e.output == nil {
		return
	}
	e.output.Write([]byte(fmt.Sprintf(format, a...)))
}

func (e *Emulator) Debug(format string, a ...interface{}) {
	if e.debug == nil {
		return
	}
	e.debug.Write([]byte(fmt.Sprintf(format, a...)))
}

func (e *Emulator) SetIconName(name string) {
	e.Debug("Icon Name: %s", name)
}

func (e *Emulator) SetWindowTitle(name string) {
	e.Debug("Window Title: %s", name)
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
	if to > e.Width {
		to = e.Width
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

func (e *Emulator) InsertChars(row, col, count int) {
	if row < 0 {
		row = 0
	} else if row >= e.Height {
		row = e.Height - 1
	}
	if col < 0 {
		col = 0
	} else if col >= e.Width {
		return
	}
	if col+count >= e.Width {
		e.ClearLine(row, col, e.Width)
		return
	}
	for x := e.Width - 1; x >= col; x-- {
		if x-count >= col {
			e.Lines[row][x] = e.Lines[row][x-count]
		} else {
			e.Lines[row][x] = blank
		}
	}
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
	if debug {
		e.Debug("Emulator.Input: %s<-0x%x,%d,%s", e.state, code, code,
			string(code))
	}
	next := e.state.Input(e, code)
	if next != nil {
		if debug {
			e.Debug("Emulator.Input: %s->%s", e.state, next)
		}
		e.SetState(next)
	}
}
