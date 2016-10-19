package tbus

import (
	"io"
	"net"
	"strings"
	"sync"
)

// MsgStreamer write msg using stream
type MsgStreamer struct {
	Writer io.Writer
	lock   sync.Mutex
}

// NewMsgStreamer creates a new MsgStreamer
func NewMsgStreamer(writer io.Writer) *MsgStreamer {
	return &MsgStreamer{Writer: writer}
}

// DispatchMsg implements MsgDispatcher
func (s *MsgStreamer) DispatchMsg(msg *Msg) (err error) {
	s.lock.Lock()
	err = msg.EncodeTo(s.Writer)
	s.lock.Unlock()
	return
}

// DecodeStream decode msgs from stream and pipe to dispatcher
func DecodeStream(reader io.Reader, dispatcher MsgDispatcher) error {
	for {
		msg, err := Decode(reader)
		if err == io.EOF {
			return nil
		}
		if err == nil {
			err = dispatcher.DispatchMsg(&msg)
		}
		if err != nil {
			return IgnoreClosingErr(err)
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
		d.initErr = BuildMsg().EncodeBody(0, &d.Info).Build().Dispatch(d)
	}
	d.init = false
}

// Run pipes remote msg to bus port
func (d *StreamDevice) Run() error {
	if d.initErr != nil {
		return IgnoreClosingErr(d.initErr)
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

// RemoteDevice is a remote device communicated over a connection
type RemoteDevice interface {
	Device
	io.Closer
	Run() error
}

// Dialer is abstract remote connector
type Dialer interface {
	Dial() (io.ReadWriteCloser, error)
}

// DialerFunc is func implementation of Dialer
type DialerFunc func() (io.ReadWriteCloser, error)

// Dial implements Dialer
func (d DialerFunc) Dial() (io.ReadWriteCloser, error) {
	return d()
}

// Listener accepts connections
type Listener interface {
	io.Closer
	Accept() (io.ReadWriteCloser, error)
}

// NetListenerWrapper wraps net.Listener
type NetListenerWrapper struct {
	Listener net.Listener
}

// Accept implements Listener
func (l *NetListenerWrapper) Accept() (io.ReadWriteCloser, error) {
	return l.Listener.Accept()
}

// Close implements Listener
func (l *NetListenerWrapper) Close() error {
	return l.Listener.Close()
}

// NetListener creates a wrapper for net.Listener
func NetListener(listener net.Listener) *NetListenerWrapper {
	return &NetListenerWrapper{Listener: listener}
}

// RemoteBusPort exposes a device over network
type RemoteBusPort struct {
	Dialer Dialer
	Device Device

	conn io.ReadWriteCloser
}

// NewRemoteBusPort creates a RemoteBusPort
func NewRemoteBusPort(dev Device, dialer Dialer) *RemoteBusPort {
	return &RemoteBusPort{Dialer: dialer, Device: dev}
}

// Conn returns current connection
func (p *RemoteBusPort) Conn() io.ReadWriteCloser {
	return p.conn
}

// Close closes the connection
func (p *RemoteBusPort) Close() error {
	conn := p.conn
	if conn != nil {
		return conn.Close()
	}
	return nil
}

// Run connect to remote and host the device
func (p *RemoteBusPort) Run() error {
	conn, err := p.Dialer.Dial()
	if err == nil {
		p.conn = conn
		err = p.runConn()
		p.conn = nil
	}
	return IgnoreClosingErr(err)
}

func (p *RemoteBusPort) runConn() error {
	defer p.conn.Close()

	// the first message is sending device info for bus attachment
	info := p.Device.DeviceInfo()
	err := BuildMsg().EncodeBody(0, &info).Build().EncodeTo(p.conn)
	if err != nil {
		return err
	}

	// expect a bus attachment
	info = DeviceInfo{}
	if _, err = DecodeAs(p.conn, &info); err != nil {
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
	Listener Listener
	acceptCh chan RemoteDevice
}

// NewRemoteDeviceHost creates a new remote device host
func NewRemoteDeviceHost(listener Listener) *RemoteDeviceHost {
	return &RemoteDeviceHost{
		Listener: listener,
		acceptCh: make(chan RemoteDevice),
	}
}

// AcceptChan returns chan for accepted device
func (h *RemoteDeviceHost) AcceptChan() <-chan RemoteDevice {
	return h.acceptCh
}

// Run starts accepting device connections
func (h *RemoteDeviceHost) Run() error {
	for {
		conn, err := h.Listener.Accept()
		if err != nil {
			return IgnoreClosingErr(err)
		}
		info := &DeviceInfo{}
		if _, err = DecodeAs(conn, info); err != nil {
			conn.Close()
		} else {
			h.acceptCh <- NewRemoteDevice(*info, conn)
		}
	}
}

type remoteStreamDevice struct {
	StreamDevice
	conn io.ReadWriteCloser
}

// NewRemoteDevice creates a connection backed remote device
func NewRemoteDevice(info DeviceInfo, conn io.ReadWriteCloser) RemoteDevice {
	d := &remoteStreamDevice{conn: conn}
	d.Reader = conn
	d.Writer = conn
	d.Info = info
	d.init = true
	return d
}

// Close implements RemoteDevice
func (d *remoteStreamDevice) Close() error {
	return d.conn.Close()
}

// IsErrClosing determines if the error is caused due to stream closing
func IsErrClosing(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	if opErr, ok := err.(*net.OpError); ok && opErr.Err == io.EOF {
		return true
	}
	return false
}

// IgnoreClosingErr returns nil if err is closing error
func IgnoreClosingErr(err error) error {
	if err == nil {
		return nil
	}
	if IsErrClosing(err) {
		return nil
	}
	return err
}
