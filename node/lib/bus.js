var stream = require('stream'),
    Class = require('js-class'),
    protocol = require('./protocol.js'),
    BusDevice = require('./busdev.js'),

    DeviceInfo = require('../gen/busenum_pb.js').DeviceInfo,
    BusEnumeration = require('../gen/busenum_pb.js').BusEnumeration;

var HostStream = Class(stream.Duplex, {
    constructor: function (bus) {
        stream.Duplex.call(this);
        this._bus = bus;
        this._msgs = [];
    },

    writeToHost: function (msg) {
        this._msgs.push(msg);
        this._push();
    },

    _read: function (n) {
        this._reading = true;
        this._push();
    },

    _push: function () {
        if (this._reading) {
            setImmediate(function () {
                while (this._reading && this._msgs.length > 0) {
                    this._reading = this.push(this._msgs.shift());
                }
            }.bind(this));
        }
    },

    _write: function (chunk, encoding, callback) {
        return this._bus._busDev.receiver().write(chunk, encoding, callback);
    }
});

var Bus = Class({
    constructor: function () {
        this._addressIndex = 0;
        this._devices = [];

        this._hostStream = new HostStream(this);

        this._busDev = new BusDevice(this);
        this._devices[0] = this._busDev;
        this._busDev.attach(this, 0);
    },

    device: function () {
        return this._busDev;
    },

    plug: function (device) {
        address = ++ this._addressIndex;
        this._devices[address] = device;
        device.attach(this, address)
        return this;
    },

    hostStream: function () {
        return this._hostStream;
    },

    writeToHost: function (buf) {
        this._hostStream.writeToHost(buf);
    },

    writeToBus: function (buf, done) {
        this._hostStream.write(buf, null, done);
    },

    route: function (info, callback) {
        var device = this._devices[info.addrs[0]];
        if (device == null) {
            callback(new Error('invalid address ' + info.addrs[0]));
            return;
        }
        if (info.addrs.length > 1) {
            callback(null, device.receiver(),
                new protocol.Encoder()
                    .route(info.addrs.slice(1))
                    .toRoutePrefix());
        } else {
            callback(null, device.receiver());
        }
    },

    enumerate: function (done) {
        var devices = [];
        for (var a in this._devices) {
            var dev = this._devices[a];
            var info = new DeviceInfo();
            info.setAddress(dev.address());
            info.setClassId(dev.classId());
            info.setDeviceId(dev.deviceId());
            devices.push(info);
        }
        var busenum = new BusEnumeration();
        busenum.setDeviceList(devices);
        done(null, busenum);
    }
});

module.exports = Bus;
