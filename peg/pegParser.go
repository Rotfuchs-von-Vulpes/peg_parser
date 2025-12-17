package peg

import (
	"main/parser"
)

type Node struct {
	Typ      string
	Value    string
	Children []Node
}

type Peg struct {
	parser parser.Tokenizer
}

func GetPegParser(text string) Peg {
	return Peg{parser.GetTokenizer(text)}
}

func (s *Peg) grammar() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if rule := s.rule(); rule.Typ != "" {
		nodes = append(nodes, rule)
		pos := s.parser.Mark()
		for {
			if rule := s.rule(); rule.Typ != "" {
				nodes = append(nodes, rule)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		if __ := s.__(); __.Typ != "" {
			if ok := s.parser.Expect(0); ok {
				return Node{"grammar", "", nodes}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) rule() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if name := s.name(); name.Typ != "" {
		if __ := s.__(); __.Typ != "" {
			if ok := s.parser.String(":"); ok {
				if __ := s.__(); __.Typ != "" {
					if body := s.body(); body.Typ != "" {
						nodes = append(nodes, body)
						if __ := s.__(); __.Typ != "" {
							if ok, _ := s.parser.Regex("[\\r\\n]"); ok {
								return Node{"rule", name.Value, nodes}
							}
						}
					}
				}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) body_1() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if __ := s.__(); __.Typ != "" {
		if ok := s.parser.String("|"); ok {
			if __ := s.__(); __.Typ != "" {
				if alternative := s.alternative(); alternative.Typ != "" {
					nodes = append(nodes, alternative)
					return Node{"body_1", "", nodes}
				}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) body() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if alternative := s.alternative(); alternative.Typ != "" {
		nodes = append(nodes, alternative)
		pos := s.parser.Mark()
		for {
			if body_1 := s.body_1(); body_1.Typ != "" {
				nodes = append(nodes, body_1.Children...)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Node{"body", "", nodes}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) alternative_1() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if __ := s.__(); __.Typ != "" {
		if loop := s.loop(); loop.Typ != "" {
			nodes = append(nodes, loop)
			return Node{"alternative_1", "", nodes}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) alternative() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if loop := s.loop(); loop.Typ != "" {
		nodes = append(nodes, loop)
		pos := s.parser.Mark()
		for {
			if alternative_1 := s.alternative_1(); alternative_1.Typ != "" {
				nodes = append(nodes, alternative_1.Children...)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Node{"alternative", "", nodes}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) loop_1() string {
	pos := s.parser.Mark()
	if ok := s.parser.String("+"); ok {
		return "+"
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("*"); ok {
		return "*"
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("?"); ok {
		return "?"
	}
	s.parser.Reset(pos)
	return ""
}

func (s *Peg) loop() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if atom := s.atom(); atom.Typ != "" {
		nodes = append(nodes, atom)
		return Node{"loop", s.loop_1(), nodes}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) atom() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if item := s.item(); item.Typ != "" {
		nodes = append(nodes, item)
		return Node{"atom", "", nodes}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("("); ok {
		if __ := s.__(); __.Typ != "" {
			if body := s.body(); body.Typ != "" {
				nodes = append(nodes, body)
				if __ := s.__(); __.Typ != "" {
					if ok := s.parser.String(")"); ok {
						return Node{"atom", "", nodes}
					}
				}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) item() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if ok := s.parser.String("ENDMARKER"); ok {
		return Node{"literal", "ENDMARKER", []Node{}}
	}
	s.parser.Reset(pos)
	if name := s.name(); name.Typ != "" {
		nodes = append(nodes, name)
		return Node{"item", "", nodes}
	}
	s.parser.Reset(pos)
	if chars := s.chars(); chars.Typ != "" {
		nodes = append(nodes, chars)
		return Node{"item", "", nodes}
	}
	s.parser.Reset(pos)
	if regex := s.regex(); regex.Typ != "" {
		nodes = append(nodes, regex)
		return Node{"item", "", nodes}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) name() Node {
	pos := s.parser.Mark()
	if ok, str := s.parser.Regex("[(\\w|\\a|_)+]"); ok {
		return Node{"name", str, []Node{}}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) chars() Node {
	pos := s.parser.Mark()
	if ok := s.parser.String("'"); ok {
		if ok, str := s.parser.Regex("['!+]"); ok {
			if ok := s.parser.String("'"); ok {
				return Node{"chars", str, []Node{}}
			}
		}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("\""); ok {
		if ok, str := s.parser.Regex("[\"!+]"); ok {
			if ok := s.parser.String("\""); ok {
				return Node{"chars", str, []Node{}}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) regex() Node {
	pos := s.parser.Mark()
	if ok := s.parser.String("["); ok {
		if ok, str := s.parser.Regex("[\\b+]"); ok {
			if ok := s.parser.String("]"); ok {
				return Node{"regex", "[" + str + "]", []Node{}}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) __() Node {
	pos := s.parser.Mark()
	if ok, _ := s.parser.Regex("[( |\\t)*]"); ok {
		return Node{"__", "", []Node{}}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Peg) Parse() Node {
	return s.grammar()
}
