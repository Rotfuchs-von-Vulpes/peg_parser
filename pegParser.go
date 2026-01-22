package main

import (
	"fmt"
	"os"
	"pegParser/peg"
	"pegParser/regex"
	"strings"
)

func getName(path string) string {
	str := strings.Split(path, "/")
	fileName := str[len(str)-1]
	langName, ok := strings.CutSuffix(fileName, ".peg")
	if ok {
		return langName
	} else {
		return fileName
	}
}

func main() {
	args := os.Args[1:]
	if len(args) >= 1 {
		if args[0] == "regex" {
			if len(args) == 3 {
				p := regex.GetRegexParser(args[1])
				s := regex.GetRegexStack(p.Parse())
				res, ok := regex.UseStack(s, args[2])
				if ok {
					fmt.Println("True.")
				} else {
					fmt.Println("False.")
				}
				switch res {
				case regex.UnexpectedEnd:
					fmt.Println("Unexpected end.")
				case regex.Matched:
					fmt.Println("matched.")
				case regex.UnexpectedRune:
					fmt.Println("Unexpected rune.")
				case regex.UnexpectedMore:
					fmt.Println("Unexpected more.")
				}
			}
		} else {
			file, err := os.ReadFile(args[0])
			if err != nil {
				panic(err.Error())
			}
			pegP := peg.GetPegParser(string(file))
			if ok, grammar := pegP.Parse(); ok {
				c := peg.GetPegCompiler(grammar, getName(args[0]))
				if len(args) >= 2 {
					c.Compile(args[1])
				} else {
					c.Compile("")
				}
			} else {
				panic("Parse fails")
			}
		}
	}
}
