var ServoCtl = require('../../gen/tbus/servo_tbusdev.js').ServoCtl,
    ServoPosition = require('../../gen/tbus/servo_pb.js').ServoPosition;

ServoCtl.prototype.moveTo = function (angle, done) {
    var param = new ServoPosition();
    param.setAngle(angle);
    this.setPosition(param, done);
};

module.exports = ServoCtl;
