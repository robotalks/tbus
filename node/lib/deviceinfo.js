var DeviceInfo = require('../gen/tbus/bus_pb.js').DeviceInfo;

DeviceInfo.prototype.fromDevice = function (device) {
    return this.fromObject(device.deviceInfo());
}

DeviceInfo.prototype.fromObject = function (info) {
    this.setAddress(info.address);
    this.setClassId(info.classId);
    this.setDeviceId(info.deviceId);
    if (info.labels != null) {
        var labelsMap = this.getLabelsMap();
        for (var k in info.labels) {
            labelsMap.set(k, info.labels[k]);
        }
    }
    return this;
}

module.exports = DeviceInfo;
