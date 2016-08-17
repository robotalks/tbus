package proto

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"

	gen "github.com/golang/protobuf/protoc-gen-go/generator"
)

type javascriptGenerator struct {
	internal bool
}

func (g *javascriptGenerator) Generate(def *Definition, out Output) error {
	for _, f := range def.Files {
		w, err := out.GenerateFile(SuffixFileName(f.Name, jsDeviceFileSuffix))
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

func jsSymbolName(origin string) string {
	s := gen.CamelCase(origin)
	return strings.ToLower(s[0:1]) + s[1:]
}

/*
func jsTypeFileName(typeName string) string {
	pos := strings.LastIndexByte(typeName, '.')
	if pos > 0 {
		typeName = typeName[pos+1:]
	}
	return strings.ToLower(gen.CamelCase(typeName)) + ".js"
}
*/

const (
	jsProtoFileSuffix  = "_pb.js"
	jsDeviceFileSuffix = "_tbusdev.js"

	jsHeaderInternal = `//
// GENERTED FROM {{.Source}}, DO NOT EDIT
//

var Class      = require('js-class'),
    Device     = require('../../lib/device.js'),
    Controller = require('../../lib/control.js'),
    protocol   = require('../../lib/protocol.js');
`

	jsHeader = `//
// GENERTED FROM {{.Source}}, DO NOT EDIT
//

var Class      = require('js-class'),
    Device     = require('tbus').Device,
    Controller = require('tbus').Controller,
    protocol   = require('tbus').protocol;
`

	jsSource = `
{{range .Requires -}}
require('{{.}}');
{{end}}
{{- range .Classes}}
var {{.ClassName}}Dev = Class(Device, {
    constructor: function (logic, options) {
        this.options = options || {};
        Device.prototype.constructor.call(this, {{.ClassName}}Dev.CLASS_ID, logic, this.options);
        var dev = this;
{{- range .Methods}}

        this.defineMethod({{.Index}}, '{{.Name}}', function (params, done) {
                dev['m:{{.Symbol}}'](params, done);
            });
{{- end}}
		logic.setDevice(this);
    },
{{- range .Methods}}

    'm:{{.Symbol}}': function (params, done) {
        {{- if .ParamType}}
        params = {{.ParamType}}.deserializeBinary(new Uint8Array(params))
        {{- end}}
        this.logic.{{.Symbol}}({{if .ParamType}}params, {{end}}done);
    },
{{- end}}
}, {
    statics: {
        CLASS_ID: {{.ClassID}}
    }
});

var {{.ClassName}}Ctl = Class(Controller, {
    constructor: function (master, addrs) {
        Controller.prototype.constructor.call(this, master, addrs);
    },
{{- range .Methods}}

    {{.Symbol}}: function ({{if .ParamType}}params, {{end}}done) {
        this.invoke(1, {{if .ParamType}}params{{else}}null{{end}}, function (err, reply) {
            {{- if .ReturnType}}
            if (err == null) {
                reply = {{.ReturnType}}.deserializeBinary(new Uint8Array(reply));
            }
            done(err, reply);
            {{- else}}
            done(err);
            {{- end}}
        });
    },
{{- end}}
}, {
    statics: {
        CLASS_ID: {{.ClassID}}
    }
});

{{- end}}

module.exports = {
{{- range .Classes}}
    {{.ClassName}}Dev: {{.ClassName}}Dev,
    {{.ClassName}}Ctl: {{.ClassName}}Ctl,
{{end -}}
};
`
)

var (
	jsSourceTemplate   = template.Must(template.New("source").Parse(jsHeader + jsSource))
	jsInternalTemplate = template.Must(template.New("source").Parse(jsHeaderInternal + jsSource))
)

type jsClass struct {
	ClassName string
	ClassID   string
	Methods   []jsMethod
}

type jsMethod struct {
	Index      uint32
	Name       string
	Symbol     string
	ParamType  string
	ReturnType string
}

type jsFile struct {
	Source   string
	Requires []string
	Classes  []jsClass
}

func (g *javascriptGenerator) generate(f *DefFile, w io.Writer) error {
	ctx := jsFile{Source: f.Name}
	for _, fn := range f.Deps {
		jsfn := SuffixFileName(fn, jsProtoFileSuffix)
		if strings.HasPrefix(fn, "google/") {
			ctx.Requires = append(ctx.Requires, "google-protobuf/"+jsfn)
		} else if g.internal {
			if strings.HasPrefix(fn, "tbus/") {
				ctx.Requires = append(ctx.Requires, "./"+jsfn[5:])
			} else {
				ctx.Requires = append(ctx.Requires, "../"+jsfn)
			}
		} else {
			ctx.Requires = append(ctx.Requires, "tbus/gen/"+jsfn)
		}
	}
	ctx.Requires = append(ctx.Requires, "./"+SuffixFileName(filepath.Base(f.Name), jsProtoFileSuffix))
	/*
		// for goog.require(...) style
		// find all custom types as they are defined in separated files named
		// after the type
		typeNames := make(map[string]string)
		for _, dev := range f.Devices {
			for _, m := range dev.Methods {
				if m.RequestType != "" {
					typeNames[m.RequestType] = jsTypeFileName(m.RequestType)
				}
				if m.ResponseType != "" {
					typeNames[m.ResponseType] = jsTypeFileName(m.ResponseType)
				}
			}
		}
		for _, fn := range typeNames {
			ctx.Requires = append(ctx.Requires, "./"+fn)
		}
	*/
	for _, dev := range f.Devices {
		cls := jsClass{
			ClassName: dev.Name,
			ClassID:   fmt.Sprintf("0x%04x", dev.ClassID),
		}
		for _, m := range dev.Methods {
			mtd := jsMethod{
				Index:      m.Index,
				Name:       m.Name,
				Symbol:     jsSymbolName(m.Name),
				ParamType:  m.RequestType,
				ReturnType: m.ResponseType,
			}
			if mtd.ParamType != "" {
				mtd.ParamType = "proto" + mtd.ParamType
			}
			if mtd.ReturnType != "" {
				mtd.ReturnType = "proto" + mtd.ReturnType
			}
			cls.Methods = append(cls.Methods, mtd)

		}
		ctx.Classes = append(ctx.Classes, cls)
	}
	tmpl := jsSourceTemplate
	if g.internal {
		tmpl = jsInternalTemplate
	}
	return tmpl.Execute(w, &ctx)
}

// NewJavaScriptGenerator creates javascript code generator
func NewJavaScriptGenerator(args []string) (Generator, error) {
	g := &javascriptGenerator{}
	for _, arg := range args {
		if arg == "internal" {
			g.internal = true
		}
	}
	return g, nil
}

func init() {
	Generators["js"] = NewJavaScriptGenerator
}
