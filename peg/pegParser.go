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

func (s *Peg) grammar() Node {
	nodes := []Rule{}
	if rule := s.rule(); rule != nil {
		nodes = append(nodes, rule.(Rule))
		pos := s.parser.Mark()
		for {
			if rule := s.rule(); rule != nil {
				nodes = append(nodes, rule.(Rule))
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		if s.__() {
			if ok := s.parser.Expect(0); ok {
				return Grammar{nodes}
			}
		}
	}
	return nil
}

func (s *Peg) rule() Node {
	if ok, name := s.name(); ok {
		if s.__() {
			if ok := s.parser.String(":"); ok {
				if s.__() {
					if body := s.body(); body != nil {
						if s.__() {
							if ok, _ := s.parser.Regex("\\r\\n"); ok {
								return Rule{name, body.(Body)}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *Peg) body_1() Node {
	if s.__() {
		if ok := s.parser.String("|"); ok {
			if s.__() {
				if alternative := s.alternative(); alternative != nil {
					return alternative
				}
			}
		}
	}
	return nil
}

func (s *Peg) body() Node {
	nodes := []Alternative{}
	if alternative := s.alternative(); alternative != nil {
		nodes = append(nodes, alternative.(Alternative))
		pos := s.parser.Mark()
		for {
			if body_1 := s.body_1(); body_1 != nil {
				nodes = append(nodes, body_1.(Alternative))
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Body{nodes}
	}
	return nil
}

func (s *Peg) alternative_1() Node {
	if s.__() {
		if loop := s.loop(); loop != nil {
			return loop
		}
	}
	return nil
}

func (s *Peg) alternative() Node {
	nodes := []Loop{}
	if loop := s.loop(); loop != nil {
		nodes = append(nodes, loop.(Loop))
		pos := s.parser.Mark()
		for {
			if alternative_1 := s.alternative_1(); alternative_1 != nil {
				nodes = append(nodes, alternative_1.(Loop))
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Alternative{nodes}
	}
	return nil
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

func (s *Peg) loop() Node {
	if atom := s.atom(); atom != nil {
		return Loop{s.loop_1(), atom}
	}
	return nil
}

func (s *Peg) atom() Node {
	pos := s.parser.Mark()
	if item := s.item(); item != nil {
		return item
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("("); ok {
		if s.__() {
			if body := s.body(); body != nil {
				if s.__() {
					if ok := s.parser.String(")"); ok {
						return body
					}
				}
			}
		}
	}
	s.parser.Reset(pos)
	return nil
}

func (s *Peg) item() Node {
	pos := s.parser.Mark()
	if ok := s.parser.String("ENDMARKER"); ok {
		return Literal{l_literal, "ENDMARKER"}
	}
	s.parser.Reset(pos)
	if ok, name := s.name(); ok {
		return Literal{l_name, name}
	}
	s.parser.Reset(pos)
	if ok, chars := s.chars(); ok {
		return Literal{l_string, chars}
	}
	s.parser.Reset(pos)
	if ok, regex := s.regex(); ok {
		return Literal{l_regex, regex}
	}
	s.parser.Reset(pos)
	return nil
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
		if ok, str := s.parser.Regex("([|])!.(((\\\\)!.([|]))!.)+"); ok {
			if ok := s.parser.String("]"); ok {
				return true, str
			}
		}
	}
	return false, ""
}

func (s *Peg) __() bool {
	if ok, _ := s.parser.Regex("( |\\t)*"); ok {
		return true
	}
	return false
}

func (s *Peg) Parse() Node {
	return s.grammar()
}
