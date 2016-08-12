var expect = require('chai').expect,
    flow = require('js-flow'),
    debug = require('debug'),
    protocol = require('../index.js').protocol;

describe('Protocol', function () {
    before(function () {
        var protoLog = debug('tbus:proto');
        protocol.setLogger(function (event, data) {
            protoLog("%s: %j", event, data);
        });
    });

    var log;
    beforeEach(function () {
        log = debug('tbus:test:' + this.currentTest.title);
        log('start');
    });

    it('buildMsg', function () {
        var msg = new protocol.Encoder()
            .messageId(100)
            .encodeBody(20, Uint8Array.from([1, 2, 3, 4]))
            .route([1, 1, 3])
            .buildMsg()
        expect(msg.head.flag).to.equal(0x10);
        expect(msg.head.prefix).to.equal(0xc2);
        expect(msg.head.addrs).to.eql([1, 1, 3]);
        expect(msg.head.msgId).to.equal(100);
        expect(msg.head.bodyBytes).to.equal(5);
        expect(msg.head.raw).to.have.lengthOf(7);
        expect(msg.head.rawPrefix).to.have.lengthOf(4);
        expect(msg.head.rawHead).to.have.lengthOf(3);
        expect(msg.body.flag).to.equal(20);
        expect(msg.body.data).to.have.lengthOf(4);
        expect(msg.body.raw).to.have.lengthOf(5);
        expect(msg.head.raw[0]).to.eql(0xc2);
        expect(msg.head.raw[1]).to.eql(1);
        expect(msg.head.raw[2]).to.eql(1);
        expect(msg.head.raw[3]).to.eql(3);
        expect(msg.head.raw[4]).to.eql(0x10);
        expect(msg.head.raw[5]).to.eql(100);
        expect(msg.head.raw[6]).to.eql(5);
    });

    it('decode stream', function (done) {
        var buf = new protocol.Encoder()
            .messageId(100)
            .encodeBody(20, Uint8Array.from([1, 2, 3, 4]))
            .route([1, 1, 3])
            .toBuffer();

        var s = new protocol.DecodeStream();
        var collected = []
        s.on(protocol.EVT_ROUTE, function (msg) {
            collected.push({event: protocol.EVT_ROUTE, msg: msg});
        });
        s.on(protocol.EVT_HEAD, function (msg) {
            collected.push({event: protocol.EVT_HEAD, msg: msg});
        });
        s.on(protocol.EVT_MSG, function (msg) {
            collected.push({event: protocol.EVT_MSG, msg: msg});
        });
        s.write(buf, null, function (err) {
            expect(err).to.be.oneOf([null, undefined]);
            expect(collected).to.have.lengthOf(3);
            expect(collected[0].event).to.equal(protocol.EVT_ROUTE);
            expect(collected[0].msg.head.prefix).to.equal(0xc2);
            expect(collected[0].msg.head.addrs).to.eql([1, 1, 3]);
            expect(collected[0].msg.head.raw).to.have.lengthOf(4);
            expect(collected[0].msg.head.rawPrefix).to.have.lengthOf(4);
            expect(collected[0].msg.head.rawPrefix).to.eql(collected[0].msg.head.raw);
            expect(collected[1].event).to.equal(protocol.EVT_HEAD);
            expect(collected[1].msg.head.prefix).to.equal(0xc2);
            expect(collected[1].msg.head.addrs).to.eql([1, 1, 3]);
            expect(collected[1].msg.head.flag).to.equal(0x10);
            expect(collected[1].msg.head.msgId).to.equal(100);
            expect(collected[1].msg.head.bodyBytes).to.equal(5);
            expect(collected[1].msg.head.rawPrefix).to.have.lengthOf(4);
            expect(collected[1].msg.head.rawHead).to.have.lengthOf(3);
            expect(collected[1].msg.head.raw).to.have.lengthOf(7);
            expect(collected[2].event).to.equal(protocol.EVT_MSG);
            expect(collected[2].msg.head.prefix).to.equal(0xc2);
            expect(collected[2].msg.head.addrs).to.eql([1, 1, 3]);
            expect(collected[2].msg.head.flag).to.equal(0x10);
            expect(collected[2].msg.head.msgId).to.equal(100);
            expect(collected[2].msg.head.bodyBytes).to.equal(5);
            expect(collected[2].msg.head.rawPrefix).to.have.lengthOf(4);
            expect(collected[2].msg.head.rawHead).to.have.lengthOf(3);
            expect(collected[2].msg.head.raw).to.have.lengthOf(7);
            expect(collected[2].msg.body.flag).to.equal(20);
            expect(collected[2].msg.body.data).to.have.lengthOf(4);
            expect(collected[2].msg.body.raw).to.have.lengthOf(5);
            done(err);
        });
    });
});
