var Class = require('js-class'),
    protocol = require('./protocol.js'),
    IdBitmap = require('./idbits.js'),
    Bus = require('./bus.js');

var Master = Class({
    constructor: function (options) {
        this.options = options || {};
        this._decodeStream = new protocol.DecodeStream(new protocol.MsgDecoder(this.options));
        this._decodeStream.on(protocol.EVT_MSG, this._onMessage.bind(this));
        this._msgMap = {};
        this._msgIdBits = new IdBitmap();

        if (this.options.stream) {
            this.connect(this.options.stream);
        } else if (this.options.bus) {
            this.connect(this.options.bus);
        }
    },

    connect: function (streamOrBus) {
        this.disconnect();

        if (streamOrBus instanceof Bus) {
            this._stream = streamOrBus.hostStream();
        } else {
            this._stream = streamOrBus;
        }
        if (this._stream) {
            this._decodeStream.decoder().reset();
            this._stream.pipe(this._decodeStream);
        }
        return this;
    },

    disconnect: function () {
        if (this._stream) {
            this._stream.unpipe(this._decodeStream);
        }
        this._flushMsgMap();
        delete this._stream;
    },

    connected: function () {
        return this._stream != null;
    },

    stream: function () {
        return this._stream;
    },

    invoke: function (index, params, addrs, callback) {
        if (typeof(addrs) == 'function') {
            callback = addrs;
            addrs = null;
        }

        if (this._stream) {
            var msgId = this._mapMsgId(callback);
            var encoder = new protocol.Encoder();
            if (Array.isArray(addrs) && addrs.length > 0) {
                encoder.route(addrs);
            }
            encoder
                .messageId(msgId)
                .encodeBody(index, params);
            this._stream.write(encoder.toBuffer(), null, function (err) {
                if (err != null) {
                    this._unmapMsgId(msgId);
                    callback(err);
                }
            }.bind(this));
        } else {
            callback(new Error('not connected'));
        }
        return this;
    },

    _onMessage: function (msg) {
        var callback = this._msgMap[msg.messageId]
        if (callback != null) {
            this._unmapMsgId(msg.messageId);
            var err = protocol.decodeError(msg);
            if (err != null) {
                callback(err);
            } else {
                callback(null, msg.body, msg.bodyFlags);
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
    },

    _flushMsgMap: function () {
        var err = new Error('disconnected');
        for (var id in this._msgMap) {
            var callback = this._msgMap[id];
            this._unmapMsgId(id);
            setImmediate(function () {
                callback(err);
            });
        }
    }
});

module.exports = Master;
