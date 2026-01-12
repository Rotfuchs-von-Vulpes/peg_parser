package peg

import (
	"pegParser/regex"
	sc "pegParser/scanner"
)

type Node2 struct {
	Typ      string
	Value    string
	Children []Node2
}

type Grammar struct {
	Rules []Rule
}

type Rule struct {
	name string
	body Body
}

type Body struct {
	Alts []Alternative
}

type Alternative struct {
	Loops []Loop
}

type loop_mode int

const (
	l_none loop_mode = iota
	l_zero_or_one
	l_zero_or_more
	l_one_or_more
)

type Loop struct {
	Mode  loop_mode
	Child Node
	Not   bool
}

type LiteralType int

const (
	l_literal LiteralType = iota
	l_name
	l_string
	l_regex
)

type Literal struct {
	Type  LiteralType
	Add   bool
	AddId string
	Value string
}

type Node any

type Peg struct {
	parser sc.Scanner
}

func GetPegParser(text string) Peg {
	return Peg{sc.GetScanner(text)}
}

func (s *Peg) grammar() (bool, Grammar) {
	nodes := []Rule{}
	if ok, rule := s.rule(); ok {
		nodes = append(nodes, rule)
		pos := s.parser.Mark()
		for {
			if ok, rule := s.rule(); ok {
				nodes = append(nodes, rule)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		if s.__() {
			if ok := s.parser.Expect(0); ok {
				return true, Grammar{nodes}
			}
		}
	}
	return false, Grammar{}
}

func (s *Peg) rule() (bool, Rule) {
	if ok, name := s.name(); ok {
		if s.__() {
			if ok := s.parser.String(":"); ok {
				if s.__() {
					if ok, body := s.body(); ok {
						if s.__() {
							return true, Rule{name, body}
						}
					}
				}
			}
		}
	}
	return false, Rule{}
}

func (s *Peg) body_1() (bool, Alternative) {
	if s.__() {
		if ok := s.parser.String("|"); ok {
			if s.__() {
				if ok, alternative := s.alternative(); ok {
					return true, alternative
				}
			}
		}
	}
	return false, Alternative{}
}

func (s *Peg) body() (bool, Body) {
	nodes := []Alternative{}
	if ok, alternative := s.alternative(); ok {
		nodes = append(nodes, alternative)
		pos := s.parser.Mark()
		for {
			if ok, body_1 := s.body_1(); ok {
				nodes = append(nodes, body_1)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return true, Body{nodes}
	}
	return false, Body{}
}

func (s *Peg) alternative_1() (bool, Loop) {
	if s.__() {
		if ok, loop := s.loop(); ok {
			return true, loop
		}
	}
	return false, Loop{}
}

func (s *Peg) alternative() (bool, Alternative) {
	nodes := []Loop{}
	if ok, loop := s.loop(); ok {
		nodes = append(nodes, loop)
		pos := s.parser.Mark()
		for {
			if ok, alternative_1 := s.alternative_1(); ok {
				nodes = append(nodes, alternative_1)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return true, Alternative{nodes}
	}
	return false, Alternative{}
}

func (s *Peg) loop_1() loop_mode {
	pos := s.parser.Mark()
	if ok := s.parser.String("+"); ok {
		return l_one_or_more
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("*"); ok {
		return l_zero_or_more
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("?"); ok {
		return l_zero_or_one
	}
	s.parser.Reset(pos)
	return l_none
}

func (s *Peg) loop() (bool, Loop) {
	not := false
	if ok := s.parser.String("!"); ok {
		not = true
	}
	if ok, atom := s.atom(); ok {
		return true, Loop{s.loop_1(), atom, not}
	}
	return false, Loop{}
}

func (s *Peg) atom() (bool, Node) {
	pos := s.parser.Mark()
	if ok, item := s.item(); ok {
		return true, item
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("("); ok {
		if s.__() {
			if ok, body := s.body(); ok {
				if s.__() {
					if ok := s.parser.String(")"); ok {
						return true, body
					}
				}
			}
		}
	}
	s.parser.Reset(pos)
	return false, nil
}

func (s *Peg) item_1() (bool, string) {
	addId := ""
	pos := s.parser.Mark()
	if ok, name := s.name(); ok {
		addId = name
	} else {
		s.parser.Reset(pos)
	}
	if ok := s.parser.String("."); ok {
		return true, addId
	}
	return false, ""
}

func (s *Peg) item_2() bool {
	if ok := s.__(); ok {
		if ok := s.parser.String(":"); ok {
			return true
		}
	}
	return false
}

func (s *Peg) item() (bool, Literal) {
	add := false
	addId := ""
	pos := s.parser.Mark()
	if ok, id := s.item_1(); ok {
		add = true
		addId = id
	} else {
		s.parser.Reset(pos)
	}
	pos = s.parser.Mark()
	if ok := s.parser.String("ENDMARKER"); ok {
		return true, Literal{l_literal, false, "", "ENDMARKER"}
	}
	s.parser.Reset(pos)
	if ok, name := s.name(); ok {
		if !s.item_2() {
			return true, Literal{l_name, add, addId, name}
		}
	}
	s.parser.Reset(pos)
	if ok, chars := s.chars(); ok {
		return true, Literal{l_string, add, addId, chars}
	}
	s.parser.Reset(pos)
	if ok, rgx := s.rgx(); ok {
		return true, Literal{l_regex, add, addId, rgx}
	}
	s.parser.Reset(pos)
	return false, Literal{}
}

func (s *Peg) name() (bool, string) {
	if ok, str := regex.RunRegex(&s.parser, "(\\w|\\a|_)+"); ok {
		return true, str
	}
	return false, ""
}

func (s *Peg) chars() (bool, string) {
	pos := s.parser.Mark()
	if ok := s.parser.String("'"); ok {
		if ok, str := regex.RunRegex(&s.parser, "((')!.)+"); ok {
			if ok := s.parser.String("'"); ok {
				return true, str
			}
		}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("\""); ok {
		if ok, str := regex.RunRegex(&s.parser, "((\")!.)+"); ok {
			if ok := s.parser.String("\""); ok {
				return true, str
			}
		}
	}
	s.parser.Reset(pos)
	return false, ""
}

func (s *Peg) rgx() (bool, string) {
	if ok := s.parser.String("["); ok {
		if ok, str := regex.RunRegex(&s.parser, "(\\[|\\])!.(((\\\\)!.(\\[|\\]))!.)+"); ok {
			if ok := s.parser.String("]"); ok {
				return true, str
			}
		}
	}
	return false, ""
}

func (s *Peg) __() bool {
	if ok, _ := regex.RunRegex(&s.parser, "( |\\t|\\r|\\n)*"); ok {
		return true
	}
	return false
}

func (s *Peg) Parse() (bool, Grammar) {
	return s.grammar()
}
