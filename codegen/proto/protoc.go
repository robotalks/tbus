package proto

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

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
	request *plugin.CodeGeneratorRequest
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
	if err = proto.Unmarshal(input, &req); err != nil {
		return nil, err
	}

	p.request = &req

	def := &Definition{}
	if err = def.ParseArgs(req.GetParameter()); err != nil {
		return nil, err
	}

	protoFiles := make(map[string]bool)
	for _, fn := range req.FileToGenerate {
		protoFiles[fn] = true
	}

	for _, d := range req.ProtoFile {
		if !protoFiles[d.GetName()] {
			// skip dependencies
			continue
		}

		defFile := &DefFile{
			Name:    d.GetName(),
			Package: d.GetPackage(),
			Deps:    d.GetDependency(),
			Options: FileOptions{
				GoPackage: d.GetOptions().GetGoPackage(),
			},
		}
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
				if m.GetClientStreaming() {
					return nil, fmt.Errorf("method %s.%s: client streaming not supported",
						svc.GetName(), m.GetName())
				}

				if m.GetServerStreaming() {
					// this is an event channel
					if m.GetInputType() != emptyType {
						return nil, fmt.Errorf("event channel %s.%s: input %s invalid, must be "+emptyType[1:],
							svc.GetName(), m.GetName(), m.GetInputType())
					}
					eventChn := &EventChannel{Index: val, Name: m.GetName(), EventType: m.GetOutputType()}
					if err = dev.AddEventChannel(eventChn); err != nil {
						return nil, err
					}
				} else {
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
			}
			defFile.Devices = append(defFile.Devices, dev)
		}
		if len(defFile.Devices) > 0 {
			def.Files = append(def.Files, defFile)
		}
	}

	return def, nil
}

// NewOutput implements Output
func (p *protocParser) NewOutput(writer io.Writer) (Output, error) {
	return &protocOutput{request: p.request, writer: writer}, nil
}

type protocOutput struct {
	request  *plugin.CodeGeneratorRequest
	writer   io.Writer
	response plugin.CodeGeneratorResponse
}

func (o *protocOutput) Stage(command, parameter string) ([]*GeneratedFile, error) {
	o.request.Parameter = proto.String(parameter)
	input, err := proto.Marshal(o.request)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("protoc-gen-" + command)
	cmd.Env = os.Environ()
	cmd.Stdin = bytes.NewBuffer(input)
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	err = proto.Unmarshal(out.Bytes(), &o.response)
	if err != nil {
		return nil, err
	}

	files := make([]*GeneratedFile, 0, len(o.response.File))
	for _, f := range o.response.File {
		if f.Name == nil || *f.Name == "" {
			continue
		}
		files = append(files, &GeneratedFile{Name: *f.Name, Content: f.Content})
	}
	return files, nil
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
	for _, f := range w.response.File {
		if f.Name != nil && *f.Name == *w.file.Name {
			f.Content = w.file.Content
			return nil
		}
	}
	w.response.File = append(w.response.File, w.file)
	return nil
}
