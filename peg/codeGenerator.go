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
	pointCount   int
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
	s.pointCount = strings.Count(s.program, "@1")
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

func (s *RuleProg) writeMark() {
	s.write("@3pos := s.parser.Mark()@1")
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

func (s *RuleProg) addRule(rule string, add bool) {
	if rule == "" {
		panic("empty rule name")
	}
	s.writeIf(fmt.Sprintf("if %s := s.%s(); %s.Typ != \"\"", rule, rule, rule))
	if add {
		s.write(fmt.Sprintf("@3nodes = append(nodes, %s.Children...)@1", rule))
	} else {
		s.write(fmt.Sprintf("@3nodes = append(nodes, %s)@1", rule))
	}
}

func (s *RuleProg) addString(str string) {
	if str == "" {
		panic("empty string")
	}
	if str == "\"" {
		str = "\\\""
	}
	s.writeIf(fmt.Sprintf("if ok := s.parser.String(\"%s\"); ok", str))
}

func (s *RuleProg) addLiteral(literal string) {
	switch literal {
	case "ENDMARKER":
		s.writeIf("if ok := s.parser.Expect(0); ok")
	case "NEWLINE":
		s.writeIf("if ok := s.parser.String(\"\\r\\n\"); ok")
	case "RUNE":
		s.writeIf("if ok, r := s.parser.Rune(); ok")
		s.write("@3nodes = append(nodes, Node{\"rune\", string(r), []Node{}})@1")
	case "NUMBER":
		s.writeIf("if ok, number := s.parser.Number(); ok")
		s.write("@3nodes = append(nodes, Node{\"number\", number, []Node{}})@1")
	case "HIGH_LETTER":
		s.writeIf("if ok, r := s.parser.HighLetter(); ok")
		s.write("@3nodes = append(nodes, Node{\"rune\", string(r), []Node{}})@1")
	case "LOW_LETTER":
		s.writeIf("if ok, r := s.parser.LowLetter(); ok")
		s.write("@3nodes = append(nodes, Node{\"rune\", string(r), []Node{}})@1")
	case "LETTER":
		s.writeIf("if ok, r := s.parser.Letter(); ok")
		s.write("@3nodes = append(nodes, Node{\"rune\", string(r), []Node{}})@1")
	case "NAME":
		s.writeIf("if ok, name := s.parser.Name(); ok")
		s.write("@3nodes = append(nodes, Node{\"name\", name, []Node{}})@1")
	default:
		panic(fmt.Sprintf("unknow literal: %s", literal))
	}
}

func (s *RuleProg) addSubRule(body Node) {
	s.subRuleCount += 1
	subRule := newRule(s.lang, s.name+"_"+strconv.Itoa(s.subRuleCount))
	subRule.body(body)
	subRule.writeReturnNull()
	s.subRules = append(s.subRules, subRule)
}

func getUnexpectedTypeError(want string, get string) string {
	return fmt.Sprintf("This is what you want: %s, this is what you get: %s", want, get)
}

func (s *RuleProg) item(item Node) {
	if item.typ != "item" {
		panic(getUnexpectedTypeError("item", item.typ))
	}
	if len(item.children) == 0 {
		panic("no item child as unexpected")
	} else if len(item.children) != 1 {
		panic("too much item children")
	}
	child := item.children[0]
	switch child.typ {
	case "name":
		s.addRule(child.value, false)
	case "string":
		s.addString(child.value)
	case "literal":
		s.addLiteral(child.value)
	default:
		panic(fmt.Sprintf("item has illegal child: %s", child.typ))
	}
}

func (s *RuleProg) atom(atom Node) bool {
	if atom.typ != "atom" {
		panic(getUnexpectedTypeError("atom", atom.typ))
	}
	if len(atom.children) == 0 {
		panic("no atom child as unexpected")
	} else if len(atom.children) != 1 {
		panic("too much atom children")
	}
	final := false
	child := atom.children[0]
	switch child.typ {
	case "item":
		s.item(child)
	case "name":
		s.addRule(child.value, false)
	case "string":
		s.addString(child.value)
	case "literal":
		s.addLiteral(child.value)
	case "body":
		s.addSubRule(child)
		s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true)
		final = true
	default:
		panic(fmt.Sprintf("atom has illegal child: %s", child.typ))
	}
	return final
}

func (s *RuleProg) loop(loop Node) {
	if loop.typ != "loop" {
		panic(getUnexpectedTypeError("loop", loop.typ))
	}
	if len(loop.children) == 0 {
		panic("zero atom as unexpected")
	} else if len(loop.children) != 1 {
		panic("too much loop")
	}
	switch loop.value {
	case "":
		s.atom(loop.children[0])
	case "?":
		s.atom(loop.children[0])
		s.writeReturn()
		s.writeCloseCatcher()
	case "*":
		child := loop.children[0]
		s.writeMark()
		s.writeRuleFor()
		s.atom(child)
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		s.writeCloseCatcher()
	case "+":
		child := loop.children[0]
		sub := s.atom(child)
		s.writeMark()
		s.writeRuleFor()
		if sub {
			s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true)
		} else {
			s.atom(child)
		}
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
	default:
		panic("unknow repeat operator")
	}
}

func (s *RuleProg) alt(alt Node) {
	if alt.typ != "alts" {
		panic(getUnexpectedTypeError("alts", alt.typ))
	}
	if len(alt.children) == 0 {
		panic("zero loop as unexpected")
	}
	for _, loop := range alt.children {
		s.loop(loop)
	}
	for i := range alt.children {
		if i == 0 {
			s.writeReturn()
		} else {
			s.close()
		}
	}
}

func (s *RuleProg) body(body Node) {
	if body.typ != "body" {
		panic(getUnexpectedTypeError("body", body.typ))
	}
	if len(body.children) == 0 {
		panic("no body children as unexpected")
	}
	if len(body.children) > 1 {
		s.writeMark()
		for _, alt := range body.children {
			s.alt(alt)
			s.writeReset()
		}
	} else {
		s.alt(body.children[0])
	}
}

type PegCompiler struct {
	data        Node
	lang        string
	rules       []RuleProg
	writingRule RuleProg
}

func GetPegCompiler(data Node, lang string) PegCompiler {
	if data.typ != "grammar" {
		panic(fmt.Sprintf("grammar is expected, got %s", data.typ))
	}
	return PegCompiler{data, lang, []RuleProg{}, RuleProg{}}
}

func newRule(lang, name string) RuleProg {
	r := RuleProg{[]RuleProg{}, lang, name, "", 1, 0, 0}
	r.create()
	return r
}

func (s *PegCompiler) rule(rule Node) {
	if rule.typ != "rule" {
		panic(getUnexpectedTypeError("rule", rule.typ))
	}
	if len(rule.children) == 0 {
		panic("no rule child as unexpected")
	} else if len(rule.children) != 1 {
		panic("too much rule children")
	}
	if rule.value == "" {
		panic("unnamed rule")
	}
	r := newRule(s.lang, rule.value)
	r.body(rule.children[0])
	r.writeReturnNull()
	s.rules = append(s.rules, r)
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
	for _, rule := range s.data.children {
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
	parser parser.Parser
}

func Get@1Parser(text string) @1 {
	return @1{parser.GetParser(text)}
}

`
	finalProg = strings.ReplaceAll(finalProg, "@1", s.lang)
	finalProg = strings.ReplaceAll(finalProg, "@2", name)
	for _, rule := range s.rules {
		for _, subRule := range rule.subRules {
			finalProg += subRule.program + "\n\n"
		}
		finalProg += rule.program + "\n\n"
	}
	finalProg += "func (s *" + s.lang + ") Parse() Node {\n\treturn s." + s.data.children[0].value + "()\n}"
	os.WriteFile(fmt.Sprintf(path+"%s/%s.go", name, name), []byte(finalProg), fs.ModeAppend)
}
