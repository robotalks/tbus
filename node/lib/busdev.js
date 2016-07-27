var Class = require('js-class'),
    Device = require('./device.js'),
    protocol = require('./protocol.js');

var BusDevice = Class(Device, {
    constructor: function (controller) {
        Device.prototype.constructor.call(this,
            0x0001, 0, new protocol.RouteDecoder(), controller);
        var dev = this;
        this.defineMethod(1, 'Enumerate', function (params, done) {
                dev['m:enumerate'](params, done);
            });
    },

    'm:enumerate': function (params, done) {
        this.controller.enumerate(function (err, result) {
            if (err == null) {
                result = result.serializeBinary();
            }
            done(err, result);
        });
    }
});

module.exports = BusDevice;
