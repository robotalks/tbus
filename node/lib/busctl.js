var Class = require('js-class'),
    Controller = require('./control.js'),
    Empty = require('google-protobuf/google/protobuf/empty_pb.js').Empty,
    BusEnumeration = require('../gen/busenum_pb.js').BusEnumeration;


var BusCtl = Class(Controller, {
    constructor: function (options) {
        Controller.prototype.constructor.call(this, options);
    },

    enumerate: function (done) {
        this.invoke(1, new Empty(), function (err, reply) {
            if (err == null) {
                reply = BusEnumeration.deserializeBinary(new Uint8Array(reply));
            }
            done(err, reply);
        });
    }
});

module.exports = BusCtl;
