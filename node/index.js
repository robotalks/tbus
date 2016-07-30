require('./gen/bus_pb.js');
require('./gen/led_pb.js');
require('./gen/motor_pb.js');
require('./gen/servo_pb.js');

module.exports = {
    Device:     require('./lib/device.js'),
    Controller: require('./lib/control.js'),
    Bus: require('./lib/bus.js'),
    Master: require('./lib/master.js'),

    protocol: require('./lib/protocol.js'),

    BusDev: require('./gen/bus_device.js').BusDev,
    BusCtl: require('./gen/bus_device.js').BusCtl,
    LEDDev: require('./gen/led_device.js').LEDDev,
    LEDCtl: require('./gen/led_device.js').LEDCtl,
    MotorDev: require('./gen/motor_device.js').MotorDev,
    MotorCtl: require('./gen/motor_device.js').MotorCtl,
    ServoDev: require('./gen/servo_device.js').ServoDev,
    ServoCtl: require('./gen/servo_device.js').ServoCtl,
};
