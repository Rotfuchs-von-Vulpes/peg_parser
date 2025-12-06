package main

import (
	"main/peg"
	"os"
)

func main() {
	args := os.Args[1:]
	file, err := os.ReadFile(args[0])
	if err != nil {
		panic(err.Error())
	}
	pegP := peg.GetPegParser(string(file))
	grammar := pegP.Parse()
	c := peg.GetPegCompiler(grammar, "peg")
	if len(args) == 2 {
		c.Compile(args[1])
	} else {
		c.Compile("")
	}
}
