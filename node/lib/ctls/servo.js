var ServoCtl = require('../../gen/tbus/servo_device.js').ServoCtl;

ServoCtl.prototype.moveTo = function (angle, done) {
    this.setPosition({ angle: angle }, done);
};

module.exports = Servo.Ctl;
