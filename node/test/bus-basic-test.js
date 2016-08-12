var expect = require('chai').expect,
    flow = require('js-flow'),
    debug = require('debug'),
    tbus = require('../index.js');

describe('Bus', function () {
    before(function () {
        var protoLog = debug('tbus:proto');
        tbus.protocol.setLogger(function (event, data) {
            protoLog("%s: %j", event, data);
        });
    });

    var log;
    beforeEach(function () {
        log = debug('tbus:test:' + this.currentTest.title);
        log('start');
    });

    it('enumeration', function (done) {
        var bus = new tbus.Bus();
        var master = new tbus.Master(new tbus.BusDev(bus));
        var ctl = new tbus.BusCtl(master);
        ctl.enumerate(function (err, busenum) {
            expect(err).to.be.null;
            expect(busenum).not.to.be.null;
            var devices = busenum.getDevicesList();
            expect(devices).to.be.empty;
            done();
        });
    });

    it('bus-tree', function (done) {
        var bus = new tbus.Bus();

        var bus1 = new tbus.Bus();
        var busDev1 = new tbus.BusDev(bus1);
        bus.plug(busDev1.setDeviceId(1));

        var busDev2 = new tbus.BusDev(new tbus.Bus());
        bus1.plug(busDev2.setDeviceId(2));

        var master = new tbus.Master(new tbus.BusDev(bus));
        var ctl = new tbus.BusCtl(master);

        var addrs = [];
        flow.steps()
            .chain()
            .do(function (next) {
                ctl.enumerate(next);
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.have.lengthOf(1);
                expect(devices[0].getClassId()).to.equal(tbus.BusDev.CLASS_ID);
                expect(devices[0].getDeviceId()).to.equal(1);
                addrs.push(devices[0].getAddress());
                ctl.setAddress(addrs).enumerate(next);
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.have.lengthOf(1);
                expect(devices[0].getClassId()).to.equal(tbus.BusDev.CLASS_ID);
                expect(devices[0].getDeviceId()).to.equal(2);
                addrs.push(devices[0].getAddress());
                ctl.setAddress(addrs).enumerate(next);
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.be.empty;
                next();
            })
            .run(done);
    });
});
