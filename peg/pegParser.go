package peg

import (
	"main/parser"
	"strings"
)

type Node struct {
	typ      string
	value    string
	children []Node
}

type Peg struct {
	parser parser.Tokenizer
}

func GetPegParser(text string) Peg {
	return Peg{parser.GetTokenizer(text)}
}

func (s *Peg) grammar() Node {
	pos := s.parser.Mark()
	if rule := s.rule(); rule.typ != "" {
		rules := []Node{rule}
		for {
			if rule := s.rule(); rule.typ != "" {
				rules = append(rules, rule)
			} else {
				break
			}
		}
		return Node{"grammar", "", rules}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) rule() Node {
	if name := s.name(); name.typ != "" {
		if ok := s.parser.String(": "); ok {
			if body := s.body(); body.typ != "" {
				if ok := s.parser.String("\r\n"); ok {
					return Node{"rule", name.value, []Node{body}}
				}
			}
		}
	}
	return Node{}
}

func (s *Peg) body() Node {
	if alt := s.alternative(); alt.typ != "" {
		alts := []Node{alt}
		pos := s.parser.Mark()
		for {
			if ok := s.parser.String(" | "); ok {
				if alt := s.alternative(); alt.typ != "" {
					alts = append(alts, alt)
					pos = s.parser.Mark()
				} else {
					break
				}
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Node{"body", "", alts}
	}
	return Node{}
}

func (s *Peg) alternative() Node {
	loops := []Node{}
	if loop := s.loop(); loop.typ != "" {
		loops = append(loops, loop)
		pos := s.parser.Mark()
		for {
			if ok := s.parser.Expect(' '); ok {
				if loop := s.loop(); loop.typ != "" {
					loops = append(loops, loop)
					pos = s.parser.Mark()
				} else {
					break
				}
			} else {
				break
			}
		}
		s.parser.Reset(pos)
	}
	if len(loops) > 0 {
		return Node{"alts", "", loops}
	}
	return Node{}
}

func (s *Peg) loop() Node {
	if atom := s.atom(); atom.typ != "" {
		if ok := s.parser.Expect('+'); ok {
			return Node{"loop", "+", []Node{atom}}
		} else if ok := s.parser.Expect('*'); ok {
			return Node{"loop", "*", []Node{atom}}
		} else if ok := s.parser.Expect('?'); ok {
			return Node{"loop", "?", []Node{atom}}
		}
		return Node{"loop", "", []Node{atom}}
	}
	return Node{}
}

func (s *Peg) atom() Node {
	if item := s.item(); item.typ != "" {
		return Node{"atom", "", []Node{item}}
	}
	if ok := s.parser.Expect('('); ok {
		if body := s.body(); body.typ != "" {
			if ok := s.parser.Expect(')'); ok {
				return Node{"atom", "", []Node{body}}
			}
		}
	}
	return Node{}
}

func (s *Peg) item() Node {
	if ok := s.parser.String("ENDMARKER"); ok {
		return Node{"literal", "ENDMARKER", []Node{}}
	}
	if ok := s.parser.String("NEWLINE"); ok {
		return Node{"literal", "NEWLINE", []Node{}}
	}
	if ok := s.parser.String("RUNE"); ok {
		return Node{"literal", "RUNE", []Node{}}
	}
	if ok := s.parser.String("NUMBER"); ok {
		return Node{"literal", "NUMBER", []Node{}}
	}
	if ok := s.parser.String("NAME"); ok {
		return Node{"literal", "NAME", []Node{}}
	}
	if ok := s.parser.String("HIGH_LETTER"); ok {
		return Node{"literal", "HIGH_LETTER", []Node{}}
	}
	if ok := s.parser.String("LOW_LETTER"); ok {
		return Node{"literal", "LOW_LETTER", []Node{}}
	}
	if ok := s.parser.String("LETTER"); ok {
		return Node{"literal", "LETTER", []Node{}}
	}
	if name := s.name(); name.typ != "" {
		return Node{"item", "", []Node{name}}
	}
	if str := s.string(); str.typ != "" {
		return Node{"item", "", []Node{str}}
	}
	return Node{}
}

func (s *Peg) name() Node {
	name := strings.Builder{}
	for {
		if ok, r := s.parser.Letter(); ok {
			name.WriteRune(r)
		} else if ok := s.parser.Expect('_'); ok {
			name.WriteRune('_')
		} else {
			break
		}
	}
	if name.Len() > 0 {
		return Node{"name", name.String(), []Node{}}
	}
	return Node{}
}

func (s *Peg) string() Node {
	str := strings.Builder{}
	pos := s.parser.Mark()
	if ok := s.parser.Expect('"'); ok {
		pos := s.parser.Mark()
		for {
			if ok, r := s.parser.Rune(); ok {
				if r != '"' {
					pos = s.parser.Mark()
					str.WriteRune(r)
				} else {
					break
				}
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		if ok := s.parser.Expect('"'); ok {
			if str.Len() > 0 {
				return Node{"string", str.String(), []Node{}}
			}
		}
	}
	s.parser.Reset(pos)
	if ok := s.parser.Expect('\''); ok {
		pos := s.parser.Mark()
		for {
			if ok, r := s.parser.Rune(); ok {
				if r != '\'' {
					pos = s.parser.Mark()
					str.WriteRune(r)
				} else {
					break
				}
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		if ok := s.parser.Expect('\''); ok {
			if str.Len() > 0 {
				return Node{"string", str.String(), []Node{}}
			}
		}
	}
	return Node{}
}

func (s *Peg) Parse() Node {
	return s.grammar()
}
