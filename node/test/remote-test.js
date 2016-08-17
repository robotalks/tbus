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

    function testRemote(busDev, test, done) {
        var listener = net.createServer();
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
                    test(new tbus.Master(device), next);
                });

                // device side
                new tbus.RemoteBusPort(busDev,
                    tbus.SocketConnector(function () {
                        return net.connect(port);
                    })).connect();
            })
            .run(done);
    }

    var testLEDLogic = Class({
        setDevice: function () {},

        setPowerState: function (state, done) {
            log('ledlogic: setPowerState %j, %j', state.getOn(), state.toObject());
            this.on = state.getOn();
            done();
        }
    });

    it('remote', function (done) {
        var bus = new tbus.Bus();
        var ledlogic = new testLEDLogic();
        var dev = new tbus.LEDDev(ledlogic);
        dev.setDeviceId(3);
        dev.deviceInfo().labels["name"] = "led";
        bus.plug(dev);
        testRemote(new tbus.BusDev(bus), function (master, done) {
            var ledctl;
            flow.steps()
                .chain()
                .do(function (next) {
                    new tbus.BusCtl(master).enumerate(next);
                })
                .do(function (busenum, next) {
                    log('bus enum: %j', busenum.toObject());
                    var devices = busenum.getDevicesList();
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
        }, done);
    });

    var errorLEDLogic = Class({
        setDevice: function () {},

        setPowerState: function (state, done) {
            done(new Error('errorLED'));
        }
    });

    it('error', function (done) {
        var bus = new tbus.Bus();
        bus.plug(new tbus.LEDDev(new errorLEDLogic()));
        testRemote(new tbus.BusDev(bus), function (master, done) {
            var ledctl;
            flow.steps()
                .chain()
                .do(function (next) {
                    new tbus.BusCtl(master).enumerate(next);
                })
                .do(function (busenum, next) {
                    log('bus enum: %j', busenum.toObject());
                    var devices = busenum.getDevicesList();
                    expect(devices).to.have.lengthOf(1);
                    expect(devices[0].getClassId()).to.equal(tbus.LEDDev.CLASS_ID);
                    ledctl = new tbus.LEDCtl(master, [devices[0].getAddress()]);
                    ledctl.on(next);
                })
                .run(function (err) {
                    expect(err).to.be.an('error');
                    expect(err.message).to.equal('errorLED');
                    done();
                });
        }, done);
    });

    describe('route', function () {
        it('invalid address', function (done) {
            testRemote(new tbus.BusDev(new tbus.Bus()), function (master, done) {
                new tbus.LEDCtl(master, [100]).on(function (err) {
                    expect(err).to.be.an('error');
                    expect(err.message).to.contain('invalid address');
                    done();
                });
            }, done);
        });

        it('not supported', function (done) {
            var bus = new tbus.Bus();
            bus.plug(new tbus.LEDDev(new testLEDLogic()));
            testRemote(new tbus.BusDev(bus), function (master, done) {
                flow.steps()
                    .chain()
                    .do(function (next) {
                        new tbus.BusCtl(master).enumerate(next);
                    })
                    .do(function (busenum, next) {
                        new tbus.LEDCtl(master, [
                            busenum.getDevicesList()[0].getAddress(),
                            100
                        ]).on(next);
                    })
                    .run(function (err) {
                        expect(err).to.be.an('error');
                        expect(err.message).to.contain('routing not supported');
                        done();
                    });
            }, done);
        });
    });
});
