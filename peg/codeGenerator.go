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

func (s *RuleProg) addString(str string, add bool) {
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
	s.writeIf(fmt.Sprintf("if ok := s.parser.String(\"%s\"); ok", final.String()))
	if add {
		s.write(fmt.Sprintf("@3nodes = append(nodes, Node{\"string\", \"%s\", []Node{}})@1", final.String()))
	}
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

func (s *RuleProg) addRegex(regex Node) {
	if regex.value == "" {
		panic("empty regex code")
	}
	s.writeIf("if ok, str := s.parser.Regex(\"" + bakeString(regex.value) + "\"); ok")
	s.write("@3nodes = append(nodes, Node{\"string\", str, []Node{}})@1")
}

func getUnexpectedTypeError(want string, get string) string {
	return fmt.Sprintf("This is what you want: %s, this is what you get: %s", want, get)
}

func (s *RuleProg) item(item Node, variable bool) {
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
	case "chars":
		s.addString(child.value, variable)
	case "literal":
		s.addLiteral(child.value)
	case "regex":
		s.addRegex(child)
	default:
		panic(fmt.Sprintf("item has illegal child: %s", child.typ))
	}
}

func (s *RuleProg) atom(atom Node, variable bool) bool {
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
		s.item(child, variable)
	case "name":
		s.addRule(child.value, false)
	case "chars":
		s.addString(child.value, variable)
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

func (s *RuleProg) loop(loop Node, various, pos_is_added bool) bool {
	if loop.typ != "loop" {
		panic(getUnexpectedTypeError("loop", loop.typ))
	}
	if len(loop.children) == 0 {
		panic("zero atom as unexpected")
	} else if len(loop.children) != 1 {
		panic("too much loop")
	}
	add_pos := pos_is_added
	switch loop.value {
	case "":
		s.atom(loop.children[0], various)
	case "?":
		s.atom(loop.children[0], true)
		s.writeReturn()
		s.writeCloseCatcher()
	case "*":
		child := loop.children[0]
		s.writeMark(add_pos)
		s.writeRuleFor()
		s.atom(child, various)
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		s.writeCloseCatcher()
		add_pos = true
	case "+":
		child := loop.children[0]
		sub := s.atom(child, various)
		s.writeMark(add_pos)
		s.writeRuleFor()
		if sub {
			s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true)
		} else {
			s.atom(child, various)
		}
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		add_pos = true
	default:
		panic("unknow repeat operator")
	}
	return add_pos
}

func (s *RuleProg) alt(alt Node, variable bool) {
	if alt.typ != "alts" {
		panic(getUnexpectedTypeError("alts", alt.typ))
	}
	if len(alt.children) == 0 {
		panic("zero loop as unexpected")
	}
	add_pos := false
	for _, loop := range alt.children {
		add_pos = s.loop(loop, variable, add_pos)
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
		variable := false
		for _, alt := range body.children {
			if len(alt.children) == 1 {
				loop := alt.children[0]
				atom := loop.children[0]
				if atom.children[0].typ == "item" {
					item := atom.children[0]
					if item.children[0].typ == "chars" {
						variable = true
						break
					}
				}
			}
		}
		s.writeMark(false)
		for _, alt := range body.children {
			s.alt(alt, variable)
			s.writeReset()
		}
	} else {
		s.writeMark(false)
		s.alt(body.children[0], false)
		s.writeReset()
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
	r := RuleProg{[]RuleProg{}, lang, name, "", 1, 0}
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
	parser parser.Tokenizer
}

func Get@1Parser(text string) @1 {
	return @1{parser.GetTokenizer(text)}
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
	os.Mkdir(path+name+"/", os.ModePerm)
	os.Remove(fmt.Sprintf(path+"%s/%s.go", name, name))
	os.WriteFile(fmt.Sprintf(path+"%s/%s.go", name, name), []byte(finalProg), fs.ModeAppend)
}
