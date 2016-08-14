var MotorCtl = require('../../gen/tbus/motor_tbusdev.js').MotorCtl;

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
        this.start({
            direction: speed > 0 ?
                proto.tbus.MotorDriveState.Direction.FWD :
                proto.tbus.MotorDriveState.Direction.REV,
            speed: speed < 0 ? -speed : speed
        }, done);
    }
    return this;
};

MotorCtl.prototype.setBrake = function (on, done) {
    this.brake({ on: on }, done);
    return this;
};

module.exports = MotorCtl;
