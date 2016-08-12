package tbus

import (
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type runners struct {
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func newRunners() *runners {
	return &runners{
		stopCh: make(chan struct{}),
	}
}

func (r *runners) Go(runner Runner) {
	r.wg.Add(1)
	go func() {
		runner.Run(r.stopCh)
		r.wg.Done()
	}()
}

func (r *runners) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

func TestBasicBus(t *testing.T) {
	Convey("Bus", t, func() {
		Convey("enumeration", func() {
			r := newRunners()
			bus := NewLocalBus()
			master := NewLocalMaster(bus.HostPort())
			busctl := NewBusCtl(master)
			r.Go(master)
			enum, err := busctl.Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			devs := enum.Devices
			So(devs, ShouldNotBeEmpty)
			So(devs, ShouldHaveLength, 1)
			d := devs[0]
			So(d, ShouldNotBeNil)
			So(d.ClassId, ShouldEqual, BusClassID)
			So(d.Address, ShouldEqual, 0)
			r.Stop()
		})

		Convey("bus tree", func() {
			r := newRunners()
			bus := NewLocalBus()
			master := NewLocalMaster(bus.HostPort())
			bus1 := NewLocalBus()
			bus1.Device().SetDeviceID(1)
			busDev1 := bus1.SlaveDevice()
			bus.Plug(busDev1.SetDeviceID(1))
			bus2 := NewLocalBus()
			bus2.Device().SetDeviceID(2)
			busDev2 := bus2.SlaveDevice()
			bus1.Plug(busDev2.SetDeviceID(2))
			busctl := NewBusCtl(master)
			r.Go(master)
			enum, err := busctl.Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			So(enum.Devices, ShouldHaveLength, 2)
			So(enum.Devices[1].DeviceId, ShouldEqual, 1)
			enum, err = busctl.
				SetAddress([]uint8{busDev1.Address()}).
				Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			So(enum.Devices, ShouldHaveLength, 2)
			So(enum.Devices[1].DeviceId, ShouldEqual, 2)
			enum, err = busctl.
				SetAddress([]uint8{busDev1.Address(), busDev2.Address()}).
				Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			So(enum.Devices, ShouldHaveLength, 1)
			r.Stop()
		})
	})
}
