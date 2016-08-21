package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	lang     string
	outDir   string
	incDirs  string
	protoDir string

	internal   bool
	protoFiles []string
)

const (
	protoSuffixLen = 6
)

func init() {
	flag.StringVar(&lang, "lang", "", "language: go,js")
	flag.StringVar(&incDirs, "I", "", "include directories, comma-separated")
	flag.StringVar(&outDir, "out", "", "output directory")
	flag.StringVar(&protoDir, "from", "", "proto directory")
}

func protoSuffixSubst(fn, suffix string) string {
	return fn[0:len(fn)-protoSuffixLen] + suffix
}

func protoc(plugin string) error {
	cmd := exec.Command("protoc")
	if incDirs != "" {
		incs := strings.Split(incDirs, ":")
		for _, inc := range incs {
			cmd.Args = append(cmd.Args, "-I"+inc)
		}
	}
	cmd.Args = append(cmd.Args, "-I"+protoDir, plugin+":"+outDir)
	for _, protoFn := range protoFiles {
		cmd.Args = append(cmd.Args, filepath.Join(protoDir, protoFn))
	}
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
	return protoc("--tbus_out=go" + internalParam)
}

func jsFixRequires(fn string) error {
	info, err := os.Stat(fn)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	content := string(data)
	pos := strings.Index(content, "\nvar tbus_common_options_pb = require(")
	if pos > 0 {
		nextLn := strings.Index(content[pos+1:], "\n")
		if nextLn >= 0 {
			content = content[0:pos] + content[pos+1+nextLn:]
			if err = ioutil.WriteFile(fn, []byte(content), info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func jsProtoc() error {
	internalParam := ""
	if internal {
		internalParam = ",internal"
	}
	err := protoc("--js_out=import_style=commonjs,binary")
	if err == nil {
		for _, fn := range protoFiles {
			err = jsFixRequires(filepath.Join(outDir, protoSuffixSubst(fn, "_pb.js")))
			if err != nil {
				return err
			}
		}
	}
	if err == nil {
		err = protoc("--tbus_out=js" + internalParam)
	}
	return err
}

func findProtoFiles(pkgs ...string) error {
	prefixLen := len(protoDir) + 1
	for _, pkg := range pkgs {
		files, err := filepath.Glob(filepath.Join(protoDir, pkg, "*.proto"))
		if err != nil {
			return err
		}
		for _, fn := range files {
			info, err := os.Stat(fn)
			if err != nil || info.IsDir() {
				continue
			}
			protoFiles = append(protoFiles, fn[prefixLen:])
		}
	}
	return nil
}

func run() (err error) {
	if err = findProtoFiles(flag.Args()...); err != nil {
		return err
	}
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
	internal = os.Getenv("TBUS_INTERNAL") != ""
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
