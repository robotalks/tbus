package tbus

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type testLED struct {
	LogicBase
	on bool
}

func (d *testLED) SetPowerState(param *LEDPowerState) error {
	d.on = param.On
	return nil
}

func TestStreamDevice(t *testing.T) {
	Convey("StreamDevice", t, func() {
		Convey("remote", func() {
			var hostErr, portErr, devErr error
			var wg sync.WaitGroup

			wg.Add(3)

			// host side
			var localAddr net.TCPAddr
			localAddr.IP = net.ParseIP("127.0.0.1")
			listener, err := net.ListenTCP("tcp", &localAddr)
			localAddr.Port = listener.Addr().(*net.TCPAddr).Port
			So(err, ShouldBeNil)

			host := NewNetDeviceHost(listener)
			go func() {
				hostErr = host.Run()
				wg.Done()
			}()

			// device side
			bus := NewLocalBus()
			ledLogic := &testLED{}
			led := NewLEDDev(ledLogic)
			led.SetDeviceID(3)
			bus.Plug(led)
			busDev := NewBusDev(bus)
			netPort := NewNetBusPort(busDev, func() (io.ReadWriteCloser, error) {
				return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localAddr.Port))
			})
			go func() {
				portErr = netPort.Run()
				wg.Done()
			}()

			// host side again
			dev := <-host.AcceptChan()
			master := NewLocalMaster(dev)
			busctl := NewBusCtl(master)
			go func() {
				devErr = dev.Run()
				wg.Done()
			}()

			enum, err := busctl.Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			devs := enum.Devices
			So(devs, ShouldHaveLength, 1)
			So(dev.Address(), ShouldEqual, busDev.Address())
			So(devs[0].ClassId, ShouldEqual, LEDClassID)

			// turn on led
			ledctl := NewLEDCtl(master)
			ledctl.SetAddress([]uint8{uint8(devs[0].Address)})
			err = ledctl.On()
			So(err, ShouldBeNil)
			So(ledLogic.on, ShouldBeTrue)
			err = ledctl.Off()
			So(err, ShouldBeNil)
			So(ledLogic.on, ShouldBeFalse)

			netPort.Conn().Close()
			listener.Close()
			wg.Wait()
			So(hostErr, ShouldNotBeNil)
			So(portErr, ShouldNotBeNil)
			So(devErr, ShouldBeNil)
		})
	})
}
