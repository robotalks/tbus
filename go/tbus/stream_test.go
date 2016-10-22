package tbus

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func dumpBytes(prefix string, data []byte) {
	var line string
	for i := 0; i < len(data); i++ {
		if (i & 0xf) == 0 {
			if i != 0 {
				line += "\n"
			}
			line += fmt.Sprintf("%s %04x:", prefix, i)
		}
		line += fmt.Sprintf(" %02X", uint8(data[i]))
	}
	fmt.Fprintln(os.Stderr, line)
}

func dumpError(prefix string, n int, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s ERROR %d %v\n", prefix, n, err)
	}
}

type listenerWrapper struct {
	tag      string
	listener net.Listener
}

func (l *listenerWrapper) Accept() (io.ReadWriteCloser, error) {
	conn, err := l.listener.Accept()
	return wrapConn(l.tag, conn, err)
}

func (l *listenerWrapper) Close() error {
	return l.listener.Close()
}

func wrapListener(tag string, listener net.Listener) *listenerWrapper {
	return &listenerWrapper{tag: tag, listener: listener}
}

type connWrapper struct {
	tag  string
	conn net.Conn
}

func (c *connWrapper) Read(b []byte) (n int, err error) {
	n, err = c.conn.Read(b)
	if err == nil {
		dumpBytes(c.tag+"<", b[:n])
	} else {
		dumpError(c.tag+"<", n, err)
	}
	return
}

func (c *connWrapper) Write(b []byte) (n int, err error) {
	dumpBytes(c.tag+">", b)
	n, err = c.conn.Write(b)
	if err != nil {
		dumpError(c.tag+">", n, err)
	}
	return
}

func (c *connWrapper) Close() error {
	return c.conn.Close()
}

func wrapConn(tag string, conn net.Conn, err error) (*connWrapper, error) {
	return &connWrapper{tag: tag, conn: conn}, err
}

func testRemote(busDev *BusDev, testFn func(*LocalMaster)) {
	fmt.Fprintln(os.Stderr, "testRemote Enter")
	defer fmt.Fprintln(os.Stderr, "testRemote Leave")

	var hostErr, portErr, devErr error
	var wg sync.WaitGroup

	wg.Add(3)

	// host side
	var localAddr net.TCPAddr
	localAddr.IP = net.ParseIP("127.0.0.1")
	listener, err := net.ListenTCP("tcp", &localAddr)
	localAddr.Port = listener.Addr().(*net.TCPAddr).Port
	So(err, ShouldBeNil)

	errCh := make(chan error, 3)

	host := NewRemoteDeviceHost(wrapListener("H", listener))
	go func() {
		hostErr = host.Run()
		dumpError("Host", 0, hostErr)
		if hostErr != nil {
			errCh <- hostErr
		}
		wg.Done()
	}()

	// device side
	netPort := NewRemoteBusPort(busDev, DialerFunc(func() (io.ReadWriteCloser, error) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localAddr.Port))
		return wrapConn("P", conn, err)
	}))
	go func() {
		portErr = netPort.Run()
		dumpError("Port", 0, portErr)
		if portErr != nil {
			errCh <- portErr
		}
		wg.Done()
	}()

	// host side again
	select {
	case dev := <-host.AcceptChan():
		master := NewLocalMaster(dev)
		master.InvocationTimeout = time.Second
		go func() {
			devErr = dev.Run()
			dumpError("Master", 0, devErr)
			wg.Done()
		}()
		So(dev.DeviceInfo().Address, ShouldEqual, busDev.DeviceInfo().Address)
		testFn(master)
	case err := <-errCh:
		So(err, ShouldBeNil)
	}

	listener.Close()
	netPort.Close()
	wg.Wait()
	So(hostErr, ShouldBeNil)
	So(portErr, ShouldBeNil)
	So(devErr, ShouldBeNil)
}

type testLED struct {
	LogicBase
	on bool
}

func (d *testLED) SetPowerState(param *LEDPowerState) error {
	d.on = param.On
	return nil
}

type errorLED struct {
	LogicBase
	err error
}

func (d *errorLED) SetPowerState(param *LEDPowerState) error {
	if d.err != nil {
		return d.err
	}
	return fmt.Errorf("errorLED")
}

func TestStreamDevice(t *testing.T) {
	Convey("StreamDevice", t, func() {
		Convey("remote", func() {
			bus := NewLocalBus()
			ledLogic := &testLED{}
			led := NewLEDDev(ledLogic)
			led.SetDeviceID(3).Info.AddLabel("name", "led")
			bus.Plug(led)

			testRemote(NewBusDev(bus), func(master *LocalMaster) {
				busctl := NewBusCtl(master)
				enum, err := busctl.Enumerate().Wait()
				So(err, ShouldBeNil)
				So(enum, ShouldNotBeNil)
				devs := enum.Devices
				So(devs, ShouldHaveLength, 1)
				So(devs[0].ClassId, ShouldEqual, LEDClassID)
				So(devs[0].Labels, ShouldNotBeEmpty)
				So(devs[0].Labels, ShouldContainKey, "name")
				So(devs[0].Labels["name"], ShouldEqual, "led")

				// turn on led
				ledctl := NewLEDCtl(master)
				ledctl.SetAddress(devs[0].DeviceAddress())
				err = ledctl.On().Wait()
				So(err, ShouldBeNil)
				So(ledLogic.on, ShouldBeTrue)
				err = ledctl.Off().Wait()
				So(err, ShouldBeNil)
				So(ledLogic.on, ShouldBeFalse)
			})
		})

		Convey("error", func() {
			bus := NewLocalBus()
			bus.Plug(NewLEDDev(&errorLED{}))
			testRemote(NewBusDev(bus), func(master *LocalMaster) {
				busctl := NewBusCtl(master)
				enum, err := busctl.Enumerate().Wait()
				So(err, ShouldBeNil)
				ledctl := NewLEDCtl(master)
				ledctl.SetAddress(enum.Devices[0].DeviceAddress())
				err = ledctl.On().Wait()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "errorLED")
			})
		})

		Convey("route", func() {
			Convey("invalid address", func() {
				testRemote(NewBusDev(NewLocalBus()), func(master *LocalMaster) {
					ledctl := NewLEDCtl(master)
					ledctl.SetAddress([]uint8{100})
					err := ledctl.On().Wait()
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid address")
				})
			})

			Convey("not supported", func() {
				bus := NewLocalBus()
				bus.Plug(NewLEDDev(&testLED{}))
				testRemote(NewBusDev(bus), func(master *LocalMaster) {
					busctl := NewBusCtl(master)
					enum, err := busctl.Enumerate().Wait()
					So(err, ShouldBeNil)
					ledctl := NewLEDCtl(master)
					ledctl.SetAddress(append(enum.Devices[0].DeviceAddress(), uint8(10)))
					err = ledctl.On().Wait()
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "route not supported")
				})
			})
		})
	})
}

type testButton struct {
	LogicBase
	pressed bool
}

func (b *testButton) GetState() (*ButtonState, error) {
	return &ButtonState{Pressed: b.pressed}, nil
}

func (b *testButton) simulatePressed(pressed bool) {
	b.pressed = pressed
	b.EmitEvent(ChnButtonStateID, &ButtonState{Pressed: b.pressed})
}

func TestStreamEvents(t *testing.T) {
	Convey("StreamDevice", t, func() {
		bus := NewLocalBus()
		btnLogic := &testButton{}
		btn := NewButtonDev(btnLogic)
		btn.SetDeviceID(3).Info.AddLabel("name", "button")
		bus.Plug(btn)

		testRemote(NewBusDev(bus), func(master *LocalMaster) {
			busctl := NewBusCtl(master)
			enum, err := busctl.Enumerate().Wait()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			devs := enum.Devices
			So(devs, ShouldHaveLength, 1)
			So(devs[0].ClassId, ShouldEqual, ButtonClassID)
			So(devs[0].Labels, ShouldNotBeEmpty)
			So(devs[0].Labels, ShouldContainKey, "name")
			So(devs[0].Labels["name"], ShouldEqual, "button")

			btnctl := NewButtonCtl(master)
			btnctl.SetAddress(devs[0].DeviceAddress())
			state, err := btnctl.GetState().Wait()
			So(err, ShouldBeNil)
			So(state.Pressed, ShouldBeFalse)
			chn := btnctl.State()
			go btnLogic.simulatePressed(true)
			state = <-chn.C
			So(state.Pressed, ShouldBeTrue)
		})
	})
}
