var MotorCtl = require('../../gen/tbus/motor_tbusdev.js').MotorCtl,
    MotorDriveState = require('../../gen/tbus/motor_pb.js').MotorDriveState,
    MotorBrakeState = require('../../gen/tbus/motor_pb.js').MotorBrakeState;

MotorCtl.prototype.forward = function (speed, done) {
    if (typeof(speed) == 'function') {
        done = speed;
        speed = 255;
    }
    return this.setSpeed(speed, done);
};

MotorCtl.prototype.reverse = function (speed, done) {
    if (typeof(speed) == 'function') {
        done = speed;
        speed = 255;
    }
    return this.setSpeed(-speed, done);
};

MotorCtl.prototype.setSpeed = function (speed, done) {
    if (speed == 0) {
        this.stop(done);
    } else {
        var param = new MotorDriveState();
        param.setDirection(speed > 0 ?
            proto.tbus.MotorDriveState.Direction.FWD :
            proto.tbus.MotorDriveState.Direction.REV);
        param.setSpeed(speed < 0 ? -speed : speed);
        this.start(param, done);
    }
    return this;
};

MotorCtl.prototype.setBrake = function (on, done) {
    var param = new MotorBrakeState();
    param.setOn(on);
    this.brake(param, done);
    return this;
};

module.exports = MotorCtl;
