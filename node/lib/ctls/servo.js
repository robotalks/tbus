var ServoCtl = require('../../gen/tbus/servo_tbusdev.js').ServoCtl;

ServoCtl.prototype.moveTo = function (angle, done) {
    this.setPosition({ angle: angle }, done);
};

module.exports = ServoCtl;
