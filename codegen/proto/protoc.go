package proto

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	proto "github.com/golang/protobuf/proto"
	desc "github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

const emptyType = ".google.protobuf.Empty"

var extClassID = &proto.ExtensionDesc{
	ExtendedType:  (*desc.ServiceOptions)(nil),
	ExtensionType: (*uint32)(nil),
	Field:         50000,
	Name:          "class_id",
	Tag:           "varint,50000,opt,name=class_id,json=classId",
}

var extIndex = &proto.ExtensionDesc{
	ExtendedType:  (*desc.MethodOptions)(nil),
	ExtensionType: (*uint32)(nil),
	Field:         50000,
	Name:          "index",
	Tag:           "varint,50000,opt,name=index",
}

type protocParser struct {
}

// NewProtocParser creates a parser for protoc
func NewProtocParser() Parser {
	return &protocParser{}
}

func uint32Ext(val interface{}, err error) (uint32, error) {
	if err != nil {
		return 0, err
	}
	intValPtr, ok := val.(*uint32)
	if !ok || intValPtr == nil {
		return 0, fmt.Errorf("bad value")
	}
	return *intValPtr, nil
}

func (p *protocParser) Parse(reader io.Reader) (*Definition, error) {
	input, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var req plugin.CodeGeneratorRequest
	err = proto.Unmarshal(input, &req)
	if err != nil {
		return nil, err
	}

	def := &Definition{Parameter: req.GetParameter()}

	protoFiles := make(map[string]bool)
	for _, fn := range req.FileToGenerate {
		protoFiles[fn] = true
	}

	for _, d := range req.ProtoFile {
		if !protoFiles[d.GetName()] {
			// skip dependencies
			continue
		}

		defFile := &DefFile{Name: d.GetName(), Deps: d.GetDependency()}
		for _, svc := range d.GetService() {
			val, err := uint32Ext(proto.GetExtension(svc.GetOptions(), extClassID))
			if err != nil {
				return nil, fmt.Errorf("service %s: option class_id invalid: %v",
					svc.GetName(), err)
			}
			dev := &Device{Name: svc.GetName(), ClassID: val}
			for _, m := range svc.GetMethod() {
				val, err = uint32Ext(proto.GetExtension(m.GetOptions(), extIndex))
				if err != nil {
					return nil, fmt.Errorf("method %s.%s: option index invalid: %v",
						svc.GetName(), m.GetName(), err)
				}
				method := &Method{Index: val, Name: m.GetName()}
				if t := m.GetInputType(); t != emptyType {
					method.RequestType = t
				}
				if t := m.GetOutputType(); t != emptyType {
					method.ResponseType = t
				}
				if err = dev.AddMethod(method); err != nil {
					return nil, err
				}
			}
			defFile.Devices = append(defFile.Devices, dev)
		}
		if len(defFile.Devices) > 0 {
			def.Files = append(def.Files, defFile)
		}
	}

	return def, nil
}

type protocOutput struct {
	writer   io.Writer
	response plugin.CodeGeneratorResponse
}

// NewProtocOutput creates an output for protoc
func NewProtocOutput(writer io.Writer) Output {
	return &protocOutput{writer: writer}
}

func (o *protocOutput) GenerateFile(name string) (io.WriteCloser, error) {
	return &protocFileWriter{
		response: &o.response,
		file:     &plugin.CodeGeneratorResponse_File{Name: proto.String(name)},
	}, nil
}

func (o *protocOutput) Close() error {
	data, err := proto.Marshal(&o.response)
	if err == nil {
		_, err = o.writer.Write(data)
	}
	return err
}

type protocFileWriter struct {
	response *plugin.CodeGeneratorResponse
	file     *plugin.CodeGeneratorResponse_File
	buffer   bytes.Buffer
}

func (w *protocFileWriter) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

func (w *protocFileWriter) Close() error {
	w.file.Content = proto.String(w.buffer.String())
	w.response.File = append(w.response.File, w.file)
	return nil
}
