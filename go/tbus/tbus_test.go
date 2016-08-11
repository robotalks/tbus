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
		Convey("Enumeration", func() {
			r := newRunners()
			bus := NewLocalBus()
			master := NewLocalMaster(bus.Host())
			busctl := NewBusCtl(master)
			r.Go(master)
			enum, err := busctl.Enumerate()
			So(err, ShouldBeNil)
			So(enum, ShouldNotBeNil)
			devs := enum.GetDevices()
			So(devs, ShouldNotBeEmpty)
			So(devs, ShouldHaveLength, 1)
			d := devs[0]
			So(d, ShouldNotBeNil)
			So(d.ClassId, ShouldEqual, BusClassID)
			So(d.Address, ShouldEqual, 0)
			r.Stop()
		})
	})
}
