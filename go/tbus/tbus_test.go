package tbus

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBasicBus(t *testing.T) {
	Convey("Bus", t, func() {
		Convey("enumeration", func() {
			master := NewLocalMaster(NewBusDev(NewLocalBus()))
			busctl := NewBusCtl(master)
			enum, err := busctl.Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			devs := enum.Devices
			So(devs, ShouldBeEmpty)
		})

		Convey("bus tree", func() {
			bus := NewLocalBus()
			master := NewLocalMaster(NewBusDev(bus))
			bus1 := NewLocalBus()
			busDev1 := NewBusDev(bus1)
			bus.Plug(busDev1.SetDeviceID(1))
			So(busDev1.DeviceInfo().Address, ShouldNotEqual, 0)
			busDev2 := NewBusDev(NewLocalBus())
			bus1.Plug(busDev2.SetDeviceID(2))
			So(busDev2.DeviceInfo().Address, ShouldNotEqual, 0)
			busctl := NewBusCtl(master)
			enum, err := busctl.Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			So(enum.Devices, ShouldHaveLength, 1)
			So(enum.Devices[0].ClassId, ShouldEqual, BusClassID)
			So(enum.Devices[0].DeviceId, ShouldEqual, 1)
			enum, err = busctl.
				SetAddress(DeviceAddress(busDev1)).
				Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			So(enum.Devices, ShouldHaveLength, 1)
			So(enum.Devices[0].ClassId, ShouldEqual, BusClassID)
			So(enum.Devices[0].DeviceId, ShouldEqual, 2)
			enum, err = busctl.
				SetAddress(DeviceAddress(busDev1, busDev2)).
				Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			So(enum.Devices, ShouldBeEmpty)
		})
	})
}
