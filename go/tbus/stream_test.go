package tbus

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func testRemote(busDev *BusDev, testFn func(*LocalMaster)) {
	var hostErr, portErr, devErr error
	var wg sync.WaitGroup

	wg.Add(3)

	// host side
	var localAddr net.TCPAddr
	localAddr.IP = net.ParseIP("127.0.0.1")
	listener, err := net.ListenTCP("tcp", &localAddr)
	localAddr.Port = listener.Addr().(*net.TCPAddr).Port
	So(err, ShouldBeNil)

	host := NewRemoteDeviceHost(listener)
	go func() {
		hostErr = host.Run()
		wg.Done()
	}()

	// device side
	netPort := NewRemoteBusPort(busDev, func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localAddr.Port))
	})
	go func() {
		portErr = netPort.Run()
		wg.Done()
	}()

	// host side again
	dev := <-host.AcceptChan()
	master := NewLocalMaster(dev)
	go func() {
		devErr = dev.Run()
		wg.Done()
	}()
	So(dev.DeviceInfo().Address, ShouldEqual, busDev.DeviceInfo().Address)
	testFn(master)

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
				enum, err := busctl.Enumerate()
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
				err = ledctl.On()
				So(err, ShouldBeNil)
				So(ledLogic.on, ShouldBeTrue)
				err = ledctl.Off()
				So(err, ShouldBeNil)
				So(ledLogic.on, ShouldBeFalse)
			})
		})

		Convey("error", func() {
			bus := NewLocalBus()
			bus.Plug(NewLEDDev(&errorLED{}))
			testRemote(NewBusDev(bus), func(master *LocalMaster) {
				busctl := NewBusCtl(master)
				enum, err := busctl.Enumerate()
				So(err, ShouldBeNil)
				ledctl := NewLEDCtl(master)
				ledctl.SetAddress(enum.Devices[0].DeviceAddress())
				err = ledctl.On()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "errorLED")
			})
		})

		Convey("route", func() {
			Convey("invalid address", func() {
				testRemote(NewBusDev(NewLocalBus()), func(master *LocalMaster) {
					ledctl := NewLEDCtl(master)
					ledctl.SetAddress([]uint8{100})
					err := ledctl.On()
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "invalid address")
				})
			})

			Convey("not supported", func() {
				bus := NewLocalBus()
				bus.Plug(NewLEDDev(&testLED{}))
				testRemote(NewBusDev(bus), func(master *LocalMaster) {
					busctl := NewBusCtl(master)
					enum, err := busctl.Enumerate()
					So(err, ShouldBeNil)
					ledctl := NewLEDCtl(master)
					ledctl.SetAddress(append(enum.Devices[0].DeviceAddress(), uint8(10)))
					err = ledctl.On()
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "route not supported")
				})
			})
		})
	})
}
