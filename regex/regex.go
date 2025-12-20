package regex

import (
	"fmt"
	"slices"
)

type StateIn struct {
	ID    int
	Typ   string
	Value rune
}

type State struct {
	ID   int
	Next []StateIn
}

type Stack struct {
	states  []State
	count   int
	dontAdd bool
}

func getUnexpectedTypeError(want string, get string) string {
	return fmt.Sprintf("This is what you want: %s, this is what you get: %s", want, get)
}

func (s *Stack) run(run Node, not bool) StateIn {
	if run.Typ != "rune" {
		panic(getUnexpectedTypeError("rune", run.Typ))
	}
	if len(run.Children) != 0 {
		panic("No rune children was unexpected")
	}
	if run.Value == "" {
		panic("Empty rune value")
	}
	literal := []rune(run.Value)[0]
	var mode string
	if not {
		mode = "not_rune"
	} else {
		mode = "rune"
	}
	nextPass := StateIn{s.count + 1, mode, literal}
	if !s.dontAdd {
		s.states[s.count].Next = append(s.states[s.count].Next, nextPass)
	}
	s.dontAdd = false
	return nextPass
}

func (s *Stack) meta(meta Node, not bool) StateIn {
	if meta.Typ != "meta" {
		panic(getUnexpectedTypeError("meta", meta.Typ))
	}
	if len(meta.Children) != 0 {
		panic("No meta children was unexpected")
	}
	if meta.Value == "" {
		panic("Empty meta value")
	}
	literal := []rune(meta.Value)[0]
	var mode string
	if not {
		mode = "not_meta"
	} else {
		mode = "meta"
	}
	nextPass := StateIn{s.count + 1, mode, literal}
	if !s.dontAdd {
		s.states[s.count].Next = append(s.states[s.count].Next, nextPass)
	}
	s.dontAdd = false
	return nextPass
}

func (s *Stack) char(char Node, not bool) StateIn {
	if char.Typ != "char" {
		panic(getUnexpectedTypeError("char", char.Typ))
	}
	if len(char.Children) == 0 {
		panic("No char children was unexpected")
	}
	if len(char.Children) > 1 {
		panic("Too much char children")
	}
	child := char.Children[0]
	switch child.Typ {
	case "meta":
		return s.meta(child, not)
	case "rune":
		return s.run(child, not)
	default:
		panic("char has illegal child")
	}
}

func (s *Stack) atom(atom Node) StateIn {
	if atom.Typ != "atom" {
		panic(getUnexpectedTypeError("atom", atom.Typ))
	}
	if len(atom.Children) == 0 {
		panic("No atom children was unexpected")
	}
	if len(atom.Children) > 2 {
		panic("Too much atom children")
	}
	child := atom.Children[0]
	switch child.Typ {
	case "char":
		if len(atom.Children) == 2 {
			return s.char(child, true)
		}
		return s.char(child, false)
	case "capture":
		return s.capture(child)
	default:
		panic("char has illegal child")
	}
}

func (s *Stack) mode(mode Node) StateIn {
	if mode.Typ != "mode" {
		panic(getUnexpectedTypeError("mode", mode.Typ))
	}
	if len(mode.Children) == 0 {
		panic("No mode children was unexpected")
	}
	if len(mode.Children) > 2 {
		panic("Too much mode children")
	}
	child := mode.Children[0]
	if child.Typ != "atom" {
		panic("mode has illegal child")
	}
	// if len(mode.Children) == 2 && mode.Children[1].Value == "!" {
	// 	s.states[s.count].Next = append(s.states[s.count].Next, StateIn{s.count + 1, "not", 0})
	// 	s.count += 1
	// 	s.states = append(s.states, State{s.count, []StateIn{}})
	// }
	mark := s.count
	add_not := len(mode.Children) == 2 && mode.Children[1].Value == "!"
	if add_not {
		s.states[s.count].Next = append(s.states[mark].Next, StateIn{s.count + 1, "not", 0})
		s.count += 1
		s.states = append(s.states, State{s.count, []StateIn{}})
	}
	nextPass := s.atom(child)
	if len(mode.Children) == 2 {
		repeat := mode.Children[1]
		if repeat.Typ != "string" {
			panic("mode has illegal child")
		}
		switch repeat.Value {
		case "?":
			s.states[mark].Next = append(s.states[mark].Next, StateIn{s.count + 1, "", 0})
		case "!":
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{{1, "end", 0}}})
			s.states[mark].Next = append(s.states[mark].Next, StateIn{s.count + 1, "", 0})
		case "*":
			if s.count-mark == 0 {
				s.count += 1
				s.states = append(s.states, State{s.count, []StateIn{}})
				s.states[s.count].Next = append(s.states[s.count].Next, StateIn{mark, "", 0})
				s.states[mark].Next = append(s.states[mark].Next, StateIn{s.count + 1, "", 0})
			} else {
				s.states[s.count].Next = append(s.states[s.count].Next, StateIn{mark, "", 0})
				s.states[mark].Next = append(s.states[mark].Next, StateIn{s.count + 1, "", 0})
			}
		case "+":
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{}})
			s.states[s.count].Next = append(s.states[s.count].Next, StateIn{mark, "", 0}, StateIn{s.count + 1, "", 0})
		default:
			panic("Illegal mode literal")
		}
	}
	return nextPass
}

func (s *Stack) group(group Node) StateIn {
	if group.Typ != "group" {
		panic(getUnexpectedTypeError("group", group.Typ))
	}
	if len(group.Children) == 0 {
		panic("No group children was unexpected")
	}
	first := true
	var nextPass StateIn
	for i, mode := range group.Children {
		if first {
			nextPass = s.mode(mode)
			first = false
		} else {
			s.mode(mode)
		}
		if i < len(group.Children)-1 {
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{}})
		}
	}
	return nextPass
}

func (s *Stack) capture(capture Node) StateIn {
	if capture.Typ != "capture" {
		panic(getUnexpectedTypeError("capture", capture.Typ))
	}
	if len(capture.Children) == 0 {
		panic("No capture children was unexpected")
	}
	if len(capture.Children) == 1 {
		return s.group(capture.Children[0])
	} else {
		mark := s.count
		nextList := []StateIn{}
		initList := []int{}
		endList := []int{}
		first := true
		var firstNext StateIn
		for _, group := range capture.Children {
			s.dontAdd = true
			initList = append(initList, s.count+1)
			nextList = append(nextList, s.group(group))
			endList = append(endList, s.count)
			if first {
				firstNext = nextList[len(nextList)-1]
				first = false
			}
		}
		s.states[mark].Next = []StateIn{}
		for i, next := range nextList {
			init := initList[i]
			end := endList[i]
			if end-init == -1 {
				s.states[mark].Next = append(s.states[mark].Next, StateIn{s.count + 1, next.Typ, next.Value})
			} else {
				s.states[mark].Next = append(s.states[mark].Next, StateIn{init, next.Typ, next.Value})
			}
			s.states[end].Next[0].ID = s.count + 1
		}
		return firstNext
	}
}

func (s *Stack) assemle(regex Node) {
	if regex.Typ != "regex" {
		panic(getUnexpectedTypeError("regex", regex.Typ))
	}
	if len(regex.Children) == 0 {
		panic("No regex children was unexpected")
	}
	if len(regex.Children) > 1 {
		panic("Too much regex children")
	}
	s.capture(regex.Children[0])
	s.count += 1
	s.states = append(s.states, State{s.count, []StateIn{}})
}

func GetRegexStack(regex Node) []State {
	if regex.Typ != "regex" {
		panic(getUnexpectedTypeError("regex", regex.Typ))
	}
	if len(regex.Children) == 0 {
		panic("No regex child has unexpected")
	}
	if len(regex.Children) > 1 {
		panic("Too much regex children")
	}
	final := Stack{[]State{{0, []StateIn{}}}, 0, false}
	final.assemle(regex)
	final.states[len(final.states)-1].Next = []StateIn{{0, "end", 0}}
	return final.states
}

func meta(r, meta rune) bool {
	switch meta {
	case '.':
		if r != 0 {
			return true
		}
	case 'w':
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return true
		}
	case 'a':
		if r >= '0' && r <= '9' {
			return true
		}
	case 'b':
		if r != '[' && r != ']' {
			return true
		}
	case 'r':
		if r == '\r' {
			return true
		}
	case 'n':
		if r == '\n' {
			return true
		}
	case 't':
		if r == '\t' {
			return true
		}
	}
	return false
}

func test(stack []State, runes []rune, index, pos int, flag bool) bool {
	// fmt.Println("pos: " + strconv.Itoa(pos))
	s := stack[pos]
	if index > len(runes)-1 {
		return false
	}
	r := runes[index]
	// fmt.Println("rune: " + string(r))
	nullNextList := []int{}
	for _, next := range s.Next {
		switch next.Typ {
		case "rune":
			if r == next.Value {
				index += 1
				pos = next.ID
				if test(stack, runes, index, pos, flag) {
					return true
				}
			}
		case "not_rune":
			if r != next.Value {
				index += 1
				pos = next.ID
				if test(stack, runes, index, pos, flag) {
					return true
				}
			}
		case "meta":
			if meta(r, next.Value) {
				index += 1
				pos = next.ID
				if test(stack, runes, index, pos, flag) {
					return true
				}
			} else {
				if r == next.Value {
					index += 1
					pos = next.ID
					if test(stack, runes, index, pos, flag) {
						return true
					}
				}
			}
		case "not_meta":
			if !meta(r, next.Value) {
				index += 1
				pos = next.ID
				if test(stack, runes, index, pos, flag) {
					return true
				}
			}
		case "not":
			if test(stack, runes, index, next.ID, true) {
				return false
			}
		case "end":
			if flag {
				return true
			} else {
				if pos == len(stack)-1 && index == len(runes)-1 {
					return true
				} else {
					return false
				}
			}
		case "":
			nullNextList = append(nullNextList, next.ID)
		default:
			panic(next.Typ + " não foi implementado")
		}
	}
	if len(nullNextList) != 0 {
		slices.Sort(nullNextList)
		for _, next := range nullNextList {
			if test(stack, runes, index, next, flag) {
				return true
			}
		}
	}
	return false
}

func UseStack(stack []State, str string) bool {
	runes := []rune(str)
	runes = append(runes, 0)
	return test(stack, runes, 0, 0, false)
}

var memo map[string]*[]State

func Run(regex, str string) bool {
	var s *[]State
	var ok bool
	if s, ok = memo[regex]; !ok {
		r := GetRegexParser(regex)
		ss := GetRegexStack(r.Parse())
		s = &ss
	}
	return UseStack(*s, str)
}
