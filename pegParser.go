package main

import (
	"fmt"
	"os"
	"pegParser/parser"
	"pegParser/peg"
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
	} else {
		if data, err := os.ReadFile("./input/peg_new.peg"); err == nil {
			// parser.ParseGrammar(string(data), "Sl 21,2-6; Eclo 20,3-4; 21,2.3")
			if sample, err := os.ReadFile("./input/toy.peg"); err == nil {
				parser.ParseGrammar(string(data), string(sample))
			} else {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
}
