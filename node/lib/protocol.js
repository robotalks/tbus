var Buffer = require('buffer').Buffer,
    stream = require('stream'),
    Class = require('js-class'),
    Empty = require('google-protobuf/google/protobuf/empty_pb.js').Empty,
    ErrorProto = require('../gen/tbus/error_pb.js').Error;

var PFX_ROUTING_MASK = 0xe0,
    PFX_ROUTING = 0xc0,
    PFX_ROUTING_ADDRNUM = 0x1f;

var FORMAT_MASK = 0xf0,
    FORMAT = 0x10;      // rev 1, protobuf encoded

var BF_ERROR = 0x80;    // body contains error

var EVT_ROUTE = 'route',
    EVT_HEAD  = 'head',
    EVT_MSG   = 'message';

var _logger = null;

function setLogger(logger) {
    _logger = logger;
}

var StreamDecoder = Class({
    constructor: function (initialState, options) {
        this._initialState = initialState;
        this._state = initialState;
        this._logger = _logger;
        this.options = options || {};
    },

    log: function (logger) {
        this._logger = logger;
        return this;
    },

    decode: function (buf) {
        this._buf = buf;
        while (this._result == null &&
            this._buf != null && this._buf.length > 0) {
            this._log('decode', function () {
                return {
                    state: this._state,
                    data: Array.from(this._buf, function (v) { return v.toString(16); }).join()
                }
            }.bind(this));
            this['_parse' + this._state]();
        }
        var result = this._result;
        delete this._result;
        return result;
    },

    readUInt8: function () {
        var b = this._buf.readUInt8(0);
        this.skipUInt8();
        return b;
    },

    decode7Bit: function () {
        var b = this.readUInt8();
        var v = (b & 0x7f) << (7 * (this._recvLen - 1));
        if (b & 0x80) {
            this._expLen ++;
            b &= 0x7f;
        }
        return v;
    },

    skipUInt8: function () {
        this._buf = this._buf.slice(1);
        this._recvLen ++;
    },

    transit: function (state) {
        this._log('transit', { from: this._state, to: state });
        this._state = state;
        this._expLen = 0;
        this._recvLen = 0;
        var enterFn = this['_enter' + state];
        if (enterFn != null) {
            enterFn.call(this);
        }
    },

    reset: function () {
        this.transit(this._initialState);
    },

    report: function (err, event, result) {
        this.setResult(err, event, result);
    },

    setResult: function (err, event, result) {
        this._result = {
            event: event,
            buf: this._buf.length > 0 ? this._buf : null
        };
        if (err != null) {
            this._result.err = err;
        } else {
            this._result.result = result;
        }
        this._log('report', this._result);
    },

    done: function (err, event, result) {
        this.report(err, event, result);
        this.reset();
    },

    allRecv: function () {
        return this._recvLen >= this._expLen;
    },

    _log: function (event, data) {
        if (this._logger) {
            if (typeof(data) == 'function') {
                data = data();
            }
            if (this.options.id != null) {
                event = this.options.id + ":" + event;
            }
            this._logger(event, data);
        }
    }
});

var Decoder = Class(StreamDecoder, {
    constructor: function (options) {
        StreamDecoder.prototype.constructor.call(this, 'Prefix', options);
        this._prefix = [];
    },

    _enterPrefix: function () {
        this._prefix = [];
    },

    _parsePrefix: function () {
        var b = this._buf.readUInt8(0);
        if ((b & PFX_ROUTING_MASK) == PFX_ROUTING) {
            this._addrNum = (b & PFX_ROUTING_ADDRNUM) + 1;
            this.skipUInt8();
            this._prefix.push(b);
            this.transit('Addrs');
        } else {
            this.transit('Flags');
        }
    },

    _enterAddrs: function () {
        this._expLen = this._addrNum;
    },

    _parseAddrs: function () {
        this._prefix.push(this.readUInt8());
        if (this.allRecv()) {
            var msg = {head: {prefix: this._prefix[0]}};
            msg.head.raw = new Buffer(this._prefix);
            msg.head.addrs = this._prefix.slice(1);
            msg.head.rawPrefix = msg.head.raw.slice(0);
            this.report(null, EVT_ROUTE, msg);
            this.transit('Flags');
        }
    },

    _enterFlags: function () {
        delete this._flag;
        delete this._msgId;
        delete this._bodyBytes;
        delete this._body;
        this._headBytes = 0;
        this._head = this._prefix.slice(0);
    },

    _parseFlags: function () {
        this._flag = this.readUInt8();
        this._head.push(this._flag);
        this._headBytes ++;
        if ((this._flag & FORMAT_MASK) != FORMAT) {
            this.done(new Error('unsupported protocol'));
        } else {
            this.transit('MsgId');
        }
    },

    _enterMsgId: function () {
        this._expLen = 1;
        this._msgId = 0;
    },

    _parseMsgId: function () {
        this._head.push(this._buf[0]);
        this._msgId |= this.decode7Bit();
        this._headBytes ++;
        if (this.allRecv()) {
            this.transit('BodySize');
        }
    },

    _enterBodySize: function () {
        this._expLen = 1;
        this._bodyBytes = 0;
        delete this._body;
    },

    _parseBodySize: function () {
        this._head.push(this._buf[0]);
        this._bodyBytes |= this.decode7Bit();
        this._headBytes ++;
        if (this.allRecv()) {
            this._report(EVT_HEAD);
            if (this._bodyBytes > 0) {
                this.transit('Body');
            } else {
                this.reset();
            }
        }
    },

    _enterBody: function () {
        this._expLen = this._bodyBytes;
        this._body = new Buffer(0);
    },

    _parseBody: function () {
        var size = this._buf.length;
        if (this._recvLen + size > this._expLen) {
            size = this._expLen - this._recvLen;
            this._body = Buffer.concat([this._body, this._buf.slice(0, size)]);
        } else {
            this._body = Buffer.concat([this._body, this._buf]);
        }
        this._buf = this._buf.slice(size);
        this._recvLen += size;
        if (this.allRecv()) {
            this._report(EVT_MSG);
            this.reset();
        }
    },

    _report: function (event) {
        var msg = {
            head: {
                flag: this._flag,
                msgId: this._msgId,
                bodyBytes: this._bodyBytes,
                raw: new Buffer(this._head)
            },
            body: {}
        };
        if (this._prefix.length > 0) {
            msg.head.prefix = this._prefix[0];
            msg.head.addrs = this._prefix.slice(1);
            msg.head.rawPrefix = msg.head.raw.slice(0, this._prefix.length);
            msg.head.rawHead = msg.head.raw.slice(this._prefix.length);
        } else {
            msg.head.rawHead = msg.head.raw.slice(0);
        }
        if (this._body != null && this._body.length > 0) {
            msg.body.raw = this._body;
            msg.body.flag = this._body.readUInt8(0);
            msg.body.data = this._body.slice(1);
        }
        this.report(null, event, msg);
    },
});

var DecodeStream = Class(stream.Writable, {
    constructor: function (decoder) {
        stream.Writable.call(this);
        this._decoder = decoder || new Decoder();
    },

    decoder: function () {
        return this._decoder;
    },

    close: function () {
        this.removeAllListeners();
    },

    _write: function (buf, encoding, callback) {
        while (buf != null && buf.length > 0) {
            var result = this._decoder.decode(buf);
            if (result == null) {
                break;
            }
            buf = result.buf;
            if (result.err != null) {
                this._handleErr(result.err)
            } else {
                this._handleEvent(result.event, result.result);
            }
        }
        setImmediate(callback);
    },

    _handleErr: function (err) {
        this.emit('error', err);
    },

    _handleEvent: function (event, msg) {
        setImmediate(function () {
            this.emit(event, msg);
        }.bind(this));
    }
});

function encode7Bit(bytes, value) {
    do {
        var v = value & 0x7f;
        value >>= 7;
        if (value > 0) {
            v |= 0x80;
        }
        bytes.push(v);
    } while (value > 0);
}

var Encoder = Class({
    constructor: function () {
        this._flag = FORMAT;
    },

    messageId: function (msgId) {
        this._msgId = msgId;
        return this;
    },

    body: function (body) {
        this._body = body;
        return this;
    },

    encodeError: function (err) {
        var reply = new ErrorProto();
        reply.setMessage(err.message);
        return this.encodeBody(BF_ERROR, reply.serializeBinary());
    },

    encodeProto: function (flag, pbMsg) {
        if (pbMsg == null) {
            pbMsg = new Empty();
        }
        return this.encodeBody(flag, pbMsg.serializeBinary());
    },

    encodeBody: function (flag, body) {
        if (!Buffer.isBuffer(body)) {
            body = new Buffer(body.buffer);
        }
        var buf = new Buffer(1 + body.length);
        buf.writeUInt8(flag, 0);
        body.copy(buf, 1);
        return this.body(buf);
    },

    route: function (addrs) {
        if (Array.isArray(addrs) && addrs.length > 0) {
            this._addrs = addrs;
        } else {
            delete(this._addrs);
        }
        return this;
    },

    buildMsg: function () {
        var msg = {
            head: {
                flag: this._flag,
                msgId: this._msgId
            },
            body: {
                raw: this._body
            }
        };
        var headBytes = 0;
        if (this._addrs && this._addrs.length > 0) {
            msg.head.addrs = this._addrs;
            headBytes += this._addrs.length + 1;
            var addrNum = (this._addrs.length - 1) & PFX_ROUTING_ADDRNUM;
            msg.head.prefix = PFX_ROUTING | addrNum;
        }
        var head = [this._flag];
        msg.head.bodyBytes = this._body ? this._body.length : 0;
        encode7Bit(head, this._msgId);
        encode7Bit(head, msg.head.bodyBytes);
        headBytes += head.length;

        msg.head.raw = new Buffer(headBytes);
        var off = 0;
        if (msg.head.addrs) {
            msg.head.raw.writeUInt8(msg.head.prefix, off++);
            for (var i = 0; i < msg.head.addrs.length; i ++) {
                msg.head.raw.writeUInt8(msg.head.addrs[i], off++);
            }
        }
        for (var i = 0; i < head.length; i ++) {
            msg.head.raw.writeUInt8(head[i], off++);
        }
        if (msg.head.addrs) {
            msg.head.rawPrefix = msg.head.raw.slice(0, msg.head.addrs.length + 1);
            msg.head.rawHead = msg.head.raw.slice(msg.head.addrs.length + 1);
        } else {
            msg.head.rawHead = msg.head.raw.slice(0);
        }

        if (msg.body.raw.length > 0) {
            msg.body.flag = msg.body.raw.readUInt8(0);
            msg.body.data = msg.body.raw.slice(1);
        }

        return msg;
    },

    toBuffer: function () {
        var msg = this.buildMsg();
        return Buffer.concat([msg.head.raw, msg.body.raw]);
    }
});

function decodeError(msg) {
    if (msg.body.flag & 0x80) {
        try {
            var err = ErrorProto.deserializeBinary(new Uint8Array(msg.body.data));
            return new Error(err.getMessage());
        } catch (err) {
            return err;
        }
    }
    return null;
}

function needRoute(msg) {
    return Array.isArray(msg.head.addrs) && msg.head.addrs.length > 0;
}

module.exports = {
    EVT_ROUTE:  EVT_ROUTE,
    EVT_HEAD:   EVT_HEAD,
    EVT_MSG:    EVT_MSG,

    Encoder: Encoder,
    Decoder: Decoder,
    DecodeStream: DecodeStream,

    decodeError: decodeError,
    needRoute: needRoute,
    setLogger: setLogger
};
