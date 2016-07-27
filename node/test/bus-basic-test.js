var expect = require('chai').expect,
    Empty = require('google-protobuf/google/protobuf/empty_pb.js').Empty,
    Bus = require('../lib/bus.js'),
    protocol = require('../lib/protocol.js'),

    DeviceInfo = require('../gen/busenum_pb.js').DeviceInfo,
    BusEnumeration = require('../gen/busenum_pb.js').BusEnumeration;

describe('Bus', function () {
    it('enumeration', function (done) {
        var bus = new Bus();
        var stream = new protocol.DecodeStream(new protocol.MsgDecoder());
        bus.hostStream().pipe(stream);
        stream.on(protocol.EVT_MSG, function (msg) {
            expect(msg.messageId).to.equal(0x100);
            expect(msg.bodyFlags).to.equal(0);
            expect(msg.body).not.to.be.empty;
            var busenum = BusEnumeration.deserializeBinary(msg.body);
            expect(busenum).not.to.be.null;
            var devices = busenum.getDeviceList();
            expect(devices).not.to.be.empty;
            expect(devices).to.have.lengthOf(1);
            expect(devices[0].getClassId()).to.equal(0x0001);
            done();
        });
        bus.writeToBus(new protocol.Encoder()
            .messageId(0x100)
            .encodeBody(1, new Empty().serializeBinary())
            .toBuffer(), function (err) {
            if (err != null) {
                done(err);
            }
        });
    });
});
