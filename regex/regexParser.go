package regex

type Node struct {
	Typ      string
	Value    string
	Children []Node
}

type Regex struct {
	parser Tokenizer
}

func GetRegexParser(text string) Regex {
	return Regex{GetTokenizer(text)}
}

func (s *Regex) regex() Node {
	nodes := []Node{}
	if ok := s.parser.String("["); ok {
		if capture := s.capture(); capture.Typ != "" {
			nodes = append(nodes, capture)
			if ok := s.parser.String("]"); ok {
				if ok := s.parser.Expect(0); ok {
					return Node{"regex", "", nodes}
				}
			}
		}
	}
	return Node{}
}

func (s *Regex) capture_1() Node {
	nodes := []Node{}
	if ok := s.parser.String("|"); ok {
		if group := s.group(); group.Typ != "" {
			nodes = append(nodes, group)
			return Node{"capture_1", "", nodes}
		}
	}
	return Node{}
}

func (s *Regex) capture() Node {
	nodes := []Node{}
	if group := s.group(); group.Typ != "" {
		nodes = append(nodes, group)
		pos := s.parser.Mark()
		for {
			if capture_1 := s.capture_1(); capture_1.Typ != "" {
				nodes = append(nodes, capture_1.Children...)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Node{"capture", "", nodes}
	}
	return Node{}
}

func (s *Regex) group() Node {
	nodes := []Node{}
	if mode := s.mode(); mode.Typ != "" {
		nodes = append(nodes, mode)
		pos := s.parser.Mark()
		for {
			if mode := s.mode(); mode.Typ != "" {
				nodes = append(nodes, mode)
				pos = s.parser.Mark()
			} else {
				break
			}
		}
		s.parser.Reset(pos)
		return Node{"group", "", nodes}
	}
	return Node{}
}

func (s *Regex) mode_1() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if ok := s.parser.String("?"); ok {
		nodes = append(nodes, Node{"string", "?", []Node{}})
		return Node{"mode_1", "", nodes}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("!"); ok {
		nodes = append(nodes, Node{"string", "!", []Node{}})
		return Node{"mode_1", "", nodes}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("*"); ok {
		nodes = append(nodes, Node{"string", "*", []Node{}})
		return Node{"mode_1", "", nodes}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("+"); ok {
		nodes = append(nodes, Node{"string", "+", []Node{}})
		return Node{"mode_1", "", nodes}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Regex) mode() Node {
	nodes := []Node{}
	if atom := s.atom(); atom.Typ != "" {
		nodes = append(nodes, atom)
		if mode_1 := s.mode_1(); mode_1.Typ != "" {
			nodes = append(nodes, mode_1.Children...)
			return Node{"mode", "", nodes}
		}
		return Node{"mode", "", nodes}
	}
	return Node{}
}

func (s *Regex) atom() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if char := s.char(); char.Typ != "" {
		nodes = append(nodes, char)
		if ok := s.parser.String("!"); ok {
			nodes = append(nodes, Node{"string", "!", []Node{}})
			return Node{"atom", "", nodes}
		}
		return Node{"atom", "", nodes}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("("); ok {
		if capture := s.capture(); capture.Typ != "" {
			nodes = append(nodes, capture)
			if ok := s.parser.String(")"); ok {
				return Node{"atom", "", nodes}
			}
		}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Regex) char() Node {
	nodes := []Node{}
	pos := s.parser.Mark()
	if ok := s.parser.String("\\"); ok {
		if ok, r := s.parser.Rune(); ok {
			if r == '(' || r == ')' || r == '[' || r == ']' || r == '+' || r == '*' || r == '?' || r == '!' || r == '|' || r == '.' {
				nodes = append(nodes, Node{"rune", string(r), []Node{}})
			} else {
				nodes = append(nodes, Node{"meta", string(r), []Node{}})
			}
			return Node{"char", "", nodes}
		}
	}
	s.parser.Reset(pos)
	if ok := s.parser.String("."); ok {
		nodes = append(nodes, Node{"meta", ".", []Node{}})
		return Node{"char", "", nodes}
	}
	s.parser.Reset(pos)
	if ok, r := s.parser.Rune(); ok {
		if r == '(' || r == ')' || r == '[' || r == ']' || r == '+' || r == '*' || r == '?' || r == '!' || r == '|' {
			return Node{}
		}
		nodes = append(nodes, Node{"rune", string(r), []Node{}})
		return Node{"char", "", nodes}
	}
	s.parser.Reset(pos)
	return Node{}
}

func (s *Regex) Parse() Node {
	return s.regex()
}
