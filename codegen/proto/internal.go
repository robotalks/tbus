package proto

import (
	"fmt"
	"io"
	"strings"
)

// Device represents the definition of a device
type Device struct {
	Name    string
	ClassID uint32
	Methods []*Method
}

// MethodByIndex finds method by method index
func (d *Device) MethodByIndex(index uint32) *Method {
	for _, m := range d.Methods {
		if m.Index == index {
			return m
		}
	}
	return nil
}

// AddMethod adds a method with unique index
func (d *Device) AddMethod(method *Method) error {
	if m := d.MethodByIndex(method.Index); m != nil {
		return fmt.Errorf("method %s and %s has the same index %d",
			m.Name, method.Name, m.Index)
	}
	d.Methods = append(d.Methods, method)
	return nil
}

// Method defines a method supported by a device
type Method struct {
	Index        uint32
	Name         string
	RequestType  string
	ResponseType string
}

// DefFile is a file containing definitions
type DefFile struct {
	Name    string
	Deps    []string
	Devices []*Device
}

// Definition contains all definition files to be processed
type Definition struct {
	Parameter string
	Files     []*DefFile
}

// Parser parses input and build definition
type Parser interface {
	Parse(io.Reader) (*Definition, error)
}

// Output is the output for code generators
type Output interface {
	io.Closer
	GenerateFile(name string) (io.WriteCloser, error)
}

// Generator is the abstact code generator, implement language specific
type Generator interface {
	Generate(*Definition, Output) error
}

// GeneratorFactory creates generator
type GeneratorFactory func(param string) (Generator, error)

// Generators are all registered generators
var Generators = make(map[string]GeneratorFactory)

// NewGenerator creates generator by language
func NewGenerator(lang, param string) (Generator, error) {
	factory := Generators[lang]
	if factory == nil {
		return nil, fmt.Errorf("unknown language %s", lang)
	}
	return factory(param)
}

// Writer is helper to write code
type Writer struct {
	Out         io.Writer
	IndentChars string
	IndentLevel int

	indentStr string
	parent    *Writer
}

// PrintLn prints with indent prefixed every line and newline at the end
func (w *Writer) PrintLn(content string) *Writer {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fmt.Fprintln(w.Out, w.indent()+line)
	}
	return w
}

// Indent creates a sub Writer with next-level indent
func (w *Writer) Indent() *Writer {
	return &Writer{
		Out:         w.Out,
		IndentChars: w.IndentChars,
		IndentLevel: w.IndentLevel + 1,
		indentStr:   w.indent() + w.IndentChars,
		parent:      w,
	}
}

// Unindent gets back to the parent-level writer
func (w *Writer) Unindent() *Writer {
	if w.parent == nil {
		panic("unmatch Indent/Unindent")
	}
	return w.parent
}

func (w *Writer) indent() string {
	if w.indentStr == "" && w.IndentChars != "" && w.IndentLevel > 0 {
		for i := 0; i < w.IndentLevel; i++ {
			w.indentStr += w.IndentChars
		}
	}
	return w.indentStr
}
