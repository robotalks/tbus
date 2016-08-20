package main

import (
	"log"
	"os"

	proto "github.com/robotalks/tbus/codegen/proto"
)

func generate() error {
	parser := proto.NewProtocParser()
	def, err := parser.Parse(os.Stdin)
	if err != nil {
		return err
	}

	g, err := proto.NewGenerator(def.Lang, def.Args)
	if err != nil {
		return err
	}

	out, err := parser.NewOutput(os.Stdout)
	if err != nil {
		return err
	}
	defer out.Close()

	return g.Generate(def, out)
}

func main() {
	err := generate()
	if err != nil {
		log.Fatalln(err)
	}
}
