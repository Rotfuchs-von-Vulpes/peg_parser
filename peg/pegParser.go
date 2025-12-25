package peg

import (
	"main/parser"
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
	Value string
}

type Node any

type Peg struct {
	parser parser.Tokenizer
}

func GetPegParser(text string) Peg {
	return Peg{parser.GetTokenizer(text)}
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

func (s *Peg) item_1() bool {
	if ok := s.__(); ok {
		if ok := s.parser.String(":"); ok {
			return true
		}
	}
	return false
}

func (s *Peg) item() (bool, Literal) {
	pos := s.parser.Mark()
	if ok := s.parser.String("ENDMARKER"); ok {
		return true, Literal{l_literal, "ENDMARKER"}
	}
	s.parser.Reset(pos)
	if ok, name := s.name(); ok {
		if !s.item_1() {
			return true, Literal{l_name, name}
		}
	}
	s.parser.Reset(pos)
	if ok, chars := s.chars(); ok {
		return true, Literal{l_string, chars}
	}
	s.parser.Reset(pos)
	if ok, regex := s.regex(); ok {
		return true, Literal{l_regex, regex}
	}
	s.parser.Reset(pos)
	return false, Literal{}
}

func (s *Peg) name() (bool, string) {
	if ok, str := s.parser.Regex("(\\w|\\a|_)+"); ok {
		return true, str
	}
	return false, ""
}

func (s *Peg) chars() (bool, string) {
	pos := s.parser.Mark()
	if ok := s.parser.String("'"); ok {
		if ok, str := s.parser.Regex("((')!.)+"); ok {
			if ok := s.parser.String("'"); ok {
				return true, str
			}
		}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("\""); ok {
		if ok, str := s.parser.Regex("((\")!.)+"); ok {
			if ok := s.parser.String("\""); ok {
				return true, str
			}
		}
	}
	s.parser.Reset(pos)
	return false, ""
}

func (s *Peg) regex() (bool, string) {
	if ok := s.parser.String("["); ok {
		if ok, str := s.parser.Regex("(\\[|\\])!.(((\\\\)!.(\\[|\\]))!.)+"); ok {
			if ok := s.parser.String("]"); ok {
				return true, str
			}
		}
	}
	return false, ""
}

func (s *Peg) __() bool {
	if ok, _ := s.parser.Regex("( |\\t|\\r|\\n)*"); ok {
		return true
	}
	return false
}

func (s *Peg) Parse() (bool, Grammar) {
	return s.grammar()
}
