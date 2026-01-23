package regex

import (
	"fmt"
	"pegParser/scanner"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

type StateInType int

const (
	t_rune StateInType = iota
	t_meta
	t_jump
	t_not
	t_end
)

type StateIn struct {
	ID    int
	Typ   StateInType
	Value rune
}

type State struct {
	id   int
	next []StateIn
}

type Stack struct {
	states  []State
	count   int
	dontAdd bool
}

func getUnexpectedTypeError(want string, get string) string {
	return fmt.Sprintf("This is what you want: %s, this is what you get: %s", want, get)
}

func (s *Stack) run(run Node) {
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
	s.states[s.count].next = append(s.states[s.count].next, StateIn{s.count + 1, t_rune, literal})
}

func (s *Stack) meta(meta Node) {
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
	s.states[s.count].next = append(s.states[s.count].next, StateIn{s.count + 1, t_meta, literal})
}

func (s *Stack) char(char Node) {
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
		s.meta(child)
	case "rune":
		s.run(child)
	default:
		panic("char has illegal child: " + child.Typ)
	}
}

func (s *Stack) atom(atom Node) {
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
		s.char(child)
	case "capture":
		s.capture(child)
	case "set":
		s.set(child)
	default:
		panic("atom has illegal child: " + child.Typ)
	}
}

func (s *Stack) mode(mode Node) {
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
		panic("mode has illegal child: " + child.Typ)
	}
	mark := s.count
	add_not := len(mode.Children) == 2 && mode.Children[1].Value == "!"
	if add_not {
		s.states[s.count].next = append(s.states[mark].next, StateIn{s.count + 1, t_not, 0})
		s.count += 1
		s.states = append(s.states, State{s.count, []StateIn{}})
	}
	s.atom(child)
	if len(mode.Children) == 2 {
		repeat := mode.Children[1]
		if repeat.Typ != "string" {
			panic("mode has illegal child: " + repeat.Typ)
		}
		switch repeat.Value {
		case "?":
			s.states[mark].next = append(s.states[mark].next, StateIn{s.count + 1, t_jump, 0})
		case "!":
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{{1, t_end, 0}}})
			s.states[mark].next = append(s.states[mark].next, StateIn{s.count + 1, t_jump, 0})
		case "*":
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{}})
			s.states[s.count].next = append(s.states[s.count].next, StateIn{mark, t_jump, 0})
			s.states[mark].next = append(s.states[mark].next, StateIn{s.count + 1, t_jump, 0})
		case "+":
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{}})
			s.states[s.count].next = append(s.states[s.count].next, StateIn{mark, t_jump, 0}, StateIn{s.count + 1, t_jump, 0})
		default:
			panic("Illegal mode literal: " + repeat.Value)
		}
	}
}

func (s *Stack) group(group Node) {
	if group.Typ != "group" {
		panic(getUnexpectedTypeError("group", group.Typ))
	}
	if len(group.Children) == 0 {
		panic("No group children was unexpected")
	}
	for i, mode := range group.Children {
		s.mode(mode)
		if i < len(group.Children)-1 {
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{}})
		}
	}
}

func (s *Stack) capture(capture Node) {
	if capture.Typ != "capture" {
		panic(getUnexpectedTypeError("capture", capture.Typ))
	}
	if len(capture.Children) == 0 {
		panic("No capture children was unexpected")
	}
	if len(capture.Children) == 1 {
		s.group(capture.Children[0])
	} else {
		mark := s.count
		s.count += 1
		s.states = append(s.states, State{s.count, []StateIn{}})
		endlist := []int{}
		for _, group := range capture.Children {
			s.states[mark].next = append(s.states[mark].next, StateIn{s.count, t_jump, 0})
			s.group(group)
			endlist = append(endlist, s.count)
			s.count += 1
			s.states = append(s.states, State{s.count, []StateIn{}})
		}
		for i := range capture.Children {
			end := endlist[i]
			s.states[end].next[0].ID = s.count
		}
		s.states = s.states[0 : len(s.states)-1]
		s.count -= 1
	}
}

func (s *Stack) set(set Node) {
	if set.Typ != "set" {
		panic(getUnexpectedTypeError("set", set.Typ))
	}
	if len(set.Children) == 0 {
		panic("No set children was unexpected")
	}
	switch set.Value {
	case "not":
		nodes := []Node{}
		for _, el := range set.Children {
			nodes = append(nodes, Node{"group", "", []Node{{"mode", "", []Node{{"atom", "", []Node{el}}}}}})
		}
		mode1 := Node{"mode", "", []Node{{"atom", "", []Node{{"capture", "", nodes}}}, {"string", "!", []Node{}}}}
		mode2 := Node{"mode", "", []Node{{"atom", "", []Node{{"char", "", []Node{{"meta", ".", []Node{}}}}}}}}
		s.capture(Node{"capture", "", []Node{{"group", "", []Node{mode1, mode2}}}})
	case "":
		nodes := []Node{}
		for _, el := range set.Children {
			nodes = append(nodes, Node{"group", "", []Node{{"mode", "", []Node{{"atom", "", []Node{el}}}}}})
		}
		s.capture(Node{"capture", "", nodes})
	default:
		panic("set has illegal value: " + set.Value)
	}
}

func (s *Stack) assemble(regex Node) {
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
	s.states = append(s.states, State{s.count, []StateIn{{0, t_end, 0}}})
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
	final.assemble(regex)
	return final.states
}

func meta(r, meta rune) bool {
	switch meta {
	case '.':
		if r != 0 {
			return true
		}
	case 'c':
		if unicode.IsControl(r) {
			return true
		}
	case 's':
		if unicode.IsSpace(r) {
			return true
		}
	case 'S':
		if !unicode.IsSpace(r) {
			return true
		}
	case 'p':
		if unicode.IsPrint(r) {
			return true
		}
	case 'w':
		if unicode.IsLetter(r) {
			return true
		}
	case 'W':
		if !unicode.IsLetter(r) {
			return true
		}
	case 'l':
		if unicode.IsLower(r) {
			return true
		}
	case 'u':
		if unicode.IsUpper(r) {
			return true
		}
	case 'a':
		if r >= '0' && r <= '9' {
			return true
		}
	case 'A':
		if r < '0' || r > '9' {
			return true
		}
	case 'x':
		if slices.Contains([]rune("abcdefABCDEF0123456789"), r) {
			return true
		}
	case 'g':
		if unicode.IsGraphic(r) {
			return true
		}
	case 'z':
		if unicode.IsPunct(r) {
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
	case 'v':
		if r == '\v' {
			return true
		}
	case 'f':
		if r == '\f' {
			return true
		}
	case 'b':
		if r != '[' && r != ']' {
			return true
		}
	default:
		if slices.Contains([]rune("()[]{}.|+*?!"), meta) && r == meta {
			return true
		}
	}
	return false
}

type ResultCode int

const (
	UnexpectedEnd ResultCode = iota
	UnexpectedMore
	UnexpectedRune
	Matched
)

func test(stack []State, runes []rune, index, pos int, inside_not bool, fromFront bool) (ResultCode, bool) {
	s := stack[pos]
	if index > len(runes)-1 {
		return UnexpectedEnd, false
	}
	r := runes[index]
	nullNextList := []int{}
	var final ResultCode
	for _, next := range s.next {
		switch next.Typ {
		case t_rune:
			switch r {
			case next.Value:
				res, ok := test(stack, runes, index+1, next.ID, inside_not, false)
				if ok {
					return Matched, true
				} else {
					final = res
				}
			case 0:
				final = UnexpectedEnd
			default:
				fmt.Println("rune", r, string(r), next.Value)
				final = UnexpectedRune
			}
		case t_meta:
			if meta(r, next.Value) {
				res, ok := test(stack, runes, index+1, next.ID, inside_not, false)
				if ok {
					return Matched, true
				} else {
					final = res
				}
			} else if r == 0 {
				final = UnexpectedEnd
			} else {
				fmt.Println("meta")
				final = UnexpectedRune
			}
		case t_not:
			res, ok := test(stack, runes, index, next.ID, true, fromFront)
			if ok {
				fmt.Println("not")
				return UnexpectedRune, false
			} else {
				final = res
			}
		case t_end:
			fmt.Println("hm")
			if inside_not {
				return Matched, true
			} else {
				if pos == len(stack)-1 && index == len(runes)-1 {
					return Matched, true
				} else {
					fmt.Println("hm")
					return UnexpectedMore, false
				}
			}
		case t_jump:
			nullNextList = append(nullNextList, next.ID)
		}
	}
	if len(nullNextList) != 0 {
		slices.Sort(nullNextList)
		for _, next := range nullNextList {
			if next < pos && fromFront {
				continue
			}
			res, ok := test(stack, runes, index, next, inside_not, fromFront || next < pos)
			if ok {
				return Matched, true
			} else {
				final = res
			}
		}
	}
	return final, false
}

func getNestedRegex(stack []State, id int) (final []State) {
	notCount := 1
	init := id
	for {
		s := stack[init]
		var s2 State
		s2.id = s.id - id
		init++
		for _, next := range s.next {
			n := StateIn{next.ID - id, next.Typ, next.Value}
			s2.next = append(s2.next, n)
			switch next.Typ {
			case t_not:
				notCount += 1
			case t_end:
				notCount -= 1
				if notCount == 0 {
					final = append(final, s2)
					return final
				}
			}
		}
		final = append(final, s2)
	}
}

type backTrack struct {
	pos     int
	index   int
	count   int
	choices []int
}

func RunStack(stack []State, str string) (ResultCode, bool) {
	fmt.Println(stack)
	pos := 0
	index := 0
	runes := []rune(str)
	runes = append(runes, 0)
	pointList := []backTrack{}
	var final ResultCode
loop:
	for {
		s := stack[pos]
		if index > len(runes)-1 {
			return UnexpectedEnd, false
		}
		r := runes[index]
		nullNextList := []int{}
		for _, next := range s.next {
			switch next.Typ {
			case t_rune:
				switch r {
				case next.Value:
					index += 1
					pos = next.ID
					continue loop
				case 0:
					final = UnexpectedEnd
				default:
					// fmt.Println("rune", r, string(r), next.Value)
					final = UnexpectedRune
				}
			case t_meta:
				if meta(r, next.Value) {
					index += 1
					pos = next.ID
					continue loop
				} else if r == 0 {
					final = UnexpectedEnd
				} else {
					final = UnexpectedRune
				}
			case t_not:
				res, ok := RunStack(getNestedRegex(stack, next.ID), string(runes[index:len(runes)-1]))
				if ok || res == Matched || res == UnexpectedMore {
					return UnexpectedRune, false
				}
			case t_end:
				if pos == len(stack)-1 && index == len(runes)-1 || r == 0 {
					return Matched, true
				} else {
					return UnexpectedMore, false
				}
			case t_jump:
				nullNextList = append(nullNextList, next.ID)
			}
		}
		if len(nullNextList) > 0 {
			slices.Sort(nullNextList)
			choices := nullNextList
			pointList = append(pointList, backTrack{pos, index, 0, choices})
		}
		if len(pointList) > 0 {
			p := pointList[len(pointList)-1]
			if p.count > len(p.choices)-1 {
				if len(pointList) <= 1 {
					return final, false
				}
				pointList = pointList[:len(pointList)-1]
				p = pointList[len(pointList)-1]
			}
			index = p.index
			next := p.choices[p.count]
			pointList[len(pointList)-1].count += 1
			pos = next
			continue
		} else {
			return final, false
		}
	}
}

func UseStack(stack []State, str string) (ResultCode, bool) {
	fmt.Println(stack)
	runes := []rune(str)
	runes = append(runes, 0)
	return test(stack, runes, 0, 0, false, false)
}

var memo = make(map[string][]State)

func run(regex, str string) (ResultCode, bool) {
	if s, ok := memo[regex]; ok {
		return RunStack(s, str)
	}
	r := GetRegexParser(regex)
	n := r.Parse()
	if n.Typ == "" {
		panic("failed to parse " + regex + " regex")
	}
	s := GetRegexStack(n)
	memo[regex] = s
	return RunStack(s, str)
}

var memo2 = make(map[string]*regexp.Regexp)

func RunRegex(s *scanner.Scanner, rule string) (bool, string) {
	// var r *regexp.Regexp
	// if rr, ok := memo2[rule]; ok {
	// 	r = rr
	// } else {
	// 	r = regexp.MustCompile(rule)
	// 	r.Longest()
	// 	memo2[rule] = r
	// }
	// pos := s.Mark()
	// text := s.Text()
	// loc := r.FindStringIndex(text)
	// if loc == nil {
	// 	return false, ""
	// } else {
	// 	matched := text[loc[0]:loc[1]]
	// 	if loc[0] == 0 {
	// 		s.Reset(pos + loc[1] + 1)
	// 		return true, matched
	// 	} else {
	// 		return false, ""
	// 	}
	// }

	buffer := strings.Builder{}
	cuttoff := false
	{
		_, ok := run(rule, "")
		if ok {
			cuttoff = true
		}
	}
	pos := s.Mark()
	for {
		if ok, r := s.Rune(); ok {
			res, ok := run(rule, buffer.String()+string(r))
			if ok {
				cuttoff = true
			} else if cuttoff {
				s.Reset(pos)
				return true, buffer.String()
			} else if res == UnexpectedRune || res == UnexpectedMore {
				return false, ""
			}
			pos = s.Mark()
			buffer.WriteRune(r)
		} else {
			if cuttoff {
				s.Reset(pos)
				return true, buffer.String()
			} else {
				return false, ""
			}
		}
	}
}
