var Class = require('js-class'),
    Device = require('./device.js'),
    protocol = require('./protocol.js');

var BusDevice = Class(Device, {
    constructor: function (logic, options) {
        this.options = options || {};
        Device.prototype.constructor.call(this,
            0x0001, this.options.id || 0,
            new protocol.RouteDecoder(this.options),
            logic);
        var dev = this;
        this.defineMethod(1, 'Enumerate', function (params, done) {
                dev['m:enumerate'](params, done);
            });
    },

    'm:enumerate': function (params, done) {
        this.logic.enumerate(function (err, result) {
            if (err == null) {
                result = result.serializeBinary();
            }
            done(err, result);
        });
    }
});

module.exports = BusDevice;
