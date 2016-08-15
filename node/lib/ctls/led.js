var LEDCtl = require('../../gen/tbus/led_tbusdev.js').LEDCtl;

LEDCtl.prototype.on = function (done) {
    return this.setOn(true, done);
};

LEDCtl.prototype.off = function (done) {
    return this.setOn(false, done);
};

LEDCtl.prototype.setOn = function (on, done) {
    return this.setPowerState({on: on}, done);
};

module.exports = LEDCtl;
