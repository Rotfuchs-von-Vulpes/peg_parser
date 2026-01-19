package peg

import (
	"fmt"
	"pegParser/regex"
	sc "pegParser/scanner"
	"strings"
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
	Name string
	Body Body
}

type Body struct {
	Alts []Alternative
}

type Alternative struct {
	Loops []Loop
}

type loop_mode int

const (
	L_none loop_mode = iota
	L_zero_or_one
	L_zero_or_more
	L_one_or_more
)

type Loop struct {
	Mode  loop_mode
	Child Node
	Not   bool
}

type LiteralType int

const (
	L_literal LiteralType = iota
	L_name
	L_string
	L_regex
)

type Literal struct {
	Type  LiteralType
	Add   bool
	AddId string
	Value string
}

type Node any

type Peg struct {
	pos    int
	tokens []token
}

func (s *Peg) Mark() int {
	return s.pos
}

func (s *Peg) Reset(pos int) {
	s.pos = pos
}

func (s *Peg) Expect(typ tokenKind) (bool, string) {
	t := s.tokens[s.pos]
	if t.typ == typ {
		s.pos += 1
		return true, t.value
	}
	return false, ""
}

func (s *Peg) String(str string) bool {
	t := s.tokens[s.pos]
	if t.typ == t_punct && str == t.value {
		s.pos += 1
		return true
	}
	return false
}

func GetPegParser(text string) Peg {
	return Peg{0, tokenize(sc.GetScanner(text))}
}

func (s *Peg) grammar() (bool, Grammar) {
	nodes := []Rule{}
	if ok, rule := s.rule(); ok {
		nodes = append(nodes, rule)
		pos := s.Mark()
		for {
			if ok, rule := s.rule(); ok {
				nodes = append(nodes, rule)
				pos = s.Mark()
			} else {
				break
			}
		}
		s.Reset(pos)
		if ok, _ := s.Expect(t_end); ok {
			return true, Grammar{nodes}
		} else {
			fmt.Println(s.pos, s.tokens[s.pos])
		}
	}
	return false, Grammar{}
}

func (s *Peg) rule() (bool, Rule) {
	if ok, name := s.name(); ok {
		if ok := s.String(":"); ok {
			if ok, body := s.body(); ok {
				return true, Rule{name, body}
			}
		}
	}
	return false, Rule{}
}

func (s *Peg) body_1() (bool, Alternative) {
	if ok := s.String("|"); ok {
		if ok, alternative := s.alternative(); ok {
			return true, alternative
		}
	}
	return false, Alternative{}
}

func (s *Peg) body() (bool, Body) {
	nodes := []Alternative{}
	if ok, alternative := s.alternative(); ok {
		nodes = append(nodes, alternative)
		pos := s.Mark()
		for {
			if ok, body_1 := s.body_1(); ok {
				nodes = append(nodes, body_1)
				pos = s.Mark()
			} else {
				break
			}
		}
		s.Reset(pos)
		return true, Body{nodes}
	}
	return false, Body{}
}

func (s *Peg) alternative() (bool, Alternative) {
	nodes := []Loop{}
	if ok, loop := s.loop(); ok {
		nodes = append(nodes, loop)
		pos := s.Mark()
		for {
			if ok, loop := s.loop(); ok {
				nodes = append(nodes, loop)
				pos = s.Mark()
			} else {
				break
			}
		}
		s.Reset(pos)
		return true, Alternative{nodes}
	}
	return false, Alternative{}
}

func (s *Peg) loop_1() loop_mode {
	pos := s.Mark()
	if ok := s.String("+"); ok {
		return L_one_or_more
	}
	s.Reset(pos)
	if ok := s.String("*"); ok {
		return L_zero_or_more
	}
	s.Reset(pos)
	if ok := s.String("?"); ok {
		return L_zero_or_one
	}
	s.Reset(pos)
	return L_none
}

func (s *Peg) loop() (bool, Loop) {
	not := false
	if ok := s.String("!"); ok {
		not = true
	}
	if ok, atom := s.atom(); ok {
		return true, Loop{s.loop_1(), atom, not}
	}
	return false, Loop{}
}

func (s *Peg) atom() (bool, Node) {
	pos := s.Mark()
	if ok, item := s.item(); ok {
		return true, item
	}
	s.Reset(pos)
	if ok := s.String("("); ok {
		if ok, body := s.body(); ok {
			if ok := s.String(")"); ok {
				return true, body
			}
		}
	}
	s.Reset(pos)
	return false, nil
}

func (s *Peg) item_1() (bool, string) {
	addId := ""
	pos := s.Mark()
	if ok, name := s.name(); ok {
		addId = name
	} else {
		s.Reset(pos)
	}
	if ok := s.String("."); ok {
		return true, addId
	}
	return false, ""
}

func (s *Peg) item_2() bool {
	if ok := s.String(":"); ok {
		return true
	}
	return false
}

func (s *Peg) item() (bool, Literal) {
	add := false
	addId := ""
	pos := s.Mark()
	if ok, id := s.item_1(); ok {
		add = true
		addId = id
	} else {
		s.Reset(pos)
	}
	pos = s.Mark()
	if ok := s.String("ENDMARKER"); ok {
		fmt.Println("oxe")
		return true, Literal{L_literal, false, "", "ENDMARKER"}
	}
	s.Reset(pos)
	if ok, name := s.name(); ok {
		if !s.item_2() {
			return true, Literal{L_name, add, addId, name}
		}
	}
	s.Reset(pos)
	if ok, chars := s.chars(); ok {
		return true, Literal{L_string, add, addId, chars}
	}
	s.Reset(pos)
	if ok, rgx := s.rgx(); ok {
		return true, Literal{L_regex, add, addId, rgx}
	}
	s.Reset(pos)
	return false, Literal{}
}

func (s *Peg) name() (bool, string) {
	if ok, str := s.Expect(t_identifier); ok {
		return true, str
	}
	return false, ""
}

func (s *Peg) chars() (bool, string) {
	pos := s.Mark()
	if ok, str := s.Expect(t_string); ok {
		return true, str
	}
	s.Reset(pos)
	if ok, str := s.Expect(t_string); ok {
		return true, str
	}
	s.Reset(pos)
	return false, ""
}

func (s *Peg) rgx() (bool, string) {
	if ok, str := s.Expect(t_regex); ok {
		return true, str
	}
	return false, ""
}

type tokenKind int

const (
	t_space tokenKind = iota
	t_string
	t_identifier
	t_regex
	t_punct
	t_end
)

type token struct {
	typ   tokenKind
	value string
}

func tokenize(scanner sc.Scanner) (final []token) {
	last_literal := strings.Builder{}
	add := func(typ tokenKind, str string) {
		if last_literal.Len() > 0 {
			final = append(final, token{t_punct, last_literal.String()})
			last_literal.Reset()
		}
		if typ != t_space {
			final = append(final, token{typ, str})
		}
	}
	for {
		if scanner.Expect(0) {
			break
		}
		pos := scanner.Mark()
		if ok, _ := regex.RunRegex(&scanner, "( |\\t|\\n|\\r)+"); ok {
			add(t_space, "")
			continue
		}
		scanner.Reset(pos)
		if ok := scanner.String("ENDMARKER"); ok {
			add(t_punct, "ENDMARKER")
			continue
		}
		scanner.Reset(pos)
		if ok, str := regex.RunRegex(&scanner, "(\\w|\\a|_)+"); ok {
			add(t_identifier, str)
			continue
		}
		scanner.Reset(pos)
		if ok := scanner.String("\""); ok {
			if ok, str := regex.RunRegex(&scanner, "((\")!.)+"); ok {
				if ok := scanner.String("\""); ok {
					add(t_string, str)
					continue
				}
			}
		}
		scanner.Reset(pos)
		if ok := scanner.String("'"); ok {
			if ok, str := regex.RunRegex(&scanner, "((')!.)+"); ok {
				if ok := scanner.String("'"); ok {
					add(t_string, str)
					continue
				}
			}
		}
		scanner.Reset(pos)
		if ok := scanner.String("["); ok {
			if ok, str := regex.RunRegex(&scanner, "\\b+"); ok {
				if ok := scanner.String("]"); ok {
					add(t_regex, str)
					continue
				}
			}
		}
		scanner.Reset(pos)
		if ok, r := scanner.Rune(); ok {
			last_literal.WriteRune(r)
			continue
		} else {
			break
		}
	}
	final = append(final, token{t_end, ""})
	fmt.Println(final)
	return final
}

func (s *Peg) Parse() (bool, Grammar) {
	return s.grammar()
}
