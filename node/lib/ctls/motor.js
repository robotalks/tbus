var MotorCtl = require('../../gen/tbus/motor_tbusdev.js').MotorCtl;

MotorCtl.prototype.forward = function (speed, done) {
    if (typeof(speed) == 'function') {
        done = speed;
        speed = 255;
    }
    this._op = {
        direction: proto.tbus.MotorDriveState.Direction.FWD,
        speed: speed
    };
    this.on(done);
};

MotorCtl.prototype.reverse = function (speed, done) {
    if (typeof(speed) == 'function') {
        done = speed;
        speed = 255;
    }
    this._op = {
        direction: proto.tbus.MotorDriveState.Direction.REV,
        speed: speed
    };
    this.on(done);
};

MotorCtl.prototype.on = function (done) {
    if (this._op == null) {
        this._op = {
            direction: proto.tbus.MotorDriveState.Direction.FWD,
            speed: 255
        };
    }
    if (this._op.speed == 0) {
        this.stop(done);
    } else {
        this.start(this._op, done);
    }
};

MotorCtl.prototype.brake = function (done) {
    this.brake({ on: true }, done);
};

MotorCtl.prototype.release = function (done) {
    this.brake({ on: false }, done);
};

module.exports = MotorCtl;
