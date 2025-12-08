package main

import (
	"main/peg"
	"os"
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
	file, err := os.ReadFile(args[0])
	if err != nil {
		panic(err.Error())
	}
	pegP := peg.GetPegParser(string(file))
	grammar := pegP.Parse()
	c := peg.GetPegCompiler(grammar, getName(args[0]))
	if len(args) == 2 {
		c.Compile(args[1])
	} else {
		c.Compile("")
	}
}
