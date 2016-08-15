var Class = require('js-class'),
    EventEmitter = require('events'),
    protocol = require('./protocol.js'),
    DeviceInfo = require('../gen/tbus/bus_pb.js').DeviceInfo;

var RemoteDevice = Class(EventEmitter, {
    constructor: function (host, connection) {
        EventEmitter.call(this);
        this._host = host;
        this._conn = connection;
        this._conn
            .on('data', this._onData.bind(this))
            .on('end', this._onEnd.bind(this))
            .on('error', this._onError.bind(this));
        this._decodeStream = new protocol.DecodeStream();
        this._decodeStream
            .on(protocol.EVT_MSG, this._onMsg.bind(this));

        this._init = true;
    },

    attach: function (busPort, addr) {
        this._busPort = busPort;
        this._deviceInfo.setAddress(addr);
        if (this._initialAttach) {
            this.sendMsg(new protocol.Encoder()
                .messageId(0)
                .encodeBody(0, this._deviceInfo.serializeBinary())
                .buildMsg(), function (err) {
                // TODO error
            });
            delete this._initialAttach;
        }
        return this;
    },

    address: function () {
        return this._deviceInfo.getAddress();
    },

    classId: function () {
        return this._deviceInfo.getClassId();
    },

    deviceId: function () {
        return this._deviceInfo.getDeviceId();
    },

    busPort: function () {
        return this._busPort;
    },

    setDeviceId: function (id) {
        this._deviceInfo.setDeviceId(id);
        return this;
    },

    sendMsg: function (msg, done) {
        if (this._conn != null) {
            this._conn.write(Buffer.concat([msg.head.raw, msg.body.raw]), null, done);
        } else {
            // TODO
            done();
        }
        return this;
    },

    _onData: function (data) {
        this._decodeStream.write(data, null, function () { });
    },

    _onEnd: function () {
        this._conn.removeAllListeners();
        this._conn.destroy();
        delete this._conn;
    },

    _onError: function (err) {
        if (!this._init) {
            this.emit('error', err);
        }
    },

    _onMsg: function (msg) {
        if (this._init) {
            try {
                this._deviceInfo = DeviceInfo.deserializeBinary(new Uint8Array(msg.body.data))
            } catch (err) {
                // TODO
                console.dir(err)
            }
            this._initialAttach = true;
            delete this._init;
            this._host.newDevice(this);
        } else if (this._busPort) {
            this._busPort.sendMsg(msg, function () { });
        }
    }
});

var RemoteDeviceHost = Class(EventEmitter, {
    constructor: function (server, options) {
        EventEmitter.call(this);
        this._server = server;
        this.options = options || {};
        this._server
            .on('connection', this._onConnection.bind(this));
    },

    _onConnection: function (conn) {
        new RemoteDevice(this, conn);
    },

    _newDevice: function (device) {
        this.emit('device', device);
    }
});

module.exports = RemoteDeviceHost;