package proto

import (
	"fmt"
	"io"
	"strings"
)

// common consts
const (
	BusClassID = 0x0001
)

// common errors
var (
	ErrArgsMissingLang = fmt.Errorf("missing language")
)

// Device represents the definition of a device
type Device struct {
	Name      string
	ClassID   uint32
	Methods   []*Method
	EventChns []*EventChannel
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
	if method.Index == 0 {
		return fmt.Errorf("method %s has invalid index 0", method.Name)
	}
	if m := d.MethodByIndex(method.Index); m != nil {
		return fmt.Errorf("method %s and %s has the same index %d",
			m.Name, method.Name, m.Index)
	}
	d.Methods = append(d.Methods, method)
	return nil
}

// EventChnByIndex finds event channel by index
func (d *Device) EventChnByIndex(index uint32) *EventChannel {
	for _, c := range d.EventChns {
		if c.Index == index {
			return c
		}
	}
	return nil
}

// AddEventChannel adds an event channel with unique index
func (d *Device) AddEventChannel(chn *EventChannel) error {
	if chn.Index == 0 {
		return fmt.Errorf("event channel %s has invalid index 0", chn.Name)
	}
	if c := d.EventChnByIndex(chn.Index); c != nil {
		return fmt.Errorf("event channel %s and %s has the same index %d",
			c.Name, chn.Name, c.Index)
	}
	d.EventChns = append(d.EventChns, chn)
	return nil
}

// Method defines a method supported by a device
type Method struct {
	Index        uint32
	Name         string
	RequestType  string
	ResponseType string
}

// EventChannel defines an event channel broadcasting events
type EventChannel struct {
	Index     uint32
	Name      string
	EventType string
}

// DefFile is a file containing definitions
type DefFile struct {
	Name    string
	Package string
	Deps    []string
	Options FileOptions
	Devices []*Device
}

// FileOptions contains known options from file
type FileOptions struct {
	GoPackage string
}

// Definition contains all definition files to be processed
type Definition struct {
	Lang  string
	Args  []string
	Files []*DefFile
}

// ParseArgs parses arguments in a string, comma-separated
func (d *Definition) ParseArgs(args string) error {
	tokens := strings.Split(args, ",")
	if len(tokens) == 0 {
		return ErrArgsMissingLang
	}
	d.Lang = tokens[0]
	d.Args = tokens[1:]
	return nil
}

// Parser parses input and build definition
type Parser interface {
	Parse(io.Reader) (*Definition, error)
	NewOutput(io.Writer) (Output, error)
}

// Output is the output for code generators
type Output interface {
	io.Closer
	Stage(cmd, param string) ([]*GeneratedFile, error)
	GenerateFile(name string) (io.WriteCloser, error)
}

// GeneratedFile represents a generated file
type GeneratedFile struct {
	Name    string
	Content *string
}

// Generator is the abstact code generator, implement language specific
type Generator interface {
	Generate(*Definition, Output) error
}

// GeneratorFactory creates generator
type GeneratorFactory func(args []string) (Generator, error)

// Generators are all registered generators
var Generators = make(map[string]GeneratorFactory)

// NewGenerator creates generator by language
func NewGenerator(lang string, args []string) (Generator, error) {
	factory := Generators[lang]
	if factory == nil {
		return nil, fmt.Errorf("unknown language %s", lang)
	}
	return factory(args)
}

// SuffixFileName replaces file name suffix
func SuffixFileName(origin, suffix string) string {
	pos := strings.LastIndexByte(origin, '.')
	if pos > 0 {
		return origin[0:pos] + suffix
	}
	return origin + suffix
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
