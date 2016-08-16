var LEDCtl = require('../../gen/tbus/led_tbusdev.js').LEDCtl,
    LEDPowerState = require('../../gen/tbus/led_pb.js').LEDPowerState;

LEDCtl.prototype.on = function (done) {
    return this.setOn(true, done);
};

LEDCtl.prototype.off = function (done) {
    return this.setOn(false, done);
};

LEDCtl.prototype.setOn = function (on, done) {
    var param = new LEDPowerState();
    param.setOn(on);
    return this.setPowerState(param, done);
};

module.exports = LEDCtl;
