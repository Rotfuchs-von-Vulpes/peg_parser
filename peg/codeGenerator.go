package peg

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"unicode"
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
	s.program += fmt.Sprintf("func (s *%s) p_%s () parseResult {\n\tnodes := []Node{}\n\tfields := []string{}@1\n}", s.lang, s.name)
}

func (s *RuleProg) writeCloseCatcher() {
	s.write("@1@1")
}

func (s *RuleProg) writeReturnNull() {
	s.write("@3return parseResult{ok: false, nextPos: s.scanner.Mark()}")
	if s.tabs > 1 {
		s.tabs -= 1
	}
	s.program += fmt.Sprintf("\n\nfunc (s *%s) m_%s () parseResult {\n\treturn s.memoize(\"%s\", s.p_%s)\n}", s.lang, s.name, s.name, s.name)
}

func (s *RuleProg) writeReturn() {
	s.write(fmt.Sprintf("@3return parseResult{true, Node{\"%s\", fields, nodes}, s.scanner.Mark()}", s.name))
	if s.tabs > 1 {
		s.tabs -= 1
	}
}

func (s *RuleProg) writeMark(old bool) {
	if old {
		s.write("@3pos = s.scanner.Mark()@1")
	} else {
		s.write("@3pos := s.scanner.Mark()@1")
	}
}

func (s *RuleProg) writeNewPos() {
	s.write("@3pos = s.scanner.Mark()@1")
}

func (s *RuleProg) writeClear() {
	s.write("@3fields = []string{}@1")
	s.write("@3nodes = []Node{}@1")
}

func (s *RuleProg) writeReset() {
	s.write("@3s.scanner.Reset(pos)@1")
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

func (s *RuleProg) writeElse() {
	s.write(" else {@1@3}@1")
	s.tabs += 1
}

func (s *RuleProg) addRule(rule string, add, not, relevant bool) {
	if rule == "" {
		panic("Empty rule name")
	}
	if not {
		s.writeIf(fmt.Sprintf("if res := s.m_%s(); !res.ok", rule))
	} else {
		if add {
			s.writeIf(fmt.Sprintf("if res := s.m_%s(); res.ok", rule))
			s.write("@3nodes = append(nodes, res.node.Children...)@1")
			s.write("@3fields = append(fields, res.node.Fields...)@1")
		} else {
			if relevant {
				s.writeIf(fmt.Sprintf("if res := s.m_%s(); res.ok", rule))
				s.write("@3nodes = append(nodes, res.node)@1")
			} else {
				s.writeIf(fmt.Sprintf("if res := s.m_%s(); res.ok", rule))
			}
		}
	}
}

func (s *RuleProg) addString(str string, not bool, tag string) {
	if str == "" {
		panic("Empty string")
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
		s.writeIf("if ok := s.scanner.String(\"" + final.String() + "\"); !ok")
	} else {
		s.writeIf(fmt.Sprintf("if ok := s.scanner.String(\"%s\"); ok", final.String()))
		if tag != "" {
			s.write(fmt.Sprintf("@3fields = append(fields, \"%s\")@1", tag))
		}
	}
}

func (s *RuleProg) addLiteral(literal string, not bool) {
	switch literal {
	case "ENDMARKER":
		if not {
			s.writeIf("if ok := s.scanner.Expect(0); !ok")
		} else {
			s.writeIf("if ok := s.scanner.Expect(0); ok")
		}
	default:
		panic("Unknow literal: " + literal)
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

func (s *RuleProg) addRegex(regex string, not, relevant bool) {
	if not {
		s.writeIf("if ok, _ := regex.RunRegex(&s.scanner, \"" + bakeString(regex) + "\"); !ok")
	} else {
		if relevant {
			s.writeIf("if ok, str := regex.RunRegex(&s.scanner, \"" + bakeString(regex) + "\"); ok")
			s.write("@3fields = append(fields, str)@1")
		} else {
			s.writeIf("if ok, _ := regex.RunRegex(&s.scanner, \"" + bakeString(regex) + "\"); ok")
		}
	}
}

func (s *RuleProg) atom(atom Node, not bool) bool {
	final := false
	if literal, ok := atom.(Literal); ok {
		switch literal.Type {
		case L_literal:
			s.addLiteral(literal.Value, not)
		case L_name:
			s.addRule(literal.Value, false, not, literal.Add)
		case L_regex:
			s.addRegex(literal.Value, not, literal.Add)
		case L_string:
			s.addString(literal.Value, not, literal.AddId)
		}
	} else if body, ok := atom.(Body); ok {
		s.addSubRule(body)
		s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true, not, false)
		final = true
	} else {
		panic("Atom has illegal type")
	}
	return final
}

func (s *RuleProg) loop(loop Loop, pos_is_added bool) bool {
	add_pos := pos_is_added
	switch loop.Mode {
	case L_none:
		s.atom(loop.Child, loop.Not)
	case L_zero_or_one:
		s.writeMark(add_pos)
		s.atom(loop.Child, loop.Not)
		s.close()
		s.writeElse()
		s.writeReset()
		s.close()
		s.writeCloseCatcher()
	case L_zero_or_more:
		s.writeMark(add_pos)
		s.writeRuleFor()
		s.atom(loop.Child, loop.Not)
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		s.writeCloseCatcher()
		add_pos = true
	case L_one_or_more:
		sub := s.atom(loop.Child, loop.Not)
		s.writeMark(add_pos)
		s.writeRuleFor()
		if sub {
			s.addRule(s.name+"_"+strconv.Itoa(s.subRuleCount), true, loop.Not, false)
		} else {
			s.atom(loop.Child, loop.Not)
		}
		s.writeNewPos()
		s.close()
		s.writeElseBreak()
		s.writeReset()
		add_pos = true
	}
	return add_pos
}

func (s *RuleProg) alternative(alt Alternative) {
	if len(alt.Loops) == 0 {
		panic("Zero loop as unexpected")
	}
	add_pos := false
	for _, loop := range alt.Loops {
		add_pos = s.loop(loop, add_pos)
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
		panic("No body children as unexpected")
	}
	if len(body.Alts) > 1 {
		s.writeMark(false)
		for i, alt := range body.Alts {
			s.alternative(alt)
			if i < len(body.Alts)-1 {
				s.writeClear()
				s.writeReset()
			}
		}
	} else {
		s.alternative(body.Alts[0])
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
	r := newRule(s.lang, rule.Name)
	b := strings.Builder{}
	capitalize := true
	for _, r := range rule.Name {
		if capitalize {
			r2 := unicode.ToUpper(r)
			b.WriteRune(r2)
			capitalize = false
		} else {
			if r != '_' {
				b.WriteRune(r)
			}
		}
		if r == '_' {
			capitalize = true
		}
	}
	r.body(rule.Body)
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
	"test/regex"
	"test/scanner"
)

type Node struct {
	Typ      string
	Fields   []string
	Children []Node
}

type parseResult struct {
	ok      bool
	node    Node
	nextPos int
}

type cacheKey struct {
	rule string
	pos  int
}

type @1 struct {
	scanner scanner.Scanner
	cache   map[cacheKey]parseResult
}

func Get@1Parser(text string) @1 {
	return @1{scanner.GetScanner(text), make(map[cacheKey]parseResult)}
}

func (s *@1) memoize(ruleName string, parseFunc func() parseResult) parseResult {
	key := cacheKey{ruleName, s.scanner.Mark()}
    if result, found := s.cache[key]; found {
		s.scanner.Reset(result.nextPos)
        return result
    }
    result := parseFunc()
    s.cache[key] = result
    return result
}

`
	finalProg = strings.ReplaceAll(finalProg, "@1", s.lang)
	finalProg = strings.ReplaceAll(finalProg, "@2", name)
	for _, rule := range s.rules {
		finalProg += s.eachSubRule(rule)
		finalProg += rule.program + "\n\n"
	}
	finalProg += "func (s *" + s.lang + ") Parse() (bool, Node) {\n\tres := s.m_" + s.data.Rules[0].Name + "()\n\treturn res.ok, res.node\n}"
	os.Mkdir(path+name+"/", os.ModePerm)
	os.Remove(fmt.Sprintf(path+"%s/%s.go", name, name))
	os.WriteFile(fmt.Sprintf(path+"%s/%s.go", name, name), []byte(finalProg), fs.ModeAppend)
}
