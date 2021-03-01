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

type action func(e *Emulator, state *state, ch int)

func actError(e *Emulator, state *state, ch int) {
	e.debug("actError: state=%s, ch=0x%x (%d) '%c'\n", state, ch, ch, ch)
	e.setState(stStart)
}

func actInsertChar(e *Emulator, state *state, ch int) {
	e.insertChar(ch)
}

func actInsertSpace(e *Emulator, state *state, ch int) {
	e.insertChar(' ')
}

func actC0Control(e *Emulator, state *state, ch int) {
	switch ch {
	case 0x08: // BS
		e.moveTo(e.Cursor.Y, e.Cursor.X-1)
	case 0x09: // Horizontal Tabulation.
		var x = e.Cursor.X + 1
		for ; x%8 != 0; x++ {
		}
		e.moveTo(e.Cursor.Y, x)
	case 0x0a: // Linefeed
		e.moveTo(e.Cursor.Y+1, e.Cursor.X)
	case 0x0d: // Carriage Return
		e.moveTo(e.Cursor.Y, 0)

	default:
		e.debug("actC0Control: %s: %s0x%x\n",
			state, string(state.parameters), ch)
	}
}

func actC1Control(e *Emulator, state *state, ch int) {
	switch ch {
	case 'D': // Index, moves down one line same column regardless of NL
		e.moveTo(e.Cursor.Y+1, e.Cursor.X)
	case 'E': // NEw Line, moves done one line and to first column (CR+LF)
		e.moveTo(e.Cursor.Y+1, 0)
	case 'M': // Reverse Index, go up one line, reverse scroll if necessary
		e.moveTo(e.Cursor.Y-1, e.Cursor.X)
	default:
		e.debug("actC1Control: %s: %s0x%x\n",
			state, string(state.parameters), ch)
	}
}

func actTwoCharEscape(e *Emulator, state *state, ch int) {
	switch ch {
	case 'c': // RIS - Reset to Initial State (VT100 does a power-on reset)
		e.Reset()

	default:
		e.debug("actTwoCharEscape: %s: %s0x%x\n",
			state, string(state.parameters), ch)
	}
}

func actAppendParam(e *Emulator, state *state, ch int) {
	state.parameters = append(state.parameters, rune(ch))
}

func actPrivateFunction(e *Emulator, state *state, ch int) {
	switch ch {
	case '8':
		switch string(state.parameters) {
		case "#": // DECALN - Alignment display, fill screen with "E"
			e.display.DECALN(e.Size)

		default:
			e.debug("Unsupported actPrivateFunction: %s%c",
				string(state.parameters), ch)
		}

	default:
		e.debug("Unsupported actPrivateFunction: %s%c",
			string(state.parameters), ch)
	}
}

func actOSC(e *Emulator, state *state, ch int) {
	params := state.params()
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

func actCSI(e *Emulator, state *state, ch int) {
	if debug {
		e.debug("actCSI: ESC[%s%c (0x%x)", string(state.parameters), ch, ch)
	}
	switch ch {
	case '@': // ICH - Insert CHaracter
		e.insertChars(e.Cursor.Y, e.Cursor.X, state.csiParam(1))

	case 'A': // CUU - CUrsor Up
		e.moveTo(e.Cursor.Y-state.csiParam(1), e.Cursor.X)

	case 'B': // CUD - CUrsor Down
		row := e.Cursor.Y + state.csiParam(1)
		if row >= e.Size.Y {
			row = e.Size.Y - 1
		}
		e.moveTo(row, e.Cursor.X)

	case 'C': // CUF - CUrsor Forward
		e.moveTo(e.Cursor.Y, e.Cursor.X+state.csiParam(1))

	case 'D': // CUB - CUrsor Backward
		e.moveTo(e.Cursor.Y, e.Cursor.X-state.csiParam(1))

	case 'G': // CHA - Cursor Horizontal position Absolute
		e.moveTo(e.Cursor.Y, state.csiParam(1)-1)

	case 'K': // EL  - Erase in Line (cursor does not move)
		switch state.csiParam(0) {
		case 0:
			e.clearLine(e.Cursor.Y, e.Cursor.X, e.Size.X)
		case 1:
			e.clearLine(e.Cursor.Y, 0, e.Cursor.X)
		case 2:
			e.clearLine(e.Cursor.Y, 0, e.Size.X)
		}

	case 'P':
		e.deleteChars(e.Cursor.Y, e.Cursor.X, state.csiParam(1))

	case 'H': // CUP - CUrsor Position
		_, row, col := state.csiParams(1, 1)
		e.moveTo(row-1, col-1)

	case 'J': // Erase in Display (cursor does not move)
		switch state.csiParam(0) {
		case 0: // Erase from current position to end (inclusive)
			e.clear(false, true)
		case 1: // Erase from beginning ot current position (inclusive)
			e.clear(true, false)
		case 2: // Erase entire display
			e.clear(true, true)
		}

	case 'c':
		e.output("\x1b[?62;1;2;7;8;9;15;18;21;44;45;46c")

	case 'd': // VPA - Vertical Position Absolute (depends on PUM)
		e.moveTo(state.csiParam(1)-1, e.Cursor.X)

	case 'f': // HVP - Horizontal and Vertical Position (depends on PUM)
		_, row, col := state.csiParams(1, 1)
		e.moveTo(row-1, col-1)

	case 'h':
		prefix, mode := state.csiPrefixParam(0)
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
				e.clear(true, true)
				e.Resize(132, e.Size.Y)
				e.moveTo(0, 0)

			case 1034: // Interpret "meta" key, sets eight bit (eightBitInput)

			default:
				e.debug("Unsupported ESC[%sh", string(state.parameters))
			}
		}

	case 'l':
		prefix, mode := state.csiPrefixParam(0)
		switch prefix {
		case "":
			e.debug("Unsupported ESC[%sl", string(state.parameters))

		case "?": // DEC*
			switch mode {
			case 3: // DECCOLM - 80 characters per line (erases screen)
				e.clear(true, true)
				e.Resize(80, e.Size.Y)
				e.moveTo(0, 0)

			default:
				e.debug("DEC*: unknown mode %d", mode)
			}
		}

	default:
		e.debug("actCSI: unsupported: ESC[%s%c (0x%x)\n",
			string(state.parameters), ch, ch)
	}
}

type transition struct {
	action action
	next   *state
}

type state struct {
	name          string
	defaultAction action
	parameters    []rune
	transitions   map[int]*transition
}

func (s *state) String() string {
	return s.name
}

func (s *state) reset() {
	s.parameters = nil
}

func (s *state) addActions(from, to int, act action, next *state) {
	transition := &transition{
		action: act,
		next:   next,
	}

	for ; from <= to; from++ {
		s.transitions[from] = transition
	}
}

func (s *state) input(e *Emulator, code int) *state {
	var act action
	var next *state

	transition, ok := s.transitions[code]
	if ok {
		act = transition.action
		next = transition.next
	} else {
		act = s.defaultAction
	}

	if act != nil {
		act(e, s, code)
	}

	return next
}

func (s *state) params() []string {
	return strings.Split(string(s.parameters), ";")
}

func (s *state) csiParam(a int) int {
	_, values := s.parseCSIParam([]int{a})
	return values[0]
}

func (s *state) csiPrefixParam(a int) (string, int) {
	prefix, values := s.parseCSIParam([]int{a})
	return prefix, values[0]
}

func (s *state) csiParams(a, b int) (string, int, int) {
	prefix, values := s.parseCSIParam([]int{a, b})
	return prefix, values[0], values[1]
}

var reParam = regexp.MustCompilePOSIX("^([^0-9;:]*)([0-9;:]*)$")

func (s *state) parseCSIParam(defaults []int) (string, []int) {
	matches := reParam.FindStringSubmatch(string(s.parameters))
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

func newState(name string, def action) *state {
	return &state{
		name:          name,
		defaultAction: def,
		transitions:   make(map[int]*transition),
	}
}

var (
	stStart  = newState("start", actInsertChar)
	stESC    = newState("ESC", actError)
	stCSI    = newState("CSI", actError)
	stESCSeq = newState("ESCSeq", actError)
	stOSC    = newState("OSC", actError)
)

func init() {
	stStart.addActions(0x00, 0x1f, actC0Control, nil)
	stStart.addActions(0x9b, 0x9b, nil, stCSI)
	stStart.addActions(0x1b, 0x1b, nil, stESC)

	stESC.addActions(0x20, 0x2f, actAppendParam, nil)
	stESC.addActions(0x30, 0x3f, actPrivateFunction, stStart)
	stESC.addActions(0x40, 0x5f, actC1Control, stStart)
	stESC.addActions(0x60, 0x7e, actTwoCharEscape, stStart)
	stESC.addActions(0x7f, 0x7f, nil, nil)            // Delete always ignored
	stESC.addActions(0x20, 0x20, actInsertSpace, nil) // Always space
	stESC.addActions(0xa0, 0xa0, actInsertSpace, nil) // Always space
	stESC.addActions('[', '[', nil, stCSI)
	stESC.addActions(']', ']', nil, stOSC)

	stOSC.addActions(0x20, 0x7e, actAppendParam, nil)
	stOSC.addActions(0x07, 0x07, actOSC, stStart)
	stOSC.addActions(0x9c, 0x9c, actOSC, stStart)

	stCSI.addActions(0x30, 0x3f, actAppendParam, nil)
	stCSI.addActions(0x40, 0x7e, actCSI, stStart)
}
