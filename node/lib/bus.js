var Class = require('js-class'),
    protocol = require('./protocol.js'),

    DeviceInfo = require('../gen/tbus/bus_pb.js').DeviceInfo,
    BusEnumeration = require('../gen/tbus/bus_pb.js').BusEnumeration;

// extensions
require('./deviceinfo.js');

var Bus = Class({
    constructor: function (options) {
        this.options = options || {};

        this._addressIndex = 0;
        this._devices = [];
    },

    // implement BusLogic
    setDevice: function (dev) {
        this._device = dev;
        return this;
    },

    // implement BusPort
    dispatchMsg: function (msg, done) {
        if (this._device) {
            this._device.busPort().dispatchMsg(msg, done);
        } else {
            done(new Error("no device associated"));
        }
        return this;
    },

    // implement Bus
    plug: function (device) {
        var address = ++ this._addressIndex;
        this._devices[address] = device;
        device.attach(this, address)
        return this;
    },

    // implement Bus
    unplug: function (device) {
        var address = device.address();
        device.attach(null, 0);
        delete this._devices[address];
        return this;
    },

    routeMsg: function (msg, callback) {
        var addr = msg.head.addrs[0];
        var device = this._devices[addr];
        if (device == null) {
            callback(new Error('invalid address ' + addr));
            return this;
        }
        msg.head.addrs = msg.head.addrs.slice(1);
        if (msg.head.addrs.length == 0) {
            delete msg.head.addrs
            delete msg.head.prefix
            delete msg.head.rawPrefix
            msg.head.raw = msg.head.raw.slice(2);
        } else {
            msg.head.raw = msg.head.raw.slice(1);
            msg.head.raw.writeUInt8(msg.head.prefix, 0);
            msg.head.rawPrefix = msg.head.raw.slice(0, msg.head.addrs.length + 1);
        }
        device.dispatchMsg(msg, callback);
        return this;
    },

    enumerate: function (done) {
        var devices = [];
        for (var a in this._devices) {
            var dev = this._devices[a];
            devices.push(new DeviceInfo().fromDevice(dev));
        }
        var busenum = new BusEnumeration();
        busenum.setDevicesList(devices);
        done(null, busenum);
    }
});

module.exports = Bus;
