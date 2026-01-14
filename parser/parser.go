package parser

import (
	"fmt"
	"pegParser/peg"
	"pegParser/regex"
	"pegParser/scanner"
)

type Node struct {
	Typ    string
	Fields []string
	Childs []Node
}

type rulePos struct {
	rule string
	pos  int
}

type result struct {
	node Node
	end  int
	ok   bool
}

type memoEntry struct {
	res      result
	progress bool
}

var memo = make(map[int]memoEntry)
var ruleSet = make(map[string]peg.Body)
var s scanner.Scanner

func testRule(rule string) result {
	start := s.Mark()
	key := start
	if entry, ok := memo[key]; ok {
		if entry.progress {
			return result{ok: false}
		}
		s.Reset(entry.res.end)
		return entry.res
	} else {
		memo[key] = memoEntry{progress: true}
		r := eval(rule, ruleSet[rule], start)
		memo[key] = memoEntry{r, false}
		s.Reset(r.end)
		return r
	}
}

func testAtom(ruleName string, atom peg.Node) result {
	nodes := []Node{}
	fields := []string{}
	if literal, ok := atom.(peg.Literal); ok {
		switch literal.Type {
		case peg.L_literal:
			if literal.Value == "ENDMARKER" {
				if ok := s.Expect(0); ok {
					return result{ok: true}
				}
			}
			return result{ok: false}
		case peg.L_name:
			res := testRule(literal.Value)
			if res.ok {
				if literal.Add {
					if literal.AddId != "" {
						res.node.Typ = literal.AddId
					}
					nodes = append(nodes, res.node)
				}
				return result{Node{"atom", fields, nodes}, s.Mark(), true}
			} else {
				return result{ok: false}
			}
		case peg.L_regex:
			fmt.Println("hm")
			if ok, str := regex.RunRegex(&s, literal.Value); ok {
				if literal.Add {
					fields = append(fields, str)
				}
				return result{Node{"atom", fields, nodes}, s.Mark(), true}
			} else {
				return result{ok: false}
			}
		case peg.L_string:
			fmt.Println(literal.Value)
			if ok := s.String(literal.Value); ok {
				if literal.Add {
					fields = append(fields, literal.AddId)
				}
				return result{Node{"atom", fields, nodes}, s.Mark(), true}
			} else {
				return result{ok: false}
			}
		default:
			return result{ok: false}
		}
	} else if body, ok := atom.(peg.Body); ok {
		start := s.Mark()
		if res := eval(ruleName, body, start); res.ok {
			nodes = append(nodes, res.node.Childs...)
			fields = append(fields, res.node.Fields...)
			return result{Node{"atom", fields, nodes}, s.Mark(), true}
		} else {
			return result{ok: false}
		}
	} else {
		return result{ok: false}
	}
}

func runLoop(ruleName string, atom peg.Node, loop peg.Loop) result {
	fields := []string{}
	nodes := []Node{}
	switch loop.Mode {
	case peg.L_none:
		if res := testAtom(ruleName, atom); res.ok {
			nodes = append(nodes, res.node.Childs...)
			fields = append(fields, res.node.Fields...)
			return result{Node{ruleName, fields, nodes}, s.Mark(), true}
		} else {
			return result{ok: false}
		}
	case peg.L_one_or_more:
		if res := testAtom(ruleName, atom); res.ok {
			nodes = append(nodes, res.node.Childs...)
			fields = append(fields, res.node.Fields...)
			pos := s.Mark()
			for {
				start := s.Mark()
				if res := testAtom(ruleName, atom); res.ok {
					end := s.Mark()
					if end == start {
						break
					}
					nodes = append(nodes, res.node.Childs...)
					fields = append(fields, res.node.Fields...)
					pos = s.Mark()
				} else {
					s.Reset(pos)
					break
				}
			}
			return result{Node{ruleName, fields, nodes}, s.Mark(), true}
		} else {
			return result{ok: false}
		}
	case peg.L_zero_or_more:
		pos := s.Mark()
		for {
			start := s.Mark()
			if res := testAtom(ruleName, atom); res.ok {
				end := s.Mark()
				if end == start {
					break
				}
				nodes = append(nodes, res.node.Childs...)
				fields = append(fields, res.node.Fields...)
				pos = s.Mark()
			} else {
				s.Reset(pos)
				break
			}
		}
		return result{Node{ruleName, fields, nodes}, s.Mark(), true}
	case peg.L_zero_or_one:
		if res := testAtom(ruleName, atom); res.ok {
			nodes = append(nodes, res.node.Childs...)
			fields = append(fields, res.node.Fields...)
		}
		return result{Node{ruleName, fields, nodes}, s.Mark(), true}
	}
	return result{ok: false}
}

func eval(ruleName string, rule peg.Body, start int) result {
	fmt.Println(ruleName)
	s.Reset(start)
	for _, alt := range rule.Alts {
		s.Reset(start)
		matched := true
		fields := []string{}
		nodes := []Node{}
		for _, loop := range alt.Loops {
			if loop.Not {
				pos := s.Mark()
				res := testAtom(ruleName, loop.Child)
				if res.ok {
					matched = false
					break
				}
				s.Reset(pos)
			} else {
				res := runLoop(ruleName, loop.Child, loop)
				if res.ok {
					nodes = append(nodes, res.node.Childs...)
					fields = append(fields, res.node.Fields...)
				} else {
					matched = false
					break
				}
			}
		}
		if matched {
			return result{Node{ruleName, fields, nodes}, s.Mark(), true}
		}
	}
	return result{ok: false}
}

func ParseGrammar(grammar string, text string) {
	p := peg.GetPegParser(grammar)
	if ok, grammarAST := p.Parse(); ok {
		s = scanner.GetScanner(text)
		for _, rule := range grammarAST.Rules {
			ruleSet[rule.Name] = rule.Body
		}
		rule := grammarAST.Rules[0]
		r := eval(rule.Name, rule.Body, 0)
		fmt.Println(r.node)
	}
}
