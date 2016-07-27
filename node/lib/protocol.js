var Buffer = require('buffer').Buffer,
    stream = require('stream'),
    Class = require('js-class');

var PFX_ROUTING_MASK = 0xe0,
    PFX_ROUTING = 0xc0,
    PFX_ROUTING_ADDRNUM = 0x1f;

var F_REVISION  = 0xc0,
    F_ENCODING  = 0x30,
    F_MSGIDSIZE = 0x0c,
    F_LONGBODY  = 0x02;

var S_MSGIDSIZE = 2;

var REVISION = 0,
    ENCODING = 0x10,    // protobuf
    MSGID_1B = 0x00,
    MSGID_2B = 0x04,
    MSGID_3B = 0x08,
    MSGID_4B = 0x0c;

var EVT_HEAD  = 'head',
    EVT_MSG   = 'message',
    EVT_ROUTE = 'route',
    EVT_FWD   = 'forward';

var MsgFlags = Class({
    constructor: function (flags) {
        this.flags = flags;
    },

    supported: function () {
        return (this.flags & (F_REVISION | F_ENCODING)) == (REVISION | ENCODING);
    },

    msgIdSize: function () {
        return ((this.flags & F_MSGIDSIZE) >> S_MSGIDSIZE) + 1;
    },

    longBody: function () {
        return (this.flags & F_LONGBODY) != 0;
    }
});

var StreamDecoder = Class({
    constructor: function (initialState) {
        this._initialState = initialState;
        this._state = initialState;
    },

    decode: function (buf) {
        this._buf = buf;
        while (this._result != null && this._buf.length > 0) {
            this['_parse' + this._state]();
        }
        var result = this._result;
        delete(this._result);
        return result;
    },

    readUInt8: function () {
        var b = this._buf.readUInt8(0);
        this.skipUInt8();
        return b;
    },

    skipUInt8: function () {
        this._buf = this._buf.slice(1);
    },

    shiftUInt8: function (v) {
        var b = this.readUInt8();
        v = (v << 8) | (b & 0xff);
        this._recvLen ++;
        return v;
    },

    transit: function (state) {
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
    },

    done: function (err, event, result) {
        this.report(err, event, result);
        this.reset();
    },

    allRecv: function () {
        return this._recvLen >= this._expLen;
    }
});

var MsgDecoder = Class(StreamDecoder, {
    constructor: function () {
        StreamDecoder.prototype.constructor.call(this, 'Flags');
    },

    _enterFlags: function () {
        delete(this._flags);
        delete(this._msgId);
        delete(this._bodyBytes);
        delete(this._body);
    },

    _parseFlags: function () {
        this._flags = new MsgFlags(this.readUInt8());
        if (!this._flags.supported()) {
            this.done(new Error('unsupported protocol'));
        } else {
            this.transit('MsgId');
        }
    },

    _enterMsgId: function () {
        this._expLen = this._flags.msgIdSize();
        this._msgId = 0;
    },

    _parseMsgId: function () {
        this._msgId = this.shiftUInt8(this._msgId);
        if (this.allRecv()) {
            this.transit('BodySize');
        }
    },

    _enterBodySize: function () {
        this._expLen = this._flags.longBody() ? 2 : 1;
        this._bodyBytes = 0;
        delete(this._body);
    },

    _parseBodySize: function () {
        this._bodyBytes = this.shiftUInt8(this._bodyBytes);
        if (this.allRecv()) {
            this._bodyBytes <<= (this._flags.longBody() ? 2 : 1);
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
    constructor: function () {
        StreamDecoder.prototype.constructor.call(this, 'Prefix');
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
        this._msgDecoder = new MsgDecoder();
        this._msgBuf = this._buf;
    },

    _parseMsgHead: function () {
        var result = this._msgDecoder.decode(this._buf);
        if (result != null) {
            if (result.err != null) {
                this.done(result.err);
                return;
            }
            var h = result.result;
            this._msgBytes = 2 + h.flags.msgIdSize() + h.bodyBytes;
            if (h.flags.longBody()) {
                this._msgBytes ++;
            }
            var msg = {
                addrs: this._addrs,
                msgHead: result.result,
                partialBuf: this._msgBuf,
                partialBody: result.buf
            };
            this._buf = this._msgBuf;
            this.report(null, EVT_ROUTE, msg);
            this.transit('MsgFwd');
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
        this.report(null, EVT_FWD, { addrs: this._addrs, buf: this._buf.slice(0, size) });
        this._buf = this._buf.slice(size);
        this._recvLen += size;
        if (this.allRecv()) {
            this.reset();
        }
    },

    _enterMsg: function () {
        this._msgDecoder = new MsgDecoder();
    },

    _parseMsg: function () {
        this._result = this._msgDecoder.decode(this._buf);
        if (this._result != null && (this._result.err != null ||
            this._result.event == EVT_MSG ||
            this._result.result.bodyBytes == 0)) {
            this.reset();
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

    _write: function (chunk, encoding, callback) {
        var buf = chunk;
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

var Encoder = Class({
    constructor: function () {
        this._flags = REVISION | ENCODING;
    },

    messageId: function (msgId) {
        this._msgId = msgId;
        if (msgId & 0xff000000) {
            this._flags |= MSGID_4B;   // 4-byte message id
        } else if (msgId & 0x00ff0000) {
            this._flags |= MSGID_3B;   // 3-byte message id
        } else if (msgId & 0x0000ff00) {
            this._flags |= MSGID_2B;   // 2-byte message id
        }
        return this;
    },

    body: function (body) {
        this._body = body;
        this._paddings = 0;
        var size = body.length;
        if (size & 1) {
            size ++;
            this._paddings ++;
        }
        size >>= 1;
        if (size > 255) {
            if (size & 1) {
                size ++;
                this._paddings += 2;
            }
            size >>= 1;
            this._bodySize = size & 0xffff;
            this._flags |= F_LONGBODY;
        } else {
            this._bodySize = size & 0xff;
            this._flags &= ~F_LONGBODY;
        }
        return this;
    },

    encodeBody: function (flags, body) {
        if (!Buffer.isBuffer(body)) {
            body = new Buffer(body);
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
        var flags = new MsgFlags(this._flags);
        var size = 2 + flags.msgIdSize() + this._body.length + this._paddings;
        if (flags.longBody()) {
            size ++;
        }
        if (this._addrs) {
            size += this._addrs.length + 1;
        }
        var buf = new Buffer(size), off = 0;
        if (this._addrs) {
            off = this._writeRoutePrefix(buf, off);
        }
        buf.writeUInt8(this._flags, off++);
        switch (flags.msgIdSize()) {
        case 4:
            buf.writeUInt32LE(this._msgId, off);
            off += 4;
            break;
        case 3:
            buf.writeUInt16LE(this._msgId & 0xffff, off);
            buf.writeUInt8(this._msgId >> 16, off+2);
            off += 3;
            break;
        case 2:
            buf.writeUInt16LE(this._msgId, off);
            off += 2;
            break;
        default:
            buf.writeUInt8(this._msgId, off++);
        }
        if (flags.longBody()) {
            buf.writeUInt16LE(this._bodySize, off);
            off += 2;
        } else {
            buf.writeUInt8(this._bodySize, off++);
        }
        this._body.copy(buf, off);
        off += this._body.length;
        for (var i = 0; i < this._paddings; i ++) {
            buf.writeUInt8(0, off++);
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

module.exports = {
    EVT_HEAD:  EVT_HEAD,
    EVT_MSG:   EVT_MSG,
    EVT_ROUTE: EVT_ROUTE,
    EVT_FWD:   EVT_FWD,

    MsgDecoder:   MsgDecoder,
    RouteDecoder: RouteDecoder,
    DecodeStream: DecodeStream,
    Encoder:      Encoder,
};
