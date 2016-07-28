var Class = require('js-class'),
    protocol = require('./protocol.js');

var Controller = Class({
    constructor: function (options) {
        this.options = options || {};
        this._decodeStream = new protocol.DecodeStream(new protocol.MsgDecoder(this.options));
        this._decodeStream.on('message', this._onMessage.bind(this));
        this._msgMap = {};

        if (this.options.stream) {
            this.setStream(this.options.stream);
        }
    },

    connect: function (stream, addrs) {
        this.setStream(stream);
        if (addrs !== undefined) {
            this.setAddress(addrs);
        }
        return this;
    },

    setStream: function (stream) {
        if (this._stream) {
            this._stream.unpipe(this._decodeStream);
        }
        this._decodeStream.decoder().reset();
        this._flushMsgMap();
        this._stream = stream;
        if (this._stream) {
            this._stream.pipe(this._decodeStream);
        }
        return this;
    },

    setAddress: function (addrs) {
        this._addrs = addrs;
        return this;
    },

    invoke: function (index, params, callback) {
        if (this._stream) {
            var msgId = this._genMsgId();
            var encoder = new protocol.Encoder();
            if (this._addrs && this._addrs.length > 0) {
                encoder.route(this._addrs);
            }
            if (typeof(params.serializeBinary) == 'function') {
                // protobuf encoding
                params = params.serializeBinary();
            }
            encoder
                .messageId(msgId)
                .encodeBody(index, params);
            this._msgMap[msgId] = callback;
            this._stream.write(encoder.toBuffer(), null, function (err) {
                if (err != null) {
                    delete this._msgMap[msgId];
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
            delete this._msgMap[msg.messageId];
            var err = protocol.decodeError(msg);
            if (err != null) {
                callback(err);
            } else {
                callback(null, msg.body, msg.bodyFlags);
            }
        }
    },

    _genMsgId: function () {
        // TODO
        if (typeof(this.options.idgen) == 'function') {
            return this.options.idgen();
        }
        return 0;
    },

    _flushMsgMap: function () {
        var err = new Error('disconnected');
        for (var id in this._msgMap) {
            var callback = this._msgMap[id];
            setImmediate(function () {
                callback(err);
            });
        }
        this._msgMap = {};
    }
});

module.exports = Controller;
