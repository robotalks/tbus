var LEDCtl = require('../../gen/led_device.js').LEDCtl;

LEDCtl.prototype.on = function (done) {
    return this.setPowerState({ on: true }, done);
};

LEDCtl.prototype.off = function (done) {
    return this.setPowerState({ on: false }, done);
};

LEDCtl.prototype.setBrightness = function (brightness, done) {
    return this.setPowerState({ on: true, brightness: brightness }, done);
};

module.exports = LEDCtl;
