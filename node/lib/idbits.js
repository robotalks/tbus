var Class = require('js-class');

var BITS_ALL = 0xffffffff,
    BITS_N = 32;

var IdBitmap = Class({
    constructor: function () {
        this._bits = [1];
        this._allocated = 1;
        this._scanStart = 0;
    },

    alloc: function() {
        var index;
        if (this._allocated < this._bits.length * BITS_N) {
            var iStart = Math.floor(this._scanStart / BITS_N);
            var jStart = this._scanStart % BITS_N;
            index = this._lookup(iStart, jStart, this._bits.length);
            if (index == null && this._scanStart != 0) {
                index = this._lookup(0, 0, this.iStart + 1);
            }
        }
        if (index == null) {
            var i = this._bits.length;
            this._bits[i] = 1;
            index = index * BITS_N;
        } else {
            this._scanStart = index + 1;
        }
        this._allocated ++;
        return index;
    },

    release: function (id) {
        var i = Math.floor(id / BITS_N);
        var j = id % BITS_N;
        var s = 1 << j;
        var bits = this._bits[i];
        if (bits != null && (bits & s) != 0) {
            this._allocated --;
            bits &= ~s;
            if (bits == 0) {
                delete this._bits[i];
            } else {
                this._bits[i] = bits;
            }
        }
    },

    _lookup: function (iStart, jStart, iMax) {
        if (iMax > this._bits.length) {
            iMax = this._bits.length;
        }
        for (var i = iStart; i < iMax; i ++) {
            var bits = this._bits[i];
            if (bits == null) {
                this._bits[i] = 1;
                return i * BITS_N;
            }
            if (bits != BITS_ALL) {
                for (var j = jStart; j < BITS_N; j ++) {
                    var s = 1 << j;
                    if ((bits & s) == 0) {
                        this._bits[i] |= s;
                        return i * BITS_N + j;
                    }
                }
            }
            jStart = 0;
        }
        return null;
    }
});

module.exports = IdBitmap;
