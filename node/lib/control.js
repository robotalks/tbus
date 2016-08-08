var Class = require('js-class'),
    protocol = require('./protocol.js');

var Controller = Class({
    constructor: function (master, addrs) {
        this._master = master;
        if (Array.isArray(addrs) && addrs.length > 0) {
            this._addrs = addrs;
        }
    },

    master: function () {
        return this._master;
    },

    address: function () {
        return this._addrs;
    },

    setAddress: function (addrs) {
        if (Array.isArray(addrs) && addrs.length > 0) {
            this._addrs = addrs;
        } else {
            throw new Error('invalid parameter');
        }
        return this;
    },

    invoke: function (index, params, callback) {
        this._master.invoke(index, params, this._addrs, callback);
        return this;
    }
});

module.exports = Controller;
