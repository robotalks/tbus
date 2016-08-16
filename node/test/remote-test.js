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
            log('ledlogic: setPowerState %j, %j', state.getOn(), state.toObject());
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
                    log('host: incoming device connection');
                    master = new tbus.Master(device);
                    new tbus.BusCtl(master).enumerate(next);
                });

                // device side
                var bus = new tbus.Bus();
                var dev = new tbus.LEDDev(ledlogic);
                dev.setDeviceId(3);
                dev.deviceInfo().labels["name"] = "led";
                bus.plug(dev);

                new tbus.RemoteBusPort(new tbus.BusDev(bus),
                    tbus.SocketConnector(function () {
                        return net.connect(port);
                    })).connect();
            })
            .do(function (busenum, next) {
                var devices = busenum.getDevicesList();
                log('bus enum: %j', busenum.toObject());
                expect(devices).to.have.lengthOf(1);
                expect(devices[0].getClassId()).to.equal(tbus.LEDDev.CLASS_ID);
                expect(devices[0].getLabelsMap().has("name")).to.be.true;
                expect(devices[0].getLabelsMap().get("name")).to.equal("led");
                ledctl = new tbus.LEDCtl(master, [devices[0].getAddress()]);
                ledctl.on(next);
            })
            .do(function (next) {
                expect(ledlogic.on).to.be.true;
                ledctl.off(next);
            })
            .do(function (next) {
                expect(ledlogic.on).to.be.false;
                next();
            })
            .run(done);
    });
});
