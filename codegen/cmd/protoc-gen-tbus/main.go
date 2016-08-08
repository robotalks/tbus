package main

import (
	"log"
	"os"

	proto "github.com/evo-bots/tbus/codegen/proto"
)

func generate() error {
	def, err := proto.NewProtocParser().Parse(os.Stdin)
	if err != nil {
		return err
	}
	g, err := proto.NewGenerator(def.Lang, def.Args)
	if err != nil {
		return err
	}

	out := proto.NewProtocOutput(os.Stdout)
	defer out.Close()

	return g.Generate(def, out)
}

func main() {
	err := generate()
	if err != nil {
		log.Fatalln(err)
	}
}
