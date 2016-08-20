package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

var (
	lang     string
	internal bool
	outDir   string
	protoDir string
)

func init() {
	flag.StringVar(&lang, "lang", "", "language: go,js")
	flag.BoolVar(&internal, "internal", false, "internal use for tbus")
	flag.StringVar(&outDir, "out", "", "output directory")
	flag.StringVar(&protoDir, "proto-dir", "", "proto directory")
}

func protoc(plugin string, protos ...string) error {
	cmd := exec.Command("protoc", "-I"+protoDir, plugin+":"+outDir)
	cmd.Args = append(cmd.Args, protos...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func goProtoc() error {
	internalParam := ""
	if internal {
		internalParam = ",internal"
	}
	return protoc("--tbus_out=go"+internalParam, flag.Args()...)
}

func jsProtoc() error {
	internalParam := ""
	if internal {
		err := protoc("--js_out=import_style=commonjs,binary", protoDir+"/tbus/common/options.proto")
		if err != nil {
			return err
		}
		internalParam = ",internal"
	}
	err := protoc("--js_out=import_style=commonjs,binary", flag.Args()...)
	if err == nil {
		err = protoc("--tbus_out=js"+internalParam, flag.Args()...)
	}
	return err
}

func run() (err error) {
	switch lang {
	case "go":
		err = goProtoc()
	case "js":
		err = jsProtoc()
	default:
		err = fmt.Errorf("Invalid language: %s", lang)
	}
	return
}

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
