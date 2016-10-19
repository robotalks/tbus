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
        this._info.address = addr;
        if (this._initialAttach) {
            this.dispatchMsg(new protocol.Encoder()
                .encodeProto(0, new DeviceInfo().fromDevice(this))
                .buildMsg(), this._handleErr.bind(this));
            delete this._initialAttach;
        }
        return this;
    },

    deviceInfo: function () {
        return this._info;
    },

    address: function () {
        return this._info.address;
    },

    classId: function () {
        return this._info.classId;
    },

    deviceId: function () {
        return this._info.deviceId;
    },

    busPort: function () {
        return this._busPort;
    },

    setDeviceId: function (id) {
        this._deviceInfo.setDeviceId(id);
        return this;
    },

    dispatchMsg: function (msg, done) {
        if (this._conn != null) {
            this._conn.write(Buffer.concat([msg.head.raw, msg.body.raw]), null, done);
        } else {
            done(new Error('not connected'));
        }
        return this;
    },

    _onData: function (data) {
        this._decodeStream.write(data, null, this._handleErr.bind(this));
    },

    _onEnd: function () {
        this._conn.removeAllListeners();
        this._conn.destroy();
        delete this._conn;
    },

    _onError: function (err) {
        if (!this._init) {
            this._handleErr(err);
        }
    },

    _onMsg: function (msg) {
        if (this._init) {
            try {
                this._info = DeviceInfo
                    .deserializeBinary(new Uint8Array(msg.body.data))
                    .toObject();
            } catch (err) {
                this._handleErr(err);
                return;
            }
            this._initialAttach = true;
            delete this._init;
            this._host._newDevice(this);
        } else if (this._busPort) {
            this._busPort.dispatchMsg(msg, this._handleErr.bind(this));
        }
    },

    _handleErr: function (err) {
        if (err != null) {
            this.emit('error', err);
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
