package peg

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

type RuleProg struct {
	subRules     []RuleProg
	lang         string
	name         string
	program      string
	tabs         int
	subRuleCount int
}

func (s *RuleProg) getTabs(c int) string {
	final := strings.Builder{}
	final.WriteRune('\n')
	count := c
	for {
		final.WriteRune('\t')
		count -= 1
		if count == 0 {
			return final.String()
		}
	}
}

func (s *RuleProg) close() {
	s.program = strings.Replace(s.program, "@1", "", 1)
	if s.tabs > 1 {
		s.tabs -= 1
	}
}

func (s *RuleProg) write(code string) {
	if strings.Contains(s.program, "@1") {
		code = strings.ReplaceAll(code, "@3", s.getTabs(s.tabs))
		s.program = strings.Replace(s.program, "@1", "%s", 1)
		s.program = fmt.Sprintf(s.program, code)
	} else {
		fmt.Println("Não existe code point para inserir codigo, regra: " + s.name + ", codigo: " + code)
	}
}

func (s *RuleProg) create() {
	s.program += fmt.Sprintf("func (s *%s) %s () Node {\n\tnodes := []Node{}@1\n}", s.lang, s.name)
}

func (s *RuleProg) writeCloseCatcher() {
	s.write("@1@1")
}

func (s *RuleProg) writeReturnNull() {
	s.write("@3return Node{}")
	if s.tabs > 1 {
		s.tabs -= 1
	}
}

func (s *RuleProg) writeReturn() {
	s.write(fmt.Sprintf("@3return Node{\"%s\", \"\", nodes}", s.name))
	if s.tabs > 1 {
		s.tabs -= 1
	}
}

func (s *RuleProg) writeMark(old bool) {
	if old {
		s.write("@3pos = s.parser.Mark()@1")
	} else {
		s.write("@3pos := s.parser.Mark()@1")
	}
}

func (s *RuleProg) writeNewPos() {
	s.write("@3pos = s.parser.Mark()@1")
}

func (s *RuleProg) writeReset() {
	s.write("@3s.parser.Reset(pos)@1")
}

func (s *RuleProg) writeIf(ifStr string) {
	s.write(fmt.Sprintf("@3%s {@1@3}@1", ifStr))
	s.tabs += 1
}

func (s *RuleProg) writeRuleFor() {
	s.write("@3for {@1@3}@1")
	s.tabs += 1
}

func (s *RuleProg) writeElseBreak() {
	s.write(" else {@3\tbreak@3}")
	s.tabs -= 1
}

func (s *RuleProg) addRule(rule string, add bool, not bool) {
	if rule == "" {
		panic("empty rule name")
	}
	if not {
		s.writeIf(fmt.Sprintf("if %s := s.%s(); %s.Typ == \"\"", rule, rule, rule))
	} else {
		s.writeIf(fmt.Sprintf("if %s := s.%s(); %s.Typ != \"\"", rule, rule, rule))
	}
	if add {
		s.write(fmt.Sprintf("@3nodes = append(nodes, %s.Children...)@1", rule))
	} else {
		s.write(fmt.Sprintf("@3nodes = append(nodes, %s)@1", rule))
	}
}

func (s *RuleProg) addString(str string, add, not bool) {
	if str == "" {
		panic("empty string")
	}
	final := strings.Builder{}
	for _, r := range str {
		switch r {
		case '\\':
			final.WriteString("\\\\")
		case '"':
			final.WriteString("\\\"")
		default:
			final.WriteRune(r)
		}
	}
	if not {
		s.writeIf("if ok := s.parser.String(\"" + final.String() + "\"); !ok")
	} else {
		s.writeIf(fmt.Sprintf("if ok := s.parser.String(\"%s\"); ok", final.String()))
		if add {
			s.write(fmt.Sprintf("@3nodes = append(nodes, Node{\"string\", \"%s\", []Node{}})@1", final.String()))
		}
	}
}

func (s *RuleProg) addLiteral(literal string, not bool) {
	switch literal {
	case "ENDMARKER":
		if not {
			s.writeIf("if ok := s.parser.Expect(0); !ok")
		} else {
			s.writeIf("if ok := s.parser.Expect(0); ok")
		}
	default:
		panic("unknow literal: " + literal)
	}
}

func (s *RuleProg) addSubRule(body Body) {
	s.subRuleCount += 1
	subRule := newRule(s.lang, s.name+"_"+strconv.Itoa(s.subRuleCount))
	subRule.body(body)
	subRule.writeReturnNull()
	s.subRules = append(s.subRules, subRule)
}

func bakeString(str string) string {
	final := strings.Builder{}
	for _, r := range str {
		switch r {
		case '"':
			final.WriteString("\\\"")
		case '\\':
			final.WriteString("\\\\")
		default:
			final.WriteRune(r)
		}
	}
	return final.String()
}

func (s *RuleProg) addRegex(regex string, not bool) {
	if not {
		s.writeIf("if ok, _ := s.parser.Regex(\"" + bakeString(regex) + "\"); !ok")
	} else {
		s.writeIf("if ok, str := s.parser.Regex(\"" + bakeString(regex) + "\"); ok")
		s.write("@3nodes = append(nodes, Node{\"string\", str, []Node{}})@1")
	}
}

func (s *RuleProg) atom(atom Node, variable bool, not bool) bool {
	if atom == nil {
		panic("atom is nil")
	}
	final := false
	if literal, ok := atom.(Literal); ok {
		switch literal.Type {
		case l_literal:
			s.addLiteral(literal.Value, not)
		case l_name:
			s.addRule(literal.Value, false, not)
		case l_regex:
			s.addRegex(literal.Value, not)
		case l_string:
			s.addString(literal.Value, variable, not)
		}
	} else if body, ok := atom.(Body); ok {
		s.addSubRule(body)
		s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true, not)
		final = true
	} else {
		panic("atom has illegal type")
	}
	return final
}

func (s *RuleProg) loop(loop Loop, various, pos_is_added bool) bool {
	add_pos := pos_is_added
	switch loop.Mode {
	case l_none:
		s.atom(loop.Child, various, loop.Not)
	case l_zero_or_one:
		s.atom(loop.Child, true, loop.Not)
		s.close()
		s.writeCloseCatcher()
	case l_zero_or_more:
		s.writeMark(add_pos)
		s.writeRuleFor()
		s.atom(loop.Child, various, loop.Not)
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		s.writeCloseCatcher()
		add_pos = true
	case l_one_or_more:
		sub := s.atom(loop.Child, various, loop.Not)
		s.writeMark(add_pos)
		s.writeRuleFor()
		if sub {
			s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true, loop.Not)
		} else {
			s.atom(loop.Child, various, loop.Not)
		}
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		add_pos = true
	}
	return add_pos
}

func (s *RuleProg) alternative(alt Alternative, variable bool) {
	if len(alt.Loops) == 0 {
		panic("zero loop as unexpected")
	}
	add_pos := false
	for _, loop := range alt.Loops {
		add_pos = s.loop(loop, variable, add_pos)
	}
	for i := range alt.Loops {
		if i == 0 {
			s.writeReturn()
		} else {
			s.close()
		}
	}
}

func (s *RuleProg) body(body Body) {
	if len(body.Alts) == 0 {
		panic("no body children as unexpected")
	}
	if len(body.Alts) > 1 {
		variable := false
		for _, alt := range body.Alts {
			if len(alt.Loops) == 1 {
				loop := alt.Loops[0]
				atom := loop.Child
				if literal, ok := atom.(Literal); ok {
					if literal.Type == l_string {
						variable = true
						break
					}
				}
			}
		}
		s.writeMark(false)
		for _, alt := range body.Alts {
			s.alternative(alt, variable)
			s.writeReset()
		}
	} else {
		s.alternative(body.Alts[0], false)
	}
}

type PegCompiler struct {
	data        Grammar
	lang        string
	rules       []RuleProg
	writingRule RuleProg
}

func GetPegCompiler(data Grammar, lang string) PegCompiler {
	return PegCompiler{data, lang, []RuleProg{}, RuleProg{}}
}

func newRule(lang, name string) RuleProg {
	r := RuleProg{[]RuleProg{}, lang, name, "", 1, 0}
	r.create()
	return r
}

func (s *PegCompiler) rule(rule Rule) {
	r := newRule(s.lang, rule.name)
	r.body(rule.body)
	r.writeReturnNull()
	s.rules = append(s.rules, r)
}

func (s *PegCompiler) eachSubRule(rule RuleProg) string {
	var final strings.Builder
	for _, subRule := range rule.subRules {
		final.WriteString(s.eachSubRule(subRule))
		final.WriteString(subRule.program + "\n\n")
	}
	return final.String()
}

func (s *PegCompiler) Compile(path string) {
	name := s.lang
	langB := strings.Builder{}
	for i, r := range s.lang {
		if i == 0 {
			str := strings.ToUpper(string(r))
			langB.WriteString(str)
		} else {
			langB.WriteRune(r)
		}
	}
	s.lang = langB.String()
	for _, rule := range s.data.Rules {
		s.rule(rule)
	}
	finalProg := `package @2

import (
	"main/parser"
)

type Node struct {
	Typ      string
	Value    string
	Children []Node
}

type @1 struct {
	parser parser.Tokenizer
}

func Get@1Parser(text string) @1 {
	return @1{parser.GetTokenizer(text)}
}

`
	finalProg = strings.ReplaceAll(finalProg, "@1", s.lang)
	finalProg = strings.ReplaceAll(finalProg, "@2", name)
	for _, rule := range s.rules {
		finalProg += s.eachSubRule(rule)
		finalProg += rule.program + "\n\n"
	}
	finalProg += "func (s *" + s.lang + ") Parse() Node {\n\treturn s." + s.data.Rules[0].name + "()\n}"
	os.Mkdir(path+name+"/", os.ModePerm)
	os.Remove(fmt.Sprintf(path+"%s/%s.go", name, name))
	os.WriteFile(fmt.Sprintf(path+"%s/%s.go", name, name), []byte(finalProg), fs.ModeAppend)
}
