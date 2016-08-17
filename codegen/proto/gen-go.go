package proto

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"

	gen "github.com/golang/protobuf/protoc-gen-go/generator"
)

type goGenerator struct {
	internal bool
}

func (g *goGenerator) Generate(def *Definition, out Output) error {
	for _, f := range def.Files {
		w, err := out.GenerateFile(SuffixFileName(f.Name, goDeviceFileSuffix))
		if err != nil {
			return err
		}
		err = g.generate(f, w)
		w.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	goDeviceFileSuffix = ".tbusdev.go"

	goSource = `//
// GENERTED FROM {{.Source}}, DO NOT EDIT
// {{- $tbus := .PkgPfx}}

package {{.Package}}

import (
    prot "github.com/evo-bots/tbus/go/tbus/protocol"
    proto "github.com/golang/protobuf/proto"
{{- range .Imports}}
    {{with .Alias}}{{.}} {{end}}"{{.Pkg}}"
{{- end}}
)

{{range .Classes -}}
// {{.ClassName}}ClassID is the class ID of {{.ClassName}}
const {{.ClassName}}ClassID uint32 = {{.ClassID}}

// {{.ClassName}}Logic defines the logic interface
type {{.ClassName}}Logic interface {
	{{$tbus}}DeviceLogic
{{- if .Router}}
	{{$tbus}}MsgRouter
{{- end}}
{{- range .Methods}}
    {{.Symbol}}({{with .ParamType}}*{{.}}{{end}}) {{if .ReturnType}}(*{{.ReturnType}}, error){{else}}error{{end}}
{{- end}}
}

// {{.ClassName}}Dev is the device
type {{.ClassName}}Dev struct {
    {{$tbus}}DeviceBase
    Logic {{.ClassName}}Logic
}

// New{{.ClassName}}Dev creates a new device
func New{{.ClassName}}Dev(logic {{.ClassName}}Logic) *{{.ClassName}}Dev {
    d := &{{.ClassName}}Dev{Logic: logic}
    d.Info.ClassId = {{.ClassName}}ClassID
	logic.SetDevice(d)
    return d
}

// SendMsg implements Device
func (d *{{.ClassName}}Dev) SendMsg(msg *prot.Msg) (err error) {
	if msg.Head.NeedRoute() {
{{- if .Router}}
		return d.Logic.({{$tbus}}MsgRouter).RouteMsg(msg)
{{- else}}
		return d.Reply(msg.Head.MsgID, nil, {{$tbus}}ErrRouteNotSupport)
{{- end}}
	}
    var reply proto.Message
	switch msg.Body.Flag {
{{- range .Methods}}
    case {{.Index}}: // {{.Name}}
        {{- if .ParamType}}
        params := &{{.ParamType}}{}
        err = proto.Unmarshal(msg.Body.Data, params)
        if err == nil {
            {{if .ReturnType}}reply, {{end}}err = d.Logic.{{.Symbol}}(params)
        }
        {{- else}}
        {{if .ReturnType}}reply, {{end}}err = d.Logic.{{.Symbol}}()
        {{- end}}
{{- end}}
    default:
        err = {{$tbus}}ErrInvalidMethod
    }
    return d.Reply(msg.Head.MsgID, reply, err)
}

// SetDeviceID sets device id
func (d *{{.ClassName}}Dev) SetDeviceID(id uint32) *{{.ClassName}}Dev {
	d.Info.DeviceId = id
	return d
}

// {{.ClassName}}Ctl is the device controller
type {{.ClassName}}Ctl struct {
    {{$tbus}}Controller
}

// New{{.ClassName}}Ctl creates controller for {{.ClassName}}
func New{{.ClassName}}Ctl(master Master) *{{.ClassName}}Ctl {
	c := &{{.ClassName}}Ctl{}
	c.Master = master
	return c
}

// SetAddress sets routing address for target device
func (c *{{.ClassName}}Ctl) SetAddress(addrs []uint8) *{{.ClassName}}Ctl {
	c.Address = addrs
	return c
}

{{$class := . }}{{range .Methods -}}
// {{.Symbol}} wraps class {{$class.ClassName}}
func (c *{{$class.ClassName}}Ctl) {{.Symbol}}({{with .ParamType}}params *{{.}}{{end}}) {{if .ReturnType}}(*{{.ReturnType}}, error){{else}}error{{end}} {
	{{- if .ReturnType}}
	reply := &{{.ReturnType}}{}
	err := c.Invoke({{.Index}}, {{if .ParamType}}params{{else}}nil{{end}}, reply)
	return reply, err
	{{- else}}
	return c.Invoke({{.Index}}, {{if .ParamType}}params{{else}}nil{{end}}, nil)
	{{- end}}
}

{{end -}}
{{end -}}
`
)

var (
	goSourceTemplate = template.Must(template.New("source").Parse(goSource))
)

type goClass struct {
	ClassName string
	ClassID   string
	Router    bool
	Methods   []goMethod
}

type goMethod struct {
	Index      uint32
	Name       string
	Symbol     string
	ParamType  string
	ReturnType string
}

type goImport struct {
	Alias string
	Pkg   string
}

type goFile struct {
	Source  string
	Package string
	PkgPfx  string
	Imports []goImport
	Classes []goClass
}

func (g *goGenerator) generate(f *DefFile, w io.Writer) error {
	ctx := goFile{Source: f.Name, Package: f.Package}
	if f.Options.GoPackage != "" {
		ctx.Package = filepath.Base(f.Options.GoPackage)
	}
	if !g.internal {
		ctx.Imports = append(ctx.Imports, goImport{
			Alias: "tbus",
			Pkg:   "github.com/evo-bots/tbus/go/tbus",
		})
		ctx.PkgPfx = "tbus."
	}

	for _, dev := range f.Devices {
		cls := goClass{
			ClassName: dev.Name,
			ClassID:   fmt.Sprintf("0x%04x", dev.ClassID),
		}
		cls.Router = dev.ClassID == BusClassID
		for _, m := range dev.Methods {
			mtd := goMethod{
				Index:      m.Index,
				Name:       m.Name,
				Symbol:     gen.CamelCase(m.Name),
				ParamType:  m.RequestType,
				ReturnType: m.ResponseType,
			}
			mtd.ParamType = g.fixTypeName(mtd.ParamType)
			mtd.ReturnType = g.fixTypeName(mtd.ReturnType)
			cls.Methods = append(cls.Methods, mtd)
		}
		ctx.Classes = append(ctx.Classes, cls)
	}
	return goSourceTemplate.Execute(w, &ctx)
}

func (g *goGenerator) fixTypeName(typeName string) string {
	if g.internal && strings.HasPrefix(typeName, ".tbus.") {
		return typeName[6:]
	} else if strings.HasPrefix(typeName, ".") {
		return typeName[1:]
	}
	return typeName
}

// NewGoGenerator creates Go code generator
func NewGoGenerator(args []string) (Generator, error) {
	g := &goGenerator{}
	for _, arg := range args {
		if arg == "internal" {
			g.internal = true
		}
	}
	return g, nil
}

func init() {
	Generators["go"] = NewGoGenerator
}
