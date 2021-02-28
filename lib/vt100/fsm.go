//
// emulator.go
//
// Copyright (c) 2018-2021 Markku Rossi
//
// All rights reserved.
//

package vt100

import (
	"regexp"
	"strconv"
	"strings"
)

type Action func(e *Emulator, state *State, ch int)

func actError(e *Emulator, state *State, ch int) {
	e.debug("actError: state=%s, ch=0x%x (%d) '%c'\n", state, ch, ch, ch)
	e.setState(stStart)
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
		e.MoveTo(e.Cursor.Y, e.Cursor.X-1)
	case 0x09: // Horizontal Tabulation.
		var x = e.Cursor.X + 1
		for ; x%8 != 0; x++ {
		}
		e.MoveTo(e.Cursor.Y, x)
	case 0x0a: // Linefeed
		e.MoveTo(e.Cursor.Y+1, e.Cursor.X)
	case 0x0d: // Carriage Return
		e.MoveTo(e.Cursor.Y, 0)

	default:
		e.debug("actC0Control: %s: %s0x%x\n", state, string(state.params), ch)
	}
}

func actC1Control(e *Emulator, state *State, ch int) {
	switch ch {
	case 'D': // Index, moves down one line same column regardless of NL
		e.MoveTo(e.Cursor.Y+1, e.Cursor.X)
	case 'E': // NEw Line, moves done one line and to first column (CR+LF)
		e.MoveTo(e.Cursor.Y+1, 0)
	case 'M': // Reverse Index, go up one line, reverse scroll if necessary
		e.MoveTo(e.Cursor.Y-1, e.Cursor.X)
	default:
		e.debug("actC1Control: %s: %s0x%x\n", state, string(state.params), ch)
	}
}

func actTwoCharEscape(e *Emulator, state *State, ch int) {
	switch ch {
	case 'c': // RIS - Reset to Initial State (VT100 does a power-on reset)
		e.Reset()

	default:
		e.debug("actTwoCharEscape: %s: %s0x%x\n",
			state, string(state.params), ch)
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
			e.display.DECALN(e.Size)

		default:
			e.debug("Unsupported actPrivateFunction: %s%c",
				string(state.params), ch)
		}

	default:
		e.debug("Unsupported actPrivateFunction: %s%c",
			string(state.params), ch)
	}
}

func actOSC(e *Emulator, state *State, ch int) {
	params := state.Params()
	if len(params) != 2 {
		e.debug("OSC: invalid parameters: %v", params)
		return
	}
	switch params[0] {
	case "0":
		e.setIconName(params[1])
		e.setWindowTitle(params[1])

	case "1":
		e.setIconName(params[1])

	case "2":
		e.setWindowTitle(params[1])

	default:
		e.debug("OSC: unsupported control: %v", params)
	}
}

func actCSI(e *Emulator, state *State, ch int) {
	if debug {
		e.debug("actCSI: ESC[%s%c (0x%x)", string(state.params), ch, ch)
	}
	switch ch {
	case '@': // ICH - Insert CHaracter
		e.InsertChars(e.Cursor.Y, e.Cursor.X, state.CSIParam(1))

	case 'A': // CUU - CUrsor Up
		e.MoveTo(e.Cursor.Y-state.CSIParam(1), e.Cursor.X)

	case 'B': // CUD - CUrsor Down
		row := e.Cursor.Y + state.CSIParam(1)
		if row >= e.Size.Y {
			row = e.Size.Y - 1
		}
		e.MoveTo(row, e.Cursor.X)

	case 'C': // CUF - CUrsor Forward
		e.MoveTo(e.Cursor.Y, e.Cursor.X+state.CSIParam(1))

	case 'D': // CUB - CUrsor Backward
		e.MoveTo(e.Cursor.Y, e.Cursor.X-state.CSIParam(1))

	case 'G': // CHA - Cursor Horizontal position Absolute
		e.MoveTo(e.Cursor.Y, state.CSIParam(1)-1)

	case 'K': // EL  - Erase in Line (cursor does not move)
		switch state.CSIParam(0) {
		case 0:
			e.ClearLine(e.Cursor.Y, e.Cursor.X, e.Size.X)
		case 1:
			e.ClearLine(e.Cursor.Y, 0, e.Cursor.X)
		case 2:
			e.ClearLine(e.Cursor.Y, 0, e.Size.X)
		}

	case 'P':
		e.DeleteChars(e.Cursor.Y, e.Cursor.X, state.CSIParam(1))

	case 'H': // CUP - CUrsor Position
		_, row, col := state.CSIParams(1, 1)
		e.MoveTo(row-1, col-1)

	case 'J': // Erase in Display (cursor does not move)
		switch state.CSIParam(0) {
		case 0: // Erase from current position to end (inclusive)
			e.Clear(false, true)
		case 1: // Erase from beginning ot current position (inclusive)
			e.Clear(true, false)
		case 2: // Erase entire display
			e.Clear(true, true)
		}

	case 'c':
		e.output("\x1b[?62;1;2;7;8;9;15;18;21;44;45;46c")

	case 'd': // VPA - Vertical Position Absolute (depends on PUM)
		e.MoveTo(state.CSIParam(1)-1, e.Cursor.X)

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
				e.debug("Set Mode (SM): unknown mode %d", mode)
			}

		case "?":
			switch mode {
			case 3: // DECCOLM - COLumn mode, 132 characters per line
				e.Clear(true, true)
				e.Resize(132, e.Size.Y)
				e.MoveTo(0, 0)

			case 1034: // Interpret "meta" key, sets eight bit (eightBitInput)

			default:
				e.debug("Unsupported ESC[%sh", string(state.params))
			}
		}

	case 'l':
		prefix, mode := state.CSIPrefixParam(0)
		switch prefix {
		case "":
			e.debug("Unsupported ESC[%sl", string(state.params))

		case "?": // DEC*
			switch mode {
			case 3: // DECCOLM - 80 characters per line (erases screen)
				e.Clear(true, true)
				e.Resize(80, e.Size.Y)
				e.MoveTo(0, 0)

			default:
				e.debug("DEC*: unknown mode %d", mode)
			}
		}

	default:
		e.debug("actCSI: unsupported: ESC[%s%c (0x%x)\n", string(state.params),
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
		if err != nil || i == 0 {
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
	stESC.AddActions(0x60, 0x7e, actTwoCharEscape, stStart)
	stESC.AddActions(0x7f, 0x7f, nil, nil)            // Delete always ignored
	stESC.AddActions(0x20, 0x20, actInsertSpace, nil) // Always space
	stESC.AddActions(0xa0, 0xa0, actInsertSpace, nil) // Always space
	stESC.AddActions('[', '[', nil, stCSI)
	stESC.AddActions(']', ']', nil, stOSC)

	stOSC.AddActions(0x20, 0x7e, actAppendParam, nil)
	stOSC.AddActions(0x07, 0x07, actOSC, stStart)
	stOSC.AddActions(0x9c, 0x9c, actOSC, stStart)

	stCSI.AddActions(0x30, 0x3f, actAppendParam, nil)
	stCSI.AddActions(0x40, 0x7e, actCSI, stStart)
}
