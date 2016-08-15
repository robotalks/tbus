var Class = require('js-class'),
    protocol = require('./protocol.js'),
    IdBitmap = require('./idbits.js'),
    Bus = require('./bus.js');

var Master = Class({
    constructor: function (device, options) {
        this.options = options || {};
        this._device = device;

        this._msgMap = {};
        this._msgIdBits = new IdBitmap();

        device.attach(this, 0);
    },

    device: function () {
        return this._device;
    },

    invoke: function (index, params, addrs, callback) {
        if (typeof(addrs) == 'function') {
            callback = addrs;
            addrs = null;
        }

        var msgId = this._mapMsgId(callback);
        var encoder = new protocol.Encoder();
        if (Array.isArray(addrs) && addrs.length > 0) {
            encoder.route(addrs);
        }
        encoder
            .messageId(msgId)
            .encodeBody(index, params);
        this._device.sendMsg(encoder.buildMsg(), function (err) {
            if (err != null) {
                this._unmapMsgId(msgId);
                callback(err);
            }
        }.bind(this));
        return this;
    },

    // implement BusPort
    sendMsg: function (msg, done) {
        var callback = this._msgMap[msg.head.msgId]
        if (callback != null) {
            this._unmapMsgId(msg.head.msgId);
            var err = protocol.decodeError(msg);
            if (err != null) {
                callback(err);
            } else {
                callback(null, msg.body.data);
            }
        }
    },

    _mapMsgId: function (callback) {
        var id = this._msgIdBits.alloc();
        this._msgMap[id] = callback;
        return id;
    },

    _unmapMsgId: function (id) {
        delete this._msgMap[id];
        this._msgIdBits.release(id);
    }
});

module.exports = Master;
