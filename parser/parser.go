package parser

import (
	"fmt"
	"strings"
)

type Tokengen struct {
	text  string
	runes []rune
	pos   int
}

func getTokengen(text string) Tokengen {
	runes := []rune{}
	for _, r := range text {
		runes = append(runes, r)
	}
	runes = append(runes, 0)
	return Tokengen{text, runes, 0}
}

func (s *Tokengen) next() rune {
	if s.pos >= len(s.runes) {
		return 0
	}
	r := s.runes[s.pos]
	s.pos += 1
	return r
}

type Tokenizer struct {
	tokengen Tokengen
	runes    []rune
	pos      int
}

func getTokenizer(text string) Tokenizer {
	return Tokenizer{
		getTokengen(text),
		[]rune{},
		0,
	}
}

func (s *Tokenizer) peekRune() rune {
	if s.pos == len(s.runes) {
		s.runes = append(s.runes, s.tokengen.next())
	}
	return s.runes[s.pos]
}

func (s *Tokenizer) mark() int {
	return s.pos
}

func (s *Tokenizer) reset(p int) {
	s.pos = p
}

func (s *Tokenizer) getRune() rune {
	r := s.peekRune()
	s.pos = s.pos + 1
	return r
}

type Parser struct {
	tokenizer Tokenizer
}

func GetParser(text string) Parser {
	return Parser{getTokenizer(text)}
}

func (s *Parser) Mark() int {
	return s.tokenizer.mark()
}

func (s *Parser) Reset(p int) {
	s.tokenizer.reset(p)
}

func (s *Parser) Expect(arg rune) bool {
	if arg == 0 {
		for {
			if ok := s.Expect(' '); ok {
				continue
			} else if ok := s.Expect('\n'); ok {
				continue
			} else if ok := s.Expect('\r'); ok {
				continue
			}
			break
		}
	}
	r := s.tokenizer.peekRune()
	if r == arg {
		s.tokenizer.pos += 1
		return true
	}
	return false
}

func (s *Parser) Rune() (bool, rune) {
	r := s.tokenizer.peekRune()
	if r == 0 {
		return false, 0
	} else {
		return true, s.tokenizer.getRune()
	}
}

func (s *Parser) String(arg string) bool {
	if arg == "" {
		return false
	}
	pos := s.Mark()
	for _, r1 := range arg {
		ok, r2 := s.Rune()
		if ok {
			if r1 != r2 {
				if arg == "\r\n" {
					fmt.Println(r2)
				}
				s.Reset(pos)
				return false
			}
		} else {
			s.Reset(pos)
			return false
		}
	}
	return true
}

func (s *Parser) LowLetter() (bool, rune) {
	r := s.tokenizer.peekRune()
	if r < 'a' || r > 'z' {
		return false, 0
	} else {
		return true, s.tokenizer.getRune()
	}
}

func (s *Parser) HighLetter() (bool, rune) {
	r := s.tokenizer.peekRune()
	if r < 'A' || r > 'Z' {
		return false, 0
	} else {
		return true, s.tokenizer.getRune()
	}
}

func (s *Parser) Letter() (bool, rune) {
	if ok, r := s.HighLetter(); !ok {
		if ok, r := s.LowLetter(); ok {
			return true, r
		} else {
			return false, 0
		}
	} else {
		return true, r
	}
}

func (s *Parser) Num() (bool, rune) {
	r := s.tokenizer.peekRune()
	if (r < '1' || r > '9') && r != '0' {
		return false, 0
	} else {
		return true, s.tokenizer.getRune()
	}
}

func (s *Parser) Number() (bool, string) {
	name := strings.Builder{}
	no_point := false
	for {
		if ok, r := s.Num(); ok {
			name.WriteRune(r)
		} else if ok := s.Expect('.'); ok && !no_point {
			no_point = true
			name.WriteRune('.')
		} else {
			break
		}
	}
	if name.Len() > 0 {
		return true, name.String()
	} else {
		return false, ""
	}
}

func (s *Parser) Name() (bool, string) {
	name := strings.Builder{}
	for {
		if ok, r := s.Letter(); ok {
			name.WriteRune(r)
		} else if ok, r := s.Num(); ok {
			name.WriteRune(r)
		} else if ok := s.Expect('_'); ok {
			name.WriteRune('_')
		} else {
			break
		}
	}
	if name.Len() > 0 {
		return true, name.String()
	} else {
		return false, ""
	}
}
