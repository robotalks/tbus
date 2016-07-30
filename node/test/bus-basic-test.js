var expect = require('chai').expect,
    flow = require('js-flow'),
    debug = require('debug'),
    protocol = require('../lib/protocol.js'),
    Bus = require('../lib/bus.js'),
    Master = require('../lib/master.js'),
    BusCtl = require('../gen/bus_device.js').BusCtl;

function bailout(err, done) {
    return function (err) {
        if (err != null) {
            done(err);
        }
    }
}

describe('Bus', function () {
    before(function () {
        var protoLog = debug('tbus:proto');
        protocol.setLogger(function (event, data) {
            protoLog("%s: %j", event, data);
        });
    });

    var log;
    beforeEach(function () {
        log = debug('tbus:test:' + this.currentTest.title);
        log('start');
    });

    it('enumeration', function (done) {
        var bus = new Bus();
        var master = new Master({bus: bus});
        var ctl = new BusCtl(master);
        ctl.enumerate(function (err, busenum) {
            expect(err).to.be.null;
            expect(busenum).not.to.be.null;
            var devices = busenum.getDevicesList();
            expect(devices).not.to.be.empty;
            expect(devices).to.have.lengthOf(1);
            expect(devices[0].getClassId()).to.equal(0x0001);
            expect(devices[0].getAddress()).to.equal(0);
            done();
        });
    });

    it('bus-tree', function (done) {
        var bus = new Bus({ id: 1 });

        var bus1 = new Bus({ id: 2, slave: true });
        bus.plug(bus1.slaveDevice());

        var bus2 = new Bus({ id: 3, slave: true });
        bus1.plug(bus2.slaveDevice());

        var master = new Master({bus: bus});

        var addrs = [];
        flow.steps()
            .chain()
            .do(function (next) {
                new BusCtl(master).enumerate(next);
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.have.lengthOf(2);
                addrs.push(devices[1].getAddress());
                new BusCtl(master, addrs).enumerate(next);
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.have.lengthOf(2);
                addrs.push(devices[1].getAddress());
                new BusCtl(master, addrs).enumerate(next);
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.have.lengthOf(1);
                next();
            })
            .run(done);
    });
});
