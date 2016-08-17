var Class = require('js-class'),
    EventEmitter = require('events'),
    protocol = require('./protocol.js'),
    DeviceInfo = require('../gen/tbus/bus_pb.js').DeviceInfo;

var RemoteBusPort = Class(EventEmitter, {
    constructor: function (device, connector, options) {
        EventEmitter.call(this);
        this._device = device;
        this._connector = connector;
        this.options = options || {};
    },

    connect: function () {
        this._connector(function (err, connection) {
            if (err != null) {
                this.emit('error', err);
            } else {
                this._setConnection(connection);
            }
        }.bind(this));
        return this;
    },

    connected: function () {
        return this._connection != null;
    },

    sendMsg: function (msg, done) {
        if (this._connection != null) {
            this._connection.write(Buffer.concat([msg.head.raw, msg.body.raw]), null, done);
        } else {
            done(new Error('not connected'));
        }
        return this;
    },

    _setConnection: function (conn) {
        if (this._connection) {
            this._decodeStream.close();
            this._connection.removeAllListeners();
            this._connection.destroy();
            this.emit('disconnected');
            delete this._decodeStream;
        }
        this._connection = conn;
        if (this._connection) {
            this._connection
                .on('data', this._onData.bind(this))
                .on('end', this._onEnd.bind(this))
                .on('error', this._onError.bind(this));
            this._decodeStream = new protocol.DecodeStream();
            this._decodeStream
                .on(protocol.EVT_MSG, this._onMsg.bind(this));
            this.emit('connected');

            this._attaching = true;
            var info = new DeviceInfo().fromDevice(this._device);
            this.sendMsg(new protocol.Encoder()
                .messageId(0)
                .encodeProto(0, info)
                .buildMsg(), this._onError.bind(this));
        }
    },

    _onData: function (data) {
        this._decodeStream.write(data, null, function () { });
    },

    _onEnd: function () {
        this._setConnection(null);
    },

    _onError: function (err) {
        if (err != null) {
            this.emit('error', err);
        }
    },

    _onMsg: function (msg) {
        if (this._attaching) {
            var info;
            try {
                info = DeviceInfo.deserializeBinary(new Uint8Array(msg.body.data));
                this._device.attach(this, info.getAddress());
            } catch (err) {
                this._onError(err);
                return this;
            }
            delete this._attaching;
            this.emit('attached', info.getAddress());
        } else {
            this._device.sendMsg(msg, this._onError.bind(this));
        }
    }
});

module.exports = RemoteBusPort;
