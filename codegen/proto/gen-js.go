package proto

import (
	"fmt"
	"html/template"
	"io"
	"strings"

	gen "github.com/golang/protobuf/protoc-gen-go/generator"
)

type javascriptGenerator struct {
	internal bool
}

func (g *javascriptGenerator) Generate(def *Definition, out Output) error {
	for _, f := range def.Files {
		w, err := out.GenerateFile(jsFileName(f.Name, jsDeviceFileSuffix))
		if err != nil {
			return err
		}
		err = jsGenerate(g, f, w)
		w.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func jsFileName(origin, suffix string) string {
	pos := strings.LastIndexByte(origin, '.')
	if pos > 0 {
		return origin[0:pos] + suffix
	}
	return origin + suffix
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
	jsDeviceFileSuffix = "_device.js"

	busClassID = 0x0001

	headerInternal = `//
// GENERTED FROM {{.Source}}, DO NOT EDIT
//

var Class      = require('js-class'),
    Device     = require('../lib/device.js'),
    Controller = require('../lib/control.js'),
    protocol   = require('../lib/protocol.js');
`

	header = `//
// GENERTED FROM {{.Source}}, DO NOT EDIT
//

var Class      = require('js-class'),
    Device     = require('tbus').Device,
    Controller = require('tbus').Controller,
    protocol   = require('tbus').protocol;
`

	source = `
{{range .Requires -}}
require('{{.}}');
{{end}}

{{range .Classes -}}
var {{.ClassName}}Dev = Class(Device, {
    constructor: function (logic, options) {
        this.options = options || {};
        Device.prototype.constructor.call(this,
            {{.ClassName}}Dev.CLASS_ID, this.options.id || 0,
            new protocol.{{.DecoderName}}(this.options),
            logic);
        var dev = this;
{{range .Methods}}
        this.defineMethod({{.Index}}, '{{.Name}}', function (params, done) {
                dev['m:{{.Symbol}}'](params, done);
            });
{{end}}
    },

{{range .Methods}}
    'm:{{.Symbol}}': function (params, done) {
        {{- if .ParamType}}
        params = {{.ParamType}}.deserializeBinary(new Uint8Array(params))
        {{- end}}
        this.logic.{{.Symbol}}({{if .ParamType}}params, {{end}}function (err, result) {
            if (err == null) {
                result = {{if .ReturnType}}result{{else}}new proto.google.protobuf.Empty(){{end}}.serializeBinary();
            }
            done(err, result);
        });
    },
{{end}}
}, {
    statics: {
        CLASS_ID: {{.ClassID}}
    }
});

var {{.ClassName}}Ctl = Class(Controller, {
    constructor: function (master, addrs) {
        Controller.prototype.constructor.call(this, master, addrs);
    },

{{range .Methods}}
    {{.Symbol}}: function ({{if .ParamType}}params, {{end}}done) {
        this.invoke(1, {{if .ParamType}}params{{else}}new proto.google.protobuf.Empty(){{end}}.serializeBinary(), function (err, reply) {
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
{{end}}
});
{{end}}

module.exports = {
{{- range .Classes}}
    {{.ClassName}}Dev: {{.ClassName}}Dev,
    {{.ClassName}}Ctl: {{.ClassName}}Ctl,
{{end -}}
};
`
)

var (
	sourceTemplate   = template.Must(template.New("source").Parse(header + source))
	internalTemplate = template.Must(template.New("source").Parse(headerInternal + source))
)

type jsClass struct {
	ClassName   string
	ClassID     string
	DecoderName string
	Methods     []jsMethod
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

func jsGenerate(g *javascriptGenerator, f *DefFile, w io.Writer) error {
	ctx := jsFile{Source: f.Name}
	for _, fn := range f.Deps {
		jsfn := jsFileName(fn, jsProtoFileSuffix)
		if strings.HasPrefix(fn, "google/") {
			ctx.Requires = append(ctx.Requires, "google-protobuf/"+jsfn)
		} else {
			ctx.Requires = append(ctx.Requires, "./"+jsfn)
		}
	}
	ctx.Requires = append(ctx.Requires, "./"+jsFileName(f.Name, jsProtoFileSuffix))
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
		if dev.ClassID == busClassID {
			cls.DecoderName = "RouteDecoder"
		} else {
			cls.DecoderName = "MsgDecoder"
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
	tmpl := sourceTemplate
	if g.internal {
		tmpl = internalTemplate
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
