package tbus

import (
	"io"
	"net"
	"sync"

	prot "github.com/evo-bots/tbus/go/tbus/protocol"
	proto "github.com/golang/protobuf/proto"
)

// MsgStreamer read/write msg using stream
type MsgStreamer struct {
	Writer io.Writer
	lock   sync.Mutex
}

// NewMsgStreamer creates a new MsgStreamer
func NewMsgStreamer(writer io.Writer) *MsgStreamer {
	return &MsgStreamer{Writer: writer}
}

// SendMsg implements MsgSender
func (s *MsgStreamer) SendMsg(msg *prot.Msg) (err error) {
	s.lock.Lock()
	_, err = s.Writer.Write(msg.Head.Raw)
	if err == nil {
		_, err = s.Writer.Write(msg.Body.Raw)
	}
	s.lock.Unlock()
	return
}

// DecodeStream decode msgs from stream and pipe to sender
func DecodeStream(reader io.Reader, sender MsgSender) error {
	for {
		msg, err := prot.Decode(reader)
		if err == io.EOF {
			return nil
		}
		if err == nil {
			err = sender.SendMsg(&msg)
		}
		if err != nil {
			return err
		}
	}
}

// StreamDevice sends msg to a writer
type StreamDevice struct {
	MsgStreamer
	Info   DeviceInfo
	Reader io.Reader

	busPort BusPort
	init    bool
	initErr error
}

// NewStreamDevice creates a stream device
func NewStreamDevice(classID uint32, rw io.ReadWriter) *StreamDevice {
	dev := &StreamDevice{Reader: rw, init: true}
	dev.Writer = rw
	dev.Info.ClassId = classID
	return dev
}

// DeviceInfo returns device information
func (d *StreamDevice) DeviceInfo() DeviceInfo {
	return d.Info
}

// BusPort implements Device
func (d *StreamDevice) BusPort() BusPort {
	return d.busPort
}

// SetDeviceID sets device ID
func (d *StreamDevice) SetDeviceID(id uint32) *StreamDevice {
	d.Info.DeviceId = id
	return d
}

// AttachTo implements Device
func (d *StreamDevice) AttachTo(busPort BusPort, addr uint8) {
	d.busPort = busPort
	d.Info.Address = uint32(addr)
	if d.init && busPort != nil {
		// during init send attach info to stream
		encoded, _ := proto.Marshal(&d.Info)
		msg, err := prot.EncodeAsMsg(nil, 0, 0, encoded)
		if err == nil {
			err = d.SendMsg(msg)
		}
		if err != nil {
			d.initErr = err
		}
	}
	d.init = false
}

// Run pipes remote msg to bus port
func (d *StreamDevice) Run() error {
	if d.initErr != nil {
		return d.initErr
	}
	return DecodeStream(d.Reader, d.busPort)
}

// StreamBusPort exposes a device to remote
type StreamBusPort struct {
	MsgStreamer
	Reader io.Reader
	Device Device
}

// NewStreamBusPort creates a stream bus port
func NewStreamBusPort(rw io.ReadWriter, dev Device, addr uint8) *StreamBusPort {
	p := &StreamBusPort{Reader: rw}
	p.Writer = rw
	p.Device = dev
	p.Device.AttachTo(p, addr)
	return p
}

// Run pipes remote msg to device
func (p *StreamBusPort) Run() error {
	return DecodeStream(p.Reader, p.Device)
}

// Dial is abstract remote connector
type Dial func() (io.ReadWriteCloser, error)

// RemoteBusPort exposes a device over network
type RemoteBusPort struct {
	Dialer Dial
	Device Device

	conn io.ReadWriteCloser
}

// NewRemoteBusPort creates a RemoteBusPort
func NewRemoteBusPort(dev Device, dialer Dial) *RemoteBusPort {
	return &RemoteBusPort{Dialer: dialer, Device: dev}
}

// Conn returns current connection
func (p *RemoteBusPort) Conn() io.ReadWriteCloser {
	return p.conn
}

// Run connect to remote and host the device
func (p *RemoteBusPort) Run() error {
	conn, err := p.Dialer()
	if err == nil {
		p.conn = conn
		err = p.runConn()
		p.conn = nil
	}
	return err
}

func (p *RemoteBusPort) runConn() error {
	defer p.conn.Close()

	// the first message is sending device info for bus attachment
	info := p.Device.DeviceInfo()
	encoded, err := proto.Marshal(&info)
	if err != nil {
		return err
	}
	if _, err = prot.Encode(p.conn, nil, 0, 0, encoded); err != nil {
		return err
	}

	// expect a bus attachment
	msg, err := prot.Decode(p.conn)
	if err != nil {
		return err
	}
	info = DeviceInfo{}
	if err = proto.Unmarshal(msg.Body.Data, &info); err != nil {
		return err
	}

	// do a bus attach
	port := NewStreamBusPort(p.conn, p.Device, uint8(info.Address))
	err = port.Run()
	p.Device.AttachTo(nil, 0)
	return err
}

// RemoteDeviceHost accepts connections from RemoteBusPort
// and creates StreamDevice for each connection.
type RemoteDeviceHost struct {
	Listener net.Listener
	acceptCh chan *StreamDevice
}

// NewRemoteDeviceHost creates a new remote device host
func NewRemoteDeviceHost(listener net.Listener) *RemoteDeviceHost {
	return &RemoteDeviceHost{
		Listener: listener,
		acceptCh: make(chan *StreamDevice),
	}
}

// AcceptChan returns chan for accepted device
func (h *RemoteDeviceHost) AcceptChan() <-chan *StreamDevice {
	return h.acceptCh
}

// Run starts accepting device connections
func (h *RemoteDeviceHost) Run() error {
	for {
		conn, err := h.Listener.Accept()
		if err != nil {
			return err
		}
		msg, err := prot.Decode(conn)
		if err != nil {
			conn.Close()
		} else {
			info := &DeviceInfo{}
			err = proto.Unmarshal(msg.Body.Data, info)
			if err != nil {
				conn.Close()
			} else {
				device := NewStreamDevice(info.ClassId, conn)
				device.SetDeviceID(info.DeviceId)
				h.acceptCh <- device
			}
		}
	}
}
