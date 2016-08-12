var Class = require('js-class'),
    protocol = require('./protocol.js');

var Device = Class({
    constructor: function (classId, logic, options) {
        this.options = options || {}
        this._classId  = classId;
        this._deviceId = this.options.id || 0;
        this._methods = [];
        this.logic = logic;
        logic.setDevice(this);
    },

    attach: function (busPort, addr) {
        this._busPort = busPort;
        this._address = addr;
        return this;
    },

    address: function () {
        return this._address;
    },

    classId: function () {
        return this._classId;
    },

    deviceId: function () {
        return this._deviceId;
    },

    busPort: function () {
        return this._busPort;
    },

    setDeviceId: function (id) {
        this._deviceId = id;
        return this;
    },

    sendMsg: function (msg, done) {
        if (protocol.needRoute(msg)) {
            this.routeMsg(msg, done);
        } else {
            var methodIndex = msg.body.flag;
            var methodParams = msg.body.body;
            var method = this._methods[methodIndex];
            if (method == null) {
                done(new Error('unknown method ' + methodIndex));
            } else {
                var self = this;
                setImmediate(function () {
                    method.func(methodParams, function (err, reply) {
                        if (err != null) {
                            // TODO encode error
                            console.error(err);
                        } else {
                            self.reply(new protocol.Encoder()
                                .messageId(msg.head.msgId)
                                .encodeBody(0, reply)
                                .buildMsg()
                            );
                        }
                    });
                });
                done();
            }
        }
        return this;
    },

    routeMsg: function (msg, done) {
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

    reply: function (msg) {
        if (this._busPort) {
            this._busPort.sendMsg(msg, function () { });
        }
        return this;
    }
});

module.exports = Device;
