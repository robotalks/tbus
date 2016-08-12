var LEDCtl = require('../../gen/tbus/led_tbusdev.js').LEDCtl;

LEDCtl.prototype.on = function (done) {
    return this.setPowerState({ on: true }, done);
};

LEDCtl.prototype.off = function (done) {
    return this.setPowerState({ on: false }, done);
};

module.exports = LEDCtl;
