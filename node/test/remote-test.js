var expect = require('chai').expect,
    net = require('net'),
    Class = require('js-class'),
    flow = require('js-flow'),
    debug = require('debug'),
    tbus = require('../index.js');

describe('RemoteDevice', function () {
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

    var testLEDLogic = Class({
        setDevice: function () {},

        setPowerState: function (state, done) {
            this.on = state.getOn();
            done();
        }
    });

    it('remote', function (done) {
        var listener = net.createServer();
        var master, ledctl, ledlogic = new testLEDLogic();
        flow.steps()
            .chain()
            .do(function (next) {
                listener.listen(0, '127.0.0.1', function () {
                    next(listener.address().port);
                });
            })
            .do(function (port, next) {
                // host side
                var host = new tbus.RemoteDeviceHost(listener);
                host.on('device', function (device) {
                    master = new tbus.Master(device);
                    new tbus.BusCtl(master).enumerate(next);
                });

                // device side
                var bus = new tbus.Bus();
                var dev = new tbus.LEDDev(ledlogic);
                dev.setDeviceId(3);
                bus.plug(dev);

                new tbus.RemoteBusPort(new tbus.BusDev(bus), function (done) {
                    net.connect(port, done);
                }).connect();
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                expect(devices).to.have.lengthOf(1);
                expect(devices[0].getClassId).to.equal(tbus.LEDDev.CLASS_ID);
                ledctl = new tbus.LEDCtl(master, [devices[0].getAddress()]);
                ledctl.On(next);
            })
            .do(function (next) {
                expect(ledlogic.on).to.be.true;
                ledctl.Off(next);
            })
            .do(function (next) {
                expect(ledlogic.on).to.be.false;
                next();
            })
            .run(done);
    });
});
