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

    invoke: function (index, params, callback) {
        this._master.invoke(index, params, this._addrs, callback);
        return this;
    }
});

module.exports = Controller;
