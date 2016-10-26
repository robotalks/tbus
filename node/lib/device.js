var Class = require('js-class'),
    protocol = require('./protocol.js');

var Device = Class({
    constructor: function (classId, logic, options) {
        this.options = options || {}
        this._info = {
            address: 0,
            classId: classId,
            deviceId: this.options.id || 0,
            labels: {}
        };
        this._methods = [];
        this.logic = logic;
        logic.setDevice(this);
    },

    deviceInfo: function () {
        return this._info;
    },

    attach: function (busPort, addr) {
        this._busPort = busPort;
        this._info.address = addr;
        return this;
    },

    address: function () {
        return this._info.address;
    },

    classId: function () {
        return this._info.classId;
    },

    deviceId: function () {
        return this._info.deviceId;
    },

    busPort: function () {
        return this._busPort;
    },

    setDeviceId: function (id) {
        this._info.deviceId = id;
        return this;
    },

    dispatchMsg: function (msg, done) {
        setImmediate(done);
        var self = this;
        if (protocol.needRoute(msg)) {
            this._routeMsg(msg, function (err) {
                if (err != null) {
                    self.reply(msg.head.msgId, null, err);
                }
            });
        } else {
            var methodIndex = msg.body.flag;
            var methodParams = msg.body.data;

            if (methodIndex == 0) {   // for device info
                setImmediate(function() {
                    self.reply(msg.head.msgId, new DeviceInfo().fromDevice(self));
                });
            } else {
                var method = this._methods[methodIndex];
                if (method == null) {
                    this.reply(msg.head.msgId, null,
                        new Error('unknown method ' + methodIndex));
                } else {
                    setImmediate(function () {
                        method.func(methodParams, function (err, reply) {
                            self.reply(msg.head.msgId, reply, err);
                        });
                    });
                }
            }
        }
        return this;
    },

    _routeMsg: function (msg, done) {
        var routeFn = this.logic.routeMsg;
        if (typeof(routeFn) == 'function') {
            routeFn.call(this.logic, msg, done);
        } else {
            done(new Error('routing not supported'));
        }
    },

    // for sub-class use
    defineMethod: function (index, name, fn) {
        this._methods[index] = {
            name: name,
            func: fn
        };
        return this;
    },

    reply: function (msgId, reply, err) {
        if (this._busPort) {
            var encoder = new protocol.Encoder();
            encoder.messageId(msgId);
            if (err != null) {
                encoder.encodeError(err);
            } else {
                encoder.encodeProto(0, reply);
            }
            this._busPort.dispatchMsg(encoder.buildMsg(), function (err) {
                if (err != null) {
                    console.error('Device.reply error: %j, %j\n%j', err, this._info, err.stack);
                }
            }.bind(this));
        }
        return this;
    }
});

module.exports = Device;
