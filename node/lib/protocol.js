var Buffer = require('buffer').Buffer,
    stream = require('stream'),
    Class = require('js-class');

var PFX_ROUTING_MASK = 0xe0,
    PFX_ROUTING = 0xc0,
    PFX_ROUTING_ADDRNUM = 0x1f;

var FORMAT_MASK = 0xf0,
    FORMAT = 0x10;      // rev 1, protobuf encoded

var BF_ERROR = 0x80;    // body contains error

var EVT_HEAD  = 'head',
    EVT_MSG   = 'message',
    EVT_ROUTE = 'route',
    EVT_FWD   = 'forward';

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

var MsgDecoder = Class(StreamDecoder, {
    constructor: function (options) {
        StreamDecoder.prototype.constructor.call(this, 'Flags', options);
        this._headBytes = 0;
    },

    _enterFlags: function () {
        delete this._flags;
        delete this._msgId;
        delete this._bodyBytes;
        delete this._body;
        this._headBytes = 0;
    },

    _parseFlags: function () {
        this._flags = this.readUInt8();
        this._headBytes ++;
        if ((this._flags & FORMAT_MASK) != FORMAT) {
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
            flags: this._flags,
            messageId: this._msgId,
            headBytes: this._headBytes,
            bodyBytes: this._bodyBytes
        };
        if (this._body != null && this._body.length > 0) {
            msg.bodyFlags = this._body.readUInt8(0);
            msg.body = this._body.slice(1);
        }
        this.report(null, event, msg);
    },
});

var RouteDecoder = Class(StreamDecoder, {
    constructor: function (options) {
        StreamDecoder.prototype.constructor.call(this, 'Prefix', options);
    },

    _parsePrefix: function () {
        var b = this._buf.readUInt8(0);
        if ((b & PFX_ROUTING_MASK) == PFX_ROUTING) {
            this._addrNum = (b & PFX_ROUTING_ADDRNUM) + 1;
            this.skipUInt8();
            this.transit('Addrs');
        } else {
            this.transit('Msg');
        }
    },

    _enterAddrs: function () {
        this._expLen = this._addrNum;
        this._addrs = [];
    },

    _parseAddrs: function () {
        this._addrs.push(this.readUInt8());
        if (this.allRecv()) {
            this.transit('MsgHead');
        }
    },

    _enterMsgHead: function () {
        this._msgDecoder = new MsgDecoder(this.options);
        this._msgBufs = [];
    },

    _parseMsgHead: function () {
        this._msgBufs.push(this._buf);
        var result = this._msgDecoder.decode(this._buf);
        if (result != null) {
            if (result.err != null) {
                this.done(result.err);
                return;
            }
            var head = result.result;
            this._msgBytes = head.headBytes + head.bodyBytes;
            this._buf = Buffer.concat(this._msgBufs);
            var msg = {
                addrs: this._addrs,
                msgHead: head,
                partialBuf: this._buf,
                partialBody: result.buf
            };
            this.report(null, EVT_ROUTE, msg);
            this.transit('MsgFwd');
        } else {
            this._buf = this._msgDecoder._buf;
        }
    },

    _enterMsgFwd: function () {
        this._expLen = this._msgBytes;
        this._recvLen = 0;
    },

    _parseMsgFwd: function () {
        var size = this._buf.length;
        if (this._recvLen + size > this._expLen) {
            size = this._expLen - this._recvLen;
        }
        var fwdBuf = this._buf.slice(0, size);
        this._buf = this._buf.slice(size);
        this._recvLen += size;
        this.report(null, EVT_FWD, { addrs: this._addrs, buf: fwdBuf });
        if (this.allRecv()) {
            this.reset();
        }
    },

    _enterMsg: function () {
        this._msgDecoder = new MsgDecoder(this.options);
    },

    _parseMsg: function () {
        this._result = this._msgDecoder.decode(this._buf);
        if (this._result != null) {
            if (this._result.err != null ||
                this._result.event == EVT_MSG ||
                this._result.result.bodyBytes == 0) {
                this.reset();
            }
        } else {
            this._buf = this._msgDecoder._buf;
        }
    },
});

var DecodeStream = Class(stream.Writable, {
    constructor: function (decoder) {
        stream.Writable.call(this);
        this._decoder = decoder;
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
        // TODO
        console.error(err);
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
        this._flags = FORMAT;
    },

    messageId: function (msgId) {
        this._msgId = msgId;
        return this;
    },

    body: function (body) {
        this._body = body;
        return this;
    },

    encodeBody: function (flags, body) {
        if (!Buffer.isBuffer(body)) {
            body = new Buffer(body.buffer);
        }
        var buf = new Buffer(1 + body.length);
        buf.writeUInt8(flags, 0);
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

    toRoutePrefix: function () {
        if (this._addrs && this._addrs.length > 0) {
            var buf = new Buffer(this._addrs.length + 1);
            this._writeRoutePrefix(buf, 0);
            return buf;
        }
        return null;
    },

    toBuffer: function () {
        var routePfx = this.toRoutePrefix();
        var off = routePfx ? routePfx.length : 0;
        var head = [this._flags];
        var bodyBytes = this._body ? this._body.length : 0;
        encode7Bit(head, this._msgId);
        encode7Bit(head, bodyBytes);
        var buf = new Buffer(off + head.length + bodyBytes);
        if (routePfx) {
            routePfx.copy(buf, 0);
        }
        head.forEach(function (v, index) { buf.writeUInt8(v, index + off); });
        if (bodyBytes > 0) {
            this._body.copy(buf, off + head.length);
        }
        return buf;
    },

    _writeRoutePrefix: function (buf, off) {
        var addrNum = (this._addrs.length - 1) & PFX_ROUTING_ADDRNUM;
        buf.writeUInt8(PFX_ROUTING | addrNum, off++);
        for (var i = 0; i < addrNum + 1; i ++) {
            buf.writeUInt8(this._addrs[i], off++);
        }
        return off;
    }
});

function encodeError(err) {
    // TODO
    return {
        bodyFlags: BF_ERROR,
        body: new Buffer(0),
    };
}

function decodeError(msg) {
    if (msg.bodyFlags & 0x80) {
        // TODO

    }
    return null;
}

module.exports = {
    EVT_HEAD:  EVT_HEAD,
    EVT_MSG:   EVT_MSG,
    EVT_ROUTE: EVT_ROUTE,
    EVT_FWD:   EVT_FWD,

    MsgDecoder:   MsgDecoder,
    RouteDecoder: RouteDecoder,
    DecodeStream: DecodeStream,
    Encoder:      Encoder,

    encodeError: encodeError,
    decodeError: decodeError,

    setLogger: setLogger
};
