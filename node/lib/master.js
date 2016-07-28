var Class = require('js-class'),
    protocol = require('./protocol.js'),
    Bus = require('./bus.js');

var BITS_ALL = 0xffffffff,
    BITS_N = 32;

var IdBitmap = Class({
    constructor: function () {
        this._bits = [1];
        this._allocated = 1;
        this._scanStart = 0;
    },

    alloc: function() {
        var index;
        if (this._allocated < this._bits.length * BITS_N) {
            var iStart = Math.floor(this._scanStart / BITS_N);
            var jStart = this._scanStart % BITS_N;
            index = this._lookup(iStart, jStart, this._bits.length);
            if (index == null && this._scanStart != 0) {
                index = this._lookup(0, 0, this.iStart + 1);
            }
        }
        if (index == null) {
            var i = this._bits.length;
            this._bits[i] = 1;
            index = index * BITS_N;
        } else {
            this._scanStart = index + 1;
        }
        this._allocated ++;
        return index;
    },

    release: function (id) {
        var i = Math.floor(id / BITS_N);
        var j = id % BITS_N;
        var s = 1 << j;
        var bits = this._bits[i];
        if (bits != null && (bits & s) != 0) {
            this._allocated --;
            bits &= ~s;
            if (bits == 0) {
                delete this._bits[i];
            } else {
                this._bits[i] = bits;
            }
        }
    },

    _lookup: function (iStart, jStart, iMax) {
        if (iMax > this._bits.length) {
            iMax = this._bits.length;
        }
        for (var i = iStart; i < iMax; i ++) {
            var bits = this._bits[i];
            if (bits == null) {
                this._bits[i] = 1;
                return i * BITS_N;
            }
            if (bits != BITS_ALL) {
                for (var j = jStart; j < BITS_N; j ++) {
                    var s = 1 << j;
                    if ((bits & s) == 0) {
                        this._bits[i] |= s;
                        return i * BITS_N + j;
                    }
                }
            }
            jStart = 0;
        }
        return null;
    }
});

var Master = Class({
    constructor: function (options) {
        this.options = options || {};
        this._decodeStream = new protocol.DecodeStream(new protocol.MsgDecoder(this.options));
        this._decodeStream.on('message', this._onMessage.bind(this));
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
