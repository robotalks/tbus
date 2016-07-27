var Class = require('js-class'),
    protocol = require('./protocol.js');

var Device = Class({
    constructor: function (classId, deviceId, decoder, controller) {
        this._classId  = classId;
        this._deviceId = deviceId;
        this._decoder  = decoder;
        this.controller = controller;
        this._methods = [];
    },

    attach: function (bus, addr) {
        this._bus = bus;
        this._address = addr;
        this._decoder.reset();
        this._recvStream = new protocol.DecodeStream(this._decoder);
        this._recvStream
            .on(protocol.EVT_MSG, this._onMessage.bind(this))
            .on(protocol.EVT_ROUTE, this._onRoute.bind(this))
            .on(protocol.EVT_FWD, this._onForward.bind(this));
    },

    detach: function () {
        if (this._recvStream) {
            this._recvStream.close();
            delete this._recvStream;
        }
        delete this._bus;
        delete this._address;
    },

    classId: function () {
        return this._classId;
    },

    deviceId: function () {
        return this._deviceId;
    },

    receiver: function () {
        return this._recvStream;
    },

    address: function () {
        return this._address;
    },

    bus: function () {
        return this._bus;
    },

    _onMessage: function (msg) {
        var method = this._methods[msg.bodyFlags];
        if (method == null) {
            // TODO generate error
            console.error(new Error('unknown method ' + msg.bodyFlags))
        } else {
            method.func(params, function (err, reply) {
                if (err != null) {
                    // TODO
                    console.error(err);
                } else {
                    this.bus().writeToHost(new protocol.Encoder()
                        .messageId(msg.messageId)
                        .encodeBody(0, reply)
                        .toBuffer()
                    );
                }
            }.bind(this));
        }
    },

    _onRoute: function (info) {
        delete this._fwdStream;
        var routeFn = this.controller.route;
        if (typeof(routeFn) == 'function') {
            routeFn.call(this.controller, info, this._setFwdStream.bind(this));
        } else {
            this._setFwdStream(new Error('routing not supported'));
        }
    },

    _setFwdStream: function (err, fwdStream, buf) {
        if (err != null) {
            // TODO
            console.error(err);
        } else {
            this._fwdStream = fwdStream;
            if (buf && buf.length > 0) {
                this._forward(buf);
            }
        }
    },

    _onForward: function (info) {
        this._forward(info.buf);
    },

    _forward: function (buf) {
        if (this._fwdStream) {
            this._fwdStream.write(buf, null, function (err) {
                if (err != null) {
                    // TODO
                    console.error(err);
                }
            }.bind(this));
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
});

module.exports = Device;
