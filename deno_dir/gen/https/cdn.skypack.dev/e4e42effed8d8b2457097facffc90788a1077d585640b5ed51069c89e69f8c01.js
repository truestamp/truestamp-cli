var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __commonJS = (cb, mod2)=>function __require() {
        return mod2 || (0, cb[__getOwnPropNames(cb)[0]])((mod2 = {
            exports: {}
        }).exports, mod2), mod2.exports;
    };
var __copyProps = (to, from, except, desc)=>{
    if (from && typeof from === "object" || typeof from === "function") {
        for (let key of __getOwnPropNames(from))if (!__hasOwnProp.call(to, key) && key !== except) __defProp(to, key, {
            get: ()=>from[key],
            enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable
        });
    }
    return to;
};
var __toESM = (mod2, isNodeMode, target)=>(target = mod2 != null ? __create(__getProtoOf(mod2)) : {}, __copyProps(isNodeMode || !mod2 || !mod2.__esModule ? __defProp(target, "default", {
        value: mod2,
        enumerable: true
    }) : target, mod2));
var require_int = __commonJS({
    "node_modules/@stablelib/int/lib/int.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        function imulShim(a, b) {
            var ah = a >>> 16 & 65535, al = a & 65535;
            var bh = b >>> 16 & 65535, bl = b & 65535;
            return al * bl + (ah * bl + al * bh << 16 >>> 0) | 0;
        }
        exports.mul = Math.imul || imulShim;
        function add(a, b) {
            return a + b | 0;
        }
        exports.add = add;
        function sub(a, b) {
            return a - b | 0;
        }
        exports.sub = sub;
        function rotl(x, n) {
            return x << n | x >>> 32 - n;
        }
        exports.rotl = rotl;
        function rotr(x, n) {
            return x << 32 - n | x >>> n;
        }
        exports.rotr = rotr;
        function isIntegerShim(n) {
            return typeof n === "number" && isFinite(n) && Math.floor(n) === n;
        }
        exports.isInteger = Number.isInteger || isIntegerShim;
        exports.MAX_SAFE_INTEGER = 9007199254740991;
        exports.isSafeInteger = function(n) {
            return exports.isInteger(n) && n >= -exports.MAX_SAFE_INTEGER && n <= exports.MAX_SAFE_INTEGER;
        };
    }
});
var require_binary = __commonJS({
    "node_modules/@stablelib/binary/lib/binary.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        var int_1 = require_int();
        function readInt16BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return (array[offset + 0] << 8 | array[offset + 1]) << 16 >> 16;
        }
        exports.readInt16BE = readInt16BE;
        function readUint16BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return (array[offset + 0] << 8 | array[offset + 1]) >>> 0;
        }
        exports.readUint16BE = readUint16BE;
        function readInt16LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return (array[offset + 1] << 8 | array[offset]) << 16 >> 16;
        }
        exports.readInt16LE = readInt16LE;
        function readUint16LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return (array[offset + 1] << 8 | array[offset]) >>> 0;
        }
        exports.readUint16LE = readUint16LE;
        function writeUint16BE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(2);
            }
            if (offset === void 0) {
                offset = 0;
            }
            out[offset + 0] = value >>> 8;
            out[offset + 1] = value >>> 0;
            return out;
        }
        exports.writeUint16BE = writeUint16BE;
        exports.writeInt16BE = writeUint16BE;
        function writeUint16LE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(2);
            }
            if (offset === void 0) {
                offset = 0;
            }
            out[offset + 0] = value >>> 0;
            out[offset + 1] = value >>> 8;
            return out;
        }
        exports.writeUint16LE = writeUint16LE;
        exports.writeInt16LE = writeUint16LE;
        function readInt32BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return array[offset] << 24 | array[offset + 1] << 16 | array[offset + 2] << 8 | array[offset + 3];
        }
        exports.readInt32BE = readInt32BE;
        function readUint32BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return (array[offset] << 24 | array[offset + 1] << 16 | array[offset + 2] << 8 | array[offset + 3]) >>> 0;
        }
        exports.readUint32BE = readUint32BE;
        function readInt32LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return array[offset + 3] << 24 | array[offset + 2] << 16 | array[offset + 1] << 8 | array[offset];
        }
        exports.readInt32LE = readInt32LE;
        function readUint32LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            return (array[offset + 3] << 24 | array[offset + 2] << 16 | array[offset + 1] << 8 | array[offset]) >>> 0;
        }
        exports.readUint32LE = readUint32LE;
        function writeUint32BE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(4);
            }
            if (offset === void 0) {
                offset = 0;
            }
            out[offset + 0] = value >>> 24;
            out[offset + 1] = value >>> 16;
            out[offset + 2] = value >>> 8;
            out[offset + 3] = value >>> 0;
            return out;
        }
        exports.writeUint32BE = writeUint32BE;
        exports.writeInt32BE = writeUint32BE;
        function writeUint32LE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(4);
            }
            if (offset === void 0) {
                offset = 0;
            }
            out[offset + 0] = value >>> 0;
            out[offset + 1] = value >>> 8;
            out[offset + 2] = value >>> 16;
            out[offset + 3] = value >>> 24;
            return out;
        }
        exports.writeUint32LE = writeUint32LE;
        exports.writeInt32LE = writeUint32LE;
        function readInt64BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var hi = readInt32BE(array, offset);
            var lo = readInt32BE(array, offset + 4);
            return hi * 4294967296 + lo - (lo >> 31) * 4294967296;
        }
        exports.readInt64BE = readInt64BE;
        function readUint64BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var hi = readUint32BE(array, offset);
            var lo = readUint32BE(array, offset + 4);
            return hi * 4294967296 + lo;
        }
        exports.readUint64BE = readUint64BE;
        function readInt64LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var lo = readInt32LE(array, offset);
            var hi = readInt32LE(array, offset + 4);
            return hi * 4294967296 + lo - (lo >> 31) * 4294967296;
        }
        exports.readInt64LE = readInt64LE;
        function readUint64LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var lo = readUint32LE(array, offset);
            var hi = readUint32LE(array, offset + 4);
            return hi * 4294967296 + lo;
        }
        exports.readUint64LE = readUint64LE;
        function writeUint64BE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(8);
            }
            if (offset === void 0) {
                offset = 0;
            }
            writeUint32BE(value / 4294967296 >>> 0, out, offset);
            writeUint32BE(value >>> 0, out, offset + 4);
            return out;
        }
        exports.writeUint64BE = writeUint64BE;
        exports.writeInt64BE = writeUint64BE;
        function writeUint64LE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(8);
            }
            if (offset === void 0) {
                offset = 0;
            }
            writeUint32LE(value >>> 0, out, offset);
            writeUint32LE(value / 4294967296 >>> 0, out, offset + 4);
            return out;
        }
        exports.writeUint64LE = writeUint64LE;
        exports.writeInt64LE = writeUint64LE;
        function readUintBE(bitLength, array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            if (bitLength % 8 !== 0) {
                throw new Error("readUintBE supports only bitLengths divisible by 8");
            }
            if (bitLength / 8 > array.length - offset) {
                throw new Error("readUintBE: array is too short for the given bitLength");
            }
            var result = 0;
            var mul = 1;
            for(var i = bitLength / 8 + offset - 1; i >= offset; i--){
                result += array[i] * mul;
                mul *= 256;
            }
            return result;
        }
        exports.readUintBE = readUintBE;
        function readUintLE(bitLength, array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            if (bitLength % 8 !== 0) {
                throw new Error("readUintLE supports only bitLengths divisible by 8");
            }
            if (bitLength / 8 > array.length - offset) {
                throw new Error("readUintLE: array is too short for the given bitLength");
            }
            var result = 0;
            var mul = 1;
            for(var i = offset; i < offset + bitLength / 8; i++){
                result += array[i] * mul;
                mul *= 256;
            }
            return result;
        }
        exports.readUintLE = readUintLE;
        function writeUintBE(bitLength, value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(bitLength / 8);
            }
            if (offset === void 0) {
                offset = 0;
            }
            if (bitLength % 8 !== 0) {
                throw new Error("writeUintBE supports only bitLengths divisible by 8");
            }
            if (!int_1.isSafeInteger(value)) {
                throw new Error("writeUintBE value must be an integer");
            }
            var div = 1;
            for(var i = bitLength / 8 + offset - 1; i >= offset; i--){
                out[i] = value / div & 255;
                div *= 256;
            }
            return out;
        }
        exports.writeUintBE = writeUintBE;
        function writeUintLE(bitLength, value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(bitLength / 8);
            }
            if (offset === void 0) {
                offset = 0;
            }
            if (bitLength % 8 !== 0) {
                throw new Error("writeUintLE supports only bitLengths divisible by 8");
            }
            if (!int_1.isSafeInteger(value)) {
                throw new Error("writeUintLE value must be an integer");
            }
            var div = 1;
            for(var i = offset; i < offset + bitLength / 8; i++){
                out[i] = value / div & 255;
                div *= 256;
            }
            return out;
        }
        exports.writeUintLE = writeUintLE;
        function readFloat32BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(array.buffer, array.byteOffset, array.byteLength);
            return view.getFloat32(offset);
        }
        exports.readFloat32BE = readFloat32BE;
        function readFloat32LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(array.buffer, array.byteOffset, array.byteLength);
            return view.getFloat32(offset, true);
        }
        exports.readFloat32LE = readFloat32LE;
        function readFloat64BE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(array.buffer, array.byteOffset, array.byteLength);
            return view.getFloat64(offset);
        }
        exports.readFloat64BE = readFloat64BE;
        function readFloat64LE(array, offset) {
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(array.buffer, array.byteOffset, array.byteLength);
            return view.getFloat64(offset, true);
        }
        exports.readFloat64LE = readFloat64LE;
        function writeFloat32BE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(4);
            }
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(out.buffer, out.byteOffset, out.byteLength);
            view.setFloat32(offset, value);
            return out;
        }
        exports.writeFloat32BE = writeFloat32BE;
        function writeFloat32LE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(4);
            }
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(out.buffer, out.byteOffset, out.byteLength);
            view.setFloat32(offset, value, true);
            return out;
        }
        exports.writeFloat32LE = writeFloat32LE;
        function writeFloat64BE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(8);
            }
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(out.buffer, out.byteOffset, out.byteLength);
            view.setFloat64(offset, value);
            return out;
        }
        exports.writeFloat64BE = writeFloat64BE;
        function writeFloat64LE(value, out, offset) {
            if (out === void 0) {
                out = new Uint8Array(8);
            }
            if (offset === void 0) {
                offset = 0;
            }
            var view = new DataView(out.buffer, out.byteOffset, out.byteLength);
            view.setFloat64(offset, value, true);
            return out;
        }
        exports.writeFloat64LE = writeFloat64LE;
    }
});
var require_wipe = __commonJS({
    "node_modules/@stablelib/wipe/lib/wipe.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        function wipe(array) {
            for(var i = 0; i < array.length; i++){
                array[i] = 0;
            }
            return array;
        }
        exports.wipe = wipe;
    }
});
var require_sha256 = __commonJS({
    "node_modules/@stablelib/sha256/lib/sha256.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        var binary_1 = require_binary();
        var wipe_1 = require_wipe();
        exports.DIGEST_LENGTH = 32;
        exports.BLOCK_SIZE = 64;
        var SHA256 = function() {
            function SHA2562() {
                this.digestLength = exports.DIGEST_LENGTH;
                this.blockSize = exports.BLOCK_SIZE;
                this._state = new Int32Array(8);
                this._temp = new Int32Array(64);
                this._buffer = new Uint8Array(128);
                this._bufferLength = 0;
                this._bytesHashed = 0;
                this._finished = false;
                this.reset();
            }
            SHA2562.prototype._initState = function() {
                this._state[0] = 1779033703;
                this._state[1] = 3144134277;
                this._state[2] = 1013904242;
                this._state[3] = 2773480762;
                this._state[4] = 1359893119;
                this._state[5] = 2600822924;
                this._state[6] = 528734635;
                this._state[7] = 1541459225;
            };
            SHA2562.prototype.reset = function() {
                this._initState();
                this._bufferLength = 0;
                this._bytesHashed = 0;
                this._finished = false;
                return this;
            };
            SHA2562.prototype.clean = function() {
                wipe_1.wipe(this._buffer);
                wipe_1.wipe(this._temp);
                this.reset();
            };
            SHA2562.prototype.update = function(data, dataLength) {
                if (dataLength === void 0) {
                    dataLength = data.length;
                }
                if (this._finished) {
                    throw new Error("SHA256: can't update because hash was finished.");
                }
                var dataPos = 0;
                this._bytesHashed += dataLength;
                if (this._bufferLength > 0) {
                    while(this._bufferLength < this.blockSize && dataLength > 0){
                        this._buffer[this._bufferLength++] = data[dataPos++];
                        dataLength--;
                    }
                    if (this._bufferLength === this.blockSize) {
                        hashBlocks(this._temp, this._state, this._buffer, 0, this.blockSize);
                        this._bufferLength = 0;
                    }
                }
                if (dataLength >= this.blockSize) {
                    dataPos = hashBlocks(this._temp, this._state, data, dataPos, dataLength);
                    dataLength %= this.blockSize;
                }
                while(dataLength > 0){
                    this._buffer[this._bufferLength++] = data[dataPos++];
                    dataLength--;
                }
                return this;
            };
            SHA2562.prototype.finish = function(out) {
                if (!this._finished) {
                    var bytesHashed = this._bytesHashed;
                    var left = this._bufferLength;
                    var bitLenHi = bytesHashed / 536870912 | 0;
                    var bitLenLo = bytesHashed << 3;
                    var padLength = bytesHashed % 64 < 56 ? 64 : 128;
                    this._buffer[left] = 128;
                    for(var i = left + 1; i < padLength - 8; i++){
                        this._buffer[i] = 0;
                    }
                    binary_1.writeUint32BE(bitLenHi, this._buffer, padLength - 8);
                    binary_1.writeUint32BE(bitLenLo, this._buffer, padLength - 4);
                    hashBlocks(this._temp, this._state, this._buffer, 0, padLength);
                    this._finished = true;
                }
                for(var i = 0; i < this.digestLength / 4; i++){
                    binary_1.writeUint32BE(this._state[i], out, i * 4);
                }
                return this;
            };
            SHA2562.prototype.digest = function() {
                var out = new Uint8Array(this.digestLength);
                this.finish(out);
                return out;
            };
            SHA2562.prototype.saveState = function() {
                if (this._finished) {
                    throw new Error("SHA256: cannot save finished state");
                }
                return {
                    state: new Int32Array(this._state),
                    buffer: this._bufferLength > 0 ? new Uint8Array(this._buffer) : void 0,
                    bufferLength: this._bufferLength,
                    bytesHashed: this._bytesHashed
                };
            };
            SHA2562.prototype.restoreState = function(savedState) {
                this._state.set(savedState.state);
                this._bufferLength = savedState.bufferLength;
                if (savedState.buffer) {
                    this._buffer.set(savedState.buffer);
                }
                this._bytesHashed = savedState.bytesHashed;
                this._finished = false;
                return this;
            };
            SHA2562.prototype.cleanSavedState = function(savedState) {
                wipe_1.wipe(savedState.state);
                if (savedState.buffer) {
                    wipe_1.wipe(savedState.buffer);
                }
                savedState.bufferLength = 0;
                savedState.bytesHashed = 0;
            };
            return SHA2562;
        }();
        exports.SHA256 = SHA256;
        var K = new Int32Array([
            1116352408,
            1899447441,
            3049323471,
            3921009573,
            961987163,
            1508970993,
            2453635748,
            2870763221,
            3624381080,
            310598401,
            607225278,
            1426881987,
            1925078388,
            2162078206,
            2614888103,
            3248222580,
            3835390401,
            4022224774,
            264347078,
            604807628,
            770255983,
            1249150122,
            1555081692,
            1996064986,
            2554220882,
            2821834349,
            2952996808,
            3210313671,
            3336571891,
            3584528711,
            113926993,
            338241895,
            666307205,
            773529912,
            1294757372,
            1396182291,
            1695183700,
            1986661051,
            2177026350,
            2456956037,
            2730485921,
            2820302411,
            3259730800,
            3345764771,
            3516065817,
            3600352804,
            4094571909,
            275423344,
            430227734,
            506948616,
            659060556,
            883997877,
            958139571,
            1322822218,
            1537002063,
            1747873779,
            1955562222,
            2024104815,
            2227730452,
            2361852424,
            2428436474,
            2756734187,
            3204031479,
            3329325298
        ]);
        function hashBlocks(w, v, p, pos, len) {
            while(len >= 64){
                var a = v[0];
                var b = v[1];
                var c = v[2];
                var d = v[3];
                var e = v[4];
                var f = v[5];
                var g = v[6];
                var h = v[7];
                for(var i = 0; i < 16; i++){
                    var j = pos + i * 4;
                    w[i] = binary_1.readUint32BE(p, j);
                }
                for(var i = 16; i < 64; i++){
                    var u = w[i - 2];
                    var t1 = (u >>> 17 | u << 32 - 17) ^ (u >>> 19 | u << 32 - 19) ^ u >>> 10;
                    u = w[i - 15];
                    var t2 = (u >>> 7 | u << 32 - 7) ^ (u >>> 18 | u << 32 - 18) ^ u >>> 3;
                    w[i] = (t1 + w[i - 7] | 0) + (t2 + w[i - 16] | 0);
                }
                for(var i = 0; i < 64; i++){
                    var t1 = (((e >>> 6 | e << 32 - 6) ^ (e >>> 11 | e << 32 - 11) ^ (e >>> 25 | e << 32 - 25)) + (e & f ^ ~e & g) | 0) + (h + (K[i] + w[i] | 0) | 0) | 0;
                    var t2 = ((a >>> 2 | a << 32 - 2) ^ (a >>> 13 | a << 32 - 13) ^ (a >>> 22 | a << 32 - 22)) + (a & b ^ a & c ^ b & c) | 0;
                    h = g;
                    g = f;
                    f = e;
                    e = d + t1 | 0;
                    d = c;
                    c = b;
                    b = a;
                    a = t1 + t2 | 0;
                }
                v[0] += a;
                v[1] += b;
                v[2] += c;
                v[3] += d;
                v[4] += e;
                v[5] += f;
                v[6] += g;
                v[7] += h;
                pos += 64;
                len -= 64;
            }
            return pos;
        }
        function hash(data) {
            var h = new SHA256();
            h.update(data);
            var digest = h.digest();
            h.clean();
            return digest;
        }
        exports.hash = hash;
    }
});
var require_halfsiphash = __commonJS({
    "node_modules/@stablelib/halfsiphash/lib/halfsiphash.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        var binary_1 = require_binary();
        var int_1 = require_int();
        exports.KEY_LENGTH = 8;
        exports.DIGEST_LENGTH = 4;
        function halfSipHash2(key, data) {
            if (key.length !== exports.KEY_LENGTH) {
                throw new Error("halfSipHash: incorrect key length");
            }
            var k0 = binary_1.readUint32LE(key, 0);
            var k1 = binary_1.readUint32LE(key, 4);
            return binary_1.writeUint32LE(halfSipHashNum(k0, k1, data));
        }
        exports.halfSipHash = halfSipHash2;
        function halfSipHashNum(k0, k1, data) {
            var v0 = k0;
            var v1 = k1;
            var v2 = k0 ^ 1819895653;
            var v3 = k1 ^ 1952801890;
            var pos = 0;
            var len = data.length;
            var fin = len % 256 << 24;
            while(len >= 4){
                var m = binary_1.readUint32LE(data, pos);
                v3 ^= m;
                v0 = int_1.add(v0, v1);
                v1 = int_1.rotl(v1, 5);
                v1 ^= v0;
                v0 = int_1.rotl(v0, 16);
                v2 = int_1.add(v2, v3);
                v3 = int_1.rotl(v3, 8);
                v3 ^= v2;
                v0 = int_1.add(v0, v3);
                v3 = int_1.rotl(v3, 7);
                v3 ^= v0;
                v2 = int_1.add(v2, v1);
                v1 = int_1.rotl(v1, 13);
                v1 ^= v2;
                v2 = int_1.rotl(v2, 16);
                v0 = int_1.add(v0, v1);
                v1 = int_1.rotl(v1, 5);
                v1 ^= v0;
                v0 = int_1.rotl(v0, 16);
                v2 = int_1.add(v2, v3);
                v3 = int_1.rotl(v3, 8);
                v3 ^= v2;
                v0 = int_1.add(v0, v3);
                v3 = int_1.rotl(v3, 7);
                v3 ^= v0;
                v2 = int_1.add(v2, v1);
                v1 = int_1.rotl(v1, 13);
                v1 ^= v2;
                v2 = int_1.rotl(v2, 16);
                v0 ^= m;
                pos += 4;
                len -= 4;
            }
            switch(len){
                case 3:
                    fin |= data[pos + 2] << 16;
                case 2:
                    fin |= data[pos + 1] << 8;
                case 1:
                    fin |= data[pos];
            }
            v3 ^= fin;
            v0 = int_1.add(v0, v1);
            v1 = int_1.rotl(v1, 5);
            v1 ^= v0;
            v0 = int_1.rotl(v0, 16);
            v2 = int_1.add(v2, v3);
            v3 = int_1.rotl(v3, 8);
            v3 ^= v2;
            v0 = int_1.add(v0, v3);
            v3 = int_1.rotl(v3, 7);
            v3 ^= v0;
            v2 = int_1.add(v2, v1);
            v1 = int_1.rotl(v1, 13);
            v1 ^= v2;
            v2 = int_1.rotl(v2, 16);
            v0 = int_1.add(v0, v1);
            v1 = int_1.rotl(v1, 5);
            v1 ^= v0;
            v0 = int_1.rotl(v0, 16);
            v2 = int_1.add(v2, v3);
            v3 = int_1.rotl(v3, 8);
            v3 ^= v2;
            v0 = int_1.add(v0, v3);
            v3 = int_1.rotl(v3, 7);
            v3 ^= v0;
            v2 = int_1.add(v2, v1);
            v1 = int_1.rotl(v1, 13);
            v1 ^= v2;
            v2 = int_1.rotl(v2, 16);
            v0 ^= fin;
            v2 ^= 255;
            v0 = int_1.add(v0, v1);
            v1 = int_1.rotl(v1, 5);
            v1 ^= v0;
            v0 = int_1.rotl(v0, 16);
            v2 = int_1.add(v2, v3);
            v3 = int_1.rotl(v3, 8);
            v3 ^= v2;
            v0 = int_1.add(v0, v3);
            v3 = int_1.rotl(v3, 7);
            v3 ^= v0;
            v2 = int_1.add(v2, v1);
            v1 = int_1.rotl(v1, 13);
            v1 ^= v2;
            v2 = int_1.rotl(v2, 16);
            v0 = int_1.add(v0, v1);
            v1 = int_1.rotl(v1, 5);
            v1 ^= v0;
            v0 = int_1.rotl(v0, 16);
            v2 = int_1.add(v2, v3);
            v3 = int_1.rotl(v3, 8);
            v3 ^= v2;
            v0 = int_1.add(v0, v3);
            v3 = int_1.rotl(v3, 7);
            v3 ^= v0;
            v2 = int_1.add(v2, v1);
            v1 = int_1.rotl(v1, 13);
            v1 ^= v2;
            v2 = int_1.rotl(v2, 16);
            v0 = int_1.add(v0, v1);
            v1 = int_1.rotl(v1, 5);
            v1 ^= v0;
            v0 = int_1.rotl(v0, 16);
            v2 = int_1.add(v2, v3);
            v3 = int_1.rotl(v3, 8);
            v3 ^= v2;
            v0 = int_1.add(v0, v3);
            v3 = int_1.rotl(v3, 7);
            v3 ^= v0;
            v2 = int_1.add(v2, v1);
            v1 = int_1.rotl(v1, 13);
            v1 ^= v2;
            v2 = int_1.rotl(v2, 16);
            v0 = int_1.add(v0, v1);
            v1 = int_1.rotl(v1, 5);
            v1 ^= v0;
            v0 = int_1.rotl(v0, 16);
            v2 = int_1.add(v2, v3);
            v3 = int_1.rotl(v3, 8);
            v3 ^= v2;
            v0 = int_1.add(v0, v3);
            v3 = int_1.rotl(v3, 7);
            v3 ^= v0;
            v2 = int_1.add(v2, v1);
            v1 = int_1.rotl(v1, 13);
            v1 ^= v2;
            return (v1 ^ v3) >>> 0;
        }
        exports.halfSipHashNum = halfSipHashNum;
    }
});
var require_constant_time = __commonJS({
    "node_modules/@stablelib/constant-time/lib/constant-time.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        function select(subject, resultIfOne, resultIfZero) {
            return ~(subject - 1) & resultIfOne | subject - 1 & resultIfZero;
        }
        exports.select = select;
        function lessOrEqual(a, b) {
            return (a | 0) - (b | 0) - 1 >>> 31 & 1;
        }
        exports.lessOrEqual = lessOrEqual;
        function compare(a, b) {
            if (a.length !== b.length) {
                return 0;
            }
            var result = 0;
            for(var i = 0; i < a.length; i++){
                result |= a[i] ^ b[i];
            }
            return 1 & result - 1 >>> 8;
        }
        exports.compare = compare;
        function equal2(a, b) {
            if (a.length === 0 || b.length === 0) {
                return false;
            }
            return compare(a, b) !== 0;
        }
        exports.equal = equal2;
    }
});
var require_lib = __commonJS({
    "node_modules/@scure/base/lib/index.js" (exports) {
        Object.defineProperty(exports, "__esModule", {
            value: true
        });
        exports.bytes = exports.stringToBytes = exports.str = exports.bytesToString = exports.hex = exports.utf8 = exports.bech32m = exports.bech32 = exports.base58check = exports.base58xmr = exports.base58xrp = exports.base58flickr = exports.base58 = exports.base64url = exports.base64 = exports.base32crockford = exports.base32hex = exports.base32 = exports.base16 = exports.utils = exports.assertNumber = void 0;
        function assertNumber(n) {
            if (!Number.isSafeInteger(n)) throw new Error(`Wrong integer: ${n}`);
        }
        exports.assertNumber = assertNumber;
        function chain(...args) {
            const wrap = (a, b)=>(c)=>a(b(c));
            const encode2 = Array.from(args).reverse().reduce((acc, i)=>acc ? wrap(acc, i.encode) : i.encode, void 0);
            const decode2 = args.reduce((acc, i)=>acc ? wrap(acc, i.decode) : i.decode, void 0);
            return {
                encode: encode2,
                decode: decode2
            };
        }
        function alphabet(alphabet2) {
            return {
                encode: (digits)=>{
                    if (!Array.isArray(digits) || digits.length && typeof digits[0] !== "number") throw new Error("alphabet.encode input should be an array of numbers");
                    return digits.map((i)=>{
                        assertNumber(i);
                        if (i < 0 || i >= alphabet2.length) throw new Error(`Digit index outside alphabet: ${i} (alphabet: ${alphabet2.length})`);
                        return alphabet2[i];
                    });
                },
                decode: (input)=>{
                    if (!Array.isArray(input) || input.length && typeof input[0] !== "string") throw new Error("alphabet.decode input should be array of strings");
                    return input.map((letter)=>{
                        if (typeof letter !== "string") throw new Error(`alphabet.decode: not string element=${letter}`);
                        const index = alphabet2.indexOf(letter);
                        if (index === -1) throw new Error(`Unknown letter: "${letter}". Allowed: ${alphabet2}`);
                        return index;
                    });
                }
            };
        }
        function join(separator = "") {
            if (typeof separator !== "string") throw new Error("join separator should be string");
            return {
                encode: (from)=>{
                    if (!Array.isArray(from) || from.length && typeof from[0] !== "string") throw new Error("join.encode input should be array of strings");
                    for (let i of from)if (typeof i !== "string") throw new Error(`join.encode: non-string input=${i}`);
                    return from.join(separator);
                },
                decode: (to)=>{
                    if (typeof to !== "string") throw new Error("join.decode input should be string");
                    return to.split(separator);
                }
            };
        }
        function padding(bits, chr = "=") {
            assertNumber(bits);
            if (typeof chr !== "string") throw new Error("padding chr should be string");
            return {
                encode (data) {
                    if (!Array.isArray(data) || data.length && typeof data[0] !== "string") throw new Error("padding.encode input should be array of strings");
                    for (let i of data)if (typeof i !== "string") throw new Error(`padding.encode: non-string input=${i}`);
                    while(data.length * bits % 8)data.push(chr);
                    return data;
                },
                decode (input) {
                    if (!Array.isArray(input) || input.length && typeof input[0] !== "string") throw new Error("padding.encode input should be array of strings");
                    for (let i of input)if (typeof i !== "string") throw new Error(`padding.decode: non-string input=${i}`);
                    let end = input.length;
                    if (end * bits % 8) throw new Error("Invalid padding: string should have whole number of bytes");
                    for(; end > 0 && input[end - 1] === chr; end--){
                        if (!((end - 1) * bits % 8)) throw new Error("Invalid padding: string has too much padding");
                    }
                    return input.slice(0, end);
                }
            };
        }
        function normalize(fn) {
            if (typeof fn !== "function") throw new Error("normalize fn should be function");
            return {
                encode: (from)=>from,
                decode: (to)=>fn(to)
            };
        }
        function convertRadix(data, from, to) {
            if (from < 2) throw new Error(`convertRadix: wrong from=${from}, base cannot be less than 2`);
            if (to < 2) throw new Error(`convertRadix: wrong to=${to}, base cannot be less than 2`);
            if (!Array.isArray(data)) throw new Error("convertRadix: data should be array");
            if (!data.length) return [];
            let pos = 0;
            const res = [];
            const digits = Array.from(data);
            digits.forEach((d)=>{
                assertNumber(d);
                if (d < 0 || d >= from) throw new Error(`Wrong integer: ${d}`);
            });
            while(true){
                let carry = 0;
                let done = true;
                for(let i = pos; i < digits.length; i++){
                    const digit = digits[i];
                    const digitBase = from * carry + digit;
                    if (!Number.isSafeInteger(digitBase) || from * carry / from !== carry || digitBase - digit !== from * carry) {
                        throw new Error("convertRadix: carry overflow");
                    }
                    carry = digitBase % to;
                    digits[i] = Math.floor(digitBase / to);
                    if (!Number.isSafeInteger(digits[i]) || digits[i] * to + carry !== digitBase) throw new Error("convertRadix: carry overflow");
                    if (!done) continue;
                    else if (!digits[i]) pos = i;
                    else done = false;
                }
                res.push(carry);
                if (done) break;
            }
            for(let i1 = 0; i1 < data.length - 1 && data[i1] === 0; i1++)res.push(0);
            return res.reverse();
        }
        var gcd = (a, b)=>!b ? a : gcd(b, a % b);
        var radix2carry = (from, to)=>from + (to - gcd(from, to));
        function convertRadix2(data, from, to, padding2) {
            if (!Array.isArray(data)) throw new Error("convertRadix2: data should be array");
            if (from <= 0 || from > 32) throw new Error(`convertRadix2: wrong from=${from}`);
            if (to <= 0 || to > 32) throw new Error(`convertRadix2: wrong to=${to}`);
            if (radix2carry(from, to) > 32) {
                throw new Error(`convertRadix2: carry overflow from=${from} to=${to} carryBits=${radix2carry(from, to)}`);
            }
            let carry = 0;
            let pos = 0;
            const mask = 2 ** to - 1;
            const res = [];
            for (const n of data){
                assertNumber(n);
                if (n >= 2 ** from) throw new Error(`convertRadix2: invalid data word=${n} from=${from}`);
                carry = carry << from | n;
                if (pos + from > 32) throw new Error(`convertRadix2: carry overflow pos=${pos} from=${from}`);
                pos += from;
                for(; pos >= to; pos -= to)res.push((carry >> pos - to & mask) >>> 0);
                carry &= 2 ** pos - 1;
            }
            carry = carry << to - pos & mask;
            if (!padding2 && pos >= from) throw new Error("Excess padding");
            if (!padding2 && carry) throw new Error(`Non-zero padding: ${carry}`);
            if (padding2 && pos > 0) res.push(carry >>> 0);
            return res;
        }
        function radix(num) {
            assertNumber(num);
            return {
                encode: (bytes2)=>{
                    if (!(bytes2 instanceof Uint8Array)) throw new Error("radix.encode input should be Uint8Array");
                    return convertRadix(Array.from(bytes2), 2 ** 8, num);
                },
                decode: (digits)=>{
                    if (!Array.isArray(digits) || digits.length && typeof digits[0] !== "number") throw new Error("radix.decode input should be array of strings");
                    return Uint8Array.from(convertRadix(digits, num, 2 ** 8));
                }
            };
        }
        function radix2(bits, revPadding = false) {
            assertNumber(bits);
            if (bits <= 0 || bits > 32) throw new Error("radix2: bits should be in (0..32]");
            if (radix2carry(8, bits) > 32 || radix2carry(bits, 8) > 32) throw new Error("radix2: carry overflow");
            return {
                encode: (bytes2)=>{
                    if (!(bytes2 instanceof Uint8Array)) throw new Error("radix2.encode input should be Uint8Array");
                    return convertRadix2(Array.from(bytes2), 8, bits, !revPadding);
                },
                decode: (digits)=>{
                    if (!Array.isArray(digits) || digits.length && typeof digits[0] !== "number") throw new Error("radix2.decode input should be array of strings");
                    return Uint8Array.from(convertRadix2(digits, bits, 8, revPadding));
                }
            };
        }
        function unsafeWrapper(fn) {
            if (typeof fn !== "function") throw new Error("unsafeWrapper fn should be function");
            return function(...args) {
                try {
                    return fn.apply(null, args);
                } catch (e) {}
            };
        }
        function checksum(len, fn) {
            assertNumber(len);
            if (typeof fn !== "function") throw new Error("checksum fn should be function");
            return {
                encode (data) {
                    if (!(data instanceof Uint8Array)) throw new Error("checksum.encode: input should be Uint8Array");
                    const checksum2 = fn(data).slice(0, len);
                    const res = new Uint8Array(data.length + len);
                    res.set(data);
                    res.set(checksum2, data.length);
                    return res;
                },
                decode (data) {
                    if (!(data instanceof Uint8Array)) throw new Error("checksum.decode: input should be Uint8Array");
                    const payload = data.slice(0, -len);
                    const newChecksum = fn(payload).slice(0, len);
                    const oldChecksum = data.slice(-len);
                    for(let i = 0; i < len; i++)if (newChecksum[i] !== oldChecksum[i]) throw new Error("Invalid checksum");
                    return payload;
                }
            };
        }
        exports.utils = {
            alphabet,
            chain,
            checksum,
            radix,
            radix2,
            join,
            padding
        };
        exports.base16 = chain(radix2(4), alphabet("0123456789ABCDEF"), join(""));
        exports.base32 = chain(radix2(5), alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"), padding(5), join(""));
        exports.base32hex = chain(radix2(5), alphabet("0123456789ABCDEFGHIJKLMNOPQRSTUV"), padding(5), join(""));
        exports.base32crockford = chain(radix2(5), alphabet("0123456789ABCDEFGHJKMNPQRSTVWXYZ"), join(""), normalize((s)=>s.toUpperCase().replace(/O/g, "0").replace(/[IL]/g, "1")));
        exports.base64 = chain(radix2(6), alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"), padding(6), join(""));
        exports.base64url = chain(radix2(6), alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"), padding(6), join(""));
        var genBase58 = (abc)=>chain(radix(58), alphabet(abc), join(""));
        exports.base58 = genBase58("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz");
        exports.base58flickr = genBase58("123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ");
        exports.base58xrp = genBase58("rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz");
        var XMR_BLOCK_LEN = [
            0,
            2,
            3,
            5,
            6,
            7,
            9,
            10,
            11
        ];
        exports.base58xmr = {
            encode (data) {
                let res = "";
                for(let i = 0; i < data.length; i += 8){
                    const block = data.subarray(i, i + 8);
                    res += exports.base58.encode(block).padStart(XMR_BLOCK_LEN[block.length], "1");
                }
                return res;
            },
            decode (str) {
                let res = [];
                for(let i = 0; i < str.length; i += 11){
                    const slice = str.slice(i, i + 11);
                    const blockLen = XMR_BLOCK_LEN.indexOf(slice.length);
                    const block = exports.base58.decode(slice);
                    for(let j = 0; j < block.length - blockLen; j++){
                        if (block[j] !== 0) throw new Error("base58xmr: wrong padding");
                    }
                    res = res.concat(Array.from(block.slice(block.length - blockLen)));
                }
                return Uint8Array.from(res);
            }
        };
        var base58check2 = (sha2562)=>chain(checksum(4, (data)=>sha2562(sha2562(data))), exports.base58);
        exports.base58check = base58check2;
        var BECH_ALPHABET = chain(alphabet("qpzry9x8gf2tvdw0s3jn54khce6mua7l"), join(""));
        var POLYMOD_GENERATORS = [
            996825010,
            642813549,
            513874426,
            1027748829,
            705979059
        ];
        function bech32Polymod(pre) {
            const b = pre >> 25;
            let chk = (pre & 33554431) << 5;
            for(let i = 0; i < POLYMOD_GENERATORS.length; i++){
                if ((b >> i & 1) === 1) chk ^= POLYMOD_GENERATORS[i];
            }
            return chk;
        }
        function bechChecksum(prefix, words, encodingConst = 1) {
            const len = prefix.length;
            let chk = 1;
            for(let i = 0; i < len; i++){
                const c = prefix.charCodeAt(i);
                if (c < 33 || c > 126) throw new Error(`Invalid prefix (${prefix})`);
                chk = bech32Polymod(chk) ^ c >> 5;
            }
            chk = bech32Polymod(chk);
            for(let i1 = 0; i1 < len; i1++)chk = bech32Polymod(chk) ^ prefix.charCodeAt(i1) & 31;
            for (let v of words)chk = bech32Polymod(chk) ^ v;
            for(let i2 = 0; i2 < 6; i2++)chk = bech32Polymod(chk);
            chk ^= encodingConst;
            return BECH_ALPHABET.encode(convertRadix2([
                chk % 2 ** 30
            ], 30, 5, false));
        }
        function genBech32(encoding) {
            const ENCODING_CONST = encoding === "bech32" ? 1 : 734539939;
            const _words = radix2(5);
            const fromWords = _words.decode;
            const toWords = _words.encode;
            const fromWordsUnsafe = unsafeWrapper(fromWords);
            function encode2(prefix, words, limit = 90) {
                if (typeof prefix !== "string") throw new Error(`bech32.encode prefix should be string, not ${typeof prefix}`);
                if (!Array.isArray(words) || words.length && typeof words[0] !== "number") throw new Error(`bech32.encode words should be array of numbers, not ${typeof words}`);
                const actualLength = prefix.length + 7 + words.length;
                if (limit !== false && actualLength > limit) throw new TypeError(`Length ${actualLength} exceeds limit ${limit}`);
                prefix = prefix.toLowerCase();
                return `${prefix}1${BECH_ALPHABET.encode(words)}${bechChecksum(prefix, words, ENCODING_CONST)}`;
            }
            function decode2(str, limit = 90) {
                if (typeof str !== "string") throw new Error(`bech32.decode input should be string, not ${typeof str}`);
                if (str.length < 8 || limit !== false && str.length > limit) throw new TypeError(`Wrong string length: ${str.length} (${str}). Expected (8..${limit})`);
                const lowered = str.toLowerCase();
                if (str !== lowered && str !== str.toUpperCase()) throw new Error(`String must be lowercase or uppercase`);
                str = lowered;
                const sepIndex = str.lastIndexOf("1");
                if (sepIndex === 0 || sepIndex === -1) throw new Error(`Letter "1" must be present between prefix and data only`);
                const prefix = str.slice(0, sepIndex);
                const _words2 = str.slice(sepIndex + 1);
                if (_words2.length < 6) throw new Error("Data must be at least 6 characters long");
                const words = BECH_ALPHABET.decode(_words2).slice(0, -6);
                const sum = bechChecksum(prefix, words, ENCODING_CONST);
                if (!_words2.endsWith(sum)) throw new Error(`Invalid checksum in ${str}: expected "${sum}"`);
                return {
                    prefix,
                    words
                };
            }
            const decodeUnsafe = unsafeWrapper(decode2);
            function decodeToBytes(str) {
                const { prefix , words  } = decode2(str, false);
                return {
                    prefix,
                    words,
                    bytes: fromWords(words)
                };
            }
            return {
                encode: encode2,
                decode: decode2,
                decodeToBytes,
                decodeUnsafe,
                fromWords,
                fromWordsUnsafe,
                toWords
            };
        }
        exports.bech32 = genBech32("bech32");
        exports.bech32m = genBech32("bech32m");
        exports.utf8 = {
            encode: (data)=>new TextDecoder().decode(data),
            decode: (str)=>new TextEncoder().encode(str)
        };
        exports.hex = chain(radix2(4), alphabet("0123456789abcdef"), join(""), normalize((s)=>{
            if (typeof s !== "string" || s.length % 2) throw new TypeError(`hex.decode: expected string, got ${typeof s} with length ${s.length}`);
            return s.toLowerCase();
        }));
        var CODERS = {
            utf8: exports.utf8,
            hex: exports.hex,
            base16: exports.base16,
            base32: exports.base32,
            base64: exports.base64,
            base64url: exports.base64url,
            base58: exports.base58,
            base58xmr: exports.base58xmr
        };
        var coderTypeError = `Invalid encoding type. Available types: ${Object.keys(CODERS).join(", ")}`;
        var bytesToString = (type, bytes2)=>{
            if (typeof type !== "string" || !CODERS.hasOwnProperty(type)) throw new TypeError(coderTypeError);
            if (!(bytes2 instanceof Uint8Array)) throw new TypeError("bytesToString() expects Uint8Array");
            return CODERS[type].encode(bytes2);
        };
        exports.bytesToString = bytesToString;
        exports.str = exports.bytesToString;
        var stringToBytes = (type, str)=>{
            if (!CODERS.hasOwnProperty(type)) throw new TypeError(coderTypeError);
            if (typeof str !== "string") throw new TypeError("stringToBytes() expects string");
            return CODERS[type].decode(str);
        };
        exports.stringToBytes = stringToBytes;
        exports.bytes = exports.stringToBytes;
    }
});
function createContext(size = 4096) {
    const buffer = new ArrayBuffer(size);
    return {
        i: 0,
        view: new DataView(buffer),
        bytes: new Uint8Array(buffer)
    };
}
function growContext(ctx) {
    ctx.bytes = new Uint8Array(ctx.bytes.length * 2);
    ctx.view = new DataView(ctx.bytes.buffer);
}
function contextSer(ctx, ser, data) {
    while(true){
        const limit = ctx.bytes.length - 8;
        ctx.i = 0;
        try {
            ser(ctx, data);
            if (ctx.i < limit) return ctx.bytes;
        } catch (error) {
            if (ctx.i < limit) throw error;
        }
        growContext(ctx);
    }
}
function contextDes(ctx, des, bytes2) {
    const { length  } = bytes2;
    if (length < 4096) {
        ctx.bytes.set(bytes2);
        ctx.i = 0;
        const data = des(ctx);
        if (ctx.i > length) throw RangeError();
        return data;
    } else {
        return des(contextFromBytes(bytes2));
    }
}
function contextFromBytes(array2) {
    return {
        i: 0,
        bytes: array2,
        view: new DataView(array2.buffer, array2.byteOffset, array2.byteLength)
    };
}
function define(ser, des) {
    return {
        ser,
        des
    };
}
var latin1 = {
    encode (ctx, data) {
        const { length  } = data;
        for(let i = 0; i < length; i++){
            ctx.view.setUint8(ctx.i++, data.charCodeAt(i));
        }
    },
    decode (ctx, size) {
        const codes = new Array(size);
        for(let i = 0; i < size; i++){
            codes[i] = ctx.view.getUint8(ctx.i++);
        }
        return String.fromCharCode(...codes);
    }
};
var string = (encoding, headSd)=>define((ctx, data)=>{
        const head = ctx.i;
        headSd.ser(ctx, data.length);
        const begin = ctx.i;
        const headSize = begin - head;
        encoding.encode(ctx, data);
        const end = ctx.i;
        const size = end - begin;
        if (size === data.length) return;
        headSd.ser(ctx, size);
        const requiredHeadSize = ctx.i - end;
        if (headSize !== requiredHeadSize) {
            ctx.bytes.copyWithin(head + requiredHeadSize, begin, end);
        }
        ctx.i = head;
        headSd.ser(ctx, size);
        ctx.i = end + (requiredHeadSize - headSize);
    }, (ctx)=>encoding.decode(ctx, headSd.des(ctx)));
var bytes = (headSd)=>define((ctx, data)=>{
        const { byteLength  } = data;
        headSd.ser(ctx, byteLength);
        ctx.bytes.set(data, ctx.i);
        ctx.i += byteLength;
    }, (ctx)=>{
        const byteLength = headSd.des(ctx);
        return ctx.bytes.subarray(ctx.i, ctx.i += byteLength);
    });
var boolean = define((ctx, data)=>void ctx.view.setUint8(ctx.i++, +data), (ctx)=>!!ctx.view.getUint8(ctx.i++));
function use({ ser , des  }) {
    const ctx = createContext();
    return {
        ser,
        des,
        toBytes: (data)=>contextSer(ctx, ser, data).slice(0, ctx.i),
        toUnsafeBytes: (data)=>contextSer(ctx, ser, data).subarray(0, ctx.i),
        fromBytes: (bytes2)=>contextDes(ctx, des, bytes2)
    };
}
var uint8 = define((ctx, data)=>ctx.view.setUint8(ctx.i++, data), (ctx)=>ctx.view.getUint8(ctx.i++));
var bigUint64 = define((ctx, data)=>{
    ctx.view.setBigUint64(ctx.i, data);
    ctx.i += 8;
}, (ctx)=>{
    const data = ctx.view.getBigUint64(ctx.i);
    ctx.i += 8;
    return data;
});
var struct = (definition)=>{
    const obj = definition instanceof Array ? ()=>[] : ()=>({});
    return define((ctx, data)=>{
        for(const key in definition){
            definition[key].ser(ctx, data[key]);
        }
    }, (ctx)=>{
        const data = obj();
        for(const key in definition){
            data[key] = definition[key].des(ctx);
        }
        return data;
    });
};
var import_sha256 = __toESM(require_sha256(), 1);
var import_halfsiphash = __toESM(require_halfsiphash(), 1);
var import_constant_time = __toESM(require_constant_time(), 1);
var util;
(function(util2) {
    function assertEqual(_cond) {}
    util2.assertEqual = assertEqual;
    function assertNever(_x) {
        throw new Error();
    }
    util2.assertNever = assertNever;
    util2.arrayToEnum = (items)=>{
        const obj = {};
        for (const item of items){
            obj[item] = item;
        }
        return obj;
    };
    util2.getValidEnumValues = (obj)=>{
        const validKeys = util2.objectKeys(obj).filter((k)=>typeof obj[obj[k]] !== "number");
        const filtered = {};
        for (const k of validKeys){
            filtered[k] = obj[k];
        }
        return util2.objectValues(filtered);
    };
    util2.objectValues = (obj)=>{
        return util2.objectKeys(obj).map(function(e) {
            return obj[e];
        });
    };
    util2.objectKeys = typeof Object.keys === "function" ? (obj)=>Object.keys(obj) : (object)=>{
        const keys = [];
        for(const key in object){
            if (Object.prototype.hasOwnProperty.call(object, key)) {
                keys.push(key);
            }
        }
        return keys;
    };
    util2.find = (arr, checker)=>{
        for (const item of arr){
            if (checker(item)) return item;
        }
        return void 0;
    };
    util2.isInteger = typeof Number.isInteger === "function" ? (val)=>Number.isInteger(val) : (val)=>typeof val === "number" && isFinite(val) && Math.floor(val) === val;
    function joinValues(array, separator = " | ") {
        return array.map((val)=>typeof val === "string" ? `'${val}'` : val).join(separator);
    }
    util2.joinValues = joinValues;
})(util || (util = {}));
var ZodParsedType = util.arrayToEnum([
    "string",
    "nan",
    "number",
    "integer",
    "float",
    "boolean",
    "date",
    "bigint",
    "symbol",
    "function",
    "undefined",
    "null",
    "array",
    "object",
    "unknown",
    "promise",
    "void",
    "never",
    "map",
    "set"
]);
var getParsedType = (data)=>{
    const t = typeof data;
    switch(t){
        case "undefined":
            return ZodParsedType.undefined;
        case "string":
            return ZodParsedType.string;
        case "number":
            return isNaN(data) ? ZodParsedType.nan : ZodParsedType.number;
        case "boolean":
            return ZodParsedType.boolean;
        case "function":
            return ZodParsedType.function;
        case "bigint":
            return ZodParsedType.bigint;
        case "object":
            if (Array.isArray(data)) {
                return ZodParsedType.array;
            }
            if (data === null) {
                return ZodParsedType.null;
            }
            if (data.then && typeof data.then === "function" && data.catch && typeof data.catch === "function") {
                return ZodParsedType.promise;
            }
            if (typeof Map !== "undefined" && data instanceof Map) {
                return ZodParsedType.map;
            }
            if (typeof Set !== "undefined" && data instanceof Set) {
                return ZodParsedType.set;
            }
            if (typeof Date !== "undefined" && data instanceof Date) {
                return ZodParsedType.date;
            }
            return ZodParsedType.object;
        default:
            return ZodParsedType.unknown;
    }
};
var ZodIssueCode = util.arrayToEnum([
    "invalid_type",
    "invalid_literal",
    "custom",
    "invalid_union",
    "invalid_union_discriminator",
    "invalid_enum_value",
    "unrecognized_keys",
    "invalid_arguments",
    "invalid_return_type",
    "invalid_date",
    "invalid_string",
    "too_small",
    "too_big",
    "invalid_intersection_types",
    "not_multiple_of"
]);
var quotelessJson = (obj)=>{
    const json = JSON.stringify(obj, null, 2);
    return json.replace(/"([^"]+)":/g, "$1:");
};
var ZodError = class extends Error {
    constructor(issues){
        super();
        this.issues = [];
        this.addIssue = (sub)=>{
            this.issues = [
                ...this.issues,
                sub
            ];
        };
        this.addIssues = (subs = [])=>{
            this.issues = [
                ...this.issues,
                ...subs
            ];
        };
        const actualProto = new.target.prototype;
        if (Object.setPrototypeOf) {
            Object.setPrototypeOf(this, actualProto);
        } else {
            this.__proto__ = actualProto;
        }
        this.name = "ZodError";
        this.issues = issues;
    }
    get errors() {
        return this.issues;
    }
    format(_mapper) {
        const mapper = _mapper || function(issue) {
            return issue.message;
        };
        const fieldErrors = {
            _errors: []
        };
        const processError = (error)=>{
            for (const issue of error.issues){
                if (issue.code === "invalid_union") {
                    issue.unionErrors.map(processError);
                } else if (issue.code === "invalid_return_type") {
                    processError(issue.returnTypeError);
                } else if (issue.code === "invalid_arguments") {
                    processError(issue.argumentsError);
                } else if (issue.path.length === 0) {
                    fieldErrors._errors.push(mapper(issue));
                } else {
                    let curr = fieldErrors;
                    let i = 0;
                    while(i < issue.path.length){
                        const el = issue.path[i];
                        const terminal = i === issue.path.length - 1;
                        if (!terminal) {
                            curr[el] = curr[el] || {
                                _errors: []
                            };
                        } else {
                            curr[el] = curr[el] || {
                                _errors: []
                            };
                            curr[el]._errors.push(mapper(issue));
                        }
                        curr = curr[el];
                        i++;
                    }
                }
            }
        };
        processError(this);
        return fieldErrors;
    }
    toString() {
        return this.message;
    }
    get message() {
        return JSON.stringify(this.issues, jsonStringifyReplacer, 2);
    }
    get isEmpty() {
        return this.issues.length === 0;
    }
    flatten(mapper = (issue)=>issue.message) {
        const fieldErrors = {};
        const formErrors = [];
        for (const sub of this.issues){
            if (sub.path.length > 0) {
                fieldErrors[sub.path[0]] = fieldErrors[sub.path[0]] || [];
                fieldErrors[sub.path[0]].push(mapper(sub));
            } else {
                formErrors.push(mapper(sub));
            }
        }
        return {
            formErrors,
            fieldErrors
        };
    }
    get formErrors() {
        return this.flatten();
    }
};
ZodError.create = (issues)=>{
    const error = new ZodError(issues);
    return error;
};
var defaultErrorMap = (issue, _ctx)=>{
    let message;
    switch(issue.code){
        case ZodIssueCode.invalid_type:
            if (issue.received === ZodParsedType.undefined) {
                message = "Required";
            } else {
                message = `Expected ${issue.expected}, received ${issue.received}`;
            }
            break;
        case ZodIssueCode.invalid_literal:
            message = `Invalid literal value, expected ${JSON.stringify(issue.expected, jsonStringifyReplacer)}`;
            break;
        case ZodIssueCode.unrecognized_keys:
            message = `Unrecognized key(s) in object: ${util.joinValues(issue.keys, ", ")}`;
            break;
        case ZodIssueCode.invalid_union:
            message = `Invalid input`;
            break;
        case ZodIssueCode.invalid_union_discriminator:
            message = `Invalid discriminator value. Expected ${util.joinValues(issue.options)}`;
            break;
        case ZodIssueCode.invalid_enum_value:
            message = `Invalid enum value. Expected ${util.joinValues(issue.options)}, received '${issue.received}'`;
            break;
        case ZodIssueCode.invalid_arguments:
            message = `Invalid function arguments`;
            break;
        case ZodIssueCode.invalid_return_type:
            message = `Invalid function return type`;
            break;
        case ZodIssueCode.invalid_date:
            message = `Invalid date`;
            break;
        case ZodIssueCode.invalid_string:
            if (typeof issue.validation === "object") {
                if ("startsWith" in issue.validation) {
                    message = `Invalid input: must start with "${issue.validation.startsWith}"`;
                } else if ("endsWith" in issue.validation) {
                    message = `Invalid input: must end with "${issue.validation.endsWith}"`;
                } else {
                    util.assertNever(issue.validation);
                }
            } else if (issue.validation !== "regex") {
                message = `Invalid ${issue.validation}`;
            } else {
                message = "Invalid";
            }
            break;
        case ZodIssueCode.too_small:
            if (issue.type === "array") message = `Array must contain ${issue.inclusive ? `at least` : `more than`} ${issue.minimum} element(s)`;
            else if (issue.type === "string") message = `String must contain ${issue.inclusive ? `at least` : `over`} ${issue.minimum} character(s)`;
            else if (issue.type === "number") message = `Number must be greater than ${issue.inclusive ? `or equal to ` : ``}${issue.minimum}`;
            else if (issue.type === "date") message = `Date must be greater than ${issue.inclusive ? `or equal to ` : ``}${new Date(issue.minimum)}`;
            else message = "Invalid input";
            break;
        case ZodIssueCode.too_big:
            if (issue.type === "array") message = `Array must contain ${issue.inclusive ? `at most` : `less than`} ${issue.maximum} element(s)`;
            else if (issue.type === "string") message = `String must contain ${issue.inclusive ? `at most` : `under`} ${issue.maximum} character(s)`;
            else if (issue.type === "number") message = `Number must be less than ${issue.inclusive ? `or equal to ` : ``}${issue.maximum}`;
            else if (issue.type === "date") message = `Date must be smaller than ${issue.inclusive ? `or equal to ` : ``}${new Date(issue.maximum)}`;
            else message = "Invalid input";
            break;
        case ZodIssueCode.custom:
            message = `Invalid input`;
            break;
        case ZodIssueCode.invalid_intersection_types:
            message = `Intersection results could not be merged`;
            break;
        case ZodIssueCode.not_multiple_of:
            message = `Number must be a multiple of ${issue.multipleOf}`;
            break;
        default:
            message = _ctx.defaultError;
            util.assertNever(issue);
    }
    return {
        message
    };
};
var overrideErrorMap = defaultErrorMap;
function setErrorMap(map) {
    overrideErrorMap = map;
}
function getErrorMap() {
    return overrideErrorMap;
}
var makeIssue = (params)=>{
    const { data , path , errorMaps , issueData  } = params;
    const fullPath = [
        ...path,
        ...issueData.path || []
    ];
    const fullIssue = {
        ...issueData,
        path: fullPath
    };
    let errorMessage = "";
    const maps = errorMaps.filter((m)=>!!m).slice().reverse();
    for (const map of maps){
        errorMessage = map(fullIssue, {
            data,
            defaultError: errorMessage
        }).message;
    }
    return {
        ...issueData,
        path: fullPath,
        message: issueData.message || errorMessage
    };
};
var EMPTY_PATH = [];
function addIssueToContext(ctx, issueData) {
    const issue = makeIssue({
        issueData,
        data: ctx.data,
        path: ctx.path,
        errorMaps: [
            ctx.common.contextualErrorMap,
            ctx.schemaErrorMap,
            getErrorMap(),
            defaultErrorMap
        ].filter((x)=>!!x)
    });
    ctx.common.issues.push(issue);
}
var ParseStatus = class {
    constructor(){
        this.value = "valid";
    }
    dirty() {
        if (this.value === "valid") this.value = "dirty";
    }
    abort() {
        if (this.value !== "aborted") this.value = "aborted";
    }
    static mergeArray(status, results) {
        const arrayValue = [];
        for (const s of results){
            if (s.status === "aborted") return INVALID;
            if (s.status === "dirty") status.dirty();
            arrayValue.push(s.value);
        }
        return {
            status: status.value,
            value: arrayValue
        };
    }
    static async mergeObjectAsync(status, pairs) {
        const syncPairs = [];
        for (const pair of pairs){
            syncPairs.push({
                key: await pair.key,
                value: await pair.value
            });
        }
        return ParseStatus.mergeObjectSync(status, syncPairs);
    }
    static mergeObjectSync(status, pairs) {
        const finalObject = {};
        for (const pair of pairs){
            const { key , value  } = pair;
            if (key.status === "aborted") return INVALID;
            if (value.status === "aborted") return INVALID;
            if (key.status === "dirty") status.dirty();
            if (value.status === "dirty") status.dirty();
            if (typeof value.value !== "undefined" || pair.alwaysSet) {
                finalObject[key.value] = value.value;
            }
        }
        return {
            status: status.value,
            value: finalObject
        };
    }
};
var INVALID = Object.freeze({
    status: "aborted"
});
var DIRTY = (value)=>({
        status: "dirty",
        value
    });
var OK = (value)=>({
        status: "valid",
        value
    });
var isAborted = (x)=>x.status === "aborted";
var isDirty = (x)=>x.status === "dirty";
var isValid = (x)=>x.status === "valid";
var isAsync = (x)=>typeof Promise !== void 0 && x instanceof Promise;
var jsonStringifyReplacer = (_, value)=>{
    if (typeof value === "bigint") {
        return value.toString();
    }
    return value;
};
var errorUtil;
(function(errorUtil2) {
    errorUtil2.errToObj = (message)=>typeof message === "string" ? {
            message
        } : message || {};
    errorUtil2.toString = (message)=>typeof message === "string" ? message : message === null || message === void 0 ? void 0 : message.message;
})(errorUtil || (errorUtil = {}));
var ParseInputLazyPath = class {
    constructor(parent, value, path, key){
        this.parent = parent;
        this.data = value;
        this._path = path;
        this._key = key;
    }
    get path() {
        return this._path.concat(this._key);
    }
};
var handleResult = (ctx, result)=>{
    if (isValid(result)) {
        return {
            success: true,
            data: result.value
        };
    } else {
        if (!ctx.common.issues.length) {
            throw new Error("Validation failed but no issues detected.");
        }
        const error = new ZodError(ctx.common.issues);
        return {
            success: false,
            error
        };
    }
};
function processCreateParams(params) {
    if (!params) return {};
    const { errorMap , invalid_type_error , required_error , description  } = params;
    if (errorMap && (invalid_type_error || required_error)) {
        throw new Error(`Can't use "invalid" or "required" in conjunction with custom error map.`);
    }
    if (errorMap) return {
        errorMap,
        description
    };
    const customMap = (iss, ctx)=>{
        if (iss.code !== "invalid_type") return {
            message: ctx.defaultError
        };
        if (typeof ctx.data === "undefined") {
            return {
                message: required_error !== null && required_error !== void 0 ? required_error : ctx.defaultError
            };
        }
        return {
            message: invalid_type_error !== null && invalid_type_error !== void 0 ? invalid_type_error : ctx.defaultError
        };
    };
    return {
        errorMap: customMap,
        description
    };
}
var ZodType = class {
    constructor(def){
        this.spa = this.safeParseAsync;
        this.superRefine = this._refinement;
        this._def = def;
        this.parse = this.parse.bind(this);
        this.safeParse = this.safeParse.bind(this);
        this.parseAsync = this.parseAsync.bind(this);
        this.safeParseAsync = this.safeParseAsync.bind(this);
        this.spa = this.spa.bind(this);
        this.refine = this.refine.bind(this);
        this.refinement = this.refinement.bind(this);
        this.superRefine = this.superRefine.bind(this);
        this.optional = this.optional.bind(this);
        this.nullable = this.nullable.bind(this);
        this.nullish = this.nullish.bind(this);
        this.array = this.array.bind(this);
        this.promise = this.promise.bind(this);
        this.or = this.or.bind(this);
        this.and = this.and.bind(this);
        this.transform = this.transform.bind(this);
        this.default = this.default.bind(this);
        this.describe = this.describe.bind(this);
        this.isNullable = this.isNullable.bind(this);
        this.isOptional = this.isOptional.bind(this);
    }
    get description() {
        return this._def.description;
    }
    _getType(input) {
        return getParsedType(input.data);
    }
    _getOrReturnCtx(input, ctx) {
        return ctx || {
            common: input.parent.common,
            data: input.data,
            parsedType: getParsedType(input.data),
            schemaErrorMap: this._def.errorMap,
            path: input.path,
            parent: input.parent
        };
    }
    _processInputParams(input) {
        return {
            status: new ParseStatus(),
            ctx: {
                common: input.parent.common,
                data: input.data,
                parsedType: getParsedType(input.data),
                schemaErrorMap: this._def.errorMap,
                path: input.path,
                parent: input.parent
            }
        };
    }
    _parseSync(input) {
        const result = this._parse(input);
        if (isAsync(result)) {
            throw new Error("Synchronous parse encountered promise.");
        }
        return result;
    }
    _parseAsync(input) {
        const result = this._parse(input);
        return Promise.resolve(result);
    }
    parse(data, params) {
        const result = this.safeParse(data, params);
        if (result.success) return result.data;
        throw result.error;
    }
    safeParse(data, params) {
        var _a;
        const ctx = {
            common: {
                issues: [],
                async: (_a = params === null || params === void 0 ? void 0 : params.async) !== null && _a !== void 0 ? _a : false,
                contextualErrorMap: params === null || params === void 0 ? void 0 : params.errorMap
            },
            path: (params === null || params === void 0 ? void 0 : params.path) || [],
            schemaErrorMap: this._def.errorMap,
            parent: null,
            data,
            parsedType: getParsedType(data)
        };
        const result = this._parseSync({
            data,
            path: ctx.path,
            parent: ctx
        });
        return handleResult(ctx, result);
    }
    async parseAsync(data, params) {
        const result = await this.safeParseAsync(data, params);
        if (result.success) return result.data;
        throw result.error;
    }
    async safeParseAsync(data, params) {
        const ctx = {
            common: {
                issues: [],
                contextualErrorMap: params === null || params === void 0 ? void 0 : params.errorMap,
                async: true
            },
            path: (params === null || params === void 0 ? void 0 : params.path) || [],
            schemaErrorMap: this._def.errorMap,
            parent: null,
            data,
            parsedType: getParsedType(data)
        };
        const maybeAsyncResult = this._parse({
            data,
            path: [],
            parent: ctx
        });
        const result = await (isAsync(maybeAsyncResult) ? maybeAsyncResult : Promise.resolve(maybeAsyncResult));
        return handleResult(ctx, result);
    }
    refine(check, message) {
        const getIssueProperties = (val)=>{
            if (typeof message === "string" || typeof message === "undefined") {
                return {
                    message
                };
            } else if (typeof message === "function") {
                return message(val);
            } else {
                return message;
            }
        };
        return this._refinement((val, ctx)=>{
            const result = check(val);
            const setError = ()=>ctx.addIssue({
                    code: ZodIssueCode.custom,
                    ...getIssueProperties(val)
                });
            if (typeof Promise !== "undefined" && result instanceof Promise) {
                return result.then((data)=>{
                    if (!data) {
                        setError();
                        return false;
                    } else {
                        return true;
                    }
                });
            }
            if (!result) {
                setError();
                return false;
            } else {
                return true;
            }
        });
    }
    refinement(check, refinementData) {
        return this._refinement((val, ctx)=>{
            if (!check(val)) {
                ctx.addIssue(typeof refinementData === "function" ? refinementData(val, ctx) : refinementData);
                return false;
            } else {
                return true;
            }
        });
    }
    _refinement(refinement) {
        return new ZodEffects({
            schema: this,
            typeName: ZodFirstPartyTypeKind.ZodEffects,
            effect: {
                type: "refinement",
                refinement
            }
        });
    }
    optional() {
        return ZodOptional.create(this);
    }
    nullable() {
        return ZodNullable.create(this);
    }
    nullish() {
        return this.optional().nullable();
    }
    array() {
        return ZodArray.create(this);
    }
    promise() {
        return ZodPromise.create(this);
    }
    or(option) {
        return ZodUnion.create([
            this,
            option
        ]);
    }
    and(incoming) {
        return ZodIntersection.create(this, incoming);
    }
    transform(transform) {
        return new ZodEffects({
            schema: this,
            typeName: ZodFirstPartyTypeKind.ZodEffects,
            effect: {
                type: "transform",
                transform
            }
        });
    }
    default(def) {
        const defaultValueFunc = typeof def === "function" ? def : ()=>def;
        return new ZodDefault({
            innerType: this,
            defaultValue: defaultValueFunc,
            typeName: ZodFirstPartyTypeKind.ZodDefault
        });
    }
    describe(description) {
        const This = this.constructor;
        return new This({
            ...this._def,
            description
        });
    }
    isOptional() {
        return this.safeParse(void 0).success;
    }
    isNullable() {
        return this.safeParse(null).success;
    }
};
var cuidRegex = /^c[^\s-]{8,}$/i;
var uuidRegex = /^([a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[a-f0-9]{4}-[a-f0-9]{12}|00000000-0000-0000-0000-000000000000)$/i;
var emailRegex = /^(([^<>()[\]\.,;:\s@\"]+(\.[^<>()[\]\.,;:\s@\"]+)*)|(\".+\"))@(([^<>()[\]\.,;:\s@\"]+\.)+[^<>()[\]\.,;:\s@\"]{2,})$/i;
var ZodString = class extends ZodType {
    constructor(){
        super(...arguments);
        this._regex = (regex, validation, message)=>this.refinement((data)=>regex.test(data), {
                validation,
                code: ZodIssueCode.invalid_string,
                ...errorUtil.errToObj(message)
            });
        this.nonempty = (message)=>this.min(1, errorUtil.errToObj(message));
        this.trim = ()=>new ZodString({
                ...this._def,
                checks: [
                    ...this._def.checks,
                    {
                        kind: "trim"
                    }
                ]
            });
    }
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.string) {
            const ctx2 = this._getOrReturnCtx(input);
            addIssueToContext(ctx2, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.string,
                received: ctx2.parsedType
            });
            return INVALID;
        }
        const status = new ParseStatus();
        let ctx = void 0;
        for (const check of this._def.checks){
            if (check.kind === "min") {
                if (input.data.length < check.value) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_small,
                        minimum: check.value,
                        type: "string",
                        inclusive: true,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "max") {
                if (input.data.length > check.value) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_big,
                        maximum: check.value,
                        type: "string",
                        inclusive: true,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "email") {
                if (!emailRegex.test(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "email",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "uuid") {
                if (!uuidRegex.test(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "uuid",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "cuid") {
                if (!cuidRegex.test(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "cuid",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "url") {
                try {
                    new URL(input.data);
                } catch (_a) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "url",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "regex") {
                check.regex.lastIndex = 0;
                const testResult = check.regex.test(input.data);
                if (!testResult) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "regex",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "trim") {
                input.data = input.data.trim();
            } else if (check.kind === "startsWith") {
                if (!input.data.startsWith(check.value)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.invalid_string,
                        validation: {
                            startsWith: check.value
                        },
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "endsWith") {
                if (!input.data.endsWith(check.value)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.invalid_string,
                        validation: {
                            endsWith: check.value
                        },
                        message: check.message
                    });
                    status.dirty();
                }
            } else {
                util.assertNever(check);
            }
        }
        return {
            status: status.value,
            value: input.data
        };
    }
    _addCheck(check) {
        return new ZodString({
            ...this._def,
            checks: [
                ...this._def.checks,
                check
            ]
        });
    }
    email(message) {
        return this._addCheck({
            kind: "email",
            ...errorUtil.errToObj(message)
        });
    }
    url(message) {
        return this._addCheck({
            kind: "url",
            ...errorUtil.errToObj(message)
        });
    }
    uuid(message) {
        return this._addCheck({
            kind: "uuid",
            ...errorUtil.errToObj(message)
        });
    }
    cuid(message) {
        return this._addCheck({
            kind: "cuid",
            ...errorUtil.errToObj(message)
        });
    }
    regex(regex, message) {
        return this._addCheck({
            kind: "regex",
            regex,
            ...errorUtil.errToObj(message)
        });
    }
    startsWith(value, message) {
        return this._addCheck({
            kind: "startsWith",
            value,
            ...errorUtil.errToObj(message)
        });
    }
    endsWith(value, message) {
        return this._addCheck({
            kind: "endsWith",
            value,
            ...errorUtil.errToObj(message)
        });
    }
    min(minLength, message) {
        return this._addCheck({
            kind: "min",
            value: minLength,
            ...errorUtil.errToObj(message)
        });
    }
    max(maxLength, message) {
        return this._addCheck({
            kind: "max",
            value: maxLength,
            ...errorUtil.errToObj(message)
        });
    }
    length(len, message) {
        return this.min(len, message).max(len, message);
    }
    get isEmail() {
        return !!this._def.checks.find((ch)=>ch.kind === "email");
    }
    get isURL() {
        return !!this._def.checks.find((ch)=>ch.kind === "url");
    }
    get isUUID() {
        return !!this._def.checks.find((ch)=>ch.kind === "uuid");
    }
    get isCUID() {
        return !!this._def.checks.find((ch)=>ch.kind === "cuid");
    }
    get minLength() {
        let min = null;
        for (const ch of this._def.checks){
            if (ch.kind === "min") {
                if (min === null || ch.value > min) min = ch.value;
            }
        }
        return min;
    }
    get maxLength() {
        let max = null;
        for (const ch of this._def.checks){
            if (ch.kind === "max") {
                if (max === null || ch.value < max) max = ch.value;
            }
        }
        return max;
    }
};
ZodString.create = (params)=>{
    return new ZodString({
        checks: [],
        typeName: ZodFirstPartyTypeKind.ZodString,
        ...processCreateParams(params)
    });
};
function floatSafeRemainder(val, step) {
    const valDecCount = (val.toString().split(".")[1] || "").length;
    const stepDecCount = (step.toString().split(".")[1] || "").length;
    const decCount = valDecCount > stepDecCount ? valDecCount : stepDecCount;
    const valInt = parseInt(val.toFixed(decCount).replace(".", ""));
    const stepInt = parseInt(step.toFixed(decCount).replace(".", ""));
    return valInt % stepInt / Math.pow(10, decCount);
}
var ZodNumber = class extends ZodType {
    constructor(){
        super(...arguments);
        this.min = this.gte;
        this.max = this.lte;
        this.step = this.multipleOf;
    }
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.number) {
            const ctx2 = this._getOrReturnCtx(input);
            addIssueToContext(ctx2, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.number,
                received: ctx2.parsedType
            });
            return INVALID;
        }
        let ctx = void 0;
        const status = new ParseStatus();
        for (const check of this._def.checks){
            if (check.kind === "int") {
                if (!util.isInteger(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.invalid_type,
                        expected: "integer",
                        received: "float",
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "min") {
                const tooSmall = check.inclusive ? input.data < check.value : input.data <= check.value;
                if (tooSmall) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_small,
                        minimum: check.value,
                        type: "number",
                        inclusive: check.inclusive,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "max") {
                const tooBig = check.inclusive ? input.data > check.value : input.data >= check.value;
                if (tooBig) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_big,
                        maximum: check.value,
                        type: "number",
                        inclusive: check.inclusive,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "multipleOf") {
                if (floatSafeRemainder(input.data, check.value) !== 0) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.not_multiple_of,
                        multipleOf: check.value,
                        message: check.message
                    });
                    status.dirty();
                }
            } else {
                util.assertNever(check);
            }
        }
        return {
            status: status.value,
            value: input.data
        };
    }
    gte(value, message) {
        return this.setLimit("min", value, true, errorUtil.toString(message));
    }
    gt(value, message) {
        return this.setLimit("min", value, false, errorUtil.toString(message));
    }
    lte(value, message) {
        return this.setLimit("max", value, true, errorUtil.toString(message));
    }
    lt(value, message) {
        return this.setLimit("max", value, false, errorUtil.toString(message));
    }
    setLimit(kind, value, inclusive, message) {
        return new ZodNumber({
            ...this._def,
            checks: [
                ...this._def.checks,
                {
                    kind,
                    value,
                    inclusive,
                    message: errorUtil.toString(message)
                }
            ]
        });
    }
    _addCheck(check) {
        return new ZodNumber({
            ...this._def,
            checks: [
                ...this._def.checks,
                check
            ]
        });
    }
    int(message) {
        return this._addCheck({
            kind: "int",
            message: errorUtil.toString(message)
        });
    }
    positive(message) {
        return this._addCheck({
            kind: "min",
            value: 0,
            inclusive: false,
            message: errorUtil.toString(message)
        });
    }
    negative(message) {
        return this._addCheck({
            kind: "max",
            value: 0,
            inclusive: false,
            message: errorUtil.toString(message)
        });
    }
    nonpositive(message) {
        return this._addCheck({
            kind: "max",
            value: 0,
            inclusive: true,
            message: errorUtil.toString(message)
        });
    }
    nonnegative(message) {
        return this._addCheck({
            kind: "min",
            value: 0,
            inclusive: true,
            message: errorUtil.toString(message)
        });
    }
    multipleOf(value, message) {
        return this._addCheck({
            kind: "multipleOf",
            value,
            message: errorUtil.toString(message)
        });
    }
    get minValue() {
        let min = null;
        for (const ch of this._def.checks){
            if (ch.kind === "min") {
                if (min === null || ch.value > min) min = ch.value;
            }
        }
        return min;
    }
    get maxValue() {
        let max = null;
        for (const ch of this._def.checks){
            if (ch.kind === "max") {
                if (max === null || ch.value < max) max = ch.value;
            }
        }
        return max;
    }
    get isInt() {
        return !!this._def.checks.find((ch)=>ch.kind === "int");
    }
};
ZodNumber.create = (params)=>{
    return new ZodNumber({
        checks: [],
        typeName: ZodFirstPartyTypeKind.ZodNumber,
        ...processCreateParams(params)
    });
};
var ZodBigInt = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.bigint) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.bigint,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
};
ZodBigInt.create = (params)=>{
    return new ZodBigInt({
        typeName: ZodFirstPartyTypeKind.ZodBigInt,
        ...processCreateParams(params)
    });
};
var ZodBoolean = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.boolean) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.boolean,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
};
ZodBoolean.create = (params)=>{
    return new ZodBoolean({
        typeName: ZodFirstPartyTypeKind.ZodBoolean,
        ...processCreateParams(params)
    });
};
var ZodDate = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.date) {
            const ctx2 = this._getOrReturnCtx(input);
            addIssueToContext(ctx2, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.date,
                received: ctx2.parsedType
            });
            return INVALID;
        }
        if (isNaN(input.data.getTime())) {
            const ctx21 = this._getOrReturnCtx(input);
            addIssueToContext(ctx21, {
                code: ZodIssueCode.invalid_date
            });
            return INVALID;
        }
        const status = new ParseStatus();
        let ctx = void 0;
        for (const check of this._def.checks){
            if (check.kind === "min") {
                if (input.data.getTime() < check.value) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_small,
                        message: check.message,
                        inclusive: true,
                        minimum: check.value,
                        type: "date"
                    });
                    status.dirty();
                }
            } else if (check.kind === "max") {
                if (input.data.getTime() > check.value) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_big,
                        message: check.message,
                        inclusive: true,
                        maximum: check.value,
                        type: "date"
                    });
                    status.dirty();
                }
            } else {
                util.assertNever(check);
            }
        }
        return {
            status: status.value,
            value: new Date(input.data.getTime())
        };
    }
    _addCheck(check) {
        return new ZodDate({
            ...this._def,
            checks: [
                ...this._def.checks,
                check
            ]
        });
    }
    min(minDate, message) {
        return this._addCheck({
            kind: "min",
            value: minDate.getTime(),
            message: errorUtil.toString(message)
        });
    }
    max(maxDate, message) {
        return this._addCheck({
            kind: "max",
            value: maxDate.getTime(),
            message: errorUtil.toString(message)
        });
    }
    get minDate() {
        let min = null;
        for (const ch of this._def.checks){
            if (ch.kind === "min") {
                if (min === null || ch.value > min) min = ch.value;
            }
        }
        return min != null ? new Date(min) : null;
    }
    get maxDate() {
        let max = null;
        for (const ch of this._def.checks){
            if (ch.kind === "max") {
                if (max === null || ch.value < max) max = ch.value;
            }
        }
        return max != null ? new Date(max) : null;
    }
};
ZodDate.create = (params)=>{
    return new ZodDate({
        checks: [],
        typeName: ZodFirstPartyTypeKind.ZodDate,
        ...processCreateParams(params)
    });
};
var ZodUndefined = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.undefined) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.undefined,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
};
ZodUndefined.create = (params)=>{
    return new ZodUndefined({
        typeName: ZodFirstPartyTypeKind.ZodUndefined,
        ...processCreateParams(params)
    });
};
var ZodNull = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.null) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.null,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
};
ZodNull.create = (params)=>{
    return new ZodNull({
        typeName: ZodFirstPartyTypeKind.ZodNull,
        ...processCreateParams(params)
    });
};
var ZodAny = class extends ZodType {
    constructor(){
        super(...arguments);
        this._any = true;
    }
    _parse(input) {
        return OK(input.data);
    }
};
ZodAny.create = (params)=>{
    return new ZodAny({
        typeName: ZodFirstPartyTypeKind.ZodAny,
        ...processCreateParams(params)
    });
};
var ZodUnknown = class extends ZodType {
    constructor(){
        super(...arguments);
        this._unknown = true;
    }
    _parse(input) {
        return OK(input.data);
    }
};
ZodUnknown.create = (params)=>{
    return new ZodUnknown({
        typeName: ZodFirstPartyTypeKind.ZodUnknown,
        ...processCreateParams(params)
    });
};
var ZodNever = class extends ZodType {
    _parse(input) {
        const ctx = this._getOrReturnCtx(input);
        addIssueToContext(ctx, {
            code: ZodIssueCode.invalid_type,
            expected: ZodParsedType.never,
            received: ctx.parsedType
        });
        return INVALID;
    }
};
ZodNever.create = (params)=>{
    return new ZodNever({
        typeName: ZodFirstPartyTypeKind.ZodNever,
        ...processCreateParams(params)
    });
};
var ZodVoid = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.undefined) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.void,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
};
ZodVoid.create = (params)=>{
    return new ZodVoid({
        typeName: ZodFirstPartyTypeKind.ZodVoid,
        ...processCreateParams(params)
    });
};
var ZodArray = class extends ZodType {
    _parse(input) {
        const { ctx , status  } = this._processInputParams(input);
        const def = this._def;
        if (ctx.parsedType !== ZodParsedType.array) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.array,
                received: ctx.parsedType
            });
            return INVALID;
        }
        if (def.minLength !== null) {
            if (ctx.data.length < def.minLength.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_small,
                    minimum: def.minLength.value,
                    type: "array",
                    inclusive: true,
                    message: def.minLength.message
                });
                status.dirty();
            }
        }
        if (def.maxLength !== null) {
            if (ctx.data.length > def.maxLength.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_big,
                    maximum: def.maxLength.value,
                    type: "array",
                    inclusive: true,
                    message: def.maxLength.message
                });
                status.dirty();
            }
        }
        if (ctx.common.async) {
            return Promise.all(ctx.data.map((item, i)=>{
                return def.type._parseAsync(new ParseInputLazyPath(ctx, item, ctx.path, i));
            })).then((result2)=>{
                return ParseStatus.mergeArray(status, result2);
            });
        }
        const result = ctx.data.map((item, i)=>{
            return def.type._parseSync(new ParseInputLazyPath(ctx, item, ctx.path, i));
        });
        return ParseStatus.mergeArray(status, result);
    }
    get element() {
        return this._def.type;
    }
    min(minLength, message) {
        return new ZodArray({
            ...this._def,
            minLength: {
                value: minLength,
                message: errorUtil.toString(message)
            }
        });
    }
    max(maxLength, message) {
        return new ZodArray({
            ...this._def,
            maxLength: {
                value: maxLength,
                message: errorUtil.toString(message)
            }
        });
    }
    length(len, message) {
        return this.min(len, message).max(len, message);
    }
    nonempty(message) {
        return this.min(1, message);
    }
};
ZodArray.create = (schema, params)=>{
    return new ZodArray({
        type: schema,
        minLength: null,
        maxLength: null,
        typeName: ZodFirstPartyTypeKind.ZodArray,
        ...processCreateParams(params)
    });
};
var objectUtil;
(function(objectUtil2) {
    objectUtil2.mergeShapes = (first, second)=>{
        return {
            ...first,
            ...second
        };
    };
})(objectUtil || (objectUtil = {}));
var AugmentFactory = (def)=>(augmentation)=>{
        return new ZodObject({
            ...def,
            shape: ()=>({
                    ...def.shape(),
                    ...augmentation
                })
        });
    };
function deepPartialify(schema) {
    if (schema instanceof ZodObject) {
        const newShape = {};
        for(const key in schema.shape){
            const fieldSchema = schema.shape[key];
            newShape[key] = ZodOptional.create(deepPartialify(fieldSchema));
        }
        return new ZodObject({
            ...schema._def,
            shape: ()=>newShape
        });
    } else if (schema instanceof ZodArray) {
        return ZodArray.create(deepPartialify(schema.element));
    } else if (schema instanceof ZodOptional) {
        return ZodOptional.create(deepPartialify(schema.unwrap()));
    } else if (schema instanceof ZodNullable) {
        return ZodNullable.create(deepPartialify(schema.unwrap()));
    } else if (schema instanceof ZodTuple) {
        return ZodTuple.create(schema.items.map((item)=>deepPartialify(item)));
    } else {
        return schema;
    }
}
var ZodObject = class extends ZodType {
    constructor(){
        super(...arguments);
        this._cached = null;
        this.nonstrict = this.passthrough;
        this.augment = AugmentFactory(this._def);
        this.extend = AugmentFactory(this._def);
    }
    _getCached() {
        if (this._cached !== null) return this._cached;
        const shape = this._def.shape();
        const keys = util.objectKeys(shape);
        return this._cached = {
            shape,
            keys
        };
    }
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.object) {
            const ctx2 = this._getOrReturnCtx(input);
            addIssueToContext(ctx2, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.object,
                received: ctx2.parsedType
            });
            return INVALID;
        }
        const { status , ctx  } = this._processInputParams(input);
        const { shape , keys: shapeKeys  } = this._getCached();
        const extraKeys = [];
        for(const key in ctx.data){
            if (!shapeKeys.includes(key)) {
                extraKeys.push(key);
            }
        }
        const pairs = [];
        for (const key1 of shapeKeys){
            const keyValidator = shape[key1];
            const value = ctx.data[key1];
            pairs.push({
                key: {
                    status: "valid",
                    value: key1
                },
                value: keyValidator._parse(new ParseInputLazyPath(ctx, value, ctx.path, key1)),
                alwaysSet: key1 in ctx.data
            });
        }
        if (this._def.catchall instanceof ZodNever) {
            const unknownKeys = this._def.unknownKeys;
            if (unknownKeys === "passthrough") {
                for (const key2 of extraKeys){
                    pairs.push({
                        key: {
                            status: "valid",
                            value: key2
                        },
                        value: {
                            status: "valid",
                            value: ctx.data[key2]
                        }
                    });
                }
            } else if (unknownKeys === "strict") {
                if (extraKeys.length > 0) {
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.unrecognized_keys,
                        keys: extraKeys
                    });
                    status.dirty();
                }
            } else if (unknownKeys === "strip") ;
            else {
                throw new Error(`Internal ZodObject error: invalid unknownKeys value.`);
            }
        } else {
            const catchall = this._def.catchall;
            for (const key3 of extraKeys){
                const value1 = ctx.data[key3];
                pairs.push({
                    key: {
                        status: "valid",
                        value: key3
                    },
                    value: catchall._parse(new ParseInputLazyPath(ctx, value1, ctx.path, key3)),
                    alwaysSet: key3 in ctx.data
                });
            }
        }
        if (ctx.common.async) {
            return Promise.resolve().then(async ()=>{
                const syncPairs = [];
                for (const pair of pairs){
                    const key = await pair.key;
                    syncPairs.push({
                        key,
                        value: await pair.value,
                        alwaysSet: pair.alwaysSet
                    });
                }
                return syncPairs;
            }).then((syncPairs)=>{
                return ParseStatus.mergeObjectSync(status, syncPairs);
            });
        } else {
            return ParseStatus.mergeObjectSync(status, pairs);
        }
    }
    get shape() {
        return this._def.shape();
    }
    strict(message) {
        errorUtil.errToObj;
        return new ZodObject({
            ...this._def,
            unknownKeys: "strict",
            ...message !== void 0 ? {
                errorMap: (issue, ctx)=>{
                    var _a, _b, _c, _d;
                    const defaultError = (_c = (_b = (_a = this._def).errorMap) === null || _b === void 0 ? void 0 : _b.call(_a, issue, ctx).message) !== null && _c !== void 0 ? _c : ctx.defaultError;
                    if (issue.code === "unrecognized_keys") return {
                        message: (_d = errorUtil.errToObj(message).message) !== null && _d !== void 0 ? _d : defaultError
                    };
                    return {
                        message: defaultError
                    };
                }
            } : {}
        });
    }
    strip() {
        return new ZodObject({
            ...this._def,
            unknownKeys: "strip"
        });
    }
    passthrough() {
        return new ZodObject({
            ...this._def,
            unknownKeys: "passthrough"
        });
    }
    setKey(key, schema) {
        return this.augment({
            [key]: schema
        });
    }
    merge(merging) {
        const merged = new ZodObject({
            unknownKeys: merging._def.unknownKeys,
            catchall: merging._def.catchall,
            shape: ()=>objectUtil.mergeShapes(this._def.shape(), merging._def.shape()),
            typeName: ZodFirstPartyTypeKind.ZodObject
        });
        return merged;
    }
    catchall(index) {
        return new ZodObject({
            ...this._def,
            catchall: index
        });
    }
    pick(mask) {
        const shape = {};
        util.objectKeys(mask).map((key)=>{
            if (this.shape[key]) shape[key] = this.shape[key];
        });
        return new ZodObject({
            ...this._def,
            shape: ()=>shape
        });
    }
    omit(mask) {
        const shape = {};
        util.objectKeys(this.shape).map((key)=>{
            if (util.objectKeys(mask).indexOf(key) === -1) {
                shape[key] = this.shape[key];
            }
        });
        return new ZodObject({
            ...this._def,
            shape: ()=>shape
        });
    }
    deepPartial() {
        return deepPartialify(this);
    }
    partial(mask) {
        const newShape = {};
        if (mask) {
            util.objectKeys(this.shape).map((key)=>{
                if (util.objectKeys(mask).indexOf(key) === -1) {
                    newShape[key] = this.shape[key];
                } else {
                    newShape[key] = this.shape[key].optional();
                }
            });
            return new ZodObject({
                ...this._def,
                shape: ()=>newShape
            });
        } else {
            for(const key in this.shape){
                const fieldSchema = this.shape[key];
                newShape[key] = fieldSchema.optional();
            }
        }
        return new ZodObject({
            ...this._def,
            shape: ()=>newShape
        });
    }
    required() {
        const newShape = {};
        for(const key in this.shape){
            const fieldSchema = this.shape[key];
            let newField = fieldSchema;
            while(newField instanceof ZodOptional){
                newField = newField._def.innerType;
            }
            newShape[key] = newField;
        }
        return new ZodObject({
            ...this._def,
            shape: ()=>newShape
        });
    }
    keyof() {
        return createZodEnum(util.objectKeys(this.shape));
    }
};
ZodObject.create = (shape, params)=>{
    return new ZodObject({
        shape: ()=>shape,
        unknownKeys: "strip",
        catchall: ZodNever.create(),
        typeName: ZodFirstPartyTypeKind.ZodObject,
        ...processCreateParams(params)
    });
};
ZodObject.strictCreate = (shape, params)=>{
    return new ZodObject({
        shape: ()=>shape,
        unknownKeys: "strict",
        catchall: ZodNever.create(),
        typeName: ZodFirstPartyTypeKind.ZodObject,
        ...processCreateParams(params)
    });
};
ZodObject.lazycreate = (shape, params)=>{
    return new ZodObject({
        shape,
        unknownKeys: "strip",
        catchall: ZodNever.create(),
        typeName: ZodFirstPartyTypeKind.ZodObject,
        ...processCreateParams(params)
    });
};
var ZodUnion = class extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        const options = this._def.options;
        function handleResults(results) {
            for (const result of results){
                if (result.result.status === "valid") {
                    return result.result;
                }
            }
            for (const result1 of results){
                if (result1.result.status === "dirty") {
                    ctx.common.issues.push(...result1.ctx.common.issues);
                    return result1.result;
                }
            }
            const unionErrors = results.map((result)=>new ZodError(result.ctx.common.issues));
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_union,
                unionErrors
            });
            return INVALID;
        }
        if (ctx.common.async) {
            return Promise.all(options.map(async (option)=>{
                const childCtx = {
                    ...ctx,
                    common: {
                        ...ctx.common,
                        issues: []
                    },
                    parent: null
                };
                return {
                    result: await option._parseAsync({
                        data: ctx.data,
                        path: ctx.path,
                        parent: childCtx
                    }),
                    ctx: childCtx
                };
            })).then(handleResults);
        } else {
            let dirty = void 0;
            const issues = [];
            for (const option of options){
                const childCtx = {
                    ...ctx,
                    common: {
                        ...ctx.common,
                        issues: []
                    },
                    parent: null
                };
                const result = option._parseSync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: childCtx
                });
                if (result.status === "valid") {
                    return result;
                } else if (result.status === "dirty" && !dirty) {
                    dirty = {
                        result,
                        ctx: childCtx
                    };
                }
                if (childCtx.common.issues.length) {
                    issues.push(childCtx.common.issues);
                }
            }
            if (dirty) {
                ctx.common.issues.push(...dirty.ctx.common.issues);
                return dirty.result;
            }
            const unionErrors = issues.map((issues2)=>new ZodError(issues2));
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_union,
                unionErrors
            });
            return INVALID;
        }
    }
    get options() {
        return this._def.options;
    }
};
ZodUnion.create = (types, params)=>{
    return new ZodUnion({
        options: types,
        typeName: ZodFirstPartyTypeKind.ZodUnion,
        ...processCreateParams(params)
    });
};
var ZodDiscriminatedUnion = class extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.object) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.object,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const discriminator = this.discriminator;
        const discriminatorValue = ctx.data[discriminator];
        const option = this.options.get(discriminatorValue);
        if (!option) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_union_discriminator,
                options: this.validDiscriminatorValues,
                path: [
                    discriminator
                ]
            });
            return INVALID;
        }
        if (ctx.common.async) {
            return option._parseAsync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            });
        } else {
            return option._parseSync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            });
        }
    }
    get discriminator() {
        return this._def.discriminator;
    }
    get validDiscriminatorValues() {
        return Array.from(this.options.keys());
    }
    get options() {
        return this._def.options;
    }
    static create(discriminator, types, params) {
        const options = /* @__PURE__ */ new Map();
        try {
            types.forEach((type)=>{
                const discriminatorValue = type.shape[discriminator].value;
                options.set(discriminatorValue, type);
            });
        } catch (e) {
            throw new Error("The discriminator value could not be extracted from all the provided schemas");
        }
        if (options.size !== types.length) {
            throw new Error("Some of the discriminator values are not unique");
        }
        return new ZodDiscriminatedUnion({
            typeName: ZodFirstPartyTypeKind.ZodDiscriminatedUnion,
            discriminator,
            options,
            ...processCreateParams(params)
        });
    }
};
function mergeValues(a, b) {
    const aType = getParsedType(a);
    const bType = getParsedType(b);
    if (a === b) {
        return {
            valid: true,
            data: a
        };
    } else if (aType === ZodParsedType.object && bType === ZodParsedType.object) {
        const bKeys = util.objectKeys(b);
        const sharedKeys = util.objectKeys(a).filter((key)=>bKeys.indexOf(key) !== -1);
        const newObj = {
            ...a,
            ...b
        };
        for (const key of sharedKeys){
            const sharedValue = mergeValues(a[key], b[key]);
            if (!sharedValue.valid) {
                return {
                    valid: false
                };
            }
            newObj[key] = sharedValue.data;
        }
        return {
            valid: true,
            data: newObj
        };
    } else if (aType === ZodParsedType.array && bType === ZodParsedType.array) {
        if (a.length !== b.length) {
            return {
                valid: false
            };
        }
        const newArray = [];
        for(let index = 0; index < a.length; index++){
            const itemA = a[index];
            const itemB = b[index];
            const sharedValue1 = mergeValues(itemA, itemB);
            if (!sharedValue1.valid) {
                return {
                    valid: false
                };
            }
            newArray.push(sharedValue1.data);
        }
        return {
            valid: true,
            data: newArray
        };
    } else if (aType === ZodParsedType.date && bType === ZodParsedType.date && +a === +b) {
        return {
            valid: true,
            data: a
        };
    } else {
        return {
            valid: false
        };
    }
}
var ZodIntersection = class extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        const handleParsed = (parsedLeft, parsedRight)=>{
            if (isAborted(parsedLeft) || isAborted(parsedRight)) {
                return INVALID;
            }
            const merged = mergeValues(parsedLeft.value, parsedRight.value);
            if (!merged.valid) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.invalid_intersection_types
                });
                return INVALID;
            }
            if (isDirty(parsedLeft) || isDirty(parsedRight)) {
                status.dirty();
            }
            return {
                status: status.value,
                value: merged.data
            };
        };
        if (ctx.common.async) {
            return Promise.all([
                this._def.left._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }),
                this._def.right._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                })
            ]).then(([left, right])=>handleParsed(left, right));
        } else {
            return handleParsed(this._def.left._parseSync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            }), this._def.right._parseSync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            }));
        }
    }
};
ZodIntersection.create = (left, right, params)=>{
    return new ZodIntersection({
        left,
        right,
        typeName: ZodFirstPartyTypeKind.ZodIntersection,
        ...processCreateParams(params)
    });
};
var ZodTuple = class extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.array) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.array,
                received: ctx.parsedType
            });
            return INVALID;
        }
        if (ctx.data.length < this._def.items.length) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.too_small,
                minimum: this._def.items.length,
                inclusive: true,
                type: "array"
            });
            return INVALID;
        }
        const rest = this._def.rest;
        if (!rest && ctx.data.length > this._def.items.length) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.too_big,
                maximum: this._def.items.length,
                inclusive: true,
                type: "array"
            });
            status.dirty();
        }
        const items = ctx.data.map((item, itemIndex)=>{
            const schema = this._def.items[itemIndex] || this._def.rest;
            if (!schema) return null;
            return schema._parse(new ParseInputLazyPath(ctx, item, ctx.path, itemIndex));
        }).filter((x)=>!!x);
        if (ctx.common.async) {
            return Promise.all(items).then((results)=>{
                return ParseStatus.mergeArray(status, results);
            });
        } else {
            return ParseStatus.mergeArray(status, items);
        }
    }
    get items() {
        return this._def.items;
    }
    rest(rest) {
        return new ZodTuple({
            ...this._def,
            rest
        });
    }
};
ZodTuple.create = (schemas, params)=>{
    return new ZodTuple({
        items: schemas,
        typeName: ZodFirstPartyTypeKind.ZodTuple,
        rest: null,
        ...processCreateParams(params)
    });
};
var ZodRecord = class extends ZodType {
    get keySchema() {
        return this._def.keyType;
    }
    get valueSchema() {
        return this._def.valueType;
    }
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.object) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.object,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const pairs = [];
        const keyType = this._def.keyType;
        const valueType = this._def.valueType;
        for(const key in ctx.data){
            pairs.push({
                key: keyType._parse(new ParseInputLazyPath(ctx, key, ctx.path, key)),
                value: valueType._parse(new ParseInputLazyPath(ctx, ctx.data[key], ctx.path, key))
            });
        }
        if (ctx.common.async) {
            return ParseStatus.mergeObjectAsync(status, pairs);
        } else {
            return ParseStatus.mergeObjectSync(status, pairs);
        }
    }
    get element() {
        return this._def.valueType;
    }
    static create(first, second, third) {
        if (second instanceof ZodType) {
            return new ZodRecord({
                keyType: first,
                valueType: second,
                typeName: ZodFirstPartyTypeKind.ZodRecord,
                ...processCreateParams(third)
            });
        }
        return new ZodRecord({
            keyType: ZodString.create(),
            valueType: first,
            typeName: ZodFirstPartyTypeKind.ZodRecord,
            ...processCreateParams(second)
        });
    }
};
var ZodMap = class extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.map) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.map,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const keyType = this._def.keyType;
        const valueType = this._def.valueType;
        const pairs = [
            ...ctx.data.entries()
        ].map(([key, value], index)=>{
            return {
                key: keyType._parse(new ParseInputLazyPath(ctx, key, ctx.path, [
                    index,
                    "key"
                ])),
                value: valueType._parse(new ParseInputLazyPath(ctx, value, ctx.path, [
                    index,
                    "value"
                ]))
            };
        });
        if (ctx.common.async) {
            const finalMap = /* @__PURE__ */ new Map();
            return Promise.resolve().then(async ()=>{
                for (const pair of pairs){
                    const key = await pair.key;
                    const value = await pair.value;
                    if (key.status === "aborted" || value.status === "aborted") {
                        return INVALID;
                    }
                    if (key.status === "dirty" || value.status === "dirty") {
                        status.dirty();
                    }
                    finalMap.set(key.value, value.value);
                }
                return {
                    status: status.value,
                    value: finalMap
                };
            });
        } else {
            const finalMap1 = /* @__PURE__ */ new Map();
            for (const pair of pairs){
                const key = pair.key;
                const value = pair.value;
                if (key.status === "aborted" || value.status === "aborted") {
                    return INVALID;
                }
                if (key.status === "dirty" || value.status === "dirty") {
                    status.dirty();
                }
                finalMap1.set(key.value, value.value);
            }
            return {
                status: status.value,
                value: finalMap1
            };
        }
    }
};
ZodMap.create = (keyType, valueType, params)=>{
    return new ZodMap({
        valueType,
        keyType,
        typeName: ZodFirstPartyTypeKind.ZodMap,
        ...processCreateParams(params)
    });
};
var ZodSet = class extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.set) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.set,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const def = this._def;
        if (def.minSize !== null) {
            if (ctx.data.size < def.minSize.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_small,
                    minimum: def.minSize.value,
                    type: "set",
                    inclusive: true,
                    message: def.minSize.message
                });
                status.dirty();
            }
        }
        if (def.maxSize !== null) {
            if (ctx.data.size > def.maxSize.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_big,
                    maximum: def.maxSize.value,
                    type: "set",
                    inclusive: true,
                    message: def.maxSize.message
                });
                status.dirty();
            }
        }
        const valueType = this._def.valueType;
        function finalizeSet(elements2) {
            const parsedSet = /* @__PURE__ */ new Set();
            for (const element of elements2){
                if (element.status === "aborted") return INVALID;
                if (element.status === "dirty") status.dirty();
                parsedSet.add(element.value);
            }
            return {
                status: status.value,
                value: parsedSet
            };
        }
        const elements = [
            ...ctx.data.values()
        ].map((item, i)=>valueType._parse(new ParseInputLazyPath(ctx, item, ctx.path, i)));
        if (ctx.common.async) {
            return Promise.all(elements).then((elements2)=>finalizeSet(elements2));
        } else {
            return finalizeSet(elements);
        }
    }
    min(minSize, message) {
        return new ZodSet({
            ...this._def,
            minSize: {
                value: minSize,
                message: errorUtil.toString(message)
            }
        });
    }
    max(maxSize, message) {
        return new ZodSet({
            ...this._def,
            maxSize: {
                value: maxSize,
                message: errorUtil.toString(message)
            }
        });
    }
    size(size, message) {
        return this.min(size, message).max(size, message);
    }
    nonempty(message) {
        return this.min(1, message);
    }
};
ZodSet.create = (valueType, params)=>{
    return new ZodSet({
        valueType,
        minSize: null,
        maxSize: null,
        typeName: ZodFirstPartyTypeKind.ZodSet,
        ...processCreateParams(params)
    });
};
var ZodFunction = class extends ZodType {
    constructor(){
        super(...arguments);
        this.validate = this.implement;
    }
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.function) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.function,
                received: ctx.parsedType
            });
            return INVALID;
        }
        function makeArgsIssue(args, error) {
            return makeIssue({
                data: args,
                path: ctx.path,
                errorMaps: [
                    ctx.common.contextualErrorMap,
                    ctx.schemaErrorMap,
                    getErrorMap(),
                    defaultErrorMap
                ].filter((x)=>!!x),
                issueData: {
                    code: ZodIssueCode.invalid_arguments,
                    argumentsError: error
                }
            });
        }
        function makeReturnsIssue(returns, error) {
            return makeIssue({
                data: returns,
                path: ctx.path,
                errorMaps: [
                    ctx.common.contextualErrorMap,
                    ctx.schemaErrorMap,
                    getErrorMap(),
                    defaultErrorMap
                ].filter((x)=>!!x),
                issueData: {
                    code: ZodIssueCode.invalid_return_type,
                    returnTypeError: error
                }
            });
        }
        const params = {
            errorMap: ctx.common.contextualErrorMap
        };
        const fn = ctx.data;
        if (this._def.returns instanceof ZodPromise) {
            return OK(async (...args)=>{
                const error = new ZodError([]);
                const parsedArgs = await this._def.args.parseAsync(args, params).catch((e)=>{
                    error.addIssue(makeArgsIssue(args, e));
                    throw error;
                });
                const result = await fn(...parsedArgs);
                const parsedReturns = await this._def.returns._def.type.parseAsync(result, params).catch((e)=>{
                    error.addIssue(makeReturnsIssue(result, e));
                    throw error;
                });
                return parsedReturns;
            });
        } else {
            return OK((...args)=>{
                const parsedArgs = this._def.args.safeParse(args, params);
                if (!parsedArgs.success) {
                    throw new ZodError([
                        makeArgsIssue(args, parsedArgs.error)
                    ]);
                }
                const result = fn(...parsedArgs.data);
                const parsedReturns = this._def.returns.safeParse(result, params);
                if (!parsedReturns.success) {
                    throw new ZodError([
                        makeReturnsIssue(result, parsedReturns.error)
                    ]);
                }
                return parsedReturns.data;
            });
        }
    }
    parameters() {
        return this._def.args;
    }
    returnType() {
        return this._def.returns;
    }
    args(...items) {
        return new ZodFunction({
            ...this._def,
            args: ZodTuple.create(items).rest(ZodUnknown.create())
        });
    }
    returns(returnType) {
        return new ZodFunction({
            ...this._def,
            returns: returnType
        });
    }
    implement(func) {
        const validatedFunc = this.parse(func);
        return validatedFunc;
    }
    strictImplement(func) {
        const validatedFunc = this.parse(func);
        return validatedFunc;
    }
};
ZodFunction.create = (args, returns, params)=>{
    return new ZodFunction({
        args: args ? args.rest(ZodUnknown.create()) : ZodTuple.create([]).rest(ZodUnknown.create()),
        returns: returns || ZodUnknown.create(),
        typeName: ZodFirstPartyTypeKind.ZodFunction,
        ...processCreateParams(params)
    });
};
var ZodLazy = class extends ZodType {
    get schema() {
        return this._def.getter();
    }
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        const lazySchema = this._def.getter();
        return lazySchema._parse({
            data: ctx.data,
            path: ctx.path,
            parent: ctx
        });
    }
};
ZodLazy.create = (getter, params)=>{
    return new ZodLazy({
        getter,
        typeName: ZodFirstPartyTypeKind.ZodLazy,
        ...processCreateParams(params)
    });
};
var ZodLiteral = class extends ZodType {
    _parse(input) {
        if (input.data !== this._def.value) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_literal,
                expected: this._def.value
            });
            return INVALID;
        }
        return {
            status: "valid",
            value: input.data
        };
    }
    get value() {
        return this._def.value;
    }
};
ZodLiteral.create = (value, params)=>{
    return new ZodLiteral({
        value,
        typeName: ZodFirstPartyTypeKind.ZodLiteral,
        ...processCreateParams(params)
    });
};
function createZodEnum(values, params) {
    return new ZodEnum({
        values,
        typeName: ZodFirstPartyTypeKind.ZodEnum,
        ...processCreateParams(params)
    });
}
var ZodEnum = class extends ZodType {
    _parse(input) {
        if (typeof input.data !== "string") {
            const ctx = this._getOrReturnCtx(input);
            const expectedValues = this._def.values;
            addIssueToContext(ctx, {
                expected: util.joinValues(expectedValues),
                received: ctx.parsedType,
                code: ZodIssueCode.invalid_type
            });
            return INVALID;
        }
        if (this._def.values.indexOf(input.data) === -1) {
            const ctx1 = this._getOrReturnCtx(input);
            const expectedValues1 = this._def.values;
            addIssueToContext(ctx1, {
                received: ctx1.data,
                code: ZodIssueCode.invalid_enum_value,
                options: expectedValues1
            });
            return INVALID;
        }
        return OK(input.data);
    }
    get options() {
        return this._def.values;
    }
    get enum() {
        const enumValues = {};
        for (const val of this._def.values){
            enumValues[val] = val;
        }
        return enumValues;
    }
    get Values() {
        const enumValues = {};
        for (const val of this._def.values){
            enumValues[val] = val;
        }
        return enumValues;
    }
    get Enum() {
        const enumValues = {};
        for (const val of this._def.values){
            enumValues[val] = val;
        }
        return enumValues;
    }
};
ZodEnum.create = createZodEnum;
var ZodNativeEnum = class extends ZodType {
    _parse(input) {
        const nativeEnumValues = util.getValidEnumValues(this._def.values);
        const ctx = this._getOrReturnCtx(input);
        if (ctx.parsedType !== ZodParsedType.string && ctx.parsedType !== ZodParsedType.number) {
            const expectedValues = util.objectValues(nativeEnumValues);
            addIssueToContext(ctx, {
                expected: util.joinValues(expectedValues),
                received: ctx.parsedType,
                code: ZodIssueCode.invalid_type
            });
            return INVALID;
        }
        if (nativeEnumValues.indexOf(input.data) === -1) {
            const expectedValues1 = util.objectValues(nativeEnumValues);
            addIssueToContext(ctx, {
                received: ctx.data,
                code: ZodIssueCode.invalid_enum_value,
                options: expectedValues1
            });
            return INVALID;
        }
        return OK(input.data);
    }
    get enum() {
        return this._def.values;
    }
};
ZodNativeEnum.create = (values, params)=>{
    return new ZodNativeEnum({
        values,
        typeName: ZodFirstPartyTypeKind.ZodNativeEnum,
        ...processCreateParams(params)
    });
};
var ZodPromise = class extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.promise && ctx.common.async === false) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.promise,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const promisified = ctx.parsedType === ZodParsedType.promise ? ctx.data : Promise.resolve(ctx.data);
        return OK(promisified.then((data)=>{
            return this._def.type.parseAsync(data, {
                path: ctx.path,
                errorMap: ctx.common.contextualErrorMap
            });
        }));
    }
};
ZodPromise.create = (schema, params)=>{
    return new ZodPromise({
        type: schema,
        typeName: ZodFirstPartyTypeKind.ZodPromise,
        ...processCreateParams(params)
    });
};
var ZodEffects = class extends ZodType {
    innerType() {
        return this._def.schema;
    }
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        const effect = this._def.effect || null;
        if (effect.type === "preprocess") {
            const processed = effect.transform(ctx.data);
            if (ctx.common.async) {
                return Promise.resolve(processed).then((processed2)=>{
                    return this._def.schema._parseAsync({
                        data: processed2,
                        path: ctx.path,
                        parent: ctx
                    });
                });
            } else {
                return this._def.schema._parseSync({
                    data: processed,
                    path: ctx.path,
                    parent: ctx
                });
            }
        }
        const checkCtx = {
            addIssue: (arg)=>{
                addIssueToContext(ctx, arg);
                if (arg.fatal) {
                    status.abort();
                } else {
                    status.dirty();
                }
            },
            get path () {
                return ctx.path;
            }
        };
        checkCtx.addIssue = checkCtx.addIssue.bind(checkCtx);
        if (effect.type === "refinement") {
            const executeRefinement = (acc)=>{
                const result = effect.refinement(acc, checkCtx);
                if (ctx.common.async) {
                    return Promise.resolve(result);
                }
                if (result instanceof Promise) {
                    throw new Error("Async refinement encountered during synchronous parse operation. Use .parseAsync instead.");
                }
                return acc;
            };
            if (ctx.common.async === false) {
                const inner = this._def.schema._parseSync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                });
                if (inner.status === "aborted") return INVALID;
                if (inner.status === "dirty") status.dirty();
                executeRefinement(inner.value);
                return {
                    status: status.value,
                    value: inner.value
                };
            } else {
                return this._def.schema._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }).then((inner)=>{
                    if (inner.status === "aborted") return INVALID;
                    if (inner.status === "dirty") status.dirty();
                    return executeRefinement(inner.value).then(()=>{
                        return {
                            status: status.value,
                            value: inner.value
                        };
                    });
                });
            }
        }
        if (effect.type === "transform") {
            if (ctx.common.async === false) {
                const base = this._def.schema._parseSync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                });
                if (!isValid(base)) return base;
                const result = effect.transform(base.value, checkCtx);
                if (result instanceof Promise) {
                    throw new Error(`Asynchronous transform encountered during synchronous parse operation. Use .parseAsync instead.`);
                }
                return {
                    status: status.value,
                    value: result
                };
            } else {
                return this._def.schema._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }).then((base)=>{
                    if (!isValid(base)) return base;
                    return Promise.resolve(effect.transform(base.value, checkCtx)).then((result)=>({
                            status: status.value,
                            value: result
                        }));
                });
            }
        }
        util.assertNever(effect);
    }
};
ZodEffects.create = (schema, effect, params)=>{
    return new ZodEffects({
        schema,
        typeName: ZodFirstPartyTypeKind.ZodEffects,
        effect,
        ...processCreateParams(params)
    });
};
ZodEffects.createWithPreprocess = (preprocess, schema, params)=>{
    return new ZodEffects({
        schema,
        effect: {
            type: "preprocess",
            transform: preprocess
        },
        typeName: ZodFirstPartyTypeKind.ZodEffects,
        ...processCreateParams(params)
    });
};
var ZodOptional = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType === ZodParsedType.undefined) {
            return OK(void 0);
        }
        return this._def.innerType._parse(input);
    }
    unwrap() {
        return this._def.innerType;
    }
};
ZodOptional.create = (type, params)=>{
    return new ZodOptional({
        innerType: type,
        typeName: ZodFirstPartyTypeKind.ZodOptional,
        ...processCreateParams(params)
    });
};
var ZodNullable = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType === ZodParsedType.null) {
            return OK(null);
        }
        return this._def.innerType._parse(input);
    }
    unwrap() {
        return this._def.innerType;
    }
};
ZodNullable.create = (type, params)=>{
    return new ZodNullable({
        innerType: type,
        typeName: ZodFirstPartyTypeKind.ZodNullable,
        ...processCreateParams(params)
    });
};
var ZodDefault = class extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        let data = ctx.data;
        if (ctx.parsedType === ZodParsedType.undefined) {
            data = this._def.defaultValue();
        }
        return this._def.innerType._parse({
            data,
            path: ctx.path,
            parent: ctx
        });
    }
    removeDefault() {
        return this._def.innerType;
    }
};
ZodDefault.create = (type, params)=>{
    return new ZodOptional({
        innerType: type,
        typeName: ZodFirstPartyTypeKind.ZodOptional,
        ...processCreateParams(params)
    });
};
var ZodNaN = class extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.nan) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.nan,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return {
            status: "valid",
            value: input.data
        };
    }
};
ZodNaN.create = (params)=>{
    return new ZodNaN({
        typeName: ZodFirstPartyTypeKind.ZodNaN,
        ...processCreateParams(params)
    });
};
var custom = (check, params = {}, fatal)=>{
    if (check) return ZodAny.create().superRefine((data, ctx)=>{
        if (!check(data)) {
            const p = typeof params === "function" ? params(data) : params;
            const p2 = typeof p === "string" ? {
                message: p
            } : p;
            ctx.addIssue({
                code: "custom",
                ...p2,
                fatal
            });
        }
    });
    return ZodAny.create();
};
var late = {
    object: ZodObject.lazycreate
};
var ZodFirstPartyTypeKind;
(function(ZodFirstPartyTypeKind2) {
    ZodFirstPartyTypeKind2["ZodString"] = "ZodString";
    ZodFirstPartyTypeKind2["ZodNumber"] = "ZodNumber";
    ZodFirstPartyTypeKind2["ZodNaN"] = "ZodNaN";
    ZodFirstPartyTypeKind2["ZodBigInt"] = "ZodBigInt";
    ZodFirstPartyTypeKind2["ZodBoolean"] = "ZodBoolean";
    ZodFirstPartyTypeKind2["ZodDate"] = "ZodDate";
    ZodFirstPartyTypeKind2["ZodUndefined"] = "ZodUndefined";
    ZodFirstPartyTypeKind2["ZodNull"] = "ZodNull";
    ZodFirstPartyTypeKind2["ZodAny"] = "ZodAny";
    ZodFirstPartyTypeKind2["ZodUnknown"] = "ZodUnknown";
    ZodFirstPartyTypeKind2["ZodNever"] = "ZodNever";
    ZodFirstPartyTypeKind2["ZodVoid"] = "ZodVoid";
    ZodFirstPartyTypeKind2["ZodArray"] = "ZodArray";
    ZodFirstPartyTypeKind2["ZodObject"] = "ZodObject";
    ZodFirstPartyTypeKind2["ZodUnion"] = "ZodUnion";
    ZodFirstPartyTypeKind2["ZodDiscriminatedUnion"] = "ZodDiscriminatedUnion";
    ZodFirstPartyTypeKind2["ZodIntersection"] = "ZodIntersection";
    ZodFirstPartyTypeKind2["ZodTuple"] = "ZodTuple";
    ZodFirstPartyTypeKind2["ZodRecord"] = "ZodRecord";
    ZodFirstPartyTypeKind2["ZodMap"] = "ZodMap";
    ZodFirstPartyTypeKind2["ZodSet"] = "ZodSet";
    ZodFirstPartyTypeKind2["ZodFunction"] = "ZodFunction";
    ZodFirstPartyTypeKind2["ZodLazy"] = "ZodLazy";
    ZodFirstPartyTypeKind2["ZodLiteral"] = "ZodLiteral";
    ZodFirstPartyTypeKind2["ZodEnum"] = "ZodEnum";
    ZodFirstPartyTypeKind2["ZodEffects"] = "ZodEffects";
    ZodFirstPartyTypeKind2["ZodNativeEnum"] = "ZodNativeEnum";
    ZodFirstPartyTypeKind2["ZodOptional"] = "ZodOptional";
    ZodFirstPartyTypeKind2["ZodNullable"] = "ZodNullable";
    ZodFirstPartyTypeKind2["ZodDefault"] = "ZodDefault";
    ZodFirstPartyTypeKind2["ZodPromise"] = "ZodPromise";
})(ZodFirstPartyTypeKind || (ZodFirstPartyTypeKind = {}));
var instanceOfType = (cls, params = {
    message: `Input not instance of ${cls.name}`
})=>custom((data)=>data instanceof cls, params, true);
var stringType = ZodString.create;
var numberType = ZodNumber.create;
var nanType = ZodNaN.create;
var bigIntType = ZodBigInt.create;
var booleanType = ZodBoolean.create;
var dateType = ZodDate.create;
var undefinedType = ZodUndefined.create;
var nullType = ZodNull.create;
var anyType = ZodAny.create;
var unknownType = ZodUnknown.create;
var neverType = ZodNever.create;
var voidType = ZodVoid.create;
var arrayType = ZodArray.create;
var objectType = ZodObject.create;
var strictObjectType = ZodObject.strictCreate;
var unionType = ZodUnion.create;
var discriminatedUnionType = ZodDiscriminatedUnion.create;
var intersectionType = ZodIntersection.create;
var tupleType = ZodTuple.create;
var recordType = ZodRecord.create;
var mapType = ZodMap.create;
var setType = ZodSet.create;
var functionType = ZodFunction.create;
var lazyType = ZodLazy.create;
var literalType = ZodLiteral.create;
var enumType = ZodEnum.create;
var nativeEnumType = ZodNativeEnum.create;
var promiseType = ZodPromise.create;
var effectsType = ZodEffects.create;
var optionalType = ZodOptional.create;
var nullableType = ZodNullable.create;
var preprocessType = ZodEffects.createWithPreprocess;
var ostring = ()=>stringType().optional();
var onumber = ()=>numberType().optional();
var oboolean = ()=>booleanType().optional();
var mod = /* @__PURE__ */ Object.freeze({
    __proto__: null,
    getParsedType,
    ZodParsedType,
    makeIssue,
    EMPTY_PATH,
    addIssueToContext,
    ParseStatus,
    INVALID,
    DIRTY,
    OK,
    isAborted,
    isDirty,
    isValid,
    isAsync,
    jsonStringifyReplacer,
    ZodType,
    ZodString,
    ZodNumber,
    ZodBigInt,
    ZodBoolean,
    ZodDate,
    ZodUndefined,
    ZodNull,
    ZodAny,
    ZodUnknown,
    ZodNever,
    ZodVoid,
    ZodArray,
    get objectUtil () {
        return objectUtil;
    },
    ZodObject,
    ZodUnion,
    ZodDiscriminatedUnion,
    ZodIntersection,
    ZodTuple,
    ZodRecord,
    ZodMap,
    ZodSet,
    ZodFunction,
    ZodLazy,
    ZodLiteral,
    ZodEnum,
    ZodNativeEnum,
    ZodPromise,
    ZodEffects,
    ZodTransformer: ZodEffects,
    ZodOptional,
    ZodNullable,
    ZodDefault,
    ZodNaN,
    custom,
    Schema: ZodType,
    ZodSchema: ZodType,
    late,
    get ZodFirstPartyTypeKind () {
        return ZodFirstPartyTypeKind;
    },
    any: anyType,
    array: arrayType,
    bigint: bigIntType,
    boolean: booleanType,
    date: dateType,
    discriminatedUnion: discriminatedUnionType,
    effect: effectsType,
    enum: enumType,
    function: functionType,
    instanceof: instanceOfType,
    intersection: intersectionType,
    lazy: lazyType,
    literal: literalType,
    map: mapType,
    nan: nanType,
    nativeEnum: nativeEnumType,
    never: neverType,
    null: nullType,
    nullable: nullableType,
    number: numberType,
    object: objectType,
    oboolean,
    onumber,
    optional: optionalType,
    ostring,
    preprocess: preprocessType,
    promise: promiseType,
    record: recordType,
    set: setType,
    strictObject: strictObjectType,
    string: stringType,
    transformer: effectsType,
    tuple: tupleType,
    undefined: undefinedType,
    union: unionType,
    unknown: unknownType,
    void: voidType,
    ZodIssueCode,
    quotelessJson,
    ZodError,
    defaultErrorMap,
    setErrorMap,
    getErrorMap
});
function hasComma(num) {
    return num === 0 ? "" : ",";
}
function canonify(object) {
    if (object === null || typeof object === "undefined" || typeof object === "boolean" || typeof object === "number" || typeof object === "string") {
        return JSON.stringify(object);
    }
    if (typeof object === "bigint") {
        throw new TypeError("BigInt value can't be serialized in JSON");
    }
    if (typeof object === "function" || typeof object === "symbol") {
        return canonify(void 0);
    }
    if (object.toJSON instanceof Function) {
        return canonify(object.toJSON());
    }
    if (Array.isArray(object)) {
        const values2 = object.reduce((t, cv, ci)=>{
            const value = cv === void 0 || typeof cv === "symbol" || typeof cv === "function" ? null : cv;
            return `${t}${hasComma(ci)}${canonify(value)}`;
        }, "");
        return `[${values2}]`;
    }
    const values = Object.keys(object).sort().reduce((t, cv)=>{
        if (object[cv] === void 0 || typeof object[cv] === "symbol" || typeof object[cv] === "function") {
            return t;
        }
        return `${t}${hasComma(t.length)}${canonify(cv)}:${canonify(object[cv])}`;
    }, "");
    return `{${values}}`;
}
var { base58check  } = require_lib();
var ID_PREFIX = "truestamp";
var ID_SEPARATOR = "-";
var REGEX_ULID = /^[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$/;
var idBinaryStruct = use(struct({
    test: boolean,
    timestamp: bigUint64,
    ulid: string(latin1, uint8),
    siphash: bytes(uint8)
}));
var SipHashKeyStruct = mod.instanceof(Uint8Array).refine((val)=>val.length === 8, {
    message: `SipHash key must be ${import_halfsiphash.KEY_LENGTH} bytes`
});
var SipHashHashStruct = mod.instanceof(Uint8Array).refine((value)=>{
    return value.length === 4;
}, {
    message: `SipHash hash should be 4 bytes`
});
var PayloadStruct = mod.object({
    test: mod.boolean().default(false),
    timestamp: mod.string(),
    ulid: mod.string().regex(REGEX_ULID)
});
function hashPayload(payload, key) {
    if (!PayloadStruct.safeParse(payload).success) {
        throw new Error(`invalid payload`);
    }
    if (!SipHashKeyStruct.safeParse(key).success) {
        throw new Error(`SipHash key must be ${import_halfsiphash.KEY_LENGTH} bytes`);
    }
    const canonicalizedPayload = canonify(payload);
    if (!canonicalizedPayload) {
        throw new Error(`payload canonicalization error`);
    }
    const hash = (0, import_halfsiphash.halfSipHash)(key, new TextEncoder().encode(canonicalizedPayload));
    if (!SipHashHashStruct.safeParse(hash).success) {
        throw new Error(`invalid SipHash hash generated`);
    }
    return hash;
}
function parseId(id) {
    const splitId = id.split(ID_SEPARATOR);
    if (splitId.length !== 1 && splitId.length !== 2) {
        throw new Error(`string format error`);
    }
    const idString = splitId.length === 2 ? splitId[1] : splitId[0];
    const decodedId = base58check(import_sha256.hash).decode(idString);
    const deserializedId = idBinaryStruct.fromBytes(decodedId);
    const { siphash , ...payload } = deserializedId;
    const payloadWithStringTimestamp = {
        ...payload,
        timestamp: payload.timestamp.toString()
    };
    if (!PayloadStruct.safeParse(payloadWithStringTimestamp).success) {
        throw new Error(`invalid payload`);
    }
    return [
        payloadWithStringTimestamp,
        siphash
    ];
}
var encode = (payload, key)=>{
    try {
        const hash = hashPayload(payload, key);
        const idBytes = idBinaryStruct.toBytes({
            siphash: hash,
            test: payload.test,
            timestamp: BigInt(payload.timestamp),
            ulid: payload.ulid
        });
        const idString = base58check(import_sha256.hash).encode(idBytes);
        return `${ID_PREFIX}${ID_SEPARATOR}${idString}`;
    } catch (error) {
        if (error instanceof mod.ZodError) {
            const joinedIssues = error.issues.map((issue)=>{
                return `${issue.code} : [${issue.path.join(", ")}] : ${issue.message}`;
            }).join("; ");
            throw new Error(`Invalid ID:  ${joinedIssues}`);
        } else if (error instanceof Error) {
            throw new Error(`Invalid ID: ${error.message}`);
        } else {
            throw error;
        }
    }
};
var decode = (id, key)=>{
    try {
        const [payload, siphash] = parseId(id);
        const newPayloadHash = hashPayload(payload, key);
        if (!(0, import_constant_time.equal)(newPayloadHash, siphash)) {
            throw new Error(`SipHash verification error`);
        }
        return payload;
    } catch (error) {
        if (error instanceof mod.ZodError) {
            const joinedIssues = error.issues.map((issue)=>{
                return `${issue.code} : [${issue.path.join(", ")}] : ${issue.message}`;
            }).join("; ");
            throw new Error(`Invalid ID:  ${joinedIssues}`);
        } else if (error instanceof Error) {
            throw new Error(`Invalid ID: ${error.message}`);
        } else {
            throw error;
        }
    }
};
var decodeUnsafely = (id)=>{
    try {
        const [payload] = parseId(id);
        return payload;
    } catch (error) {
        if (error instanceof mod.ZodError) {
            const joinedIssues = error.issues.map((issue)=>{
                return `${issue.code} : [${issue.path.join(", ")}] : ${issue.message}`;
            }).join("; ");
            throw new Error(`Invalid ID:  ${joinedIssues}`);
        } else if (error instanceof Error) {
            throw new Error(`Invalid ID: ${error.message}`);
        } else {
            throw error;
        }
    }
};
var isValid2 = (id, key)=>{
    try {
        decode(id, key);
        return true;
    } catch (error) {
        return false;
    }
};
var isValidUnsafely = (id)=>{
    try {
        decodeUnsafely(id);
        return true;
    } catch (error) {
        return false;
    }
};
/*! scure-base - MIT License (c) 2022 Paul Miller (paulmillr.com) */ export { PayloadStruct, SipHashHashStruct, SipHashKeyStruct, decode, decodeUnsafely, encode, isValid2 as isValid, isValidUnsafely };
export default null;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vY2RuLnNreXBhY2suZGV2Ly0vQHRydWVzdGFtcC9pZEB2MS4zLjEteEs4QmkwUlNQRklEbkUzMXYxdU0vZGlzdD1lczIwMTksbW9kZT1pbXBvcnRzL29wdGltaXplZC9AdHJ1ZXN0YW1wL2lkLmpzIl0sInNvdXJjZXNDb250ZW50IjpbInZhciBfX2NyZWF0ZSA9IE9iamVjdC5jcmVhdGU7XG52YXIgX19kZWZQcm9wID0gT2JqZWN0LmRlZmluZVByb3BlcnR5O1xudmFyIF9fZ2V0T3duUHJvcERlc2MgPSBPYmplY3QuZ2V0T3duUHJvcGVydHlEZXNjcmlwdG9yO1xudmFyIF9fZ2V0T3duUHJvcE5hbWVzID0gT2JqZWN0LmdldE93blByb3BlcnR5TmFtZXM7XG52YXIgX19nZXRQcm90b09mID0gT2JqZWN0LmdldFByb3RvdHlwZU9mO1xudmFyIF9faGFzT3duUHJvcCA9IE9iamVjdC5wcm90b3R5cGUuaGFzT3duUHJvcGVydHk7XG52YXIgX19jb21tb25KUyA9IChjYiwgbW9kMikgPT4gZnVuY3Rpb24gX19yZXF1aXJlKCkge1xuICByZXR1cm4gbW9kMiB8fCAoMCwgY2JbX19nZXRPd25Qcm9wTmFtZXMoY2IpWzBdXSkoKG1vZDIgPSB7ZXhwb3J0czoge319KS5leHBvcnRzLCBtb2QyKSwgbW9kMi5leHBvcnRzO1xufTtcbnZhciBfX2NvcHlQcm9wcyA9ICh0bywgZnJvbSwgZXhjZXB0LCBkZXNjKSA9PiB7XG4gIGlmIChmcm9tICYmIHR5cGVvZiBmcm9tID09PSBcIm9iamVjdFwiIHx8IHR5cGVvZiBmcm9tID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICBmb3IgKGxldCBrZXkgb2YgX19nZXRPd25Qcm9wTmFtZXMoZnJvbSkpXG4gICAgICBpZiAoIV9faGFzT3duUHJvcC5jYWxsKHRvLCBrZXkpICYmIGtleSAhPT0gZXhjZXB0KVxuICAgICAgICBfX2RlZlByb3AodG8sIGtleSwge2dldDogKCkgPT4gZnJvbVtrZXldLCBlbnVtZXJhYmxlOiAhKGRlc2MgPSBfX2dldE93blByb3BEZXNjKGZyb20sIGtleSkpIHx8IGRlc2MuZW51bWVyYWJsZX0pO1xuICB9XG4gIHJldHVybiB0bztcbn07XG52YXIgX190b0VTTSA9IChtb2QyLCBpc05vZGVNb2RlLCB0YXJnZXQpID0+ICh0YXJnZXQgPSBtb2QyICE9IG51bGwgPyBfX2NyZWF0ZShfX2dldFByb3RvT2YobW9kMikpIDoge30sIF9fY29weVByb3BzKGlzTm9kZU1vZGUgfHwgIW1vZDIgfHwgIW1vZDIuX19lc01vZHVsZSA/IF9fZGVmUHJvcCh0YXJnZXQsIFwiZGVmYXVsdFwiLCB7dmFsdWU6IG1vZDIsIGVudW1lcmFibGU6IHRydWV9KSA6IHRhcmdldCwgbW9kMikpO1xudmFyIHJlcXVpcmVfaW50ID0gX19jb21tb25KUyh7XG4gIFwibm9kZV9tb2R1bGVzL0BzdGFibGVsaWIvaW50L2xpYi9pbnQuanNcIihleHBvcnRzKSB7XG4gICAgT2JqZWN0LmRlZmluZVByb3BlcnR5KGV4cG9ydHMsIFwiX19lc01vZHVsZVwiLCB7dmFsdWU6IHRydWV9KTtcbiAgICBmdW5jdGlvbiBpbXVsU2hpbShhLCBiKSB7XG4gICAgICB2YXIgYWggPSBhID4+PiAxNiAmIDY1NTM1LCBhbCA9IGEgJiA2NTUzNTtcbiAgICAgIHZhciBiaCA9IGIgPj4+IDE2ICYgNjU1MzUsIGJsID0gYiAmIDY1NTM1O1xuICAgICAgcmV0dXJuIGFsICogYmwgKyAoYWggKiBibCArIGFsICogYmggPDwgMTYgPj4+IDApIHwgMDtcbiAgICB9XG4gICAgZXhwb3J0cy5tdWwgPSBNYXRoLmltdWwgfHwgaW11bFNoaW07XG4gICAgZnVuY3Rpb24gYWRkKGEsIGIpIHtcbiAgICAgIHJldHVybiBhICsgYiB8IDA7XG4gICAgfVxuICAgIGV4cG9ydHMuYWRkID0gYWRkO1xuICAgIGZ1bmN0aW9uIHN1YihhLCBiKSB7XG4gICAgICByZXR1cm4gYSAtIGIgfCAwO1xuICAgIH1cbiAgICBleHBvcnRzLnN1YiA9IHN1YjtcbiAgICBmdW5jdGlvbiByb3RsKHgsIG4pIHtcbiAgICAgIHJldHVybiB4IDw8IG4gfCB4ID4+PiAzMiAtIG47XG4gICAgfVxuICAgIGV4cG9ydHMucm90bCA9IHJvdGw7XG4gICAgZnVuY3Rpb24gcm90cih4LCBuKSB7XG4gICAgICByZXR1cm4geCA8PCAzMiAtIG4gfCB4ID4+PiBuO1xuICAgIH1cbiAgICBleHBvcnRzLnJvdHIgPSByb3RyO1xuICAgIGZ1bmN0aW9uIGlzSW50ZWdlclNoaW0obikge1xuICAgICAgcmV0dXJuIHR5cGVvZiBuID09PSBcIm51bWJlclwiICYmIGlzRmluaXRlKG4pICYmIE1hdGguZmxvb3IobikgPT09IG47XG4gICAgfVxuICAgIGV4cG9ydHMuaXNJbnRlZ2VyID0gTnVtYmVyLmlzSW50ZWdlciB8fCBpc0ludGVnZXJTaGltO1xuICAgIGV4cG9ydHMuTUFYX1NBRkVfSU5URUdFUiA9IDkwMDcxOTkyNTQ3NDA5OTE7XG4gICAgZXhwb3J0cy5pc1NhZmVJbnRlZ2VyID0gZnVuY3Rpb24obikge1xuICAgICAgcmV0dXJuIGV4cG9ydHMuaXNJbnRlZ2VyKG4pICYmIChuID49IC1leHBvcnRzLk1BWF9TQUZFX0lOVEVHRVIgJiYgbiA8PSBleHBvcnRzLk1BWF9TQUZFX0lOVEVHRVIpO1xuICAgIH07XG4gIH1cbn0pO1xudmFyIHJlcXVpcmVfYmluYXJ5ID0gX19jb21tb25KUyh7XG4gIFwibm9kZV9tb2R1bGVzL0BzdGFibGVsaWIvYmluYXJ5L2xpYi9iaW5hcnkuanNcIihleHBvcnRzKSB7XG4gICAgT2JqZWN0LmRlZmluZVByb3BlcnR5KGV4cG9ydHMsIFwiX19lc01vZHVsZVwiLCB7dmFsdWU6IHRydWV9KTtcbiAgICB2YXIgaW50XzEgPSByZXF1aXJlX2ludCgpO1xuICAgIGZ1bmN0aW9uIHJlYWRJbnQxNkJFKGFycmF5LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgcmV0dXJuIChhcnJheVtvZmZzZXQgKyAwXSA8PCA4IHwgYXJyYXlbb2Zmc2V0ICsgMV0pIDw8IDE2ID4+IDE2O1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRJbnQxNkJFID0gcmVhZEludDE2QkU7XG4gICAgZnVuY3Rpb24gcmVhZFVpbnQxNkJFKGFycmF5LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgcmV0dXJuIChhcnJheVtvZmZzZXQgKyAwXSA8PCA4IHwgYXJyYXlbb2Zmc2V0ICsgMV0pID4+PiAwO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRVaW50MTZCRSA9IHJlYWRVaW50MTZCRTtcbiAgICBmdW5jdGlvbiByZWFkSW50MTZMRShhcnJheSwgb2Zmc2V0KSB7XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIHJldHVybiAoYXJyYXlbb2Zmc2V0ICsgMV0gPDwgOCB8IGFycmF5W29mZnNldF0pIDw8IDE2ID4+IDE2O1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRJbnQxNkxFID0gcmVhZEludDE2TEU7XG4gICAgZnVuY3Rpb24gcmVhZFVpbnQxNkxFKGFycmF5LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgcmV0dXJuIChhcnJheVtvZmZzZXQgKyAxXSA8PCA4IHwgYXJyYXlbb2Zmc2V0XSkgPj4+IDA7XG4gICAgfVxuICAgIGV4cG9ydHMucmVhZFVpbnQxNkxFID0gcmVhZFVpbnQxNkxFO1xuICAgIGZ1bmN0aW9uIHdyaXRlVWludDE2QkUodmFsdWUsIG91dCwgb2Zmc2V0KSB7XG4gICAgICBpZiAob3V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb3V0ID0gbmV3IFVpbnQ4QXJyYXkoMik7XG4gICAgICB9XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIG91dFtvZmZzZXQgKyAwXSA9IHZhbHVlID4+PiA4O1xuICAgICAgb3V0W29mZnNldCArIDFdID0gdmFsdWUgPj4+IDA7XG4gICAgICByZXR1cm4gb3V0O1xuICAgIH1cbiAgICBleHBvcnRzLndyaXRlVWludDE2QkUgPSB3cml0ZVVpbnQxNkJFO1xuICAgIGV4cG9ydHMud3JpdGVJbnQxNkJFID0gd3JpdGVVaW50MTZCRTtcbiAgICBmdW5jdGlvbiB3cml0ZVVpbnQxNkxFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDIpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICBvdXRbb2Zmc2V0ICsgMF0gPSB2YWx1ZSA+Pj4gMDtcbiAgICAgIG91dFtvZmZzZXQgKyAxXSA9IHZhbHVlID4+PiA4O1xuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZVVpbnQxNkxFID0gd3JpdGVVaW50MTZMRTtcbiAgICBleHBvcnRzLndyaXRlSW50MTZMRSA9IHdyaXRlVWludDE2TEU7XG4gICAgZnVuY3Rpb24gcmVhZEludDMyQkUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICByZXR1cm4gYXJyYXlbb2Zmc2V0XSA8PCAyNCB8IGFycmF5W29mZnNldCArIDFdIDw8IDE2IHwgYXJyYXlbb2Zmc2V0ICsgMl0gPDwgOCB8IGFycmF5W29mZnNldCArIDNdO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRJbnQzMkJFID0gcmVhZEludDMyQkU7XG4gICAgZnVuY3Rpb24gcmVhZFVpbnQzMkJFKGFycmF5LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgcmV0dXJuIChhcnJheVtvZmZzZXRdIDw8IDI0IHwgYXJyYXlbb2Zmc2V0ICsgMV0gPDwgMTYgfCBhcnJheVtvZmZzZXQgKyAyXSA8PCA4IHwgYXJyYXlbb2Zmc2V0ICsgM10pID4+PiAwO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRVaW50MzJCRSA9IHJlYWRVaW50MzJCRTtcbiAgICBmdW5jdGlvbiByZWFkSW50MzJMRShhcnJheSwgb2Zmc2V0KSB7XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIHJldHVybiBhcnJheVtvZmZzZXQgKyAzXSA8PCAyNCB8IGFycmF5W29mZnNldCArIDJdIDw8IDE2IHwgYXJyYXlbb2Zmc2V0ICsgMV0gPDwgOCB8IGFycmF5W29mZnNldF07XG4gICAgfVxuICAgIGV4cG9ydHMucmVhZEludDMyTEUgPSByZWFkSW50MzJMRTtcbiAgICBmdW5jdGlvbiByZWFkVWludDMyTEUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICByZXR1cm4gKGFycmF5W29mZnNldCArIDNdIDw8IDI0IHwgYXJyYXlbb2Zmc2V0ICsgMl0gPDwgMTYgfCBhcnJheVtvZmZzZXQgKyAxXSA8PCA4IHwgYXJyYXlbb2Zmc2V0XSkgPj4+IDA7XG4gICAgfVxuICAgIGV4cG9ydHMucmVhZFVpbnQzMkxFID0gcmVhZFVpbnQzMkxFO1xuICAgIGZ1bmN0aW9uIHdyaXRlVWludDMyQkUodmFsdWUsIG91dCwgb2Zmc2V0KSB7XG4gICAgICBpZiAob3V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb3V0ID0gbmV3IFVpbnQ4QXJyYXkoNCk7XG4gICAgICB9XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIG91dFtvZmZzZXQgKyAwXSA9IHZhbHVlID4+PiAyNDtcbiAgICAgIG91dFtvZmZzZXQgKyAxXSA9IHZhbHVlID4+PiAxNjtcbiAgICAgIG91dFtvZmZzZXQgKyAyXSA9IHZhbHVlID4+PiA4O1xuICAgICAgb3V0W29mZnNldCArIDNdID0gdmFsdWUgPj4+IDA7XG4gICAgICByZXR1cm4gb3V0O1xuICAgIH1cbiAgICBleHBvcnRzLndyaXRlVWludDMyQkUgPSB3cml0ZVVpbnQzMkJFO1xuICAgIGV4cG9ydHMud3JpdGVJbnQzMkJFID0gd3JpdGVVaW50MzJCRTtcbiAgICBmdW5jdGlvbiB3cml0ZVVpbnQzMkxFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDQpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICBvdXRbb2Zmc2V0ICsgMF0gPSB2YWx1ZSA+Pj4gMDtcbiAgICAgIG91dFtvZmZzZXQgKyAxXSA9IHZhbHVlID4+PiA4O1xuICAgICAgb3V0W29mZnNldCArIDJdID0gdmFsdWUgPj4+IDE2O1xuICAgICAgb3V0W29mZnNldCArIDNdID0gdmFsdWUgPj4+IDI0O1xuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZVVpbnQzMkxFID0gd3JpdGVVaW50MzJMRTtcbiAgICBleHBvcnRzLndyaXRlSW50MzJMRSA9IHdyaXRlVWludDMyTEU7XG4gICAgZnVuY3Rpb24gcmVhZEludDY0QkUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgaGkgPSByZWFkSW50MzJCRShhcnJheSwgb2Zmc2V0KTtcbiAgICAgIHZhciBsbyA9IHJlYWRJbnQzMkJFKGFycmF5LCBvZmZzZXQgKyA0KTtcbiAgICAgIHJldHVybiBoaSAqIDQyOTQ5NjcyOTYgKyBsbyAtIChsbyA+PiAzMSkgKiA0Mjk0OTY3Mjk2O1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRJbnQ2NEJFID0gcmVhZEludDY0QkU7XG4gICAgZnVuY3Rpb24gcmVhZFVpbnQ2NEJFKGFycmF5LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgdmFyIGhpID0gcmVhZFVpbnQzMkJFKGFycmF5LCBvZmZzZXQpO1xuICAgICAgdmFyIGxvID0gcmVhZFVpbnQzMkJFKGFycmF5LCBvZmZzZXQgKyA0KTtcbiAgICAgIHJldHVybiBoaSAqIDQyOTQ5NjcyOTYgKyBsbztcbiAgICB9XG4gICAgZXhwb3J0cy5yZWFkVWludDY0QkUgPSByZWFkVWludDY0QkU7XG4gICAgZnVuY3Rpb24gcmVhZEludDY0TEUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgbG8gPSByZWFkSW50MzJMRShhcnJheSwgb2Zmc2V0KTtcbiAgICAgIHZhciBoaSA9IHJlYWRJbnQzMkxFKGFycmF5LCBvZmZzZXQgKyA0KTtcbiAgICAgIHJldHVybiBoaSAqIDQyOTQ5NjcyOTYgKyBsbyAtIChsbyA+PiAzMSkgKiA0Mjk0OTY3Mjk2O1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRJbnQ2NExFID0gcmVhZEludDY0TEU7XG4gICAgZnVuY3Rpb24gcmVhZFVpbnQ2NExFKGFycmF5LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgdmFyIGxvID0gcmVhZFVpbnQzMkxFKGFycmF5LCBvZmZzZXQpO1xuICAgICAgdmFyIGhpID0gcmVhZFVpbnQzMkxFKGFycmF5LCBvZmZzZXQgKyA0KTtcbiAgICAgIHJldHVybiBoaSAqIDQyOTQ5NjcyOTYgKyBsbztcbiAgICB9XG4gICAgZXhwb3J0cy5yZWFkVWludDY0TEUgPSByZWFkVWludDY0TEU7XG4gICAgZnVuY3Rpb24gd3JpdGVVaW50NjRCRSh2YWx1ZSwgb3V0LCBvZmZzZXQpIHtcbiAgICAgIGlmIChvdXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvdXQgPSBuZXcgVWludDhBcnJheSg4KTtcbiAgICAgIH1cbiAgICAgIGlmIChvZmZzZXQgPT09IHZvaWQgMCkge1xuICAgICAgICBvZmZzZXQgPSAwO1xuICAgICAgfVxuICAgICAgd3JpdGVVaW50MzJCRSh2YWx1ZSAvIDQyOTQ5NjcyOTYgPj4+IDAsIG91dCwgb2Zmc2V0KTtcbiAgICAgIHdyaXRlVWludDMyQkUodmFsdWUgPj4+IDAsIG91dCwgb2Zmc2V0ICsgNCk7XG4gICAgICByZXR1cm4gb3V0O1xuICAgIH1cbiAgICBleHBvcnRzLndyaXRlVWludDY0QkUgPSB3cml0ZVVpbnQ2NEJFO1xuICAgIGV4cG9ydHMud3JpdGVJbnQ2NEJFID0gd3JpdGVVaW50NjRCRTtcbiAgICBmdW5jdGlvbiB3cml0ZVVpbnQ2NExFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDgpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB3cml0ZVVpbnQzMkxFKHZhbHVlID4+PiAwLCBvdXQsIG9mZnNldCk7XG4gICAgICB3cml0ZVVpbnQzMkxFKHZhbHVlIC8gNDI5NDk2NzI5NiA+Pj4gMCwgb3V0LCBvZmZzZXQgKyA0KTtcbiAgICAgIHJldHVybiBvdXQ7XG4gICAgfVxuICAgIGV4cG9ydHMud3JpdGVVaW50NjRMRSA9IHdyaXRlVWludDY0TEU7XG4gICAgZXhwb3J0cy53cml0ZUludDY0TEUgPSB3cml0ZVVpbnQ2NExFO1xuICAgIGZ1bmN0aW9uIHJlYWRVaW50QkUoYml0TGVuZ3RoLCBhcnJheSwgb2Zmc2V0KSB7XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIGlmIChiaXRMZW5ndGggJSA4ICE9PSAwKSB7XG4gICAgICAgIHRocm93IG5ldyBFcnJvcihcInJlYWRVaW50QkUgc3VwcG9ydHMgb25seSBiaXRMZW5ndGhzIGRpdmlzaWJsZSBieSA4XCIpO1xuICAgICAgfVxuICAgICAgaWYgKGJpdExlbmd0aCAvIDggPiBhcnJheS5sZW5ndGggLSBvZmZzZXQpIHtcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicmVhZFVpbnRCRTogYXJyYXkgaXMgdG9vIHNob3J0IGZvciB0aGUgZ2l2ZW4gYml0TGVuZ3RoXCIpO1xuICAgICAgfVxuICAgICAgdmFyIHJlc3VsdCA9IDA7XG4gICAgICB2YXIgbXVsID0gMTtcbiAgICAgIGZvciAodmFyIGkgPSBiaXRMZW5ndGggLyA4ICsgb2Zmc2V0IC0gMTsgaSA+PSBvZmZzZXQ7IGktLSkge1xuICAgICAgICByZXN1bHQgKz0gYXJyYXlbaV0gKiBtdWw7XG4gICAgICAgIG11bCAqPSAyNTY7XG4gICAgICB9XG4gICAgICByZXR1cm4gcmVzdWx0O1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRVaW50QkUgPSByZWFkVWludEJFO1xuICAgIGZ1bmN0aW9uIHJlYWRVaW50TEUoYml0TGVuZ3RoLCBhcnJheSwgb2Zmc2V0KSB7XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIGlmIChiaXRMZW5ndGggJSA4ICE9PSAwKSB7XG4gICAgICAgIHRocm93IG5ldyBFcnJvcihcInJlYWRVaW50TEUgc3VwcG9ydHMgb25seSBiaXRMZW5ndGhzIGRpdmlzaWJsZSBieSA4XCIpO1xuICAgICAgfVxuICAgICAgaWYgKGJpdExlbmd0aCAvIDggPiBhcnJheS5sZW5ndGggLSBvZmZzZXQpIHtcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicmVhZFVpbnRMRTogYXJyYXkgaXMgdG9vIHNob3J0IGZvciB0aGUgZ2l2ZW4gYml0TGVuZ3RoXCIpO1xuICAgICAgfVxuICAgICAgdmFyIHJlc3VsdCA9IDA7XG4gICAgICB2YXIgbXVsID0gMTtcbiAgICAgIGZvciAodmFyIGkgPSBvZmZzZXQ7IGkgPCBvZmZzZXQgKyBiaXRMZW5ndGggLyA4OyBpKyspIHtcbiAgICAgICAgcmVzdWx0ICs9IGFycmF5W2ldICogbXVsO1xuICAgICAgICBtdWwgKj0gMjU2O1xuICAgICAgfVxuICAgICAgcmV0dXJuIHJlc3VsdDtcbiAgICB9XG4gICAgZXhwb3J0cy5yZWFkVWludExFID0gcmVhZFVpbnRMRTtcbiAgICBmdW5jdGlvbiB3cml0ZVVpbnRCRShiaXRMZW5ndGgsIHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KGJpdExlbmd0aCAvIDgpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICBpZiAoYml0TGVuZ3RoICUgOCAhPT0gMCkge1xuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJ3cml0ZVVpbnRCRSBzdXBwb3J0cyBvbmx5IGJpdExlbmd0aHMgZGl2aXNpYmxlIGJ5IDhcIik7XG4gICAgICB9XG4gICAgICBpZiAoIWludF8xLmlzU2FmZUludGVnZXIodmFsdWUpKSB7XG4gICAgICAgIHRocm93IG5ldyBFcnJvcihcIndyaXRlVWludEJFIHZhbHVlIG11c3QgYmUgYW4gaW50ZWdlclwiKTtcbiAgICAgIH1cbiAgICAgIHZhciBkaXYgPSAxO1xuICAgICAgZm9yICh2YXIgaSA9IGJpdExlbmd0aCAvIDggKyBvZmZzZXQgLSAxOyBpID49IG9mZnNldDsgaS0tKSB7XG4gICAgICAgIG91dFtpXSA9IHZhbHVlIC8gZGl2ICYgMjU1O1xuICAgICAgICBkaXYgKj0gMjU2O1xuICAgICAgfVxuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZVVpbnRCRSA9IHdyaXRlVWludEJFO1xuICAgIGZ1bmN0aW9uIHdyaXRlVWludExFKGJpdExlbmd0aCwgdmFsdWUsIG91dCwgb2Zmc2V0KSB7XG4gICAgICBpZiAob3V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb3V0ID0gbmV3IFVpbnQ4QXJyYXkoYml0TGVuZ3RoIC8gOCk7XG4gICAgICB9XG4gICAgICBpZiAob2Zmc2V0ID09PSB2b2lkIDApIHtcbiAgICAgICAgb2Zmc2V0ID0gMDtcbiAgICAgIH1cbiAgICAgIGlmIChiaXRMZW5ndGggJSA4ICE9PSAwKSB7XG4gICAgICAgIHRocm93IG5ldyBFcnJvcihcIndyaXRlVWludExFIHN1cHBvcnRzIG9ubHkgYml0TGVuZ3RocyBkaXZpc2libGUgYnkgOFwiKTtcbiAgICAgIH1cbiAgICAgIGlmICghaW50XzEuaXNTYWZlSW50ZWdlcih2YWx1ZSkpIHtcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwid3JpdGVVaW50TEUgdmFsdWUgbXVzdCBiZSBhbiBpbnRlZ2VyXCIpO1xuICAgICAgfVxuICAgICAgdmFyIGRpdiA9IDE7XG4gICAgICBmb3IgKHZhciBpID0gb2Zmc2V0OyBpIDwgb2Zmc2V0ICsgYml0TGVuZ3RoIC8gODsgaSsrKSB7XG4gICAgICAgIG91dFtpXSA9IHZhbHVlIC8gZGl2ICYgMjU1O1xuICAgICAgICBkaXYgKj0gMjU2O1xuICAgICAgfVxuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZVVpbnRMRSA9IHdyaXRlVWludExFO1xuICAgIGZ1bmN0aW9uIHJlYWRGbG9hdDMyQkUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhhcnJheS5idWZmZXIsIGFycmF5LmJ5dGVPZmZzZXQsIGFycmF5LmJ5dGVMZW5ndGgpO1xuICAgICAgcmV0dXJuIHZpZXcuZ2V0RmxvYXQzMihvZmZzZXQpO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRGbG9hdDMyQkUgPSByZWFkRmxvYXQzMkJFO1xuICAgIGZ1bmN0aW9uIHJlYWRGbG9hdDMyTEUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhhcnJheS5idWZmZXIsIGFycmF5LmJ5dGVPZmZzZXQsIGFycmF5LmJ5dGVMZW5ndGgpO1xuICAgICAgcmV0dXJuIHZpZXcuZ2V0RmxvYXQzMihvZmZzZXQsIHRydWUpO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRGbG9hdDMyTEUgPSByZWFkRmxvYXQzMkxFO1xuICAgIGZ1bmN0aW9uIHJlYWRGbG9hdDY0QkUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhhcnJheS5idWZmZXIsIGFycmF5LmJ5dGVPZmZzZXQsIGFycmF5LmJ5dGVMZW5ndGgpO1xuICAgICAgcmV0dXJuIHZpZXcuZ2V0RmxvYXQ2NChvZmZzZXQpO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRGbG9hdDY0QkUgPSByZWFkRmxvYXQ2NEJFO1xuICAgIGZ1bmN0aW9uIHJlYWRGbG9hdDY0TEUoYXJyYXksIG9mZnNldCkge1xuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhhcnJheS5idWZmZXIsIGFycmF5LmJ5dGVPZmZzZXQsIGFycmF5LmJ5dGVMZW5ndGgpO1xuICAgICAgcmV0dXJuIHZpZXcuZ2V0RmxvYXQ2NChvZmZzZXQsIHRydWUpO1xuICAgIH1cbiAgICBleHBvcnRzLnJlYWRGbG9hdDY0TEUgPSByZWFkRmxvYXQ2NExFO1xuICAgIGZ1bmN0aW9uIHdyaXRlRmxvYXQzMkJFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDQpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhvdXQuYnVmZmVyLCBvdXQuYnl0ZU9mZnNldCwgb3V0LmJ5dGVMZW5ndGgpO1xuICAgICAgdmlldy5zZXRGbG9hdDMyKG9mZnNldCwgdmFsdWUpO1xuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZUZsb2F0MzJCRSA9IHdyaXRlRmxvYXQzMkJFO1xuICAgIGZ1bmN0aW9uIHdyaXRlRmxvYXQzMkxFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDQpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhvdXQuYnVmZmVyLCBvdXQuYnl0ZU9mZnNldCwgb3V0LmJ5dGVMZW5ndGgpO1xuICAgICAgdmlldy5zZXRGbG9hdDMyKG9mZnNldCwgdmFsdWUsIHRydWUpO1xuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZUZsb2F0MzJMRSA9IHdyaXRlRmxvYXQzMkxFO1xuICAgIGZ1bmN0aW9uIHdyaXRlRmxvYXQ2NEJFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDgpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhvdXQuYnVmZmVyLCBvdXQuYnl0ZU9mZnNldCwgb3V0LmJ5dGVMZW5ndGgpO1xuICAgICAgdmlldy5zZXRGbG9hdDY0KG9mZnNldCwgdmFsdWUpO1xuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZUZsb2F0NjRCRSA9IHdyaXRlRmxvYXQ2NEJFO1xuICAgIGZ1bmN0aW9uIHdyaXRlRmxvYXQ2NExFKHZhbHVlLCBvdXQsIG9mZnNldCkge1xuICAgICAgaWYgKG91dCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG91dCA9IG5ldyBVaW50OEFycmF5KDgpO1xuICAgICAgfVxuICAgICAgaWYgKG9mZnNldCA9PT0gdm9pZCAwKSB7XG4gICAgICAgIG9mZnNldCA9IDA7XG4gICAgICB9XG4gICAgICB2YXIgdmlldyA9IG5ldyBEYXRhVmlldyhvdXQuYnVmZmVyLCBvdXQuYnl0ZU9mZnNldCwgb3V0LmJ5dGVMZW5ndGgpO1xuICAgICAgdmlldy5zZXRGbG9hdDY0KG9mZnNldCwgdmFsdWUsIHRydWUpO1xuICAgICAgcmV0dXJuIG91dDtcbiAgICB9XG4gICAgZXhwb3J0cy53cml0ZUZsb2F0NjRMRSA9IHdyaXRlRmxvYXQ2NExFO1xuICB9XG59KTtcbnZhciByZXF1aXJlX3dpcGUgPSBfX2NvbW1vbkpTKHtcbiAgXCJub2RlX21vZHVsZXMvQHN0YWJsZWxpYi93aXBlL2xpYi93aXBlLmpzXCIoZXhwb3J0cykge1xuICAgIE9iamVjdC5kZWZpbmVQcm9wZXJ0eShleHBvcnRzLCBcIl9fZXNNb2R1bGVcIiwge3ZhbHVlOiB0cnVlfSk7XG4gICAgZnVuY3Rpb24gd2lwZShhcnJheSkge1xuICAgICAgZm9yICh2YXIgaSA9IDA7IGkgPCBhcnJheS5sZW5ndGg7IGkrKykge1xuICAgICAgICBhcnJheVtpXSA9IDA7XG4gICAgICB9XG4gICAgICByZXR1cm4gYXJyYXk7XG4gICAgfVxuICAgIGV4cG9ydHMud2lwZSA9IHdpcGU7XG4gIH1cbn0pO1xudmFyIHJlcXVpcmVfc2hhMjU2ID0gX19jb21tb25KUyh7XG4gIFwibm9kZV9tb2R1bGVzL0BzdGFibGVsaWIvc2hhMjU2L2xpYi9zaGEyNTYuanNcIihleHBvcnRzKSB7XG4gICAgT2JqZWN0LmRlZmluZVByb3BlcnR5KGV4cG9ydHMsIFwiX19lc01vZHVsZVwiLCB7dmFsdWU6IHRydWV9KTtcbiAgICB2YXIgYmluYXJ5XzEgPSByZXF1aXJlX2JpbmFyeSgpO1xuICAgIHZhciB3aXBlXzEgPSByZXF1aXJlX3dpcGUoKTtcbiAgICBleHBvcnRzLkRJR0VTVF9MRU5HVEggPSAzMjtcbiAgICBleHBvcnRzLkJMT0NLX1NJWkUgPSA2NDtcbiAgICB2YXIgU0hBMjU2ID0gZnVuY3Rpb24oKSB7XG4gICAgICBmdW5jdGlvbiBTSEEyNTYyKCkge1xuICAgICAgICB0aGlzLmRpZ2VzdExlbmd0aCA9IGV4cG9ydHMuRElHRVNUX0xFTkdUSDtcbiAgICAgICAgdGhpcy5ibG9ja1NpemUgPSBleHBvcnRzLkJMT0NLX1NJWkU7XG4gICAgICAgIHRoaXMuX3N0YXRlID0gbmV3IEludDMyQXJyYXkoOCk7XG4gICAgICAgIHRoaXMuX3RlbXAgPSBuZXcgSW50MzJBcnJheSg2NCk7XG4gICAgICAgIHRoaXMuX2J1ZmZlciA9IG5ldyBVaW50OEFycmF5KDEyOCk7XG4gICAgICAgIHRoaXMuX2J1ZmZlckxlbmd0aCA9IDA7XG4gICAgICAgIHRoaXMuX2J5dGVzSGFzaGVkID0gMDtcbiAgICAgICAgdGhpcy5fZmluaXNoZWQgPSBmYWxzZTtcbiAgICAgICAgdGhpcy5yZXNldCgpO1xuICAgICAgfVxuICAgICAgU0hBMjU2Mi5wcm90b3R5cGUuX2luaXRTdGF0ZSA9IGZ1bmN0aW9uKCkge1xuICAgICAgICB0aGlzLl9zdGF0ZVswXSA9IDE3NzkwMzM3MDM7XG4gICAgICAgIHRoaXMuX3N0YXRlWzFdID0gMzE0NDEzNDI3NztcbiAgICAgICAgdGhpcy5fc3RhdGVbMl0gPSAxMDEzOTA0MjQyO1xuICAgICAgICB0aGlzLl9zdGF0ZVszXSA9IDI3NzM0ODA3NjI7XG4gICAgICAgIHRoaXMuX3N0YXRlWzRdID0gMTM1OTg5MzExOTtcbiAgICAgICAgdGhpcy5fc3RhdGVbNV0gPSAyNjAwODIyOTI0O1xuICAgICAgICB0aGlzLl9zdGF0ZVs2XSA9IDUyODczNDYzNTtcbiAgICAgICAgdGhpcy5fc3RhdGVbN10gPSAxNTQxNDU5MjI1O1xuICAgICAgfTtcbiAgICAgIFNIQTI1NjIucHJvdG90eXBlLnJlc2V0ID0gZnVuY3Rpb24oKSB7XG4gICAgICAgIHRoaXMuX2luaXRTdGF0ZSgpO1xuICAgICAgICB0aGlzLl9idWZmZXJMZW5ndGggPSAwO1xuICAgICAgICB0aGlzLl9ieXRlc0hhc2hlZCA9IDA7XG4gICAgICAgIHRoaXMuX2ZpbmlzaGVkID0gZmFsc2U7XG4gICAgICAgIHJldHVybiB0aGlzO1xuICAgICAgfTtcbiAgICAgIFNIQTI1NjIucHJvdG90eXBlLmNsZWFuID0gZnVuY3Rpb24oKSB7XG4gICAgICAgIHdpcGVfMS53aXBlKHRoaXMuX2J1ZmZlcik7XG4gICAgICAgIHdpcGVfMS53aXBlKHRoaXMuX3RlbXApO1xuICAgICAgICB0aGlzLnJlc2V0KCk7XG4gICAgICB9O1xuICAgICAgU0hBMjU2Mi5wcm90b3R5cGUudXBkYXRlID0gZnVuY3Rpb24oZGF0YSwgZGF0YUxlbmd0aCkge1xuICAgICAgICBpZiAoZGF0YUxlbmd0aCA9PT0gdm9pZCAwKSB7XG4gICAgICAgICAgZGF0YUxlbmd0aCA9IGRhdGEubGVuZ3RoO1xuICAgICAgICB9XG4gICAgICAgIGlmICh0aGlzLl9maW5pc2hlZCkge1xuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcIlNIQTI1NjogY2FuJ3QgdXBkYXRlIGJlY2F1c2UgaGFzaCB3YXMgZmluaXNoZWQuXCIpO1xuICAgICAgICB9XG4gICAgICAgIHZhciBkYXRhUG9zID0gMDtcbiAgICAgICAgdGhpcy5fYnl0ZXNIYXNoZWQgKz0gZGF0YUxlbmd0aDtcbiAgICAgICAgaWYgKHRoaXMuX2J1ZmZlckxlbmd0aCA+IDApIHtcbiAgICAgICAgICB3aGlsZSAodGhpcy5fYnVmZmVyTGVuZ3RoIDwgdGhpcy5ibG9ja1NpemUgJiYgZGF0YUxlbmd0aCA+IDApIHtcbiAgICAgICAgICAgIHRoaXMuX2J1ZmZlclt0aGlzLl9idWZmZXJMZW5ndGgrK10gPSBkYXRhW2RhdGFQb3MrK107XG4gICAgICAgICAgICBkYXRhTGVuZ3RoLS07XG4gICAgICAgICAgfVxuICAgICAgICAgIGlmICh0aGlzLl9idWZmZXJMZW5ndGggPT09IHRoaXMuYmxvY2tTaXplKSB7XG4gICAgICAgICAgICBoYXNoQmxvY2tzKHRoaXMuX3RlbXAsIHRoaXMuX3N0YXRlLCB0aGlzLl9idWZmZXIsIDAsIHRoaXMuYmxvY2tTaXplKTtcbiAgICAgICAgICAgIHRoaXMuX2J1ZmZlckxlbmd0aCA9IDA7XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICAgIGlmIChkYXRhTGVuZ3RoID49IHRoaXMuYmxvY2tTaXplKSB7XG4gICAgICAgICAgZGF0YVBvcyA9IGhhc2hCbG9ja3ModGhpcy5fdGVtcCwgdGhpcy5fc3RhdGUsIGRhdGEsIGRhdGFQb3MsIGRhdGFMZW5ndGgpO1xuICAgICAgICAgIGRhdGFMZW5ndGggJT0gdGhpcy5ibG9ja1NpemU7XG4gICAgICAgIH1cbiAgICAgICAgd2hpbGUgKGRhdGFMZW5ndGggPiAwKSB7XG4gICAgICAgICAgdGhpcy5fYnVmZmVyW3RoaXMuX2J1ZmZlckxlbmd0aCsrXSA9IGRhdGFbZGF0YVBvcysrXTtcbiAgICAgICAgICBkYXRhTGVuZ3RoLS07XG4gICAgICAgIH1cbiAgICAgICAgcmV0dXJuIHRoaXM7XG4gICAgICB9O1xuICAgICAgU0hBMjU2Mi5wcm90b3R5cGUuZmluaXNoID0gZnVuY3Rpb24ob3V0KSB7XG4gICAgICAgIGlmICghdGhpcy5fZmluaXNoZWQpIHtcbiAgICAgICAgICB2YXIgYnl0ZXNIYXNoZWQgPSB0aGlzLl9ieXRlc0hhc2hlZDtcbiAgICAgICAgICB2YXIgbGVmdCA9IHRoaXMuX2J1ZmZlckxlbmd0aDtcbiAgICAgICAgICB2YXIgYml0TGVuSGkgPSBieXRlc0hhc2hlZCAvIDUzNjg3MDkxMiB8IDA7XG4gICAgICAgICAgdmFyIGJpdExlbkxvID0gYnl0ZXNIYXNoZWQgPDwgMztcbiAgICAgICAgICB2YXIgcGFkTGVuZ3RoID0gYnl0ZXNIYXNoZWQgJSA2NCA8IDU2ID8gNjQgOiAxMjg7XG4gICAgICAgICAgdGhpcy5fYnVmZmVyW2xlZnRdID0gMTI4O1xuICAgICAgICAgIGZvciAodmFyIGkgPSBsZWZ0ICsgMTsgaSA8IHBhZExlbmd0aCAtIDg7IGkrKykge1xuICAgICAgICAgICAgdGhpcy5fYnVmZmVyW2ldID0gMDtcbiAgICAgICAgICB9XG4gICAgICAgICAgYmluYXJ5XzEud3JpdGVVaW50MzJCRShiaXRMZW5IaSwgdGhpcy5fYnVmZmVyLCBwYWRMZW5ndGggLSA4KTtcbiAgICAgICAgICBiaW5hcnlfMS53cml0ZVVpbnQzMkJFKGJpdExlbkxvLCB0aGlzLl9idWZmZXIsIHBhZExlbmd0aCAtIDQpO1xuICAgICAgICAgIGhhc2hCbG9ja3ModGhpcy5fdGVtcCwgdGhpcy5fc3RhdGUsIHRoaXMuX2J1ZmZlciwgMCwgcGFkTGVuZ3RoKTtcbiAgICAgICAgICB0aGlzLl9maW5pc2hlZCA9IHRydWU7XG4gICAgICAgIH1cbiAgICAgICAgZm9yICh2YXIgaSA9IDA7IGkgPCB0aGlzLmRpZ2VzdExlbmd0aCAvIDQ7IGkrKykge1xuICAgICAgICAgIGJpbmFyeV8xLndyaXRlVWludDMyQkUodGhpcy5fc3RhdGVbaV0sIG91dCwgaSAqIDQpO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiB0aGlzO1xuICAgICAgfTtcbiAgICAgIFNIQTI1NjIucHJvdG90eXBlLmRpZ2VzdCA9IGZ1bmN0aW9uKCkge1xuICAgICAgICB2YXIgb3V0ID0gbmV3IFVpbnQ4QXJyYXkodGhpcy5kaWdlc3RMZW5ndGgpO1xuICAgICAgICB0aGlzLmZpbmlzaChvdXQpO1xuICAgICAgICByZXR1cm4gb3V0O1xuICAgICAgfTtcbiAgICAgIFNIQTI1NjIucHJvdG90eXBlLnNhdmVTdGF0ZSA9IGZ1bmN0aW9uKCkge1xuICAgICAgICBpZiAodGhpcy5fZmluaXNoZWQpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJTSEEyNTY6IGNhbm5vdCBzYXZlIGZpbmlzaGVkIHN0YXRlXCIpO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiB7XG4gICAgICAgICAgc3RhdGU6IG5ldyBJbnQzMkFycmF5KHRoaXMuX3N0YXRlKSxcbiAgICAgICAgICBidWZmZXI6IHRoaXMuX2J1ZmZlckxlbmd0aCA+IDAgPyBuZXcgVWludDhBcnJheSh0aGlzLl9idWZmZXIpIDogdm9pZCAwLFxuICAgICAgICAgIGJ1ZmZlckxlbmd0aDogdGhpcy5fYnVmZmVyTGVuZ3RoLFxuICAgICAgICAgIGJ5dGVzSGFzaGVkOiB0aGlzLl9ieXRlc0hhc2hlZFxuICAgICAgICB9O1xuICAgICAgfTtcbiAgICAgIFNIQTI1NjIucHJvdG90eXBlLnJlc3RvcmVTdGF0ZSA9IGZ1bmN0aW9uKHNhdmVkU3RhdGUpIHtcbiAgICAgICAgdGhpcy5fc3RhdGUuc2V0KHNhdmVkU3RhdGUuc3RhdGUpO1xuICAgICAgICB0aGlzLl9idWZmZXJMZW5ndGggPSBzYXZlZFN0YXRlLmJ1ZmZlckxlbmd0aDtcbiAgICAgICAgaWYgKHNhdmVkU3RhdGUuYnVmZmVyKSB7XG4gICAgICAgICAgdGhpcy5fYnVmZmVyLnNldChzYXZlZFN0YXRlLmJ1ZmZlcik7XG4gICAgICAgIH1cbiAgICAgICAgdGhpcy5fYnl0ZXNIYXNoZWQgPSBzYXZlZFN0YXRlLmJ5dGVzSGFzaGVkO1xuICAgICAgICB0aGlzLl9maW5pc2hlZCA9IGZhbHNlO1xuICAgICAgICByZXR1cm4gdGhpcztcbiAgICAgIH07XG4gICAgICBTSEEyNTYyLnByb3RvdHlwZS5jbGVhblNhdmVkU3RhdGUgPSBmdW5jdGlvbihzYXZlZFN0YXRlKSB7XG4gICAgICAgIHdpcGVfMS53aXBlKHNhdmVkU3RhdGUuc3RhdGUpO1xuICAgICAgICBpZiAoc2F2ZWRTdGF0ZS5idWZmZXIpIHtcbiAgICAgICAgICB3aXBlXzEud2lwZShzYXZlZFN0YXRlLmJ1ZmZlcik7XG4gICAgICAgIH1cbiAgICAgICAgc2F2ZWRTdGF0ZS5idWZmZXJMZW5ndGggPSAwO1xuICAgICAgICBzYXZlZFN0YXRlLmJ5dGVzSGFzaGVkID0gMDtcbiAgICAgIH07XG4gICAgICByZXR1cm4gU0hBMjU2MjtcbiAgICB9KCk7XG4gICAgZXhwb3J0cy5TSEEyNTYgPSBTSEEyNTY7XG4gICAgdmFyIEsgPSBuZXcgSW50MzJBcnJheShbXG4gICAgICAxMTE2MzUyNDA4LFxuICAgICAgMTg5OTQ0NzQ0MSxcbiAgICAgIDMwNDkzMjM0NzEsXG4gICAgICAzOTIxMDA5NTczLFxuICAgICAgOTYxOTg3MTYzLFxuICAgICAgMTUwODk3MDk5MyxcbiAgICAgIDI0NTM2MzU3NDgsXG4gICAgICAyODcwNzYzMjIxLFxuICAgICAgMzYyNDM4MTA4MCxcbiAgICAgIDMxMDU5ODQwMSxcbiAgICAgIDYwNzIyNTI3OCxcbiAgICAgIDE0MjY4ODE5ODcsXG4gICAgICAxOTI1MDc4Mzg4LFxuICAgICAgMjE2MjA3ODIwNixcbiAgICAgIDI2MTQ4ODgxMDMsXG4gICAgICAzMjQ4MjIyNTgwLFxuICAgICAgMzgzNTM5MDQwMSxcbiAgICAgIDQwMjIyMjQ3NzQsXG4gICAgICAyNjQzNDcwNzgsXG4gICAgICA2MDQ4MDc2MjgsXG4gICAgICA3NzAyNTU5ODMsXG4gICAgICAxMjQ5MTUwMTIyLFxuICAgICAgMTU1NTA4MTY5MixcbiAgICAgIDE5OTYwNjQ5ODYsXG4gICAgICAyNTU0MjIwODgyLFxuICAgICAgMjgyMTgzNDM0OSxcbiAgICAgIDI5NTI5OTY4MDgsXG4gICAgICAzMjEwMzEzNjcxLFxuICAgICAgMzMzNjU3MTg5MSxcbiAgICAgIDM1ODQ1Mjg3MTEsXG4gICAgICAxMTM5MjY5OTMsXG4gICAgICAzMzgyNDE4OTUsXG4gICAgICA2NjYzMDcyMDUsXG4gICAgICA3NzM1Mjk5MTIsXG4gICAgICAxMjk0NzU3MzcyLFxuICAgICAgMTM5NjE4MjI5MSxcbiAgICAgIDE2OTUxODM3MDAsXG4gICAgICAxOTg2NjYxMDUxLFxuICAgICAgMjE3NzAyNjM1MCxcbiAgICAgIDI0NTY5NTYwMzcsXG4gICAgICAyNzMwNDg1OTIxLFxuICAgICAgMjgyMDMwMjQxMSxcbiAgICAgIDMyNTk3MzA4MDAsXG4gICAgICAzMzQ1NzY0NzcxLFxuICAgICAgMzUxNjA2NTgxNyxcbiAgICAgIDM2MDAzNTI4MDQsXG4gICAgICA0MDk0NTcxOTA5LFxuICAgICAgMjc1NDIzMzQ0LFxuICAgICAgNDMwMjI3NzM0LFxuICAgICAgNTA2OTQ4NjE2LFxuICAgICAgNjU5MDYwNTU2LFxuICAgICAgODgzOTk3ODc3LFxuICAgICAgOTU4MTM5NTcxLFxuICAgICAgMTMyMjgyMjIxOCxcbiAgICAgIDE1MzcwMDIwNjMsXG4gICAgICAxNzQ3ODczNzc5LFxuICAgICAgMTk1NTU2MjIyMixcbiAgICAgIDIwMjQxMDQ4MTUsXG4gICAgICAyMjI3NzMwNDUyLFxuICAgICAgMjM2MTg1MjQyNCxcbiAgICAgIDI0Mjg0MzY0NzQsXG4gICAgICAyNzU2NzM0MTg3LFxuICAgICAgMzIwNDAzMTQ3OSxcbiAgICAgIDMzMjkzMjUyOThcbiAgICBdKTtcbiAgICBmdW5jdGlvbiBoYXNoQmxvY2tzKHcsIHYsIHAsIHBvcywgbGVuKSB7XG4gICAgICB3aGlsZSAobGVuID49IDY0KSB7XG4gICAgICAgIHZhciBhID0gdlswXTtcbiAgICAgICAgdmFyIGIgPSB2WzFdO1xuICAgICAgICB2YXIgYyA9IHZbMl07XG4gICAgICAgIHZhciBkID0gdlszXTtcbiAgICAgICAgdmFyIGUgPSB2WzRdO1xuICAgICAgICB2YXIgZiA9IHZbNV07XG4gICAgICAgIHZhciBnID0gdls2XTtcbiAgICAgICAgdmFyIGggPSB2WzddO1xuICAgICAgICBmb3IgKHZhciBpID0gMDsgaSA8IDE2OyBpKyspIHtcbiAgICAgICAgICB2YXIgaiA9IHBvcyArIGkgKiA0O1xuICAgICAgICAgIHdbaV0gPSBiaW5hcnlfMS5yZWFkVWludDMyQkUocCwgaik7XG4gICAgICAgIH1cbiAgICAgICAgZm9yICh2YXIgaSA9IDE2OyBpIDwgNjQ7IGkrKykge1xuICAgICAgICAgIHZhciB1ID0gd1tpIC0gMl07XG4gICAgICAgICAgdmFyIHQxID0gKHUgPj4+IDE3IHwgdSA8PCAzMiAtIDE3KSBeICh1ID4+PiAxOSB8IHUgPDwgMzIgLSAxOSkgXiB1ID4+PiAxMDtcbiAgICAgICAgICB1ID0gd1tpIC0gMTVdO1xuICAgICAgICAgIHZhciB0MiA9ICh1ID4+PiA3IHwgdSA8PCAzMiAtIDcpIF4gKHUgPj4+IDE4IHwgdSA8PCAzMiAtIDE4KSBeIHUgPj4+IDM7XG4gICAgICAgICAgd1tpXSA9ICh0MSArIHdbaSAtIDddIHwgMCkgKyAodDIgKyB3W2kgLSAxNl0gfCAwKTtcbiAgICAgICAgfVxuICAgICAgICBmb3IgKHZhciBpID0gMDsgaSA8IDY0OyBpKyspIHtcbiAgICAgICAgICB2YXIgdDEgPSAoKChlID4+PiA2IHwgZSA8PCAzMiAtIDYpIF4gKGUgPj4+IDExIHwgZSA8PCAzMiAtIDExKSBeIChlID4+PiAyNSB8IGUgPDwgMzIgLSAyNSkpICsgKGUgJiBmIF4gfmUgJiBnKSB8IDApICsgKGggKyAoS1tpXSArIHdbaV0gfCAwKSB8IDApIHwgMDtcbiAgICAgICAgICB2YXIgdDIgPSAoKGEgPj4+IDIgfCBhIDw8IDMyIC0gMikgXiAoYSA+Pj4gMTMgfCBhIDw8IDMyIC0gMTMpIF4gKGEgPj4+IDIyIHwgYSA8PCAzMiAtIDIyKSkgKyAoYSAmIGIgXiBhICYgYyBeIGIgJiBjKSB8IDA7XG4gICAgICAgICAgaCA9IGc7XG4gICAgICAgICAgZyA9IGY7XG4gICAgICAgICAgZiA9IGU7XG4gICAgICAgICAgZSA9IGQgKyB0MSB8IDA7XG4gICAgICAgICAgZCA9IGM7XG4gICAgICAgICAgYyA9IGI7XG4gICAgICAgICAgYiA9IGE7XG4gICAgICAgICAgYSA9IHQxICsgdDIgfCAwO1xuICAgICAgICB9XG4gICAgICAgIHZbMF0gKz0gYTtcbiAgICAgICAgdlsxXSArPSBiO1xuICAgICAgICB2WzJdICs9IGM7XG4gICAgICAgIHZbM10gKz0gZDtcbiAgICAgICAgdls0XSArPSBlO1xuICAgICAgICB2WzVdICs9IGY7XG4gICAgICAgIHZbNl0gKz0gZztcbiAgICAgICAgdls3XSArPSBoO1xuICAgICAgICBwb3MgKz0gNjQ7XG4gICAgICAgIGxlbiAtPSA2NDtcbiAgICAgIH1cbiAgICAgIHJldHVybiBwb3M7XG4gICAgfVxuICAgIGZ1bmN0aW9uIGhhc2goZGF0YSkge1xuICAgICAgdmFyIGggPSBuZXcgU0hBMjU2KCk7XG4gICAgICBoLnVwZGF0ZShkYXRhKTtcbiAgICAgIHZhciBkaWdlc3QgPSBoLmRpZ2VzdCgpO1xuICAgICAgaC5jbGVhbigpO1xuICAgICAgcmV0dXJuIGRpZ2VzdDtcbiAgICB9XG4gICAgZXhwb3J0cy5oYXNoID0gaGFzaDtcbiAgfVxufSk7XG52YXIgcmVxdWlyZV9oYWxmc2lwaGFzaCA9IF9fY29tbW9uSlMoe1xuICBcIm5vZGVfbW9kdWxlcy9Ac3RhYmxlbGliL2hhbGZzaXBoYXNoL2xpYi9oYWxmc2lwaGFzaC5qc1wiKGV4cG9ydHMpIHtcbiAgICBPYmplY3QuZGVmaW5lUHJvcGVydHkoZXhwb3J0cywgXCJfX2VzTW9kdWxlXCIsIHt2YWx1ZTogdHJ1ZX0pO1xuICAgIHZhciBiaW5hcnlfMSA9IHJlcXVpcmVfYmluYXJ5KCk7XG4gICAgdmFyIGludF8xID0gcmVxdWlyZV9pbnQoKTtcbiAgICBleHBvcnRzLktFWV9MRU5HVEggPSA4O1xuICAgIGV4cG9ydHMuRElHRVNUX0xFTkdUSCA9IDQ7XG4gICAgZnVuY3Rpb24gaGFsZlNpcEhhc2gyKGtleSwgZGF0YSkge1xuICAgICAgaWYgKGtleS5sZW5ndGggIT09IGV4cG9ydHMuS0VZX0xFTkdUSCkge1xuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJoYWxmU2lwSGFzaDogaW5jb3JyZWN0IGtleSBsZW5ndGhcIik7XG4gICAgICB9XG4gICAgICB2YXIgazAgPSBiaW5hcnlfMS5yZWFkVWludDMyTEUoa2V5LCAwKTtcbiAgICAgIHZhciBrMSA9IGJpbmFyeV8xLnJlYWRVaW50MzJMRShrZXksIDQpO1xuICAgICAgcmV0dXJuIGJpbmFyeV8xLndyaXRlVWludDMyTEUoaGFsZlNpcEhhc2hOdW0oazAsIGsxLCBkYXRhKSk7XG4gICAgfVxuICAgIGV4cG9ydHMuaGFsZlNpcEhhc2ggPSBoYWxmU2lwSGFzaDI7XG4gICAgZnVuY3Rpb24gaGFsZlNpcEhhc2hOdW0oazAsIGsxLCBkYXRhKSB7XG4gICAgICB2YXIgdjAgPSBrMDtcbiAgICAgIHZhciB2MSA9IGsxO1xuICAgICAgdmFyIHYyID0gazAgXiAxODE5ODk1NjUzO1xuICAgICAgdmFyIHYzID0gazEgXiAxOTUyODAxODkwO1xuICAgICAgdmFyIHBvcyA9IDA7XG4gICAgICB2YXIgbGVuID0gZGF0YS5sZW5ndGg7XG4gICAgICB2YXIgZmluID0gbGVuICUgMjU2IDw8IDI0O1xuICAgICAgd2hpbGUgKGxlbiA+PSA0KSB7XG4gICAgICAgIHZhciBtID0gYmluYXJ5XzEucmVhZFVpbnQzMkxFKGRhdGEsIHBvcyk7XG4gICAgICAgIHYzIF49IG07XG4gICAgICAgIHYwID0gaW50XzEuYWRkKHYwLCB2MSk7XG4gICAgICAgIHYxID0gaW50XzEucm90bCh2MSwgNSk7XG4gICAgICAgIHYxIF49IHYwO1xuICAgICAgICB2MCA9IGludF8xLnJvdGwodjAsIDE2KTtcbiAgICAgICAgdjIgPSBpbnRfMS5hZGQodjIsIHYzKTtcbiAgICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgICAgdjMgXj0gdjI7XG4gICAgICAgIHYwID0gaW50XzEuYWRkKHYwLCB2Myk7XG4gICAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICAgIHYzIF49IHYwO1xuICAgICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjEpO1xuICAgICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgICAgdjEgXj0gdjI7XG4gICAgICAgIHYyID0gaW50XzEucm90bCh2MiwgMTYpO1xuICAgICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgICB2MSA9IGludF8xLnJvdGwodjEsIDUpO1xuICAgICAgICB2MSBePSB2MDtcbiAgICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2Myk7XG4gICAgICAgIHYzID0gaW50XzEucm90bCh2MywgOCk7XG4gICAgICAgIHYzIF49IHYyO1xuICAgICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjMpO1xuICAgICAgICB2MyA9IGludF8xLnJvdGwodjMsIDcpO1xuICAgICAgICB2MyBePSB2MDtcbiAgICAgICAgdjIgPSBpbnRfMS5hZGQodjIsIHYxKTtcbiAgICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCAxMyk7XG4gICAgICAgIHYxIF49IHYyO1xuICAgICAgICB2MiA9IGludF8xLnJvdGwodjIsIDE2KTtcbiAgICAgICAgdjAgXj0gbTtcbiAgICAgICAgcG9zICs9IDQ7XG4gICAgICAgIGxlbiAtPSA0O1xuICAgICAgfVxuICAgICAgc3dpdGNoIChsZW4pIHtcbiAgICAgICAgY2FzZSAzOlxuICAgICAgICAgIGZpbiB8PSBkYXRhW3BvcyArIDJdIDw8IDE2O1xuICAgICAgICBjYXNlIDI6XG4gICAgICAgICAgZmluIHw9IGRhdGFbcG9zICsgMV0gPDwgODtcbiAgICAgICAgY2FzZSAxOlxuICAgICAgICAgIGZpbiB8PSBkYXRhW3Bvc107XG4gICAgICB9XG4gICAgICB2MyBePSBmaW47XG4gICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCA1KTtcbiAgICAgIHYxIF49IHYwO1xuICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjMpO1xuICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgIHYzIF49IHYyO1xuICAgICAgdjAgPSBpbnRfMS5hZGQodjAsIHYzKTtcbiAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICB2MyBePSB2MDtcbiAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2MSk7XG4gICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgIHYxIF49IHYyO1xuICAgICAgdjIgPSBpbnRfMS5yb3RsKHYyLCAxNik7XG4gICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCA1KTtcbiAgICAgIHYxIF49IHYwO1xuICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjMpO1xuICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgIHYzIF49IHYyO1xuICAgICAgdjAgPSBpbnRfMS5hZGQodjAsIHYzKTtcbiAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICB2MyBePSB2MDtcbiAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2MSk7XG4gICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgIHYxIF49IHYyO1xuICAgICAgdjIgPSBpbnRfMS5yb3RsKHYyLCAxNik7XG4gICAgICB2MCBePSBmaW47XG4gICAgICB2MiBePSAyNTU7XG4gICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCA1KTtcbiAgICAgIHYxIF49IHYwO1xuICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjMpO1xuICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgIHYzIF49IHYyO1xuICAgICAgdjAgPSBpbnRfMS5hZGQodjAsIHYzKTtcbiAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICB2MyBePSB2MDtcbiAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2MSk7XG4gICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgIHYxIF49IHYyO1xuICAgICAgdjIgPSBpbnRfMS5yb3RsKHYyLCAxNik7XG4gICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCA1KTtcbiAgICAgIHYxIF49IHYwO1xuICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjMpO1xuICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgIHYzIF49IHYyO1xuICAgICAgdjAgPSBpbnRfMS5hZGQodjAsIHYzKTtcbiAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICB2MyBePSB2MDtcbiAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2MSk7XG4gICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgIHYxIF49IHYyO1xuICAgICAgdjIgPSBpbnRfMS5yb3RsKHYyLCAxNik7XG4gICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCA1KTtcbiAgICAgIHYxIF49IHYwO1xuICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjMpO1xuICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgIHYzIF49IHYyO1xuICAgICAgdjAgPSBpbnRfMS5hZGQodjAsIHYzKTtcbiAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICB2MyBePSB2MDtcbiAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2MSk7XG4gICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgIHYxIF49IHYyO1xuICAgICAgdjIgPSBpbnRfMS5yb3RsKHYyLCAxNik7XG4gICAgICB2MCA9IGludF8xLmFkZCh2MCwgdjEpO1xuICAgICAgdjEgPSBpbnRfMS5yb3RsKHYxLCA1KTtcbiAgICAgIHYxIF49IHYwO1xuICAgICAgdjAgPSBpbnRfMS5yb3RsKHYwLCAxNik7XG4gICAgICB2MiA9IGludF8xLmFkZCh2MiwgdjMpO1xuICAgICAgdjMgPSBpbnRfMS5yb3RsKHYzLCA4KTtcbiAgICAgIHYzIF49IHYyO1xuICAgICAgdjAgPSBpbnRfMS5hZGQodjAsIHYzKTtcbiAgICAgIHYzID0gaW50XzEucm90bCh2MywgNyk7XG4gICAgICB2MyBePSB2MDtcbiAgICAgIHYyID0gaW50XzEuYWRkKHYyLCB2MSk7XG4gICAgICB2MSA9IGludF8xLnJvdGwodjEsIDEzKTtcbiAgICAgIHYxIF49IHYyO1xuICAgICAgcmV0dXJuICh2MSBeIHYzKSA+Pj4gMDtcbiAgICB9XG4gICAgZXhwb3J0cy5oYWxmU2lwSGFzaE51bSA9IGhhbGZTaXBIYXNoTnVtO1xuICB9XG59KTtcbnZhciByZXF1aXJlX2NvbnN0YW50X3RpbWUgPSBfX2NvbW1vbkpTKHtcbiAgXCJub2RlX21vZHVsZXMvQHN0YWJsZWxpYi9jb25zdGFudC10aW1lL2xpYi9jb25zdGFudC10aW1lLmpzXCIoZXhwb3J0cykge1xuICAgIE9iamVjdC5kZWZpbmVQcm9wZXJ0eShleHBvcnRzLCBcIl9fZXNNb2R1bGVcIiwge3ZhbHVlOiB0cnVlfSk7XG4gICAgZnVuY3Rpb24gc2VsZWN0KHN1YmplY3QsIHJlc3VsdElmT25lLCByZXN1bHRJZlplcm8pIHtcbiAgICAgIHJldHVybiB+KHN1YmplY3QgLSAxKSAmIHJlc3VsdElmT25lIHwgc3ViamVjdCAtIDEgJiByZXN1bHRJZlplcm87XG4gICAgfVxuICAgIGV4cG9ydHMuc2VsZWN0ID0gc2VsZWN0O1xuICAgIGZ1bmN0aW9uIGxlc3NPckVxdWFsKGEsIGIpIHtcbiAgICAgIHJldHVybiAoYSB8IDApIC0gKGIgfCAwKSAtIDEgPj4+IDMxICYgMTtcbiAgICB9XG4gICAgZXhwb3J0cy5sZXNzT3JFcXVhbCA9IGxlc3NPckVxdWFsO1xuICAgIGZ1bmN0aW9uIGNvbXBhcmUoYSwgYikge1xuICAgICAgaWYgKGEubGVuZ3RoICE9PSBiLmxlbmd0aCkge1xuICAgICAgICByZXR1cm4gMDtcbiAgICAgIH1cbiAgICAgIHZhciByZXN1bHQgPSAwO1xuICAgICAgZm9yICh2YXIgaSA9IDA7IGkgPCBhLmxlbmd0aDsgaSsrKSB7XG4gICAgICAgIHJlc3VsdCB8PSBhW2ldIF4gYltpXTtcbiAgICAgIH1cbiAgICAgIHJldHVybiAxICYgcmVzdWx0IC0gMSA+Pj4gODtcbiAgICB9XG4gICAgZXhwb3J0cy5jb21wYXJlID0gY29tcGFyZTtcbiAgICBmdW5jdGlvbiBlcXVhbDIoYSwgYikge1xuICAgICAgaWYgKGEubGVuZ3RoID09PSAwIHx8IGIubGVuZ3RoID09PSAwKSB7XG4gICAgICAgIHJldHVybiBmYWxzZTtcbiAgICAgIH1cbiAgICAgIHJldHVybiBjb21wYXJlKGEsIGIpICE9PSAwO1xuICAgIH1cbiAgICBleHBvcnRzLmVxdWFsID0gZXF1YWwyO1xuICB9XG59KTtcbnZhciByZXF1aXJlX2xpYiA9IF9fY29tbW9uSlMoe1xuICBcIm5vZGVfbW9kdWxlcy9Ac2N1cmUvYmFzZS9saWIvaW5kZXguanNcIihleHBvcnRzKSB7XG4gICAgT2JqZWN0LmRlZmluZVByb3BlcnR5KGV4cG9ydHMsIFwiX19lc01vZHVsZVwiLCB7dmFsdWU6IHRydWV9KTtcbiAgICBleHBvcnRzLmJ5dGVzID0gZXhwb3J0cy5zdHJpbmdUb0J5dGVzID0gZXhwb3J0cy5zdHIgPSBleHBvcnRzLmJ5dGVzVG9TdHJpbmcgPSBleHBvcnRzLmhleCA9IGV4cG9ydHMudXRmOCA9IGV4cG9ydHMuYmVjaDMybSA9IGV4cG9ydHMuYmVjaDMyID0gZXhwb3J0cy5iYXNlNThjaGVjayA9IGV4cG9ydHMuYmFzZTU4eG1yID0gZXhwb3J0cy5iYXNlNTh4cnAgPSBleHBvcnRzLmJhc2U1OGZsaWNrciA9IGV4cG9ydHMuYmFzZTU4ID0gZXhwb3J0cy5iYXNlNjR1cmwgPSBleHBvcnRzLmJhc2U2NCA9IGV4cG9ydHMuYmFzZTMyY3JvY2tmb3JkID0gZXhwb3J0cy5iYXNlMzJoZXggPSBleHBvcnRzLmJhc2UzMiA9IGV4cG9ydHMuYmFzZTE2ID0gZXhwb3J0cy51dGlscyA9IGV4cG9ydHMuYXNzZXJ0TnVtYmVyID0gdm9pZCAwO1xuICAgIGZ1bmN0aW9uIGFzc2VydE51bWJlcihuKSB7XG4gICAgICBpZiAoIU51bWJlci5pc1NhZmVJbnRlZ2VyKG4pKVxuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYFdyb25nIGludGVnZXI6ICR7bn1gKTtcbiAgICB9XG4gICAgZXhwb3J0cy5hc3NlcnROdW1iZXIgPSBhc3NlcnROdW1iZXI7XG4gICAgZnVuY3Rpb24gY2hhaW4oLi4uYXJncykge1xuICAgICAgY29uc3Qgd3JhcCA9IChhLCBiKSA9PiAoYykgPT4gYShiKGMpKTtcbiAgICAgIGNvbnN0IGVuY29kZTIgPSBBcnJheS5mcm9tKGFyZ3MpLnJldmVyc2UoKS5yZWR1Y2UoKGFjYywgaSkgPT4gYWNjID8gd3JhcChhY2MsIGkuZW5jb2RlKSA6IGkuZW5jb2RlLCB2b2lkIDApO1xuICAgICAgY29uc3QgZGVjb2RlMiA9IGFyZ3MucmVkdWNlKChhY2MsIGkpID0+IGFjYyA/IHdyYXAoYWNjLCBpLmRlY29kZSkgOiBpLmRlY29kZSwgdm9pZCAwKTtcbiAgICAgIHJldHVybiB7ZW5jb2RlOiBlbmNvZGUyLCBkZWNvZGU6IGRlY29kZTJ9O1xuICAgIH1cbiAgICBmdW5jdGlvbiBhbHBoYWJldChhbHBoYWJldDIpIHtcbiAgICAgIHJldHVybiB7XG4gICAgICAgIGVuY29kZTogKGRpZ2l0cykgPT4ge1xuICAgICAgICAgIGlmICghQXJyYXkuaXNBcnJheShkaWdpdHMpIHx8IGRpZ2l0cy5sZW5ndGggJiYgdHlwZW9mIGRpZ2l0c1swXSAhPT0gXCJudW1iZXJcIilcbiAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcImFscGhhYmV0LmVuY29kZSBpbnB1dCBzaG91bGQgYmUgYW4gYXJyYXkgb2YgbnVtYmVyc1wiKTtcbiAgICAgICAgICByZXR1cm4gZGlnaXRzLm1hcCgoaSkgPT4ge1xuICAgICAgICAgICAgYXNzZXJ0TnVtYmVyKGkpO1xuICAgICAgICAgICAgaWYgKGkgPCAwIHx8IGkgPj0gYWxwaGFiZXQyLmxlbmd0aClcbiAgICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBEaWdpdCBpbmRleCBvdXRzaWRlIGFscGhhYmV0OiAke2l9IChhbHBoYWJldDogJHthbHBoYWJldDIubGVuZ3RofSlgKTtcbiAgICAgICAgICAgIHJldHVybiBhbHBoYWJldDJbaV07XG4gICAgICAgICAgfSk7XG4gICAgICAgIH0sXG4gICAgICAgIGRlY29kZTogKGlucHV0KSA9PiB7XG4gICAgICAgICAgaWYgKCFBcnJheS5pc0FycmF5KGlucHV0KSB8fCBpbnB1dC5sZW5ndGggJiYgdHlwZW9mIGlucHV0WzBdICE9PSBcInN0cmluZ1wiKVxuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiYWxwaGFiZXQuZGVjb2RlIGlucHV0IHNob3VsZCBiZSBhcnJheSBvZiBzdHJpbmdzXCIpO1xuICAgICAgICAgIHJldHVybiBpbnB1dC5tYXAoKGxldHRlcikgPT4ge1xuICAgICAgICAgICAgaWYgKHR5cGVvZiBsZXR0ZXIgIT09IFwic3RyaW5nXCIpXG4gICAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgYWxwaGFiZXQuZGVjb2RlOiBub3Qgc3RyaW5nIGVsZW1lbnQ9JHtsZXR0ZXJ9YCk7XG4gICAgICAgICAgICBjb25zdCBpbmRleCA9IGFscGhhYmV0Mi5pbmRleE9mKGxldHRlcik7XG4gICAgICAgICAgICBpZiAoaW5kZXggPT09IC0xKVxuICAgICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYFVua25vd24gbGV0dGVyOiBcIiR7bGV0dGVyfVwiLiBBbGxvd2VkOiAke2FscGhhYmV0Mn1gKTtcbiAgICAgICAgICAgIHJldHVybiBpbmRleDtcbiAgICAgICAgICB9KTtcbiAgICAgICAgfVxuICAgICAgfTtcbiAgICB9XG4gICAgZnVuY3Rpb24gam9pbihzZXBhcmF0b3IgPSBcIlwiKSB7XG4gICAgICBpZiAodHlwZW9mIHNlcGFyYXRvciAhPT0gXCJzdHJpbmdcIilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiam9pbiBzZXBhcmF0b3Igc2hvdWxkIGJlIHN0cmluZ1wiKTtcbiAgICAgIHJldHVybiB7XG4gICAgICAgIGVuY29kZTogKGZyb20pID0+IHtcbiAgICAgICAgICBpZiAoIUFycmF5LmlzQXJyYXkoZnJvbSkgfHwgZnJvbS5sZW5ndGggJiYgdHlwZW9mIGZyb21bMF0gIT09IFwic3RyaW5nXCIpXG4gICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJqb2luLmVuY29kZSBpbnB1dCBzaG91bGQgYmUgYXJyYXkgb2Ygc3RyaW5nc1wiKTtcbiAgICAgICAgICBmb3IgKGxldCBpIG9mIGZyb20pXG4gICAgICAgICAgICBpZiAodHlwZW9mIGkgIT09IFwic3RyaW5nXCIpXG4gICAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgam9pbi5lbmNvZGU6IG5vbi1zdHJpbmcgaW5wdXQ9JHtpfWApO1xuICAgICAgICAgIHJldHVybiBmcm9tLmpvaW4oc2VwYXJhdG9yKTtcbiAgICAgICAgfSxcbiAgICAgICAgZGVjb2RlOiAodG8pID0+IHtcbiAgICAgICAgICBpZiAodHlwZW9mIHRvICE9PSBcInN0cmluZ1wiKVxuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiam9pbi5kZWNvZGUgaW5wdXQgc2hvdWxkIGJlIHN0cmluZ1wiKTtcbiAgICAgICAgICByZXR1cm4gdG8uc3BsaXQoc2VwYXJhdG9yKTtcbiAgICAgICAgfVxuICAgICAgfTtcbiAgICB9XG4gICAgZnVuY3Rpb24gcGFkZGluZyhiaXRzLCBjaHIgPSBcIj1cIikge1xuICAgICAgYXNzZXJ0TnVtYmVyKGJpdHMpO1xuICAgICAgaWYgKHR5cGVvZiBjaHIgIT09IFwic3RyaW5nXCIpXG4gICAgICAgIHRocm93IG5ldyBFcnJvcihcInBhZGRpbmcgY2hyIHNob3VsZCBiZSBzdHJpbmdcIik7XG4gICAgICByZXR1cm4ge1xuICAgICAgICBlbmNvZGUoZGF0YSkge1xuICAgICAgICAgIGlmICghQXJyYXkuaXNBcnJheShkYXRhKSB8fCBkYXRhLmxlbmd0aCAmJiB0eXBlb2YgZGF0YVswXSAhPT0gXCJzdHJpbmdcIilcbiAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcInBhZGRpbmcuZW5jb2RlIGlucHV0IHNob3VsZCBiZSBhcnJheSBvZiBzdHJpbmdzXCIpO1xuICAgICAgICAgIGZvciAobGV0IGkgb2YgZGF0YSlcbiAgICAgICAgICAgIGlmICh0eXBlb2YgaSAhPT0gXCJzdHJpbmdcIilcbiAgICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBwYWRkaW5nLmVuY29kZTogbm9uLXN0cmluZyBpbnB1dD0ke2l9YCk7XG4gICAgICAgICAgd2hpbGUgKGRhdGEubGVuZ3RoICogYml0cyAlIDgpXG4gICAgICAgICAgICBkYXRhLnB1c2goY2hyKTtcbiAgICAgICAgICByZXR1cm4gZGF0YTtcbiAgICAgICAgfSxcbiAgICAgICAgZGVjb2RlKGlucHV0KSB7XG4gICAgICAgICAgaWYgKCFBcnJheS5pc0FycmF5KGlucHV0KSB8fCBpbnB1dC5sZW5ndGggJiYgdHlwZW9mIGlucHV0WzBdICE9PSBcInN0cmluZ1wiKVxuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicGFkZGluZy5lbmNvZGUgaW5wdXQgc2hvdWxkIGJlIGFycmF5IG9mIHN0cmluZ3NcIik7XG4gICAgICAgICAgZm9yIChsZXQgaSBvZiBpbnB1dClcbiAgICAgICAgICAgIGlmICh0eXBlb2YgaSAhPT0gXCJzdHJpbmdcIilcbiAgICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBwYWRkaW5nLmRlY29kZTogbm9uLXN0cmluZyBpbnB1dD0ke2l9YCk7XG4gICAgICAgICAgbGV0IGVuZCA9IGlucHV0Lmxlbmd0aDtcbiAgICAgICAgICBpZiAoZW5kICogYml0cyAlIDgpXG4gICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJJbnZhbGlkIHBhZGRpbmc6IHN0cmluZyBzaG91bGQgaGF2ZSB3aG9sZSBudW1iZXIgb2YgYnl0ZXNcIik7XG4gICAgICAgICAgZm9yICg7IGVuZCA+IDAgJiYgaW5wdXRbZW5kIC0gMV0gPT09IGNocjsgZW5kLS0pIHtcbiAgICAgICAgICAgIGlmICghKChlbmQgLSAxKSAqIGJpdHMgJSA4KSlcbiAgICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiSW52YWxpZCBwYWRkaW5nOiBzdHJpbmcgaGFzIHRvbyBtdWNoIHBhZGRpbmdcIik7XG4gICAgICAgICAgfVxuICAgICAgICAgIHJldHVybiBpbnB1dC5zbGljZSgwLCBlbmQpO1xuICAgICAgICB9XG4gICAgICB9O1xuICAgIH1cbiAgICBmdW5jdGlvbiBub3JtYWxpemUoZm4pIHtcbiAgICAgIGlmICh0eXBlb2YgZm4gIT09IFwiZnVuY3Rpb25cIilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwibm9ybWFsaXplIGZuIHNob3VsZCBiZSBmdW5jdGlvblwiKTtcbiAgICAgIHJldHVybiB7ZW5jb2RlOiAoZnJvbSkgPT4gZnJvbSwgZGVjb2RlOiAodG8pID0+IGZuKHRvKX07XG4gICAgfVxuICAgIGZ1bmN0aW9uIGNvbnZlcnRSYWRpeChkYXRhLCBmcm9tLCB0bykge1xuICAgICAgaWYgKGZyb20gPCAyKVxuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYGNvbnZlcnRSYWRpeDogd3JvbmcgZnJvbT0ke2Zyb219LCBiYXNlIGNhbm5vdCBiZSBsZXNzIHRoYW4gMmApO1xuICAgICAgaWYgKHRvIDwgMilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBjb252ZXJ0UmFkaXg6IHdyb25nIHRvPSR7dG99LCBiYXNlIGNhbm5vdCBiZSBsZXNzIHRoYW4gMmApO1xuICAgICAgaWYgKCFBcnJheS5pc0FycmF5KGRhdGEpKVxuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJjb252ZXJ0UmFkaXg6IGRhdGEgc2hvdWxkIGJlIGFycmF5XCIpO1xuICAgICAgaWYgKCFkYXRhLmxlbmd0aClcbiAgICAgICAgcmV0dXJuIFtdO1xuICAgICAgbGV0IHBvcyA9IDA7XG4gICAgICBjb25zdCByZXMgPSBbXTtcbiAgICAgIGNvbnN0IGRpZ2l0cyA9IEFycmF5LmZyb20oZGF0YSk7XG4gICAgICBkaWdpdHMuZm9yRWFjaCgoZCkgPT4ge1xuICAgICAgICBhc3NlcnROdW1iZXIoZCk7XG4gICAgICAgIGlmIChkIDwgMCB8fCBkID49IGZyb20pXG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBXcm9uZyBpbnRlZ2VyOiAke2R9YCk7XG4gICAgICB9KTtcbiAgICAgIHdoaWxlICh0cnVlKSB7XG4gICAgICAgIGxldCBjYXJyeSA9IDA7XG4gICAgICAgIGxldCBkb25lID0gdHJ1ZTtcbiAgICAgICAgZm9yIChsZXQgaSA9IHBvczsgaSA8IGRpZ2l0cy5sZW5ndGg7IGkrKykge1xuICAgICAgICAgIGNvbnN0IGRpZ2l0ID0gZGlnaXRzW2ldO1xuICAgICAgICAgIGNvbnN0IGRpZ2l0QmFzZSA9IGZyb20gKiBjYXJyeSArIGRpZ2l0O1xuICAgICAgICAgIGlmICghTnVtYmVyLmlzU2FmZUludGVnZXIoZGlnaXRCYXNlKSB8fCBmcm9tICogY2FycnkgLyBmcm9tICE9PSBjYXJyeSB8fCBkaWdpdEJhc2UgLSBkaWdpdCAhPT0gZnJvbSAqIGNhcnJ5KSB7XG4gICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJjb252ZXJ0UmFkaXg6IGNhcnJ5IG92ZXJmbG93XCIpO1xuICAgICAgICAgIH1cbiAgICAgICAgICBjYXJyeSA9IGRpZ2l0QmFzZSAlIHRvO1xuICAgICAgICAgIGRpZ2l0c1tpXSA9IE1hdGguZmxvb3IoZGlnaXRCYXNlIC8gdG8pO1xuICAgICAgICAgIGlmICghTnVtYmVyLmlzU2FmZUludGVnZXIoZGlnaXRzW2ldKSB8fCBkaWdpdHNbaV0gKiB0byArIGNhcnJ5ICE9PSBkaWdpdEJhc2UpXG4gICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJjb252ZXJ0UmFkaXg6IGNhcnJ5IG92ZXJmbG93XCIpO1xuICAgICAgICAgIGlmICghZG9uZSlcbiAgICAgICAgICAgIGNvbnRpbnVlO1xuICAgICAgICAgIGVsc2UgaWYgKCFkaWdpdHNbaV0pXG4gICAgICAgICAgICBwb3MgPSBpO1xuICAgICAgICAgIGVsc2VcbiAgICAgICAgICAgIGRvbmUgPSBmYWxzZTtcbiAgICAgICAgfVxuICAgICAgICByZXMucHVzaChjYXJyeSk7XG4gICAgICAgIGlmIChkb25lKVxuICAgICAgICAgIGJyZWFrO1xuICAgICAgfVxuICAgICAgZm9yIChsZXQgaSA9IDA7IGkgPCBkYXRhLmxlbmd0aCAtIDEgJiYgZGF0YVtpXSA9PT0gMDsgaSsrKVxuICAgICAgICByZXMucHVzaCgwKTtcbiAgICAgIHJldHVybiByZXMucmV2ZXJzZSgpO1xuICAgIH1cbiAgICB2YXIgZ2NkID0gKGEsIGIpID0+ICFiID8gYSA6IGdjZChiLCBhICUgYik7XG4gICAgdmFyIHJhZGl4MmNhcnJ5ID0gKGZyb20sIHRvKSA9PiBmcm9tICsgKHRvIC0gZ2NkKGZyb20sIHRvKSk7XG4gICAgZnVuY3Rpb24gY29udmVydFJhZGl4MihkYXRhLCBmcm9tLCB0bywgcGFkZGluZzIpIHtcbiAgICAgIGlmICghQXJyYXkuaXNBcnJheShkYXRhKSlcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiY29udmVydFJhZGl4MjogZGF0YSBzaG91bGQgYmUgYXJyYXlcIik7XG4gICAgICBpZiAoZnJvbSA8PSAwIHx8IGZyb20gPiAzMilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBjb252ZXJ0UmFkaXgyOiB3cm9uZyBmcm9tPSR7ZnJvbX1gKTtcbiAgICAgIGlmICh0byA8PSAwIHx8IHRvID4gMzIpXG4gICAgICAgIHRocm93IG5ldyBFcnJvcihgY29udmVydFJhZGl4Mjogd3JvbmcgdG89JHt0b31gKTtcbiAgICAgIGlmIChyYWRpeDJjYXJyeShmcm9tLCB0bykgPiAzMikge1xuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYGNvbnZlcnRSYWRpeDI6IGNhcnJ5IG92ZXJmbG93IGZyb209JHtmcm9tfSB0bz0ke3RvfSBjYXJyeUJpdHM9JHtyYWRpeDJjYXJyeShmcm9tLCB0byl9YCk7XG4gICAgICB9XG4gICAgICBsZXQgY2FycnkgPSAwO1xuICAgICAgbGV0IHBvcyA9IDA7XG4gICAgICBjb25zdCBtYXNrID0gMiAqKiB0byAtIDE7XG4gICAgICBjb25zdCByZXMgPSBbXTtcbiAgICAgIGZvciAoY29uc3QgbiBvZiBkYXRhKSB7XG4gICAgICAgIGFzc2VydE51bWJlcihuKTtcbiAgICAgICAgaWYgKG4gPj0gMiAqKiBmcm9tKVxuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgY29udmVydFJhZGl4MjogaW52YWxpZCBkYXRhIHdvcmQ9JHtufSBmcm9tPSR7ZnJvbX1gKTtcbiAgICAgICAgY2FycnkgPSBjYXJyeSA8PCBmcm9tIHwgbjtcbiAgICAgICAgaWYgKHBvcyArIGZyb20gPiAzMilcbiAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYGNvbnZlcnRSYWRpeDI6IGNhcnJ5IG92ZXJmbG93IHBvcz0ke3Bvc30gZnJvbT0ke2Zyb219YCk7XG4gICAgICAgIHBvcyArPSBmcm9tO1xuICAgICAgICBmb3IgKDsgcG9zID49IHRvOyBwb3MgLT0gdG8pXG4gICAgICAgICAgcmVzLnB1c2goKGNhcnJ5ID4+IHBvcyAtIHRvICYgbWFzaykgPj4+IDApO1xuICAgICAgICBjYXJyeSAmPSAyICoqIHBvcyAtIDE7XG4gICAgICB9XG4gICAgICBjYXJyeSA9IGNhcnJ5IDw8IHRvIC0gcG9zICYgbWFzaztcbiAgICAgIGlmICghcGFkZGluZzIgJiYgcG9zID49IGZyb20pXG4gICAgICAgIHRocm93IG5ldyBFcnJvcihcIkV4Y2VzcyBwYWRkaW5nXCIpO1xuICAgICAgaWYgKCFwYWRkaW5nMiAmJiBjYXJyeSlcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBOb24temVybyBwYWRkaW5nOiAke2NhcnJ5fWApO1xuICAgICAgaWYgKHBhZGRpbmcyICYmIHBvcyA+IDApXG4gICAgICAgIHJlcy5wdXNoKGNhcnJ5ID4+PiAwKTtcbiAgICAgIHJldHVybiByZXM7XG4gICAgfVxuICAgIGZ1bmN0aW9uIHJhZGl4KG51bSkge1xuICAgICAgYXNzZXJ0TnVtYmVyKG51bSk7XG4gICAgICByZXR1cm4ge1xuICAgICAgICBlbmNvZGU6IChieXRlczIpID0+IHtcbiAgICAgICAgICBpZiAoIShieXRlczIgaW5zdGFuY2VvZiBVaW50OEFycmF5KSlcbiAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcInJhZGl4LmVuY29kZSBpbnB1dCBzaG91bGQgYmUgVWludDhBcnJheVwiKTtcbiAgICAgICAgICByZXR1cm4gY29udmVydFJhZGl4KEFycmF5LmZyb20oYnl0ZXMyKSwgMiAqKiA4LCBudW0pO1xuICAgICAgICB9LFxuICAgICAgICBkZWNvZGU6IChkaWdpdHMpID0+IHtcbiAgICAgICAgICBpZiAoIUFycmF5LmlzQXJyYXkoZGlnaXRzKSB8fCBkaWdpdHMubGVuZ3RoICYmIHR5cGVvZiBkaWdpdHNbMF0gIT09IFwibnVtYmVyXCIpXG4gICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJyYWRpeC5kZWNvZGUgaW5wdXQgc2hvdWxkIGJlIGFycmF5IG9mIHN0cmluZ3NcIik7XG4gICAgICAgICAgcmV0dXJuIFVpbnQ4QXJyYXkuZnJvbShjb252ZXJ0UmFkaXgoZGlnaXRzLCBudW0sIDIgKiogOCkpO1xuICAgICAgICB9XG4gICAgICB9O1xuICAgIH1cbiAgICBmdW5jdGlvbiByYWRpeDIoYml0cywgcmV2UGFkZGluZyA9IGZhbHNlKSB7XG4gICAgICBhc3NlcnROdW1iZXIoYml0cyk7XG4gICAgICBpZiAoYml0cyA8PSAwIHx8IGJpdHMgPiAzMilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicmFkaXgyOiBiaXRzIHNob3VsZCBiZSBpbiAoMC4uMzJdXCIpO1xuICAgICAgaWYgKHJhZGl4MmNhcnJ5KDgsIGJpdHMpID4gMzIgfHwgcmFkaXgyY2FycnkoYml0cywgOCkgPiAzMilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicmFkaXgyOiBjYXJyeSBvdmVyZmxvd1wiKTtcbiAgICAgIHJldHVybiB7XG4gICAgICAgIGVuY29kZTogKGJ5dGVzMikgPT4ge1xuICAgICAgICAgIGlmICghKGJ5dGVzMiBpbnN0YW5jZW9mIFVpbnQ4QXJyYXkpKVxuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicmFkaXgyLmVuY29kZSBpbnB1dCBzaG91bGQgYmUgVWludDhBcnJheVwiKTtcbiAgICAgICAgICByZXR1cm4gY29udmVydFJhZGl4MihBcnJheS5mcm9tKGJ5dGVzMiksIDgsIGJpdHMsICFyZXZQYWRkaW5nKTtcbiAgICAgICAgfSxcbiAgICAgICAgZGVjb2RlOiAoZGlnaXRzKSA9PiB7XG4gICAgICAgICAgaWYgKCFBcnJheS5pc0FycmF5KGRpZ2l0cykgfHwgZGlnaXRzLmxlbmd0aCAmJiB0eXBlb2YgZGlnaXRzWzBdICE9PSBcIm51bWJlclwiKVxuICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwicmFkaXgyLmRlY29kZSBpbnB1dCBzaG91bGQgYmUgYXJyYXkgb2Ygc3RyaW5nc1wiKTtcbiAgICAgICAgICByZXR1cm4gVWludDhBcnJheS5mcm9tKGNvbnZlcnRSYWRpeDIoZGlnaXRzLCBiaXRzLCA4LCByZXZQYWRkaW5nKSk7XG4gICAgICAgIH1cbiAgICAgIH07XG4gICAgfVxuICAgIGZ1bmN0aW9uIHVuc2FmZVdyYXBwZXIoZm4pIHtcbiAgICAgIGlmICh0eXBlb2YgZm4gIT09IFwiZnVuY3Rpb25cIilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwidW5zYWZlV3JhcHBlciBmbiBzaG91bGQgYmUgZnVuY3Rpb25cIik7XG4gICAgICByZXR1cm4gZnVuY3Rpb24oLi4uYXJncykge1xuICAgICAgICB0cnkge1xuICAgICAgICAgIHJldHVybiBmbi5hcHBseShudWxsLCBhcmdzKTtcbiAgICAgICAgfSBjYXRjaCAoZSkge1xuICAgICAgICB9XG4gICAgICB9O1xuICAgIH1cbiAgICBmdW5jdGlvbiBjaGVja3N1bShsZW4sIGZuKSB7XG4gICAgICBhc3NlcnROdW1iZXIobGVuKTtcbiAgICAgIGlmICh0eXBlb2YgZm4gIT09IFwiZnVuY3Rpb25cIilcbiAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiY2hlY2tzdW0gZm4gc2hvdWxkIGJlIGZ1bmN0aW9uXCIpO1xuICAgICAgcmV0dXJuIHtcbiAgICAgICAgZW5jb2RlKGRhdGEpIHtcbiAgICAgICAgICBpZiAoIShkYXRhIGluc3RhbmNlb2YgVWludDhBcnJheSkpXG4gICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJjaGVja3N1bS5lbmNvZGU6IGlucHV0IHNob3VsZCBiZSBVaW50OEFycmF5XCIpO1xuICAgICAgICAgIGNvbnN0IGNoZWNrc3VtMiA9IGZuKGRhdGEpLnNsaWNlKDAsIGxlbik7XG4gICAgICAgICAgY29uc3QgcmVzID0gbmV3IFVpbnQ4QXJyYXkoZGF0YS5sZW5ndGggKyBsZW4pO1xuICAgICAgICAgIHJlcy5zZXQoZGF0YSk7XG4gICAgICAgICAgcmVzLnNldChjaGVja3N1bTIsIGRhdGEubGVuZ3RoKTtcbiAgICAgICAgICByZXR1cm4gcmVzO1xuICAgICAgICB9LFxuICAgICAgICBkZWNvZGUoZGF0YSkge1xuICAgICAgICAgIGlmICghKGRhdGEgaW5zdGFuY2VvZiBVaW50OEFycmF5KSlcbiAgICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcImNoZWNrc3VtLmRlY29kZTogaW5wdXQgc2hvdWxkIGJlIFVpbnQ4QXJyYXlcIik7XG4gICAgICAgICAgY29uc3QgcGF5bG9hZCA9IGRhdGEuc2xpY2UoMCwgLWxlbik7XG4gICAgICAgICAgY29uc3QgbmV3Q2hlY2tzdW0gPSBmbihwYXlsb2FkKS5zbGljZSgwLCBsZW4pO1xuICAgICAgICAgIGNvbnN0IG9sZENoZWNrc3VtID0gZGF0YS5zbGljZSgtbGVuKTtcbiAgICAgICAgICBmb3IgKGxldCBpID0gMDsgaSA8IGxlbjsgaSsrKVxuICAgICAgICAgICAgaWYgKG5ld0NoZWNrc3VtW2ldICE9PSBvbGRDaGVja3N1bVtpXSlcbiAgICAgICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiSW52YWxpZCBjaGVja3N1bVwiKTtcbiAgICAgICAgICByZXR1cm4gcGF5bG9hZDtcbiAgICAgICAgfVxuICAgICAgfTtcbiAgICB9XG4gICAgZXhwb3J0cy51dGlscyA9IHthbHBoYWJldCwgY2hhaW4sIGNoZWNrc3VtLCByYWRpeCwgcmFkaXgyLCBqb2luLCBwYWRkaW5nfTtcbiAgICBleHBvcnRzLmJhc2UxNiA9IGNoYWluKHJhZGl4Mig0KSwgYWxwaGFiZXQoXCIwMTIzNDU2Nzg5QUJDREVGXCIpLCBqb2luKFwiXCIpKTtcbiAgICBleHBvcnRzLmJhc2UzMiA9IGNoYWluKHJhZGl4Mig1KSwgYWxwaGFiZXQoXCJBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWjIzNDU2N1wiKSwgcGFkZGluZyg1KSwgam9pbihcIlwiKSk7XG4gICAgZXhwb3J0cy5iYXNlMzJoZXggPSBjaGFpbihyYWRpeDIoNSksIGFscGhhYmV0KFwiMDEyMzQ1Njc4OUFCQ0RFRkdISUpLTE1OT1BRUlNUVVZcIiksIHBhZGRpbmcoNSksIGpvaW4oXCJcIikpO1xuICAgIGV4cG9ydHMuYmFzZTMyY3JvY2tmb3JkID0gY2hhaW4ocmFkaXgyKDUpLCBhbHBoYWJldChcIjAxMjM0NTY3ODlBQkNERUZHSEpLTU5QUVJTVFZXWFlaXCIpLCBqb2luKFwiXCIpLCBub3JtYWxpemUoKHMpID0+IHMudG9VcHBlckNhc2UoKS5yZXBsYWNlKC9PL2csIFwiMFwiKS5yZXBsYWNlKC9bSUxdL2csIFwiMVwiKSkpO1xuICAgIGV4cG9ydHMuYmFzZTY0ID0gY2hhaW4ocmFkaXgyKDYpLCBhbHBoYWJldChcIkFCQ0RFRkdISUpLTE1OT1BRUlNUVVZXWFlaYWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5Ky9cIiksIHBhZGRpbmcoNiksIGpvaW4oXCJcIikpO1xuICAgIGV4cG9ydHMuYmFzZTY0dXJsID0gY2hhaW4ocmFkaXgyKDYpLCBhbHBoYWJldChcIkFCQ0RFRkdISUpLTE1OT1BRUlNUVVZXWFlaYWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXowMTIzNDU2Nzg5LV9cIiksIHBhZGRpbmcoNiksIGpvaW4oXCJcIikpO1xuICAgIHZhciBnZW5CYXNlNTggPSAoYWJjKSA9PiBjaGFpbihyYWRpeCg1OCksIGFscGhhYmV0KGFiYyksIGpvaW4oXCJcIikpO1xuICAgIGV4cG9ydHMuYmFzZTU4ID0gZ2VuQmFzZTU4KFwiMTIzNDU2Nzg5QUJDREVGR0hKS0xNTlBRUlNUVVZXWFlaYWJjZGVmZ2hpamttbm9wcXJzdHV2d3h5elwiKTtcbiAgICBleHBvcnRzLmJhc2U1OGZsaWNrciA9IGdlbkJhc2U1OChcIjEyMzQ1Njc4OWFiY2RlZmdoaWprbW5vcHFyc3R1dnd4eXpBQkNERUZHSEpLTE1OUFFSU1RVVldYWVpcIik7XG4gICAgZXhwb3J0cy5iYXNlNTh4cnAgPSBnZW5CYXNlNTgoXCJycHNobmFmMzl3QlVETkVHSEpLTE00UFFSU1Q3VldYWVoyYmNkZUNnNjVqa204b0ZxaTF0dXZBeHl6XCIpO1xuICAgIHZhciBYTVJfQkxPQ0tfTEVOID0gWzAsIDIsIDMsIDUsIDYsIDcsIDksIDEwLCAxMV07XG4gICAgZXhwb3J0cy5iYXNlNTh4bXIgPSB7XG4gICAgICBlbmNvZGUoZGF0YSkge1xuICAgICAgICBsZXQgcmVzID0gXCJcIjtcbiAgICAgICAgZm9yIChsZXQgaSA9IDA7IGkgPCBkYXRhLmxlbmd0aDsgaSArPSA4KSB7XG4gICAgICAgICAgY29uc3QgYmxvY2sgPSBkYXRhLnN1YmFycmF5KGksIGkgKyA4KTtcbiAgICAgICAgICByZXMgKz0gZXhwb3J0cy5iYXNlNTguZW5jb2RlKGJsb2NrKS5wYWRTdGFydChYTVJfQkxPQ0tfTEVOW2Jsb2NrLmxlbmd0aF0sIFwiMVwiKTtcbiAgICAgICAgfVxuICAgICAgICByZXR1cm4gcmVzO1xuICAgICAgfSxcbiAgICAgIGRlY29kZShzdHIpIHtcbiAgICAgICAgbGV0IHJlcyA9IFtdO1xuICAgICAgICBmb3IgKGxldCBpID0gMDsgaSA8IHN0ci5sZW5ndGg7IGkgKz0gMTEpIHtcbiAgICAgICAgICBjb25zdCBzbGljZSA9IHN0ci5zbGljZShpLCBpICsgMTEpO1xuICAgICAgICAgIGNvbnN0IGJsb2NrTGVuID0gWE1SX0JMT0NLX0xFTi5pbmRleE9mKHNsaWNlLmxlbmd0aCk7XG4gICAgICAgICAgY29uc3QgYmxvY2sgPSBleHBvcnRzLmJhc2U1OC5kZWNvZGUoc2xpY2UpO1xuICAgICAgICAgIGZvciAobGV0IGogPSAwOyBqIDwgYmxvY2subGVuZ3RoIC0gYmxvY2tMZW47IGorKykge1xuICAgICAgICAgICAgaWYgKGJsb2NrW2pdICE9PSAwKVxuICAgICAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXCJiYXNlNTh4bXI6IHdyb25nIHBhZGRpbmdcIik7XG4gICAgICAgICAgfVxuICAgICAgICAgIHJlcyA9IHJlcy5jb25jYXQoQXJyYXkuZnJvbShibG9jay5zbGljZShibG9jay5sZW5ndGggLSBibG9ja0xlbikpKTtcbiAgICAgICAgfVxuICAgICAgICByZXR1cm4gVWludDhBcnJheS5mcm9tKHJlcyk7XG4gICAgICB9XG4gICAgfTtcbiAgICB2YXIgYmFzZTU4Y2hlY2syID0gKHNoYTI1NjIpID0+IGNoYWluKGNoZWNrc3VtKDQsIChkYXRhKSA9PiBzaGEyNTYyKHNoYTI1NjIoZGF0YSkpKSwgZXhwb3J0cy5iYXNlNTgpO1xuICAgIGV4cG9ydHMuYmFzZTU4Y2hlY2sgPSBiYXNlNThjaGVjazI7XG4gICAgdmFyIEJFQ0hfQUxQSEFCRVQgPSBjaGFpbihhbHBoYWJldChcInFwenJ5OXg4Z2YydHZkdzBzM2puNTRraGNlNm11YTdsXCIpLCBqb2luKFwiXCIpKTtcbiAgICB2YXIgUE9MWU1PRF9HRU5FUkFUT1JTID0gWzk5NjgyNTAxMCwgNjQyODEzNTQ5LCA1MTM4NzQ0MjYsIDEwMjc3NDg4MjksIDcwNTk3OTA1OV07XG4gICAgZnVuY3Rpb24gYmVjaDMyUG9seW1vZChwcmUpIHtcbiAgICAgIGNvbnN0IGIgPSBwcmUgPj4gMjU7XG4gICAgICBsZXQgY2hrID0gKHByZSAmIDMzNTU0NDMxKSA8PCA1O1xuICAgICAgZm9yIChsZXQgaSA9IDA7IGkgPCBQT0xZTU9EX0dFTkVSQVRPUlMubGVuZ3RoOyBpKyspIHtcbiAgICAgICAgaWYgKChiID4+IGkgJiAxKSA9PT0gMSlcbiAgICAgICAgICBjaGsgXj0gUE9MWU1PRF9HRU5FUkFUT1JTW2ldO1xuICAgICAgfVxuICAgICAgcmV0dXJuIGNoaztcbiAgICB9XG4gICAgZnVuY3Rpb24gYmVjaENoZWNrc3VtKHByZWZpeCwgd29yZHMsIGVuY29kaW5nQ29uc3QgPSAxKSB7XG4gICAgICBjb25zdCBsZW4gPSBwcmVmaXgubGVuZ3RoO1xuICAgICAgbGV0IGNoayA9IDE7XG4gICAgICBmb3IgKGxldCBpID0gMDsgaSA8IGxlbjsgaSsrKSB7XG4gICAgICAgIGNvbnN0IGMgPSBwcmVmaXguY2hhckNvZGVBdChpKTtcbiAgICAgICAgaWYgKGMgPCAzMyB8fCBjID4gMTI2KVxuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgSW52YWxpZCBwcmVmaXggKCR7cHJlZml4fSlgKTtcbiAgICAgICAgY2hrID0gYmVjaDMyUG9seW1vZChjaGspIF4gYyA+PiA1O1xuICAgICAgfVxuICAgICAgY2hrID0gYmVjaDMyUG9seW1vZChjaGspO1xuICAgICAgZm9yIChsZXQgaSA9IDA7IGkgPCBsZW47IGkrKylcbiAgICAgICAgY2hrID0gYmVjaDMyUG9seW1vZChjaGspIF4gcHJlZml4LmNoYXJDb2RlQXQoaSkgJiAzMTtcbiAgICAgIGZvciAobGV0IHYgb2Ygd29yZHMpXG4gICAgICAgIGNoayA9IGJlY2gzMlBvbHltb2QoY2hrKSBeIHY7XG4gICAgICBmb3IgKGxldCBpID0gMDsgaSA8IDY7IGkrKylcbiAgICAgICAgY2hrID0gYmVjaDMyUG9seW1vZChjaGspO1xuICAgICAgY2hrIF49IGVuY29kaW5nQ29uc3Q7XG4gICAgICByZXR1cm4gQkVDSF9BTFBIQUJFVC5lbmNvZGUoY29udmVydFJhZGl4MihbY2hrICUgMiAqKiAzMF0sIDMwLCA1LCBmYWxzZSkpO1xuICAgIH1cbiAgICBmdW5jdGlvbiBnZW5CZWNoMzIoZW5jb2RpbmcpIHtcbiAgICAgIGNvbnN0IEVOQ09ESU5HX0NPTlNUID0gZW5jb2RpbmcgPT09IFwiYmVjaDMyXCIgPyAxIDogNzM0NTM5OTM5O1xuICAgICAgY29uc3QgX3dvcmRzID0gcmFkaXgyKDUpO1xuICAgICAgY29uc3QgZnJvbVdvcmRzID0gX3dvcmRzLmRlY29kZTtcbiAgICAgIGNvbnN0IHRvV29yZHMgPSBfd29yZHMuZW5jb2RlO1xuICAgICAgY29uc3QgZnJvbVdvcmRzVW5zYWZlID0gdW5zYWZlV3JhcHBlcihmcm9tV29yZHMpO1xuICAgICAgZnVuY3Rpb24gZW5jb2RlMihwcmVmaXgsIHdvcmRzLCBsaW1pdCA9IDkwKSB7XG4gICAgICAgIGlmICh0eXBlb2YgcHJlZml4ICE9PSBcInN0cmluZ1wiKVxuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgYmVjaDMyLmVuY29kZSBwcmVmaXggc2hvdWxkIGJlIHN0cmluZywgbm90ICR7dHlwZW9mIHByZWZpeH1gKTtcbiAgICAgICAgaWYgKCFBcnJheS5pc0FycmF5KHdvcmRzKSB8fCB3b3Jkcy5sZW5ndGggJiYgdHlwZW9mIHdvcmRzWzBdICE9PSBcIm51bWJlclwiKVxuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgYmVjaDMyLmVuY29kZSB3b3JkcyBzaG91bGQgYmUgYXJyYXkgb2YgbnVtYmVycywgbm90ICR7dHlwZW9mIHdvcmRzfWApO1xuICAgICAgICBjb25zdCBhY3R1YWxMZW5ndGggPSBwcmVmaXgubGVuZ3RoICsgNyArIHdvcmRzLmxlbmd0aDtcbiAgICAgICAgaWYgKGxpbWl0ICE9PSBmYWxzZSAmJiBhY3R1YWxMZW5ndGggPiBsaW1pdClcbiAgICAgICAgICB0aHJvdyBuZXcgVHlwZUVycm9yKGBMZW5ndGggJHthY3R1YWxMZW5ndGh9IGV4Y2VlZHMgbGltaXQgJHtsaW1pdH1gKTtcbiAgICAgICAgcHJlZml4ID0gcHJlZml4LnRvTG93ZXJDYXNlKCk7XG4gICAgICAgIHJldHVybiBgJHtwcmVmaXh9MSR7QkVDSF9BTFBIQUJFVC5lbmNvZGUod29yZHMpfSR7YmVjaENoZWNrc3VtKHByZWZpeCwgd29yZHMsIEVOQ09ESU5HX0NPTlNUKX1gO1xuICAgICAgfVxuICAgICAgZnVuY3Rpb24gZGVjb2RlMihzdHIsIGxpbWl0ID0gOTApIHtcbiAgICAgICAgaWYgKHR5cGVvZiBzdHIgIT09IFwic3RyaW5nXCIpXG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBiZWNoMzIuZGVjb2RlIGlucHV0IHNob3VsZCBiZSBzdHJpbmcsIG5vdCAke3R5cGVvZiBzdHJ9YCk7XG4gICAgICAgIGlmIChzdHIubGVuZ3RoIDwgOCB8fCBsaW1pdCAhPT0gZmFsc2UgJiYgc3RyLmxlbmd0aCA+IGxpbWl0KVxuICAgICAgICAgIHRocm93IG5ldyBUeXBlRXJyb3IoYFdyb25nIHN0cmluZyBsZW5ndGg6ICR7c3RyLmxlbmd0aH0gKCR7c3RyfSkuIEV4cGVjdGVkICg4Li4ke2xpbWl0fSlgKTtcbiAgICAgICAgY29uc3QgbG93ZXJlZCA9IHN0ci50b0xvd2VyQ2FzZSgpO1xuICAgICAgICBpZiAoc3RyICE9PSBsb3dlcmVkICYmIHN0ciAhPT0gc3RyLnRvVXBwZXJDYXNlKCkpXG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBTdHJpbmcgbXVzdCBiZSBsb3dlcmNhc2Ugb3IgdXBwZXJjYXNlYCk7XG4gICAgICAgIHN0ciA9IGxvd2VyZWQ7XG4gICAgICAgIGNvbnN0IHNlcEluZGV4ID0gc3RyLmxhc3RJbmRleE9mKFwiMVwiKTtcbiAgICAgICAgaWYgKHNlcEluZGV4ID09PSAwIHx8IHNlcEluZGV4ID09PSAtMSlcbiAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYExldHRlciBcIjFcIiBtdXN0IGJlIHByZXNlbnQgYmV0d2VlbiBwcmVmaXggYW5kIGRhdGEgb25seWApO1xuICAgICAgICBjb25zdCBwcmVmaXggPSBzdHIuc2xpY2UoMCwgc2VwSW5kZXgpO1xuICAgICAgICBjb25zdCBfd29yZHMyID0gc3RyLnNsaWNlKHNlcEluZGV4ICsgMSk7XG4gICAgICAgIGlmIChfd29yZHMyLmxlbmd0aCA8IDYpXG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiRGF0YSBtdXN0IGJlIGF0IGxlYXN0IDYgY2hhcmFjdGVycyBsb25nXCIpO1xuICAgICAgICBjb25zdCB3b3JkcyA9IEJFQ0hfQUxQSEFCRVQuZGVjb2RlKF93b3JkczIpLnNsaWNlKDAsIC02KTtcbiAgICAgICAgY29uc3Qgc3VtID0gYmVjaENoZWNrc3VtKHByZWZpeCwgd29yZHMsIEVOQ09ESU5HX0NPTlNUKTtcbiAgICAgICAgaWYgKCFfd29yZHMyLmVuZHNXaXRoKHN1bSkpXG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKGBJbnZhbGlkIGNoZWNrc3VtIGluICR7c3RyfTogZXhwZWN0ZWQgXCIke3N1bX1cImApO1xuICAgICAgICByZXR1cm4ge3ByZWZpeCwgd29yZHN9O1xuICAgICAgfVxuICAgICAgY29uc3QgZGVjb2RlVW5zYWZlID0gdW5zYWZlV3JhcHBlcihkZWNvZGUyKTtcbiAgICAgIGZ1bmN0aW9uIGRlY29kZVRvQnl0ZXMoc3RyKSB7XG4gICAgICAgIGNvbnN0IHtwcmVmaXgsIHdvcmRzfSA9IGRlY29kZTIoc3RyLCBmYWxzZSk7XG4gICAgICAgIHJldHVybiB7cHJlZml4LCB3b3JkcywgYnl0ZXM6IGZyb21Xb3Jkcyh3b3Jkcyl9O1xuICAgICAgfVxuICAgICAgcmV0dXJuIHtlbmNvZGU6IGVuY29kZTIsIGRlY29kZTogZGVjb2RlMiwgZGVjb2RlVG9CeXRlcywgZGVjb2RlVW5zYWZlLCBmcm9tV29yZHMsIGZyb21Xb3Jkc1Vuc2FmZSwgdG9Xb3Jkc307XG4gICAgfVxuICAgIGV4cG9ydHMuYmVjaDMyID0gZ2VuQmVjaDMyKFwiYmVjaDMyXCIpO1xuICAgIGV4cG9ydHMuYmVjaDMybSA9IGdlbkJlY2gzMihcImJlY2gzMm1cIik7XG4gICAgZXhwb3J0cy51dGY4ID0ge1xuICAgICAgZW5jb2RlOiAoZGF0YSkgPT4gbmV3IFRleHREZWNvZGVyKCkuZGVjb2RlKGRhdGEpLFxuICAgICAgZGVjb2RlOiAoc3RyKSA9PiBuZXcgVGV4dEVuY29kZXIoKS5lbmNvZGUoc3RyKVxuICAgIH07XG4gICAgZXhwb3J0cy5oZXggPSBjaGFpbihyYWRpeDIoNCksIGFscGhhYmV0KFwiMDEyMzQ1Njc4OWFiY2RlZlwiKSwgam9pbihcIlwiKSwgbm9ybWFsaXplKChzKSA9PiB7XG4gICAgICBpZiAodHlwZW9mIHMgIT09IFwic3RyaW5nXCIgfHwgcy5sZW5ndGggJSAyKVxuICAgICAgICB0aHJvdyBuZXcgVHlwZUVycm9yKGBoZXguZGVjb2RlOiBleHBlY3RlZCBzdHJpbmcsIGdvdCAke3R5cGVvZiBzfSB3aXRoIGxlbmd0aCAke3MubGVuZ3RofWApO1xuICAgICAgcmV0dXJuIHMudG9Mb3dlckNhc2UoKTtcbiAgICB9KSk7XG4gICAgdmFyIENPREVSUyA9IHtcbiAgICAgIHV0Zjg6IGV4cG9ydHMudXRmOCxcbiAgICAgIGhleDogZXhwb3J0cy5oZXgsXG4gICAgICBiYXNlMTY6IGV4cG9ydHMuYmFzZTE2LFxuICAgICAgYmFzZTMyOiBleHBvcnRzLmJhc2UzMixcbiAgICAgIGJhc2U2NDogZXhwb3J0cy5iYXNlNjQsXG4gICAgICBiYXNlNjR1cmw6IGV4cG9ydHMuYmFzZTY0dXJsLFxuICAgICAgYmFzZTU4OiBleHBvcnRzLmJhc2U1OCxcbiAgICAgIGJhc2U1OHhtcjogZXhwb3J0cy5iYXNlNTh4bXJcbiAgICB9O1xuICAgIHZhciBjb2RlclR5cGVFcnJvciA9IGBJbnZhbGlkIGVuY29kaW5nIHR5cGUuIEF2YWlsYWJsZSB0eXBlczogJHtPYmplY3Qua2V5cyhDT0RFUlMpLmpvaW4oXCIsIFwiKX1gO1xuICAgIHZhciBieXRlc1RvU3RyaW5nID0gKHR5cGUsIGJ5dGVzMikgPT4ge1xuICAgICAgaWYgKHR5cGVvZiB0eXBlICE9PSBcInN0cmluZ1wiIHx8ICFDT0RFUlMuaGFzT3duUHJvcGVydHkodHlwZSkpXG4gICAgICAgIHRocm93IG5ldyBUeXBlRXJyb3IoY29kZXJUeXBlRXJyb3IpO1xuICAgICAgaWYgKCEoYnl0ZXMyIGluc3RhbmNlb2YgVWludDhBcnJheSkpXG4gICAgICAgIHRocm93IG5ldyBUeXBlRXJyb3IoXCJieXRlc1RvU3RyaW5nKCkgZXhwZWN0cyBVaW50OEFycmF5XCIpO1xuICAgICAgcmV0dXJuIENPREVSU1t0eXBlXS5lbmNvZGUoYnl0ZXMyKTtcbiAgICB9O1xuICAgIGV4cG9ydHMuYnl0ZXNUb1N0cmluZyA9IGJ5dGVzVG9TdHJpbmc7XG4gICAgZXhwb3J0cy5zdHIgPSBleHBvcnRzLmJ5dGVzVG9TdHJpbmc7XG4gICAgdmFyIHN0cmluZ1RvQnl0ZXMgPSAodHlwZSwgc3RyKSA9PiB7XG4gICAgICBpZiAoIUNPREVSUy5oYXNPd25Qcm9wZXJ0eSh0eXBlKSlcbiAgICAgICAgdGhyb3cgbmV3IFR5cGVFcnJvcihjb2RlclR5cGVFcnJvcik7XG4gICAgICBpZiAodHlwZW9mIHN0ciAhPT0gXCJzdHJpbmdcIilcbiAgICAgICAgdGhyb3cgbmV3IFR5cGVFcnJvcihcInN0cmluZ1RvQnl0ZXMoKSBleHBlY3RzIHN0cmluZ1wiKTtcbiAgICAgIHJldHVybiBDT0RFUlNbdHlwZV0uZGVjb2RlKHN0cik7XG4gICAgfTtcbiAgICBleHBvcnRzLnN0cmluZ1RvQnl0ZXMgPSBzdHJpbmdUb0J5dGVzO1xuICAgIGV4cG9ydHMuYnl0ZXMgPSBleHBvcnRzLnN0cmluZ1RvQnl0ZXM7XG4gIH1cbn0pO1xuZnVuY3Rpb24gY3JlYXRlQ29udGV4dChzaXplID0gNDA5Nikge1xuICBjb25zdCBidWZmZXIgPSBuZXcgQXJyYXlCdWZmZXIoc2l6ZSk7XG4gIHJldHVybiB7XG4gICAgaTogMCxcbiAgICB2aWV3OiBuZXcgRGF0YVZpZXcoYnVmZmVyKSxcbiAgICBieXRlczogbmV3IFVpbnQ4QXJyYXkoYnVmZmVyKVxuICB9O1xufVxuZnVuY3Rpb24gZ3Jvd0NvbnRleHQoY3R4KSB7XG4gIGN0eC5ieXRlcyA9IG5ldyBVaW50OEFycmF5KGN0eC5ieXRlcy5sZW5ndGggKiAyKTtcbiAgY3R4LnZpZXcgPSBuZXcgRGF0YVZpZXcoY3R4LmJ5dGVzLmJ1ZmZlcik7XG59XG5mdW5jdGlvbiBjb250ZXh0U2VyKGN0eCwgc2VyLCBkYXRhKSB7XG4gIHdoaWxlICh0cnVlKSB7XG4gICAgY29uc3QgbGltaXQgPSBjdHguYnl0ZXMubGVuZ3RoIC0gODtcbiAgICBjdHguaSA9IDA7XG4gICAgdHJ5IHtcbiAgICAgIHNlcihjdHgsIGRhdGEpO1xuICAgICAgaWYgKGN0eC5pIDwgbGltaXQpXG4gICAgICAgIHJldHVybiBjdHguYnl0ZXM7XG4gICAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICAgIGlmIChjdHguaSA8IGxpbWl0KVxuICAgICAgICB0aHJvdyBlcnJvcjtcbiAgICB9XG4gICAgZ3Jvd0NvbnRleHQoY3R4KTtcbiAgfVxufVxuZnVuY3Rpb24gY29udGV4dERlcyhjdHgsIGRlcywgYnl0ZXMyKSB7XG4gIGNvbnN0IHtsZW5ndGh9ID0gYnl0ZXMyO1xuICBpZiAobGVuZ3RoIDwgNDA5Nikge1xuICAgIGN0eC5ieXRlcy5zZXQoYnl0ZXMyKTtcbiAgICBjdHguaSA9IDA7XG4gICAgY29uc3QgZGF0YSA9IGRlcyhjdHgpO1xuICAgIGlmIChjdHguaSA+IGxlbmd0aClcbiAgICAgIHRocm93IFJhbmdlRXJyb3IoKTtcbiAgICByZXR1cm4gZGF0YTtcbiAgfSBlbHNlIHtcbiAgICByZXR1cm4gZGVzKGNvbnRleHRGcm9tQnl0ZXMoYnl0ZXMyKSk7XG4gIH1cbn1cbmZ1bmN0aW9uIGNvbnRleHRGcm9tQnl0ZXMoYXJyYXkyKSB7XG4gIHJldHVybiB7XG4gICAgaTogMCxcbiAgICBieXRlczogYXJyYXkyLFxuICAgIHZpZXc6IG5ldyBEYXRhVmlldyhhcnJheTIuYnVmZmVyLCBhcnJheTIuYnl0ZU9mZnNldCwgYXJyYXkyLmJ5dGVMZW5ndGgpXG4gIH07XG59XG5mdW5jdGlvbiBkZWZpbmUoc2VyLCBkZXMpIHtcbiAgcmV0dXJuIHtzZXIsIGRlc307XG59XG52YXIgbGF0aW4xID0ge1xuICBlbmNvZGUoY3R4LCBkYXRhKSB7XG4gICAgY29uc3Qge2xlbmd0aH0gPSBkYXRhO1xuICAgIGZvciAobGV0IGkgPSAwOyBpIDwgbGVuZ3RoOyBpKyspIHtcbiAgICAgIGN0eC52aWV3LnNldFVpbnQ4KGN0eC5pKyssIGRhdGEuY2hhckNvZGVBdChpKSk7XG4gICAgfVxuICB9LFxuICBkZWNvZGUoY3R4LCBzaXplKSB7XG4gICAgY29uc3QgY29kZXMgPSBuZXcgQXJyYXkoc2l6ZSk7XG4gICAgZm9yIChsZXQgaSA9IDA7IGkgPCBzaXplOyBpKyspIHtcbiAgICAgIGNvZGVzW2ldID0gY3R4LnZpZXcuZ2V0VWludDgoY3R4LmkrKyk7XG4gICAgfVxuICAgIHJldHVybiBTdHJpbmcuZnJvbUNoYXJDb2RlKC4uLmNvZGVzKTtcbiAgfVxufTtcbnZhciBzdHJpbmcgPSAoZW5jb2RpbmcsIGhlYWRTZCkgPT4gZGVmaW5lKChjdHgsIGRhdGEpID0+IHtcbiAgY29uc3QgaGVhZCA9IGN0eC5pO1xuICBoZWFkU2Quc2VyKGN0eCwgZGF0YS5sZW5ndGgpO1xuICBjb25zdCBiZWdpbiA9IGN0eC5pO1xuICBjb25zdCBoZWFkU2l6ZSA9IGJlZ2luIC0gaGVhZDtcbiAgZW5jb2RpbmcuZW5jb2RlKGN0eCwgZGF0YSk7XG4gIGNvbnN0IGVuZCA9IGN0eC5pO1xuICBjb25zdCBzaXplID0gZW5kIC0gYmVnaW47XG4gIGlmIChzaXplID09PSBkYXRhLmxlbmd0aClcbiAgICByZXR1cm47XG4gIGhlYWRTZC5zZXIoY3R4LCBzaXplKTtcbiAgY29uc3QgcmVxdWlyZWRIZWFkU2l6ZSA9IGN0eC5pIC0gZW5kO1xuICBpZiAoaGVhZFNpemUgIT09IHJlcXVpcmVkSGVhZFNpemUpIHtcbiAgICBjdHguYnl0ZXMuY29weVdpdGhpbihoZWFkICsgcmVxdWlyZWRIZWFkU2l6ZSwgYmVnaW4sIGVuZCk7XG4gIH1cbiAgY3R4LmkgPSBoZWFkO1xuICBoZWFkU2Quc2VyKGN0eCwgc2l6ZSk7XG4gIGN0eC5pID0gZW5kICsgKHJlcXVpcmVkSGVhZFNpemUgLSBoZWFkU2l6ZSk7XG59LCAoY3R4KSA9PiBlbmNvZGluZy5kZWNvZGUoY3R4LCBoZWFkU2QuZGVzKGN0eCkpKTtcbnZhciBieXRlcyA9IChoZWFkU2QpID0+IGRlZmluZSgoY3R4LCBkYXRhKSA9PiB7XG4gIGNvbnN0IHtieXRlTGVuZ3RofSA9IGRhdGE7XG4gIGhlYWRTZC5zZXIoY3R4LCBieXRlTGVuZ3RoKTtcbiAgY3R4LmJ5dGVzLnNldChkYXRhLCBjdHguaSk7XG4gIGN0eC5pICs9IGJ5dGVMZW5ndGg7XG59LCAoY3R4KSA9PiB7XG4gIGNvbnN0IGJ5dGVMZW5ndGggPSBoZWFkU2QuZGVzKGN0eCk7XG4gIHJldHVybiBjdHguYnl0ZXMuc3ViYXJyYXkoY3R4LmksIGN0eC5pICs9IGJ5dGVMZW5ndGgpO1xufSk7XG52YXIgYm9vbGVhbiA9IGRlZmluZSgoY3R4LCBkYXRhKSA9PiB2b2lkIGN0eC52aWV3LnNldFVpbnQ4KGN0eC5pKyssICtkYXRhKSwgKGN0eCkgPT4gISFjdHgudmlldy5nZXRVaW50OChjdHguaSsrKSk7XG5mdW5jdGlvbiB1c2Uoe3NlciwgZGVzfSkge1xuICBjb25zdCBjdHggPSBjcmVhdGVDb250ZXh0KCk7XG4gIHJldHVybiB7XG4gICAgc2VyLFxuICAgIGRlcyxcbiAgICB0b0J5dGVzOiAoZGF0YSkgPT4gY29udGV4dFNlcihjdHgsIHNlciwgZGF0YSkuc2xpY2UoMCwgY3R4LmkpLFxuICAgIHRvVW5zYWZlQnl0ZXM6IChkYXRhKSA9PiBjb250ZXh0U2VyKGN0eCwgc2VyLCBkYXRhKS5zdWJhcnJheSgwLCBjdHguaSksXG4gICAgZnJvbUJ5dGVzOiAoYnl0ZXMyKSA9PiBjb250ZXh0RGVzKGN0eCwgZGVzLCBieXRlczIpXG4gIH07XG59XG52YXIgdWludDggPSBkZWZpbmUoKGN0eCwgZGF0YSkgPT4gY3R4LnZpZXcuc2V0VWludDgoY3R4LmkrKywgZGF0YSksIChjdHgpID0+IGN0eC52aWV3LmdldFVpbnQ4KGN0eC5pKyspKTtcbnZhciBiaWdVaW50NjQgPSBkZWZpbmUoKGN0eCwgZGF0YSkgPT4ge1xuICBjdHgudmlldy5zZXRCaWdVaW50NjQoY3R4LmksIGRhdGEpO1xuICBjdHguaSArPSA4O1xufSwgKGN0eCkgPT4ge1xuICBjb25zdCBkYXRhID0gY3R4LnZpZXcuZ2V0QmlnVWludDY0KGN0eC5pKTtcbiAgY3R4LmkgKz0gODtcbiAgcmV0dXJuIGRhdGE7XG59KTtcbnZhciBzdHJ1Y3QgPSAoZGVmaW5pdGlvbikgPT4ge1xuICBjb25zdCBvYmogPSBkZWZpbml0aW9uIGluc3RhbmNlb2YgQXJyYXkgPyAoKSA9PiBbXSA6ICgpID0+ICh7fSk7XG4gIHJldHVybiBkZWZpbmUoKGN0eCwgZGF0YSkgPT4ge1xuICAgIGZvciAoY29uc3Qga2V5IGluIGRlZmluaXRpb24pIHtcbiAgICAgIGRlZmluaXRpb25ba2V5XS5zZXIoY3R4LCBkYXRhW2tleV0pO1xuICAgIH1cbiAgfSwgKGN0eCkgPT4ge1xuICAgIGNvbnN0IGRhdGEgPSBvYmooKTtcbiAgICBmb3IgKGNvbnN0IGtleSBpbiBkZWZpbml0aW9uKSB7XG4gICAgICBkYXRhW2tleV0gPSBkZWZpbml0aW9uW2tleV0uZGVzKGN0eCk7XG4gICAgfVxuICAgIHJldHVybiBkYXRhO1xuICB9KTtcbn07XG52YXIgaW1wb3J0X3NoYTI1NiA9IF9fdG9FU00ocmVxdWlyZV9zaGEyNTYoKSwgMSk7XG52YXIgaW1wb3J0X2hhbGZzaXBoYXNoID0gX190b0VTTShyZXF1aXJlX2hhbGZzaXBoYXNoKCksIDEpO1xudmFyIGltcG9ydF9jb25zdGFudF90aW1lID0gX190b0VTTShyZXF1aXJlX2NvbnN0YW50X3RpbWUoKSwgMSk7XG52YXIgdXRpbDtcbihmdW5jdGlvbih1dGlsMikge1xuICBmdW5jdGlvbiBhc3NlcnRFcXVhbChfY29uZCkge1xuICB9XG4gIHV0aWwyLmFzc2VydEVxdWFsID0gYXNzZXJ0RXF1YWw7XG4gIGZ1bmN0aW9uIGFzc2VydE5ldmVyKF94KSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKCk7XG4gIH1cbiAgdXRpbDIuYXNzZXJ0TmV2ZXIgPSBhc3NlcnROZXZlcjtcbiAgdXRpbDIuYXJyYXlUb0VudW0gPSAoaXRlbXMpID0+IHtcbiAgICBjb25zdCBvYmogPSB7fTtcbiAgICBmb3IgKGNvbnN0IGl0ZW0gb2YgaXRlbXMpIHtcbiAgICAgIG9ialtpdGVtXSA9IGl0ZW07XG4gICAgfVxuICAgIHJldHVybiBvYmo7XG4gIH07XG4gIHV0aWwyLmdldFZhbGlkRW51bVZhbHVlcyA9IChvYmopID0+IHtcbiAgICBjb25zdCB2YWxpZEtleXMgPSB1dGlsMi5vYmplY3RLZXlzKG9iaikuZmlsdGVyKChrKSA9PiB0eXBlb2Ygb2JqW29ialtrXV0gIT09IFwibnVtYmVyXCIpO1xuICAgIGNvbnN0IGZpbHRlcmVkID0ge307XG4gICAgZm9yIChjb25zdCBrIG9mIHZhbGlkS2V5cykge1xuICAgICAgZmlsdGVyZWRba10gPSBvYmpba107XG4gICAgfVxuICAgIHJldHVybiB1dGlsMi5vYmplY3RWYWx1ZXMoZmlsdGVyZWQpO1xuICB9O1xuICB1dGlsMi5vYmplY3RWYWx1ZXMgPSAob2JqKSA9PiB7XG4gICAgcmV0dXJuIHV0aWwyLm9iamVjdEtleXMob2JqKS5tYXAoZnVuY3Rpb24oZSkge1xuICAgICAgcmV0dXJuIG9ialtlXTtcbiAgICB9KTtcbiAgfTtcbiAgdXRpbDIub2JqZWN0S2V5cyA9IHR5cGVvZiBPYmplY3Qua2V5cyA9PT0gXCJmdW5jdGlvblwiID8gKG9iaikgPT4gT2JqZWN0LmtleXMob2JqKSA6IChvYmplY3QpID0+IHtcbiAgICBjb25zdCBrZXlzID0gW107XG4gICAgZm9yIChjb25zdCBrZXkgaW4gb2JqZWN0KSB7XG4gICAgICBpZiAoT2JqZWN0LnByb3RvdHlwZS5oYXNPd25Qcm9wZXJ0eS5jYWxsKG9iamVjdCwga2V5KSkge1xuICAgICAgICBrZXlzLnB1c2goa2V5KTtcbiAgICAgIH1cbiAgICB9XG4gICAgcmV0dXJuIGtleXM7XG4gIH07XG4gIHV0aWwyLmZpbmQgPSAoYXJyLCBjaGVja2VyKSA9PiB7XG4gICAgZm9yIChjb25zdCBpdGVtIG9mIGFycikge1xuICAgICAgaWYgKGNoZWNrZXIoaXRlbSkpXG4gICAgICAgIHJldHVybiBpdGVtO1xuICAgIH1cbiAgICByZXR1cm4gdm9pZCAwO1xuICB9O1xuICB1dGlsMi5pc0ludGVnZXIgPSB0eXBlb2YgTnVtYmVyLmlzSW50ZWdlciA9PT0gXCJmdW5jdGlvblwiID8gKHZhbCkgPT4gTnVtYmVyLmlzSW50ZWdlcih2YWwpIDogKHZhbCkgPT4gdHlwZW9mIHZhbCA9PT0gXCJudW1iZXJcIiAmJiBpc0Zpbml0ZSh2YWwpICYmIE1hdGguZmxvb3IodmFsKSA9PT0gdmFsO1xuICBmdW5jdGlvbiBqb2luVmFsdWVzKGFycmF5LCBzZXBhcmF0b3IgPSBcIiB8IFwiKSB7XG4gICAgcmV0dXJuIGFycmF5Lm1hcCgodmFsKSA9PiB0eXBlb2YgdmFsID09PSBcInN0cmluZ1wiID8gYCcke3ZhbH0nYCA6IHZhbCkuam9pbihzZXBhcmF0b3IpO1xuICB9XG4gIHV0aWwyLmpvaW5WYWx1ZXMgPSBqb2luVmFsdWVzO1xufSkodXRpbCB8fCAodXRpbCA9IHt9KSk7XG52YXIgWm9kUGFyc2VkVHlwZSA9IHV0aWwuYXJyYXlUb0VudW0oW1xuICBcInN0cmluZ1wiLFxuICBcIm5hblwiLFxuICBcIm51bWJlclwiLFxuICBcImludGVnZXJcIixcbiAgXCJmbG9hdFwiLFxuICBcImJvb2xlYW5cIixcbiAgXCJkYXRlXCIsXG4gIFwiYmlnaW50XCIsXG4gIFwic3ltYm9sXCIsXG4gIFwiZnVuY3Rpb25cIixcbiAgXCJ1bmRlZmluZWRcIixcbiAgXCJudWxsXCIsXG4gIFwiYXJyYXlcIixcbiAgXCJvYmplY3RcIixcbiAgXCJ1bmtub3duXCIsXG4gIFwicHJvbWlzZVwiLFxuICBcInZvaWRcIixcbiAgXCJuZXZlclwiLFxuICBcIm1hcFwiLFxuICBcInNldFwiXG5dKTtcbnZhciBnZXRQYXJzZWRUeXBlID0gKGRhdGEpID0+IHtcbiAgY29uc3QgdCA9IHR5cGVvZiBkYXRhO1xuICBzd2l0Y2ggKHQpIHtcbiAgICBjYXNlIFwidW5kZWZpbmVkXCI6XG4gICAgICByZXR1cm4gWm9kUGFyc2VkVHlwZS51bmRlZmluZWQ7XG4gICAgY2FzZSBcInN0cmluZ1wiOlxuICAgICAgcmV0dXJuIFpvZFBhcnNlZFR5cGUuc3RyaW5nO1xuICAgIGNhc2UgXCJudW1iZXJcIjpcbiAgICAgIHJldHVybiBpc05hTihkYXRhKSA/IFpvZFBhcnNlZFR5cGUubmFuIDogWm9kUGFyc2VkVHlwZS5udW1iZXI7XG4gICAgY2FzZSBcImJvb2xlYW5cIjpcbiAgICAgIHJldHVybiBab2RQYXJzZWRUeXBlLmJvb2xlYW47XG4gICAgY2FzZSBcImZ1bmN0aW9uXCI6XG4gICAgICByZXR1cm4gWm9kUGFyc2VkVHlwZS5mdW5jdGlvbjtcbiAgICBjYXNlIFwiYmlnaW50XCI6XG4gICAgICByZXR1cm4gWm9kUGFyc2VkVHlwZS5iaWdpbnQ7XG4gICAgY2FzZSBcIm9iamVjdFwiOlxuICAgICAgaWYgKEFycmF5LmlzQXJyYXkoZGF0YSkpIHtcbiAgICAgICAgcmV0dXJuIFpvZFBhcnNlZFR5cGUuYXJyYXk7XG4gICAgICB9XG4gICAgICBpZiAoZGF0YSA9PT0gbnVsbCkge1xuICAgICAgICByZXR1cm4gWm9kUGFyc2VkVHlwZS5udWxsO1xuICAgICAgfVxuICAgICAgaWYgKGRhdGEudGhlbiAmJiB0eXBlb2YgZGF0YS50aGVuID09PSBcImZ1bmN0aW9uXCIgJiYgZGF0YS5jYXRjaCAmJiB0eXBlb2YgZGF0YS5jYXRjaCA9PT0gXCJmdW5jdGlvblwiKSB7XG4gICAgICAgIHJldHVybiBab2RQYXJzZWRUeXBlLnByb21pc2U7XG4gICAgICB9XG4gICAgICBpZiAodHlwZW9mIE1hcCAhPT0gXCJ1bmRlZmluZWRcIiAmJiBkYXRhIGluc3RhbmNlb2YgTWFwKSB7XG4gICAgICAgIHJldHVybiBab2RQYXJzZWRUeXBlLm1hcDtcbiAgICAgIH1cbiAgICAgIGlmICh0eXBlb2YgU2V0ICE9PSBcInVuZGVmaW5lZFwiICYmIGRhdGEgaW5zdGFuY2VvZiBTZXQpIHtcbiAgICAgICAgcmV0dXJuIFpvZFBhcnNlZFR5cGUuc2V0O1xuICAgICAgfVxuICAgICAgaWYgKHR5cGVvZiBEYXRlICE9PSBcInVuZGVmaW5lZFwiICYmIGRhdGEgaW5zdGFuY2VvZiBEYXRlKSB7XG4gICAgICAgIHJldHVybiBab2RQYXJzZWRUeXBlLmRhdGU7XG4gICAgICB9XG4gICAgICByZXR1cm4gWm9kUGFyc2VkVHlwZS5vYmplY3Q7XG4gICAgZGVmYXVsdDpcbiAgICAgIHJldHVybiBab2RQYXJzZWRUeXBlLnVua25vd247XG4gIH1cbn07XG52YXIgWm9kSXNzdWVDb2RlID0gdXRpbC5hcnJheVRvRW51bShbXG4gIFwiaW52YWxpZF90eXBlXCIsXG4gIFwiaW52YWxpZF9saXRlcmFsXCIsXG4gIFwiY3VzdG9tXCIsXG4gIFwiaW52YWxpZF91bmlvblwiLFxuICBcImludmFsaWRfdW5pb25fZGlzY3JpbWluYXRvclwiLFxuICBcImludmFsaWRfZW51bV92YWx1ZVwiLFxuICBcInVucmVjb2duaXplZF9rZXlzXCIsXG4gIFwiaW52YWxpZF9hcmd1bWVudHNcIixcbiAgXCJpbnZhbGlkX3JldHVybl90eXBlXCIsXG4gIFwiaW52YWxpZF9kYXRlXCIsXG4gIFwiaW52YWxpZF9zdHJpbmdcIixcbiAgXCJ0b29fc21hbGxcIixcbiAgXCJ0b29fYmlnXCIsXG4gIFwiaW52YWxpZF9pbnRlcnNlY3Rpb25fdHlwZXNcIixcbiAgXCJub3RfbXVsdGlwbGVfb2ZcIlxuXSk7XG52YXIgcXVvdGVsZXNzSnNvbiA9IChvYmopID0+IHtcbiAgY29uc3QganNvbiA9IEpTT04uc3RyaW5naWZ5KG9iaiwgbnVsbCwgMik7XG4gIHJldHVybiBqc29uLnJlcGxhY2UoL1wiKFteXCJdKylcIjovZywgXCIkMTpcIik7XG59O1xudmFyIFpvZEVycm9yID0gY2xhc3MgZXh0ZW5kcyBFcnJvciB7XG4gIGNvbnN0cnVjdG9yKGlzc3Vlcykge1xuICAgIHN1cGVyKCk7XG4gICAgdGhpcy5pc3N1ZXMgPSBbXTtcbiAgICB0aGlzLmFkZElzc3VlID0gKHN1YikgPT4ge1xuICAgICAgdGhpcy5pc3N1ZXMgPSBbLi4udGhpcy5pc3N1ZXMsIHN1Yl07XG4gICAgfTtcbiAgICB0aGlzLmFkZElzc3VlcyA9IChzdWJzID0gW10pID0+IHtcbiAgICAgIHRoaXMuaXNzdWVzID0gWy4uLnRoaXMuaXNzdWVzLCAuLi5zdWJzXTtcbiAgICB9O1xuICAgIGNvbnN0IGFjdHVhbFByb3RvID0gbmV3LnRhcmdldC5wcm90b3R5cGU7XG4gICAgaWYgKE9iamVjdC5zZXRQcm90b3R5cGVPZikge1xuICAgICAgT2JqZWN0LnNldFByb3RvdHlwZU9mKHRoaXMsIGFjdHVhbFByb3RvKTtcbiAgICB9IGVsc2Uge1xuICAgICAgdGhpcy5fX3Byb3RvX18gPSBhY3R1YWxQcm90bztcbiAgICB9XG4gICAgdGhpcy5uYW1lID0gXCJab2RFcnJvclwiO1xuICAgIHRoaXMuaXNzdWVzID0gaXNzdWVzO1xuICB9XG4gIGdldCBlcnJvcnMoKSB7XG4gICAgcmV0dXJuIHRoaXMuaXNzdWVzO1xuICB9XG4gIGZvcm1hdChfbWFwcGVyKSB7XG4gICAgY29uc3QgbWFwcGVyID0gX21hcHBlciB8fCBmdW5jdGlvbihpc3N1ZSkge1xuICAgICAgcmV0dXJuIGlzc3VlLm1lc3NhZ2U7XG4gICAgfTtcbiAgICBjb25zdCBmaWVsZEVycm9ycyA9IHtfZXJyb3JzOiBbXX07XG4gICAgY29uc3QgcHJvY2Vzc0Vycm9yID0gKGVycm9yKSA9PiB7XG4gICAgICBmb3IgKGNvbnN0IGlzc3VlIG9mIGVycm9yLmlzc3Vlcykge1xuICAgICAgICBpZiAoaXNzdWUuY29kZSA9PT0gXCJpbnZhbGlkX3VuaW9uXCIpIHtcbiAgICAgICAgICBpc3N1ZS51bmlvbkVycm9ycy5tYXAocHJvY2Vzc0Vycm9yKTtcbiAgICAgICAgfSBlbHNlIGlmIChpc3N1ZS5jb2RlID09PSBcImludmFsaWRfcmV0dXJuX3R5cGVcIikge1xuICAgICAgICAgIHByb2Nlc3NFcnJvcihpc3N1ZS5yZXR1cm5UeXBlRXJyb3IpO1xuICAgICAgICB9IGVsc2UgaWYgKGlzc3VlLmNvZGUgPT09IFwiaW52YWxpZF9hcmd1bWVudHNcIikge1xuICAgICAgICAgIHByb2Nlc3NFcnJvcihpc3N1ZS5hcmd1bWVudHNFcnJvcik7XG4gICAgICAgIH0gZWxzZSBpZiAoaXNzdWUucGF0aC5sZW5ndGggPT09IDApIHtcbiAgICAgICAgICBmaWVsZEVycm9ycy5fZXJyb3JzLnB1c2gobWFwcGVyKGlzc3VlKSk7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgbGV0IGN1cnIgPSBmaWVsZEVycm9ycztcbiAgICAgICAgICBsZXQgaSA9IDA7XG4gICAgICAgICAgd2hpbGUgKGkgPCBpc3N1ZS5wYXRoLmxlbmd0aCkge1xuICAgICAgICAgICAgY29uc3QgZWwgPSBpc3N1ZS5wYXRoW2ldO1xuICAgICAgICAgICAgY29uc3QgdGVybWluYWwgPSBpID09PSBpc3N1ZS5wYXRoLmxlbmd0aCAtIDE7XG4gICAgICAgICAgICBpZiAoIXRlcm1pbmFsKSB7XG4gICAgICAgICAgICAgIGN1cnJbZWxdID0gY3VycltlbF0gfHwge19lcnJvcnM6IFtdfTtcbiAgICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICAgIGN1cnJbZWxdID0gY3VycltlbF0gfHwge19lcnJvcnM6IFtdfTtcbiAgICAgICAgICAgICAgY3VycltlbF0uX2Vycm9ycy5wdXNoKG1hcHBlcihpc3N1ZSkpO1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgY3VyciA9IGN1cnJbZWxdO1xuICAgICAgICAgICAgaSsrO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuICAgICAgfVxuICAgIH07XG4gICAgcHJvY2Vzc0Vycm9yKHRoaXMpO1xuICAgIHJldHVybiBmaWVsZEVycm9ycztcbiAgfVxuICB0b1N0cmluZygpIHtcbiAgICByZXR1cm4gdGhpcy5tZXNzYWdlO1xuICB9XG4gIGdldCBtZXNzYWdlKCkge1xuICAgIHJldHVybiBKU09OLnN0cmluZ2lmeSh0aGlzLmlzc3VlcywganNvblN0cmluZ2lmeVJlcGxhY2VyLCAyKTtcbiAgfVxuICBnZXQgaXNFbXB0eSgpIHtcbiAgICByZXR1cm4gdGhpcy5pc3N1ZXMubGVuZ3RoID09PSAwO1xuICB9XG4gIGZsYXR0ZW4obWFwcGVyID0gKGlzc3VlKSA9PiBpc3N1ZS5tZXNzYWdlKSB7XG4gICAgY29uc3QgZmllbGRFcnJvcnMgPSB7fTtcbiAgICBjb25zdCBmb3JtRXJyb3JzID0gW107XG4gICAgZm9yIChjb25zdCBzdWIgb2YgdGhpcy5pc3N1ZXMpIHtcbiAgICAgIGlmIChzdWIucGF0aC5sZW5ndGggPiAwKSB7XG4gICAgICAgIGZpZWxkRXJyb3JzW3N1Yi5wYXRoWzBdXSA9IGZpZWxkRXJyb3JzW3N1Yi5wYXRoWzBdXSB8fCBbXTtcbiAgICAgICAgZmllbGRFcnJvcnNbc3ViLnBhdGhbMF1dLnB1c2gobWFwcGVyKHN1YikpO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgZm9ybUVycm9ycy5wdXNoKG1hcHBlcihzdWIpKTtcbiAgICAgIH1cbiAgICB9XG4gICAgcmV0dXJuIHtmb3JtRXJyb3JzLCBmaWVsZEVycm9yc307XG4gIH1cbiAgZ2V0IGZvcm1FcnJvcnMoKSB7XG4gICAgcmV0dXJuIHRoaXMuZmxhdHRlbigpO1xuICB9XG59O1xuWm9kRXJyb3IuY3JlYXRlID0gKGlzc3VlcykgPT4ge1xuICBjb25zdCBlcnJvciA9IG5ldyBab2RFcnJvcihpc3N1ZXMpO1xuICByZXR1cm4gZXJyb3I7XG59O1xudmFyIGRlZmF1bHRFcnJvck1hcCA9IChpc3N1ZSwgX2N0eCkgPT4ge1xuICBsZXQgbWVzc2FnZTtcbiAgc3dpdGNoIChpc3N1ZS5jb2RlKSB7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlOlxuICAgICAgaWYgKGlzc3VlLnJlY2VpdmVkID09PSBab2RQYXJzZWRUeXBlLnVuZGVmaW5lZCkge1xuICAgICAgICBtZXNzYWdlID0gXCJSZXF1aXJlZFwiO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgbWVzc2FnZSA9IGBFeHBlY3RlZCAke2lzc3VlLmV4cGVjdGVkfSwgcmVjZWl2ZWQgJHtpc3N1ZS5yZWNlaXZlZH1gO1xuICAgICAgfVxuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF9saXRlcmFsOlxuICAgICAgbWVzc2FnZSA9IGBJbnZhbGlkIGxpdGVyYWwgdmFsdWUsIGV4cGVjdGVkICR7SlNPTi5zdHJpbmdpZnkoaXNzdWUuZXhwZWN0ZWQsIGpzb25TdHJpbmdpZnlSZXBsYWNlcil9YDtcbiAgICAgIGJyZWFrO1xuICAgIGNhc2UgWm9kSXNzdWVDb2RlLnVucmVjb2duaXplZF9rZXlzOlxuICAgICAgbWVzc2FnZSA9IGBVbnJlY29nbml6ZWQga2V5KHMpIGluIG9iamVjdDogJHt1dGlsLmpvaW5WYWx1ZXMoaXNzdWUua2V5cywgXCIsIFwiKX1gO1xuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF91bmlvbjpcbiAgICAgIG1lc3NhZ2UgPSBgSW52YWxpZCBpbnB1dGA7XG4gICAgICBicmVhaztcbiAgICBjYXNlIFpvZElzc3VlQ29kZS5pbnZhbGlkX3VuaW9uX2Rpc2NyaW1pbmF0b3I6XG4gICAgICBtZXNzYWdlID0gYEludmFsaWQgZGlzY3JpbWluYXRvciB2YWx1ZS4gRXhwZWN0ZWQgJHt1dGlsLmpvaW5WYWx1ZXMoaXNzdWUub3B0aW9ucyl9YDtcbiAgICAgIGJyZWFrO1xuICAgIGNhc2UgWm9kSXNzdWVDb2RlLmludmFsaWRfZW51bV92YWx1ZTpcbiAgICAgIG1lc3NhZ2UgPSBgSW52YWxpZCBlbnVtIHZhbHVlLiBFeHBlY3RlZCAke3V0aWwuam9pblZhbHVlcyhpc3N1ZS5vcHRpb25zKX0sIHJlY2VpdmVkICcke2lzc3VlLnJlY2VpdmVkfSdgO1xuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF9hcmd1bWVudHM6XG4gICAgICBtZXNzYWdlID0gYEludmFsaWQgZnVuY3Rpb24gYXJndW1lbnRzYDtcbiAgICAgIGJyZWFrO1xuICAgIGNhc2UgWm9kSXNzdWVDb2RlLmludmFsaWRfcmV0dXJuX3R5cGU6XG4gICAgICBtZXNzYWdlID0gYEludmFsaWQgZnVuY3Rpb24gcmV0dXJuIHR5cGVgO1xuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF9kYXRlOlxuICAgICAgbWVzc2FnZSA9IGBJbnZhbGlkIGRhdGVgO1xuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF9zdHJpbmc6XG4gICAgICBpZiAodHlwZW9mIGlzc3VlLnZhbGlkYXRpb24gPT09IFwib2JqZWN0XCIpIHtcbiAgICAgICAgaWYgKFwic3RhcnRzV2l0aFwiIGluIGlzc3VlLnZhbGlkYXRpb24pIHtcbiAgICAgICAgICBtZXNzYWdlID0gYEludmFsaWQgaW5wdXQ6IG11c3Qgc3RhcnQgd2l0aCBcIiR7aXNzdWUudmFsaWRhdGlvbi5zdGFydHNXaXRofVwiYDtcbiAgICAgICAgfSBlbHNlIGlmIChcImVuZHNXaXRoXCIgaW4gaXNzdWUudmFsaWRhdGlvbikge1xuICAgICAgICAgIG1lc3NhZ2UgPSBgSW52YWxpZCBpbnB1dDogbXVzdCBlbmQgd2l0aCBcIiR7aXNzdWUudmFsaWRhdGlvbi5lbmRzV2l0aH1cImA7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgdXRpbC5hc3NlcnROZXZlcihpc3N1ZS52YWxpZGF0aW9uKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChpc3N1ZS52YWxpZGF0aW9uICE9PSBcInJlZ2V4XCIpIHtcbiAgICAgICAgbWVzc2FnZSA9IGBJbnZhbGlkICR7aXNzdWUudmFsaWRhdGlvbn1gO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgbWVzc2FnZSA9IFwiSW52YWxpZFwiO1xuICAgICAgfVxuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUudG9vX3NtYWxsOlxuICAgICAgaWYgKGlzc3VlLnR5cGUgPT09IFwiYXJyYXlcIilcbiAgICAgICAgbWVzc2FnZSA9IGBBcnJheSBtdXN0IGNvbnRhaW4gJHtpc3N1ZS5pbmNsdXNpdmUgPyBgYXQgbGVhc3RgIDogYG1vcmUgdGhhbmB9ICR7aXNzdWUubWluaW11bX0gZWxlbWVudChzKWA7XG4gICAgICBlbHNlIGlmIChpc3N1ZS50eXBlID09PSBcInN0cmluZ1wiKVxuICAgICAgICBtZXNzYWdlID0gYFN0cmluZyBtdXN0IGNvbnRhaW4gJHtpc3N1ZS5pbmNsdXNpdmUgPyBgYXQgbGVhc3RgIDogYG92ZXJgfSAke2lzc3VlLm1pbmltdW19IGNoYXJhY3RlcihzKWA7XG4gICAgICBlbHNlIGlmIChpc3N1ZS50eXBlID09PSBcIm51bWJlclwiKVxuICAgICAgICBtZXNzYWdlID0gYE51bWJlciBtdXN0IGJlIGdyZWF0ZXIgdGhhbiAke2lzc3VlLmluY2x1c2l2ZSA/IGBvciBlcXVhbCB0byBgIDogYGB9JHtpc3N1ZS5taW5pbXVtfWA7XG4gICAgICBlbHNlIGlmIChpc3N1ZS50eXBlID09PSBcImRhdGVcIilcbiAgICAgICAgbWVzc2FnZSA9IGBEYXRlIG11c3QgYmUgZ3JlYXRlciB0aGFuICR7aXNzdWUuaW5jbHVzaXZlID8gYG9yIGVxdWFsIHRvIGAgOiBgYH0ke25ldyBEYXRlKGlzc3VlLm1pbmltdW0pfWA7XG4gICAgICBlbHNlXG4gICAgICAgIG1lc3NhZ2UgPSBcIkludmFsaWQgaW5wdXRcIjtcbiAgICAgIGJyZWFrO1xuICAgIGNhc2UgWm9kSXNzdWVDb2RlLnRvb19iaWc6XG4gICAgICBpZiAoaXNzdWUudHlwZSA9PT0gXCJhcnJheVwiKVxuICAgICAgICBtZXNzYWdlID0gYEFycmF5IG11c3QgY29udGFpbiAke2lzc3VlLmluY2x1c2l2ZSA/IGBhdCBtb3N0YCA6IGBsZXNzIHRoYW5gfSAke2lzc3VlLm1heGltdW19IGVsZW1lbnQocylgO1xuICAgICAgZWxzZSBpZiAoaXNzdWUudHlwZSA9PT0gXCJzdHJpbmdcIilcbiAgICAgICAgbWVzc2FnZSA9IGBTdHJpbmcgbXVzdCBjb250YWluICR7aXNzdWUuaW5jbHVzaXZlID8gYGF0IG1vc3RgIDogYHVuZGVyYH0gJHtpc3N1ZS5tYXhpbXVtfSBjaGFyYWN0ZXIocylgO1xuICAgICAgZWxzZSBpZiAoaXNzdWUudHlwZSA9PT0gXCJudW1iZXJcIilcbiAgICAgICAgbWVzc2FnZSA9IGBOdW1iZXIgbXVzdCBiZSBsZXNzIHRoYW4gJHtpc3N1ZS5pbmNsdXNpdmUgPyBgb3IgZXF1YWwgdG8gYCA6IGBgfSR7aXNzdWUubWF4aW11bX1gO1xuICAgICAgZWxzZSBpZiAoaXNzdWUudHlwZSA9PT0gXCJkYXRlXCIpXG4gICAgICAgIG1lc3NhZ2UgPSBgRGF0ZSBtdXN0IGJlIHNtYWxsZXIgdGhhbiAke2lzc3VlLmluY2x1c2l2ZSA/IGBvciBlcXVhbCB0byBgIDogYGB9JHtuZXcgRGF0ZShpc3N1ZS5tYXhpbXVtKX1gO1xuICAgICAgZWxzZVxuICAgICAgICBtZXNzYWdlID0gXCJJbnZhbGlkIGlucHV0XCI7XG4gICAgICBicmVhaztcbiAgICBjYXNlIFpvZElzc3VlQ29kZS5jdXN0b206XG4gICAgICBtZXNzYWdlID0gYEludmFsaWQgaW5wdXRgO1xuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUuaW52YWxpZF9pbnRlcnNlY3Rpb25fdHlwZXM6XG4gICAgICBtZXNzYWdlID0gYEludGVyc2VjdGlvbiByZXN1bHRzIGNvdWxkIG5vdCBiZSBtZXJnZWRgO1xuICAgICAgYnJlYWs7XG4gICAgY2FzZSBab2RJc3N1ZUNvZGUubm90X211bHRpcGxlX29mOlxuICAgICAgbWVzc2FnZSA9IGBOdW1iZXIgbXVzdCBiZSBhIG11bHRpcGxlIG9mICR7aXNzdWUubXVsdGlwbGVPZn1gO1xuICAgICAgYnJlYWs7XG4gICAgZGVmYXVsdDpcbiAgICAgIG1lc3NhZ2UgPSBfY3R4LmRlZmF1bHRFcnJvcjtcbiAgICAgIHV0aWwuYXNzZXJ0TmV2ZXIoaXNzdWUpO1xuICB9XG4gIHJldHVybiB7bWVzc2FnZX07XG59O1xudmFyIG92ZXJyaWRlRXJyb3JNYXAgPSBkZWZhdWx0RXJyb3JNYXA7XG5mdW5jdGlvbiBzZXRFcnJvck1hcChtYXApIHtcbiAgb3ZlcnJpZGVFcnJvck1hcCA9IG1hcDtcbn1cbmZ1bmN0aW9uIGdldEVycm9yTWFwKCkge1xuICByZXR1cm4gb3ZlcnJpZGVFcnJvck1hcDtcbn1cbnZhciBtYWtlSXNzdWUgPSAocGFyYW1zKSA9PiB7XG4gIGNvbnN0IHtkYXRhLCBwYXRoLCBlcnJvck1hcHMsIGlzc3VlRGF0YX0gPSBwYXJhbXM7XG4gIGNvbnN0IGZ1bGxQYXRoID0gWy4uLnBhdGgsIC4uLmlzc3VlRGF0YS5wYXRoIHx8IFtdXTtcbiAgY29uc3QgZnVsbElzc3VlID0ge1xuICAgIC4uLmlzc3VlRGF0YSxcbiAgICBwYXRoOiBmdWxsUGF0aFxuICB9O1xuICBsZXQgZXJyb3JNZXNzYWdlID0gXCJcIjtcbiAgY29uc3QgbWFwcyA9IGVycm9yTWFwcy5maWx0ZXIoKG0pID0+ICEhbSkuc2xpY2UoKS5yZXZlcnNlKCk7XG4gIGZvciAoY29uc3QgbWFwIG9mIG1hcHMpIHtcbiAgICBlcnJvck1lc3NhZ2UgPSBtYXAoZnVsbElzc3VlLCB7ZGF0YSwgZGVmYXVsdEVycm9yOiBlcnJvck1lc3NhZ2V9KS5tZXNzYWdlO1xuICB9XG4gIHJldHVybiB7XG4gICAgLi4uaXNzdWVEYXRhLFxuICAgIHBhdGg6IGZ1bGxQYXRoLFxuICAgIG1lc3NhZ2U6IGlzc3VlRGF0YS5tZXNzYWdlIHx8IGVycm9yTWVzc2FnZVxuICB9O1xufTtcbnZhciBFTVBUWV9QQVRIID0gW107XG5mdW5jdGlvbiBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIGlzc3VlRGF0YSkge1xuICBjb25zdCBpc3N1ZSA9IG1ha2VJc3N1ZSh7XG4gICAgaXNzdWVEYXRhLFxuICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgIGVycm9yTWFwczogW1xuICAgICAgY3R4LmNvbW1vbi5jb250ZXh0dWFsRXJyb3JNYXAsXG4gICAgICBjdHguc2NoZW1hRXJyb3JNYXAsXG4gICAgICBnZXRFcnJvck1hcCgpLFxuICAgICAgZGVmYXVsdEVycm9yTWFwXG4gICAgXS5maWx0ZXIoKHgpID0+ICEheClcbiAgfSk7XG4gIGN0eC5jb21tb24uaXNzdWVzLnB1c2goaXNzdWUpO1xufVxudmFyIFBhcnNlU3RhdHVzID0gY2xhc3Mge1xuICBjb25zdHJ1Y3RvcigpIHtcbiAgICB0aGlzLnZhbHVlID0gXCJ2YWxpZFwiO1xuICB9XG4gIGRpcnR5KCkge1xuICAgIGlmICh0aGlzLnZhbHVlID09PSBcInZhbGlkXCIpXG4gICAgICB0aGlzLnZhbHVlID0gXCJkaXJ0eVwiO1xuICB9XG4gIGFib3J0KCkge1xuICAgIGlmICh0aGlzLnZhbHVlICE9PSBcImFib3J0ZWRcIilcbiAgICAgIHRoaXMudmFsdWUgPSBcImFib3J0ZWRcIjtcbiAgfVxuICBzdGF0aWMgbWVyZ2VBcnJheShzdGF0dXMsIHJlc3VsdHMpIHtcbiAgICBjb25zdCBhcnJheVZhbHVlID0gW107XG4gICAgZm9yIChjb25zdCBzIG9mIHJlc3VsdHMpIHtcbiAgICAgIGlmIChzLnN0YXR1cyA9PT0gXCJhYm9ydGVkXCIpXG4gICAgICAgIHJldHVybiBJTlZBTElEO1xuICAgICAgaWYgKHMuc3RhdHVzID09PSBcImRpcnR5XCIpXG4gICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgYXJyYXlWYWx1ZS5wdXNoKHMudmFsdWUpO1xuICAgIH1cbiAgICByZXR1cm4ge3N0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogYXJyYXlWYWx1ZX07XG4gIH1cbiAgc3RhdGljIGFzeW5jIG1lcmdlT2JqZWN0QXN5bmMoc3RhdHVzLCBwYWlycykge1xuICAgIGNvbnN0IHN5bmNQYWlycyA9IFtdO1xuICAgIGZvciAoY29uc3QgcGFpciBvZiBwYWlycykge1xuICAgICAgc3luY1BhaXJzLnB1c2goe1xuICAgICAgICBrZXk6IGF3YWl0IHBhaXIua2V5LFxuICAgICAgICB2YWx1ZTogYXdhaXQgcGFpci52YWx1ZVxuICAgICAgfSk7XG4gICAgfVxuICAgIHJldHVybiBQYXJzZVN0YXR1cy5tZXJnZU9iamVjdFN5bmMoc3RhdHVzLCBzeW5jUGFpcnMpO1xuICB9XG4gIHN0YXRpYyBtZXJnZU9iamVjdFN5bmMoc3RhdHVzLCBwYWlycykge1xuICAgIGNvbnN0IGZpbmFsT2JqZWN0ID0ge307XG4gICAgZm9yIChjb25zdCBwYWlyIG9mIHBhaXJzKSB7XG4gICAgICBjb25zdCB7a2V5LCB2YWx1ZX0gPSBwYWlyO1xuICAgICAgaWYgKGtleS5zdGF0dXMgPT09IFwiYWJvcnRlZFwiKVxuICAgICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICAgIGlmICh2YWx1ZS5zdGF0dXMgPT09IFwiYWJvcnRlZFwiKVxuICAgICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICAgIGlmIChrZXkuc3RhdHVzID09PSBcImRpcnR5XCIpXG4gICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgaWYgKHZhbHVlLnN0YXR1cyA9PT0gXCJkaXJ0eVwiKVxuICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgIGlmICh0eXBlb2YgdmFsdWUudmFsdWUgIT09IFwidW5kZWZpbmVkXCIgfHwgcGFpci5hbHdheXNTZXQpIHtcbiAgICAgICAgZmluYWxPYmplY3Rba2V5LnZhbHVlXSA9IHZhbHVlLnZhbHVlO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4ge3N0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogZmluYWxPYmplY3R9O1xuICB9XG59O1xudmFyIElOVkFMSUQgPSBPYmplY3QuZnJlZXplKHtcbiAgc3RhdHVzOiBcImFib3J0ZWRcIlxufSk7XG52YXIgRElSVFkgPSAodmFsdWUpID0+ICh7c3RhdHVzOiBcImRpcnR5XCIsIHZhbHVlfSk7XG52YXIgT0sgPSAodmFsdWUpID0+ICh7c3RhdHVzOiBcInZhbGlkXCIsIHZhbHVlfSk7XG52YXIgaXNBYm9ydGVkID0gKHgpID0+IHguc3RhdHVzID09PSBcImFib3J0ZWRcIjtcbnZhciBpc0RpcnR5ID0gKHgpID0+IHguc3RhdHVzID09PSBcImRpcnR5XCI7XG52YXIgaXNWYWxpZCA9ICh4KSA9PiB4LnN0YXR1cyA9PT0gXCJ2YWxpZFwiO1xudmFyIGlzQXN5bmMgPSAoeCkgPT4gdHlwZW9mIFByb21pc2UgIT09IHZvaWQgMCAmJiB4IGluc3RhbmNlb2YgUHJvbWlzZTtcbnZhciBqc29uU3RyaW5naWZ5UmVwbGFjZXIgPSAoXywgdmFsdWUpID0+IHtcbiAgaWYgKHR5cGVvZiB2YWx1ZSA9PT0gXCJiaWdpbnRcIikge1xuICAgIHJldHVybiB2YWx1ZS50b1N0cmluZygpO1xuICB9XG4gIHJldHVybiB2YWx1ZTtcbn07XG52YXIgZXJyb3JVdGlsO1xuKGZ1bmN0aW9uKGVycm9yVXRpbDIpIHtcbiAgZXJyb3JVdGlsMi5lcnJUb09iaiA9IChtZXNzYWdlKSA9PiB0eXBlb2YgbWVzc2FnZSA9PT0gXCJzdHJpbmdcIiA/IHttZXNzYWdlfSA6IG1lc3NhZ2UgfHwge307XG4gIGVycm9yVXRpbDIudG9TdHJpbmcgPSAobWVzc2FnZSkgPT4gdHlwZW9mIG1lc3NhZ2UgPT09IFwic3RyaW5nXCIgPyBtZXNzYWdlIDogbWVzc2FnZSA9PT0gbnVsbCB8fCBtZXNzYWdlID09PSB2b2lkIDAgPyB2b2lkIDAgOiBtZXNzYWdlLm1lc3NhZ2U7XG59KShlcnJvclV0aWwgfHwgKGVycm9yVXRpbCA9IHt9KSk7XG52YXIgUGFyc2VJbnB1dExhenlQYXRoID0gY2xhc3Mge1xuICBjb25zdHJ1Y3RvcihwYXJlbnQsIHZhbHVlLCBwYXRoLCBrZXkpIHtcbiAgICB0aGlzLnBhcmVudCA9IHBhcmVudDtcbiAgICB0aGlzLmRhdGEgPSB2YWx1ZTtcbiAgICB0aGlzLl9wYXRoID0gcGF0aDtcbiAgICB0aGlzLl9rZXkgPSBrZXk7XG4gIH1cbiAgZ2V0IHBhdGgoKSB7XG4gICAgcmV0dXJuIHRoaXMuX3BhdGguY29uY2F0KHRoaXMuX2tleSk7XG4gIH1cbn07XG52YXIgaGFuZGxlUmVzdWx0ID0gKGN0eCwgcmVzdWx0KSA9PiB7XG4gIGlmIChpc1ZhbGlkKHJlc3VsdCkpIHtcbiAgICByZXR1cm4ge3N1Y2Nlc3M6IHRydWUsIGRhdGE6IHJlc3VsdC52YWx1ZX07XG4gIH0gZWxzZSB7XG4gICAgaWYgKCFjdHguY29tbW9uLmlzc3Vlcy5sZW5ndGgpIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihcIlZhbGlkYXRpb24gZmFpbGVkIGJ1dCBubyBpc3N1ZXMgZGV0ZWN0ZWQuXCIpO1xuICAgIH1cbiAgICBjb25zdCBlcnJvciA9IG5ldyBab2RFcnJvcihjdHguY29tbW9uLmlzc3Vlcyk7XG4gICAgcmV0dXJuIHtzdWNjZXNzOiBmYWxzZSwgZXJyb3J9O1xuICB9XG59O1xuZnVuY3Rpb24gcHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpIHtcbiAgaWYgKCFwYXJhbXMpXG4gICAgcmV0dXJuIHt9O1xuICBjb25zdCB7ZXJyb3JNYXAsIGludmFsaWRfdHlwZV9lcnJvciwgcmVxdWlyZWRfZXJyb3IsIGRlc2NyaXB0aW9ufSA9IHBhcmFtcztcbiAgaWYgKGVycm9yTWFwICYmIChpbnZhbGlkX3R5cGVfZXJyb3IgfHwgcmVxdWlyZWRfZXJyb3IpKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKGBDYW4ndCB1c2UgXCJpbnZhbGlkXCIgb3IgXCJyZXF1aXJlZFwiIGluIGNvbmp1bmN0aW9uIHdpdGggY3VzdG9tIGVycm9yIG1hcC5gKTtcbiAgfVxuICBpZiAoZXJyb3JNYXApXG4gICAgcmV0dXJuIHtlcnJvck1hcCwgZGVzY3JpcHRpb259O1xuICBjb25zdCBjdXN0b21NYXAgPSAoaXNzLCBjdHgpID0+IHtcbiAgICBpZiAoaXNzLmNvZGUgIT09IFwiaW52YWxpZF90eXBlXCIpXG4gICAgICByZXR1cm4ge21lc3NhZ2U6IGN0eC5kZWZhdWx0RXJyb3J9O1xuICAgIGlmICh0eXBlb2YgY3R4LmRhdGEgPT09IFwidW5kZWZpbmVkXCIpIHtcbiAgICAgIHJldHVybiB7bWVzc2FnZTogcmVxdWlyZWRfZXJyb3IgIT09IG51bGwgJiYgcmVxdWlyZWRfZXJyb3IgIT09IHZvaWQgMCA/IHJlcXVpcmVkX2Vycm9yIDogY3R4LmRlZmF1bHRFcnJvcn07XG4gICAgfVxuICAgIHJldHVybiB7bWVzc2FnZTogaW52YWxpZF90eXBlX2Vycm9yICE9PSBudWxsICYmIGludmFsaWRfdHlwZV9lcnJvciAhPT0gdm9pZCAwID8gaW52YWxpZF90eXBlX2Vycm9yIDogY3R4LmRlZmF1bHRFcnJvcn07XG4gIH07XG4gIHJldHVybiB7ZXJyb3JNYXA6IGN1c3RvbU1hcCwgZGVzY3JpcHRpb259O1xufVxudmFyIFpvZFR5cGUgPSBjbGFzcyB7XG4gIGNvbnN0cnVjdG9yKGRlZikge1xuICAgIHRoaXMuc3BhID0gdGhpcy5zYWZlUGFyc2VBc3luYztcbiAgICB0aGlzLnN1cGVyUmVmaW5lID0gdGhpcy5fcmVmaW5lbWVudDtcbiAgICB0aGlzLl9kZWYgPSBkZWY7XG4gICAgdGhpcy5wYXJzZSA9IHRoaXMucGFyc2UuYmluZCh0aGlzKTtcbiAgICB0aGlzLnNhZmVQYXJzZSA9IHRoaXMuc2FmZVBhcnNlLmJpbmQodGhpcyk7XG4gICAgdGhpcy5wYXJzZUFzeW5jID0gdGhpcy5wYXJzZUFzeW5jLmJpbmQodGhpcyk7XG4gICAgdGhpcy5zYWZlUGFyc2VBc3luYyA9IHRoaXMuc2FmZVBhcnNlQXN5bmMuYmluZCh0aGlzKTtcbiAgICB0aGlzLnNwYSA9IHRoaXMuc3BhLmJpbmQodGhpcyk7XG4gICAgdGhpcy5yZWZpbmUgPSB0aGlzLnJlZmluZS5iaW5kKHRoaXMpO1xuICAgIHRoaXMucmVmaW5lbWVudCA9IHRoaXMucmVmaW5lbWVudC5iaW5kKHRoaXMpO1xuICAgIHRoaXMuc3VwZXJSZWZpbmUgPSB0aGlzLnN1cGVyUmVmaW5lLmJpbmQodGhpcyk7XG4gICAgdGhpcy5vcHRpb25hbCA9IHRoaXMub3B0aW9uYWwuYmluZCh0aGlzKTtcbiAgICB0aGlzLm51bGxhYmxlID0gdGhpcy5udWxsYWJsZS5iaW5kKHRoaXMpO1xuICAgIHRoaXMubnVsbGlzaCA9IHRoaXMubnVsbGlzaC5iaW5kKHRoaXMpO1xuICAgIHRoaXMuYXJyYXkgPSB0aGlzLmFycmF5LmJpbmQodGhpcyk7XG4gICAgdGhpcy5wcm9taXNlID0gdGhpcy5wcm9taXNlLmJpbmQodGhpcyk7XG4gICAgdGhpcy5vciA9IHRoaXMub3IuYmluZCh0aGlzKTtcbiAgICB0aGlzLmFuZCA9IHRoaXMuYW5kLmJpbmQodGhpcyk7XG4gICAgdGhpcy50cmFuc2Zvcm0gPSB0aGlzLnRyYW5zZm9ybS5iaW5kKHRoaXMpO1xuICAgIHRoaXMuZGVmYXVsdCA9IHRoaXMuZGVmYXVsdC5iaW5kKHRoaXMpO1xuICAgIHRoaXMuZGVzY3JpYmUgPSB0aGlzLmRlc2NyaWJlLmJpbmQodGhpcyk7XG4gICAgdGhpcy5pc051bGxhYmxlID0gdGhpcy5pc051bGxhYmxlLmJpbmQodGhpcyk7XG4gICAgdGhpcy5pc09wdGlvbmFsID0gdGhpcy5pc09wdGlvbmFsLmJpbmQodGhpcyk7XG4gIH1cbiAgZ2V0IGRlc2NyaXB0aW9uKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuZGVzY3JpcHRpb247XG4gIH1cbiAgX2dldFR5cGUoaW5wdXQpIHtcbiAgICByZXR1cm4gZ2V0UGFyc2VkVHlwZShpbnB1dC5kYXRhKTtcbiAgfVxuICBfZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCkge1xuICAgIHJldHVybiBjdHggfHwge1xuICAgICAgY29tbW9uOiBpbnB1dC5wYXJlbnQuY29tbW9uLFxuICAgICAgZGF0YTogaW5wdXQuZGF0YSxcbiAgICAgIHBhcnNlZFR5cGU6IGdldFBhcnNlZFR5cGUoaW5wdXQuZGF0YSksXG4gICAgICBzY2hlbWFFcnJvck1hcDogdGhpcy5fZGVmLmVycm9yTWFwLFxuICAgICAgcGF0aDogaW5wdXQucGF0aCxcbiAgICAgIHBhcmVudDogaW5wdXQucGFyZW50XG4gICAgfTtcbiAgfVxuICBfcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KSB7XG4gICAgcmV0dXJuIHtcbiAgICAgIHN0YXR1czogbmV3IFBhcnNlU3RhdHVzKCksXG4gICAgICBjdHg6IHtcbiAgICAgICAgY29tbW9uOiBpbnB1dC5wYXJlbnQuY29tbW9uLFxuICAgICAgICBkYXRhOiBpbnB1dC5kYXRhLFxuICAgICAgICBwYXJzZWRUeXBlOiBnZXRQYXJzZWRUeXBlKGlucHV0LmRhdGEpLFxuICAgICAgICBzY2hlbWFFcnJvck1hcDogdGhpcy5fZGVmLmVycm9yTWFwLFxuICAgICAgICBwYXRoOiBpbnB1dC5wYXRoLFxuICAgICAgICBwYXJlbnQ6IGlucHV0LnBhcmVudFxuICAgICAgfVxuICAgIH07XG4gIH1cbiAgX3BhcnNlU3luYyhpbnB1dCkge1xuICAgIGNvbnN0IHJlc3VsdCA9IHRoaXMuX3BhcnNlKGlucHV0KTtcbiAgICBpZiAoaXNBc3luYyhyZXN1bHQpKSB7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoXCJTeW5jaHJvbm91cyBwYXJzZSBlbmNvdW50ZXJlZCBwcm9taXNlLlwiKTtcbiAgICB9XG4gICAgcmV0dXJuIHJlc3VsdDtcbiAgfVxuICBfcGFyc2VBc3luYyhpbnB1dCkge1xuICAgIGNvbnN0IHJlc3VsdCA9IHRoaXMuX3BhcnNlKGlucHV0KTtcbiAgICByZXR1cm4gUHJvbWlzZS5yZXNvbHZlKHJlc3VsdCk7XG4gIH1cbiAgcGFyc2UoZGF0YSwgcGFyYW1zKSB7XG4gICAgY29uc3QgcmVzdWx0ID0gdGhpcy5zYWZlUGFyc2UoZGF0YSwgcGFyYW1zKTtcbiAgICBpZiAocmVzdWx0LnN1Y2Nlc3MpXG4gICAgICByZXR1cm4gcmVzdWx0LmRhdGE7XG4gICAgdGhyb3cgcmVzdWx0LmVycm9yO1xuICB9XG4gIHNhZmVQYXJzZShkYXRhLCBwYXJhbXMpIHtcbiAgICB2YXIgX2E7XG4gICAgY29uc3QgY3R4ID0ge1xuICAgICAgY29tbW9uOiB7XG4gICAgICAgIGlzc3VlczogW10sXG4gICAgICAgIGFzeW5jOiAoX2EgPSBwYXJhbXMgPT09IG51bGwgfHwgcGFyYW1zID09PSB2b2lkIDAgPyB2b2lkIDAgOiBwYXJhbXMuYXN5bmMpICE9PSBudWxsICYmIF9hICE9PSB2b2lkIDAgPyBfYSA6IGZhbHNlLFxuICAgICAgICBjb250ZXh0dWFsRXJyb3JNYXA6IHBhcmFtcyA9PT0gbnVsbCB8fCBwYXJhbXMgPT09IHZvaWQgMCA/IHZvaWQgMCA6IHBhcmFtcy5lcnJvck1hcFxuICAgICAgfSxcbiAgICAgIHBhdGg6IChwYXJhbXMgPT09IG51bGwgfHwgcGFyYW1zID09PSB2b2lkIDAgPyB2b2lkIDAgOiBwYXJhbXMucGF0aCkgfHwgW10sXG4gICAgICBzY2hlbWFFcnJvck1hcDogdGhpcy5fZGVmLmVycm9yTWFwLFxuICAgICAgcGFyZW50OiBudWxsLFxuICAgICAgZGF0YSxcbiAgICAgIHBhcnNlZFR5cGU6IGdldFBhcnNlZFR5cGUoZGF0YSlcbiAgICB9O1xuICAgIGNvbnN0IHJlc3VsdCA9IHRoaXMuX3BhcnNlU3luYyh7ZGF0YSwgcGF0aDogY3R4LnBhdGgsIHBhcmVudDogY3R4fSk7XG4gICAgcmV0dXJuIGhhbmRsZVJlc3VsdChjdHgsIHJlc3VsdCk7XG4gIH1cbiAgYXN5bmMgcGFyc2VBc3luYyhkYXRhLCBwYXJhbXMpIHtcbiAgICBjb25zdCByZXN1bHQgPSBhd2FpdCB0aGlzLnNhZmVQYXJzZUFzeW5jKGRhdGEsIHBhcmFtcyk7XG4gICAgaWYgKHJlc3VsdC5zdWNjZXNzKVxuICAgICAgcmV0dXJuIHJlc3VsdC5kYXRhO1xuICAgIHRocm93IHJlc3VsdC5lcnJvcjtcbiAgfVxuICBhc3luYyBzYWZlUGFyc2VBc3luYyhkYXRhLCBwYXJhbXMpIHtcbiAgICBjb25zdCBjdHggPSB7XG4gICAgICBjb21tb246IHtcbiAgICAgICAgaXNzdWVzOiBbXSxcbiAgICAgICAgY29udGV4dHVhbEVycm9yTWFwOiBwYXJhbXMgPT09IG51bGwgfHwgcGFyYW1zID09PSB2b2lkIDAgPyB2b2lkIDAgOiBwYXJhbXMuZXJyb3JNYXAsXG4gICAgICAgIGFzeW5jOiB0cnVlXG4gICAgICB9LFxuICAgICAgcGF0aDogKHBhcmFtcyA9PT0gbnVsbCB8fCBwYXJhbXMgPT09IHZvaWQgMCA/IHZvaWQgMCA6IHBhcmFtcy5wYXRoKSB8fCBbXSxcbiAgICAgIHNjaGVtYUVycm9yTWFwOiB0aGlzLl9kZWYuZXJyb3JNYXAsXG4gICAgICBwYXJlbnQ6IG51bGwsXG4gICAgICBkYXRhLFxuICAgICAgcGFyc2VkVHlwZTogZ2V0UGFyc2VkVHlwZShkYXRhKVxuICAgIH07XG4gICAgY29uc3QgbWF5YmVBc3luY1Jlc3VsdCA9IHRoaXMuX3BhcnNlKHtkYXRhLCBwYXRoOiBbXSwgcGFyZW50OiBjdHh9KTtcbiAgICBjb25zdCByZXN1bHQgPSBhd2FpdCAoaXNBc3luYyhtYXliZUFzeW5jUmVzdWx0KSA/IG1heWJlQXN5bmNSZXN1bHQgOiBQcm9taXNlLnJlc29sdmUobWF5YmVBc3luY1Jlc3VsdCkpO1xuICAgIHJldHVybiBoYW5kbGVSZXN1bHQoY3R4LCByZXN1bHQpO1xuICB9XG4gIHJlZmluZShjaGVjaywgbWVzc2FnZSkge1xuICAgIGNvbnN0IGdldElzc3VlUHJvcGVydGllcyA9ICh2YWwpID0+IHtcbiAgICAgIGlmICh0eXBlb2YgbWVzc2FnZSA9PT0gXCJzdHJpbmdcIiB8fCB0eXBlb2YgbWVzc2FnZSA9PT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICByZXR1cm4ge21lc3NhZ2V9O1xuICAgICAgfSBlbHNlIGlmICh0eXBlb2YgbWVzc2FnZSA9PT0gXCJmdW5jdGlvblwiKSB7XG4gICAgICAgIHJldHVybiBtZXNzYWdlKHZhbCk7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gbWVzc2FnZTtcbiAgICAgIH1cbiAgICB9O1xuICAgIHJldHVybiB0aGlzLl9yZWZpbmVtZW50KCh2YWwsIGN0eCkgPT4ge1xuICAgICAgY29uc3QgcmVzdWx0ID0gY2hlY2sodmFsKTtcbiAgICAgIGNvbnN0IHNldEVycm9yID0gKCkgPT4gY3R4LmFkZElzc3VlKHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmN1c3RvbSxcbiAgICAgICAgLi4uZ2V0SXNzdWVQcm9wZXJ0aWVzKHZhbClcbiAgICAgIH0pO1xuICAgICAgaWYgKHR5cGVvZiBQcm9taXNlICE9PSBcInVuZGVmaW5lZFwiICYmIHJlc3VsdCBpbnN0YW5jZW9mIFByb21pc2UpIHtcbiAgICAgICAgcmV0dXJuIHJlc3VsdC50aGVuKChkYXRhKSA9PiB7XG4gICAgICAgICAgaWYgKCFkYXRhKSB7XG4gICAgICAgICAgICBzZXRFcnJvcigpO1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICByZXR1cm4gdHJ1ZTtcbiAgICAgICAgICB9XG4gICAgICAgIH0pO1xuICAgICAgfVxuICAgICAgaWYgKCFyZXN1bHQpIHtcbiAgICAgICAgc2V0RXJyb3IoKTtcbiAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmV0dXJuIHRydWU7XG4gICAgICB9XG4gICAgfSk7XG4gIH1cbiAgcmVmaW5lbWVudChjaGVjaywgcmVmaW5lbWVudERhdGEpIHtcbiAgICByZXR1cm4gdGhpcy5fcmVmaW5lbWVudCgodmFsLCBjdHgpID0+IHtcbiAgICAgIGlmICghY2hlY2sodmFsKSkge1xuICAgICAgICBjdHguYWRkSXNzdWUodHlwZW9mIHJlZmluZW1lbnREYXRhID09PSBcImZ1bmN0aW9uXCIgPyByZWZpbmVtZW50RGF0YSh2YWwsIGN0eCkgOiByZWZpbmVtZW50RGF0YSk7XG4gICAgICAgIHJldHVybiBmYWxzZTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJldHVybiB0cnVlO1xuICAgICAgfVxuICAgIH0pO1xuICB9XG4gIF9yZWZpbmVtZW50KHJlZmluZW1lbnQpIHtcbiAgICByZXR1cm4gbmV3IFpvZEVmZmVjdHMoe1xuICAgICAgc2NoZW1hOiB0aGlzLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RFZmZlY3RzLFxuICAgICAgZWZmZWN0OiB7dHlwZTogXCJyZWZpbmVtZW50XCIsIHJlZmluZW1lbnR9XG4gICAgfSk7XG4gIH1cbiAgb3B0aW9uYWwoKSB7XG4gICAgcmV0dXJuIFpvZE9wdGlvbmFsLmNyZWF0ZSh0aGlzKTtcbiAgfVxuICBudWxsYWJsZSgpIHtcbiAgICByZXR1cm4gWm9kTnVsbGFibGUuY3JlYXRlKHRoaXMpO1xuICB9XG4gIG51bGxpc2goKSB7XG4gICAgcmV0dXJuIHRoaXMub3B0aW9uYWwoKS5udWxsYWJsZSgpO1xuICB9XG4gIGFycmF5KCkge1xuICAgIHJldHVybiBab2RBcnJheS5jcmVhdGUodGhpcyk7XG4gIH1cbiAgcHJvbWlzZSgpIHtcbiAgICByZXR1cm4gWm9kUHJvbWlzZS5jcmVhdGUodGhpcyk7XG4gIH1cbiAgb3Iob3B0aW9uKSB7XG4gICAgcmV0dXJuIFpvZFVuaW9uLmNyZWF0ZShbdGhpcywgb3B0aW9uXSk7XG4gIH1cbiAgYW5kKGluY29taW5nKSB7XG4gICAgcmV0dXJuIFpvZEludGVyc2VjdGlvbi5jcmVhdGUodGhpcywgaW5jb21pbmcpO1xuICB9XG4gIHRyYW5zZm9ybSh0cmFuc2Zvcm0pIHtcbiAgICByZXR1cm4gbmV3IFpvZEVmZmVjdHMoe1xuICAgICAgc2NoZW1hOiB0aGlzLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RFZmZlY3RzLFxuICAgICAgZWZmZWN0OiB7dHlwZTogXCJ0cmFuc2Zvcm1cIiwgdHJhbnNmb3JtfVxuICAgIH0pO1xuICB9XG4gIGRlZmF1bHQoZGVmKSB7XG4gICAgY29uc3QgZGVmYXVsdFZhbHVlRnVuYyA9IHR5cGVvZiBkZWYgPT09IFwiZnVuY3Rpb25cIiA/IGRlZiA6ICgpID0+IGRlZjtcbiAgICByZXR1cm4gbmV3IFpvZERlZmF1bHQoe1xuICAgICAgaW5uZXJUeXBlOiB0aGlzLFxuICAgICAgZGVmYXVsdFZhbHVlOiBkZWZhdWx0VmFsdWVGdW5jLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2REZWZhdWx0XG4gICAgfSk7XG4gIH1cbiAgZGVzY3JpYmUoZGVzY3JpcHRpb24pIHtcbiAgICBjb25zdCBUaGlzID0gdGhpcy5jb25zdHJ1Y3RvcjtcbiAgICByZXR1cm4gbmV3IFRoaXMoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgZGVzY3JpcHRpb25cbiAgICB9KTtcbiAgfVxuICBpc09wdGlvbmFsKCkge1xuICAgIHJldHVybiB0aGlzLnNhZmVQYXJzZSh2b2lkIDApLnN1Y2Nlc3M7XG4gIH1cbiAgaXNOdWxsYWJsZSgpIHtcbiAgICByZXR1cm4gdGhpcy5zYWZlUGFyc2UobnVsbCkuc3VjY2VzcztcbiAgfVxufTtcbnZhciBjdWlkUmVnZXggPSAvXmNbXlxccy1dezgsfSQvaTtcbnZhciB1dWlkUmVnZXggPSAvXihbYS1mMC05XXs4fS1bYS1mMC05XXs0fS1bMS01XVthLWYwLTldezN9LVthLWYwLTldezR9LVthLWYwLTldezEyfXwwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDApJC9pO1xudmFyIGVtYWlsUmVnZXggPSAvXigoW148PigpW1xcXVxcLiw7Olxcc0BcXFwiXSsoXFwuW148PigpW1xcXVxcLiw7Olxcc0BcXFwiXSspKil8KFxcXCIuK1xcXCIpKUAoKFtePD4oKVtcXF1cXC4sOzpcXHNAXFxcIl0rXFwuKStbXjw+KClbXFxdXFwuLDs6XFxzQFxcXCJdezIsfSkkL2k7XG52YXIgWm9kU3RyaW5nID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgY29uc3RydWN0b3IoKSB7XG4gICAgc3VwZXIoLi4uYXJndW1lbnRzKTtcbiAgICB0aGlzLl9yZWdleCA9IChyZWdleCwgdmFsaWRhdGlvbiwgbWVzc2FnZSkgPT4gdGhpcy5yZWZpbmVtZW50KChkYXRhKSA9PiByZWdleC50ZXN0KGRhdGEpLCB7XG4gICAgICB2YWxpZGF0aW9uLFxuICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfc3RyaW5nLFxuICAgICAgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpXG4gICAgfSk7XG4gICAgdGhpcy5ub25lbXB0eSA9IChtZXNzYWdlKSA9PiB0aGlzLm1pbigxLCBlcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSkpO1xuICAgIHRoaXMudHJpbSA9ICgpID0+IG5ldyBab2RTdHJpbmcoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgY2hlY2tzOiBbLi4udGhpcy5fZGVmLmNoZWNrcywge2tpbmQ6IFwidHJpbVwifV1cbiAgICB9KTtcbiAgfVxuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuc3RyaW5nKSB7XG4gICAgICBjb25zdCBjdHgyID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4Miwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5zdHJpbmcsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgyLnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGNvbnN0IHN0YXR1cyA9IG5ldyBQYXJzZVN0YXR1cygpO1xuICAgIGxldCBjdHggPSB2b2lkIDA7XG4gICAgZm9yIChjb25zdCBjaGVjayBvZiB0aGlzLl9kZWYuY2hlY2tzKSB7XG4gICAgICBpZiAoY2hlY2sua2luZCA9PT0gXCJtaW5cIikge1xuICAgICAgICBpZiAoaW5wdXQuZGF0YS5sZW5ndGggPCBjaGVjay52YWx1ZSkge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19zbWFsbCxcbiAgICAgICAgICAgIG1pbmltdW06IGNoZWNrLnZhbHVlLFxuICAgICAgICAgICAgdHlwZTogXCJzdHJpbmdcIixcbiAgICAgICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcIm1heFwiKSB7XG4gICAgICAgIGlmIChpbnB1dC5kYXRhLmxlbmd0aCA+IGNoZWNrLnZhbHVlKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX2JpZyxcbiAgICAgICAgICAgIG1heGltdW06IGNoZWNrLnZhbHVlLFxuICAgICAgICAgICAgdHlwZTogXCJzdHJpbmdcIixcbiAgICAgICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcImVtYWlsXCIpIHtcbiAgICAgICAgaWYgKCFlbWFpbFJlZ2V4LnRlc3QoaW5wdXQuZGF0YSkpIHtcbiAgICAgICAgICBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCwgY3R4KTtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIHZhbGlkYXRpb246IFwiZW1haWxcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcInV1aWRcIikge1xuICAgICAgICBpZiAoIXV1aWRSZWdleC50ZXN0KGlucHV0LmRhdGEpKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICB2YWxpZGF0aW9uOiBcInV1aWRcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcImN1aWRcIikge1xuICAgICAgICBpZiAoIWN1aWRSZWdleC50ZXN0KGlucHV0LmRhdGEpKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICB2YWxpZGF0aW9uOiBcImN1aWRcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcInVybFwiKSB7XG4gICAgICAgIHRyeSB7XG4gICAgICAgICAgbmV3IFVSTChpbnB1dC5kYXRhKTtcbiAgICAgICAgfSBjYXRjaCAoX2EpIHtcbiAgICAgICAgICBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCwgY3R4KTtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIHZhbGlkYXRpb246IFwidXJsXCIsXG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF9zdHJpbmcsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJyZWdleFwiKSB7XG4gICAgICAgIGNoZWNrLnJlZ2V4Lmxhc3RJbmRleCA9IDA7XG4gICAgICAgIGNvbnN0IHRlc3RSZXN1bHQgPSBjaGVjay5yZWdleC50ZXN0KGlucHV0LmRhdGEpO1xuICAgICAgICBpZiAoIXRlc3RSZXN1bHQpIHtcbiAgICAgICAgICBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCwgY3R4KTtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIHZhbGlkYXRpb246IFwicmVnZXhcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcInRyaW1cIikge1xuICAgICAgICBpbnB1dC5kYXRhID0gaW5wdXQuZGF0YS50cmltKCk7XG4gICAgICB9IGVsc2UgaWYgKGNoZWNrLmtpbmQgPT09IFwic3RhcnRzV2l0aFwiKSB7XG4gICAgICAgIGlmICghaW5wdXQuZGF0YS5zdGFydHNXaXRoKGNoZWNrLnZhbHVlKSkge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfc3RyaW5nLFxuICAgICAgICAgICAgdmFsaWRhdGlvbjoge3N0YXJ0c1dpdGg6IGNoZWNrLnZhbHVlfSxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcImVuZHNXaXRoXCIpIHtcbiAgICAgICAgaWYgKCFpbnB1dC5kYXRhLmVuZHNXaXRoKGNoZWNrLnZhbHVlKSkge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfc3RyaW5nLFxuICAgICAgICAgICAgdmFsaWRhdGlvbjoge2VuZHNXaXRoOiBjaGVjay52YWx1ZX0sXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHV0aWwuYXNzZXJ0TmV2ZXIoY2hlY2spO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4ge3N0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogaW5wdXQuZGF0YX07XG4gIH1cbiAgX2FkZENoZWNrKGNoZWNrKSB7XG4gICAgcmV0dXJuIG5ldyBab2RTdHJpbmcoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgY2hlY2tzOiBbLi4udGhpcy5fZGVmLmNoZWNrcywgY2hlY2tdXG4gICAgfSk7XG4gIH1cbiAgZW1haWwobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7a2luZDogXCJlbWFpbFwiLCAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSl9KTtcbiAgfVxuICB1cmwobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7a2luZDogXCJ1cmxcIiwgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpfSk7XG4gIH1cbiAgdXVpZChtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtraW5kOiBcInV1aWRcIiwgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpfSk7XG4gIH1cbiAgY3VpZChtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtraW5kOiBcImN1aWRcIiwgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpfSk7XG4gIH1cbiAgcmVnZXgocmVnZXgsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soe1xuICAgICAga2luZDogXCJyZWdleFwiLFxuICAgICAgcmVnZXgsXG4gICAgICAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSlcbiAgICB9KTtcbiAgfVxuICBzdGFydHNXaXRoKHZhbHVlLCBtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtcbiAgICAgIGtpbmQ6IFwic3RhcnRzV2l0aFwiLFxuICAgICAgdmFsdWUsXG4gICAgICAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSlcbiAgICB9KTtcbiAgfVxuICBlbmRzV2l0aCh2YWx1ZSwgbWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcImVuZHNXaXRoXCIsXG4gICAgICB2YWx1ZSxcbiAgICAgIC4uLmVycm9yVXRpbC5lcnJUb09iaihtZXNzYWdlKVxuICAgIH0pO1xuICB9XG4gIG1pbihtaW5MZW5ndGgsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soe1xuICAgICAga2luZDogXCJtaW5cIixcbiAgICAgIHZhbHVlOiBtaW5MZW5ndGgsXG4gICAgICAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSlcbiAgICB9KTtcbiAgfVxuICBtYXgobWF4TGVuZ3RoLCBtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtcbiAgICAgIGtpbmQ6IFwibWF4XCIsXG4gICAgICB2YWx1ZTogbWF4TGVuZ3RoLFxuICAgICAgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpXG4gICAgfSk7XG4gIH1cbiAgbGVuZ3RoKGxlbiwgbWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLm1pbihsZW4sIG1lc3NhZ2UpLm1heChsZW4sIG1lc3NhZ2UpO1xuICB9XG4gIGdldCBpc0VtYWlsKCkge1xuICAgIHJldHVybiAhIXRoaXMuX2RlZi5jaGVja3MuZmluZCgoY2gpID0+IGNoLmtpbmQgPT09IFwiZW1haWxcIik7XG4gIH1cbiAgZ2V0IGlzVVJMKCkge1xuICAgIHJldHVybiAhIXRoaXMuX2RlZi5jaGVja3MuZmluZCgoY2gpID0+IGNoLmtpbmQgPT09IFwidXJsXCIpO1xuICB9XG4gIGdldCBpc1VVSUQoKSB7XG4gICAgcmV0dXJuICEhdGhpcy5fZGVmLmNoZWNrcy5maW5kKChjaCkgPT4gY2gua2luZCA9PT0gXCJ1dWlkXCIpO1xuICB9XG4gIGdldCBpc0NVSUQoKSB7XG4gICAgcmV0dXJuICEhdGhpcy5fZGVmLmNoZWNrcy5maW5kKChjaCkgPT4gY2gua2luZCA9PT0gXCJjdWlkXCIpO1xuICB9XG4gIGdldCBtaW5MZW5ndGgoKSB7XG4gICAgbGV0IG1pbiA9IG51bGw7XG4gICAgZm9yIChjb25zdCBjaCBvZiB0aGlzLl9kZWYuY2hlY2tzKSB7XG4gICAgICBpZiAoY2gua2luZCA9PT0gXCJtaW5cIikge1xuICAgICAgICBpZiAobWluID09PSBudWxsIHx8IGNoLnZhbHVlID4gbWluKVxuICAgICAgICAgIG1pbiA9IGNoLnZhbHVlO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4gbWluO1xuICB9XG4gIGdldCBtYXhMZW5ndGgoKSB7XG4gICAgbGV0IG1heCA9IG51bGw7XG4gICAgZm9yIChjb25zdCBjaCBvZiB0aGlzLl9kZWYuY2hlY2tzKSB7XG4gICAgICBpZiAoY2gua2luZCA9PT0gXCJtYXhcIikge1xuICAgICAgICBpZiAobWF4ID09PSBudWxsIHx8IGNoLnZhbHVlIDwgbWF4KVxuICAgICAgICAgIG1heCA9IGNoLnZhbHVlO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4gbWF4O1xuICB9XG59O1xuWm9kU3RyaW5nLmNyZWF0ZSA9IChwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RTdHJpbmcoe1xuICAgIGNoZWNrczogW10sXG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RTdHJpbmcsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbmZ1bmN0aW9uIGZsb2F0U2FmZVJlbWFpbmRlcih2YWwsIHN0ZXApIHtcbiAgY29uc3QgdmFsRGVjQ291bnQgPSAodmFsLnRvU3RyaW5nKCkuc3BsaXQoXCIuXCIpWzFdIHx8IFwiXCIpLmxlbmd0aDtcbiAgY29uc3Qgc3RlcERlY0NvdW50ID0gKHN0ZXAudG9TdHJpbmcoKS5zcGxpdChcIi5cIilbMV0gfHwgXCJcIikubGVuZ3RoO1xuICBjb25zdCBkZWNDb3VudCA9IHZhbERlY0NvdW50ID4gc3RlcERlY0NvdW50ID8gdmFsRGVjQ291bnQgOiBzdGVwRGVjQ291bnQ7XG4gIGNvbnN0IHZhbEludCA9IHBhcnNlSW50KHZhbC50b0ZpeGVkKGRlY0NvdW50KS5yZXBsYWNlKFwiLlwiLCBcIlwiKSk7XG4gIGNvbnN0IHN0ZXBJbnQgPSBwYXJzZUludChzdGVwLnRvRml4ZWQoZGVjQ291bnQpLnJlcGxhY2UoXCIuXCIsIFwiXCIpKTtcbiAgcmV0dXJuIHZhbEludCAlIHN0ZXBJbnQgLyBNYXRoLnBvdygxMCwgZGVjQ291bnQpO1xufVxudmFyIFpvZE51bWJlciA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIGNvbnN0cnVjdG9yKCkge1xuICAgIHN1cGVyKC4uLmFyZ3VtZW50cyk7XG4gICAgdGhpcy5taW4gPSB0aGlzLmd0ZTtcbiAgICB0aGlzLm1heCA9IHRoaXMubHRlO1xuICAgIHRoaXMuc3RlcCA9IHRoaXMubXVsdGlwbGVPZjtcbiAgfVxuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUubnVtYmVyKSB7XG4gICAgICBjb25zdCBjdHgyID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4Miwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5udW1iZXIsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgyLnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGxldCBjdHggPSB2b2lkIDA7XG4gICAgY29uc3Qgc3RhdHVzID0gbmV3IFBhcnNlU3RhdHVzKCk7XG4gICAgZm9yIChjb25zdCBjaGVjayBvZiB0aGlzLl9kZWYuY2hlY2tzKSB7XG4gICAgICBpZiAoY2hlY2sua2luZCA9PT0gXCJpbnRcIikge1xuICAgICAgICBpZiAoIXV0aWwuaXNJbnRlZ2VyKGlucHV0LmRhdGEpKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICAgICAgZXhwZWN0ZWQ6IFwiaW50ZWdlclwiLFxuICAgICAgICAgICAgcmVjZWl2ZWQ6IFwiZmxvYXRcIixcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2VcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcIm1pblwiKSB7XG4gICAgICAgIGNvbnN0IHRvb1NtYWxsID0gY2hlY2suaW5jbHVzaXZlID8gaW5wdXQuZGF0YSA8IGNoZWNrLnZhbHVlIDogaW5wdXQuZGF0YSA8PSBjaGVjay52YWx1ZTtcbiAgICAgICAgaWYgKHRvb1NtYWxsKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX3NtYWxsLFxuICAgICAgICAgICAgbWluaW11bTogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICB0eXBlOiBcIm51bWJlclwiLFxuICAgICAgICAgICAgaW5jbHVzaXZlOiBjaGVjay5pbmNsdXNpdmUsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJtYXhcIikge1xuICAgICAgICBjb25zdCB0b29CaWcgPSBjaGVjay5pbmNsdXNpdmUgPyBpbnB1dC5kYXRhID4gY2hlY2sudmFsdWUgOiBpbnB1dC5kYXRhID49IGNoZWNrLnZhbHVlO1xuICAgICAgICBpZiAodG9vQmlnKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX2JpZyxcbiAgICAgICAgICAgIG1heGltdW06IGNoZWNrLnZhbHVlLFxuICAgICAgICAgICAgdHlwZTogXCJudW1iZXJcIixcbiAgICAgICAgICAgIGluY2x1c2l2ZTogY2hlY2suaW5jbHVzaXZlLFxuICAgICAgICAgICAgbWVzc2FnZTogY2hlY2subWVzc2FnZVxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGNoZWNrLmtpbmQgPT09IFwibXVsdGlwbGVPZlwiKSB7XG4gICAgICAgIGlmIChmbG9hdFNhZmVSZW1haW5kZXIoaW5wdXQuZGF0YSwgY2hlY2sudmFsdWUpICE9PSAwKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUubm90X211bHRpcGxlX29mLFxuICAgICAgICAgICAgbXVsdGlwbGVPZjogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHV0aWwuYXNzZXJ0TmV2ZXIoY2hlY2spO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4ge3N0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogaW5wdXQuZGF0YX07XG4gIH1cbiAgZ3RlKHZhbHVlLCBtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuc2V0TGltaXQoXCJtaW5cIiwgdmFsdWUsIHRydWUsIGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSk7XG4gIH1cbiAgZ3QodmFsdWUsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5zZXRMaW1pdChcIm1pblwiLCB2YWx1ZSwgZmFsc2UsIGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSk7XG4gIH1cbiAgbHRlKHZhbHVlLCBtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuc2V0TGltaXQoXCJtYXhcIiwgdmFsdWUsIHRydWUsIGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSk7XG4gIH1cbiAgbHQodmFsdWUsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5zZXRMaW1pdChcIm1heFwiLCB2YWx1ZSwgZmFsc2UsIGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSk7XG4gIH1cbiAgc2V0TGltaXQoa2luZCwgdmFsdWUsIGluY2x1c2l2ZSwgbWVzc2FnZSkge1xuICAgIHJldHVybiBuZXcgWm9kTnVtYmVyKHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIGNoZWNrczogW1xuICAgICAgICAuLi50aGlzLl9kZWYuY2hlY2tzLFxuICAgICAgICB7XG4gICAgICAgICAga2luZCxcbiAgICAgICAgICB2YWx1ZSxcbiAgICAgICAgICBpbmNsdXNpdmUsXG4gICAgICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpXG4gICAgICAgIH1cbiAgICAgIF1cbiAgICB9KTtcbiAgfVxuICBfYWRkQ2hlY2soY2hlY2spIHtcbiAgICByZXR1cm4gbmV3IFpvZE51bWJlcih7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBjaGVja3M6IFsuLi50aGlzLl9kZWYuY2hlY2tzLCBjaGVja11cbiAgICB9KTtcbiAgfVxuICBpbnQobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcImludFwiLFxuICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpXG4gICAgfSk7XG4gIH1cbiAgcG9zaXRpdmUobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcIm1pblwiLFxuICAgICAgdmFsdWU6IDAsXG4gICAgICBpbmNsdXNpdmU6IGZhbHNlLFxuICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpXG4gICAgfSk7XG4gIH1cbiAgbmVnYXRpdmUobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcIm1heFwiLFxuICAgICAgdmFsdWU6IDAsXG4gICAgICBpbmNsdXNpdmU6IGZhbHNlLFxuICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpXG4gICAgfSk7XG4gIH1cbiAgbm9ucG9zaXRpdmUobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcIm1heFwiLFxuICAgICAgdmFsdWU6IDAsXG4gICAgICBpbmNsdXNpdmU6IHRydWUsXG4gICAgICBtZXNzYWdlOiBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSlcbiAgICB9KTtcbiAgfVxuICBub25uZWdhdGl2ZShtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtcbiAgICAgIGtpbmQ6IFwibWluXCIsXG4gICAgICB2YWx1ZTogMCxcbiAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKVxuICAgIH0pO1xuICB9XG4gIG11bHRpcGxlT2YodmFsdWUsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soe1xuICAgICAga2luZDogXCJtdWx0aXBsZU9mXCIsXG4gICAgICB2YWx1ZSxcbiAgICAgIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKVxuICAgIH0pO1xuICB9XG4gIGdldCBtaW5WYWx1ZSgpIHtcbiAgICBsZXQgbWluID0gbnVsbDtcbiAgICBmb3IgKGNvbnN0IGNoIG9mIHRoaXMuX2RlZi5jaGVja3MpIHtcbiAgICAgIGlmIChjaC5raW5kID09PSBcIm1pblwiKSB7XG4gICAgICAgIGlmIChtaW4gPT09IG51bGwgfHwgY2gudmFsdWUgPiBtaW4pXG4gICAgICAgICAgbWluID0gY2gudmFsdWU7XG4gICAgICB9XG4gICAgfVxuICAgIHJldHVybiBtaW47XG4gIH1cbiAgZ2V0IG1heFZhbHVlKCkge1xuICAgIGxldCBtYXggPSBudWxsO1xuICAgIGZvciAoY29uc3QgY2ggb2YgdGhpcy5fZGVmLmNoZWNrcykge1xuICAgICAgaWYgKGNoLmtpbmQgPT09IFwibWF4XCIpIHtcbiAgICAgICAgaWYgKG1heCA9PT0gbnVsbCB8fCBjaC52YWx1ZSA8IG1heClcbiAgICAgICAgICBtYXggPSBjaC52YWx1ZTtcbiAgICAgIH1cbiAgICB9XG4gICAgcmV0dXJuIG1heDtcbiAgfVxuICBnZXQgaXNJbnQoKSB7XG4gICAgcmV0dXJuICEhdGhpcy5fZGVmLmNoZWNrcy5maW5kKChjaCkgPT4gY2gua2luZCA9PT0gXCJpbnRcIik7XG4gIH1cbn07XG5ab2ROdW1iZXIuY3JlYXRlID0gKHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZE51bWJlcih7XG4gICAgY2hlY2tzOiBbXSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE51bWJlcixcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZEJpZ0ludCA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5iaWdpbnQpIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5iaWdpbnQsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgcmV0dXJuIE9LKGlucHV0LmRhdGEpO1xuICB9XG59O1xuWm9kQmlnSW50LmNyZWF0ZSA9IChwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RCaWdJbnQoe1xuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kQmlnSW50LFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG52YXIgWm9kQm9vbGVhbiA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5ib29sZWFuKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUuYm9vbGVhbixcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICByZXR1cm4gT0soaW5wdXQuZGF0YSk7XG4gIH1cbn07XG5ab2RCb29sZWFuLmNyZWF0ZSA9IChwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RCb29sZWFuKHtcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEJvb2xlYW4sXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2REYXRlID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3QgcGFyc2VkVHlwZSA9IHRoaXMuX2dldFR5cGUoaW5wdXQpO1xuICAgIGlmIChwYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLmRhdGUpIHtcbiAgICAgIGNvbnN0IGN0eDIgPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgyLCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLmRhdGUsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgyLnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGlmIChpc05hTihpbnB1dC5kYXRhLmdldFRpbWUoKSkpIHtcbiAgICAgIGNvbnN0IGN0eDIgPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgyLCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX2RhdGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGNvbnN0IHN0YXR1cyA9IG5ldyBQYXJzZVN0YXR1cygpO1xuICAgIGxldCBjdHggPSB2b2lkIDA7XG4gICAgZm9yIChjb25zdCBjaGVjayBvZiB0aGlzLl9kZWYuY2hlY2tzKSB7XG4gICAgICBpZiAoY2hlY2sua2luZCA9PT0gXCJtaW5cIikge1xuICAgICAgICBpZiAoaW5wdXQuZGF0YS5nZXRUaW1lKCkgPCBjaGVjay52YWx1ZSkge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19zbWFsbCxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2UsXG4gICAgICAgICAgICBpbmNsdXNpdmU6IHRydWUsXG4gICAgICAgICAgICBtaW5pbXVtOiBjaGVjay52YWx1ZSxcbiAgICAgICAgICAgIHR5cGU6IFwiZGF0ZVwiXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJtYXhcIikge1xuICAgICAgICBpZiAoaW5wdXQuZGF0YS5nZXRUaW1lKCkgPiBjaGVjay52YWx1ZSkge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19iaWcsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlLFxuICAgICAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICAgICAgbWF4aW11bTogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICB0eXBlOiBcImRhdGVcIlxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICB1dGlsLmFzc2VydE5ldmVyKGNoZWNrKTtcbiAgICAgIH1cbiAgICB9XG4gICAgcmV0dXJuIHtcbiAgICAgIHN0YXR1czogc3RhdHVzLnZhbHVlLFxuICAgICAgdmFsdWU6IG5ldyBEYXRlKGlucHV0LmRhdGEuZ2V0VGltZSgpKVxuICAgIH07XG4gIH1cbiAgX2FkZENoZWNrKGNoZWNrKSB7XG4gICAgcmV0dXJuIG5ldyBab2REYXRlKHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIGNoZWNrczogWy4uLnRoaXMuX2RlZi5jaGVja3MsIGNoZWNrXVxuICAgIH0pO1xuICB9XG4gIG1pbihtaW5EYXRlLCBtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtcbiAgICAgIGtpbmQ6IFwibWluXCIsXG4gICAgICB2YWx1ZTogbWluRGF0ZS5nZXRUaW1lKCksXG4gICAgICBtZXNzYWdlOiBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSlcbiAgICB9KTtcbiAgfVxuICBtYXgobWF4RGF0ZSwgbWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcIm1heFwiLFxuICAgICAgdmFsdWU6IG1heERhdGUuZ2V0VGltZSgpLFxuICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpXG4gICAgfSk7XG4gIH1cbiAgZ2V0IG1pbkRhdGUoKSB7XG4gICAgbGV0IG1pbiA9IG51bGw7XG4gICAgZm9yIChjb25zdCBjaCBvZiB0aGlzLl9kZWYuY2hlY2tzKSB7XG4gICAgICBpZiAoY2gua2luZCA9PT0gXCJtaW5cIikge1xuICAgICAgICBpZiAobWluID09PSBudWxsIHx8IGNoLnZhbHVlID4gbWluKVxuICAgICAgICAgIG1pbiA9IGNoLnZhbHVlO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4gbWluICE9IG51bGwgPyBuZXcgRGF0ZShtaW4pIDogbnVsbDtcbiAgfVxuICBnZXQgbWF4RGF0ZSgpIHtcbiAgICBsZXQgbWF4ID0gbnVsbDtcbiAgICBmb3IgKGNvbnN0IGNoIG9mIHRoaXMuX2RlZi5jaGVja3MpIHtcbiAgICAgIGlmIChjaC5raW5kID09PSBcIm1heFwiKSB7XG4gICAgICAgIGlmIChtYXggPT09IG51bGwgfHwgY2gudmFsdWUgPCBtYXgpXG4gICAgICAgICAgbWF4ID0gY2gudmFsdWU7XG4gICAgICB9XG4gICAgfVxuICAgIHJldHVybiBtYXggIT0gbnVsbCA/IG5ldyBEYXRlKG1heCkgOiBudWxsO1xuICB9XG59O1xuWm9kRGF0ZS5jcmVhdGUgPSAocGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kRGF0ZSh7XG4gICAgY2hlY2tzOiBbXSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZERhdGUsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2RVbmRlZmluZWQgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUudW5kZWZpbmVkKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUudW5kZWZpbmVkLFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxufTtcblpvZFVuZGVmaW5lZC5jcmVhdGUgPSAocGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kVW5kZWZpbmVkKHtcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFVuZGVmaW5lZCxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZE51bGwgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUubnVsbCkge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm51bGwsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgcmV0dXJuIE9LKGlucHV0LmRhdGEpO1xuICB9XG59O1xuWm9kTnVsbC5jcmVhdGUgPSAocGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kTnVsbCh7XG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROdWxsLFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG52YXIgWm9kQW55ID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgY29uc3RydWN0b3IoKSB7XG4gICAgc3VwZXIoLi4uYXJndW1lbnRzKTtcbiAgICB0aGlzLl9hbnkgPSB0cnVlO1xuICB9XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxufTtcblpvZEFueS5jcmVhdGUgPSAocGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kQW55KHtcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEFueSxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZFVua25vd24gPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBjb25zdHJ1Y3RvcigpIHtcbiAgICBzdXBlciguLi5hcmd1bWVudHMpO1xuICAgIHRoaXMuX3Vua25vd24gPSB0cnVlO1xuICB9XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxufTtcblpvZFVua25vd24uY3JlYXRlID0gKHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZFVua25vd24oe1xuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kVW5rbm93bixcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZE5ldmVyID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm5ldmVyLFxuICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlXG4gICAgfSk7XG4gICAgcmV0dXJuIElOVkFMSUQ7XG4gIH1cbn07XG5ab2ROZXZlci5jcmVhdGUgPSAocGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kTmV2ZXIoe1xuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTmV2ZXIsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2RWb2lkID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3QgcGFyc2VkVHlwZSA9IHRoaXMuX2dldFR5cGUoaW5wdXQpO1xuICAgIGlmIChwYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLnVuZGVmaW5lZCkge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLnZvaWQsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgcmV0dXJuIE9LKGlucHV0LmRhdGEpO1xuICB9XG59O1xuWm9kVm9pZC5jcmVhdGUgPSAocGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kVm9pZCh7XG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RWb2lkLFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG52YXIgWm9kQXJyYXkgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7Y3R4LCBzdGF0dXN9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBjb25zdCBkZWYgPSB0aGlzLl9kZWY7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLmFycmF5KSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUuYXJyYXksXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgaWYgKGRlZi5taW5MZW5ndGggIT09IG51bGwpIHtcbiAgICAgIGlmIChjdHguZGF0YS5sZW5ndGggPCBkZWYubWluTGVuZ3RoLnZhbHVlKSB7XG4gICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS50b29fc21hbGwsXG4gICAgICAgICAgbWluaW11bTogZGVmLm1pbkxlbmd0aC52YWx1ZSxcbiAgICAgICAgICB0eXBlOiBcImFycmF5XCIsXG4gICAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICAgIG1lc3NhZ2U6IGRlZi5taW5MZW5ndGgubWVzc2FnZVxuICAgICAgICB9KTtcbiAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICB9XG4gICAgfVxuICAgIGlmIChkZWYubWF4TGVuZ3RoICE9PSBudWxsKSB7XG4gICAgICBpZiAoY3R4LmRhdGEubGVuZ3RoID4gZGVmLm1heExlbmd0aC52YWx1ZSkge1xuICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX2JpZyxcbiAgICAgICAgICBtYXhpbXVtOiBkZWYubWF4TGVuZ3RoLnZhbHVlLFxuICAgICAgICAgIHR5cGU6IFwiYXJyYXlcIixcbiAgICAgICAgICBpbmNsdXNpdmU6IHRydWUsXG4gICAgICAgICAgbWVzc2FnZTogZGVmLm1heExlbmd0aC5tZXNzYWdlXG4gICAgICAgIH0pO1xuICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgIH1cbiAgICB9XG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBQcm9taXNlLmFsbChjdHguZGF0YS5tYXAoKGl0ZW0sIGkpID0+IHtcbiAgICAgICAgcmV0dXJuIGRlZi50eXBlLl9wYXJzZUFzeW5jKG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCBpdGVtLCBjdHgucGF0aCwgaSkpO1xuICAgICAgfSkpLnRoZW4oKHJlc3VsdDIpID0+IHtcbiAgICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlQXJyYXkoc3RhdHVzLCByZXN1bHQyKTtcbiAgICAgIH0pO1xuICAgIH1cbiAgICBjb25zdCByZXN1bHQgPSBjdHguZGF0YS5tYXAoKGl0ZW0sIGkpID0+IHtcbiAgICAgIHJldHVybiBkZWYudHlwZS5fcGFyc2VTeW5jKG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCBpdGVtLCBjdHgucGF0aCwgaSkpO1xuICAgIH0pO1xuICAgIHJldHVybiBQYXJzZVN0YXR1cy5tZXJnZUFycmF5KHN0YXR1cywgcmVzdWx0KTtcbiAgfVxuICBnZXQgZWxlbWVudCgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnR5cGU7XG4gIH1cbiAgbWluKG1pbkxlbmd0aCwgbWVzc2FnZSkge1xuICAgIHJldHVybiBuZXcgWm9kQXJyYXkoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgbWluTGVuZ3RoOiB7dmFsdWU6IG1pbkxlbmd0aCwgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpfVxuICAgIH0pO1xuICB9XG4gIG1heChtYXhMZW5ndGgsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gbmV3IFpvZEFycmF5KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIG1heExlbmd0aDoge3ZhbHVlOiBtYXhMZW5ndGgsIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKX1cbiAgICB9KTtcbiAgfVxuICBsZW5ndGgobGVuLCBtZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMubWluKGxlbiwgbWVzc2FnZSkubWF4KGxlbiwgbWVzc2FnZSk7XG4gIH1cbiAgbm9uZW1wdHkobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLm1pbigxLCBtZXNzYWdlKTtcbiAgfVxufTtcblpvZEFycmF5LmNyZWF0ZSA9IChzY2hlbWEsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZEFycmF5KHtcbiAgICB0eXBlOiBzY2hlbWEsXG4gICAgbWluTGVuZ3RoOiBudWxsLFxuICAgIG1heExlbmd0aDogbnVsbCxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEFycmF5LFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG52YXIgb2JqZWN0VXRpbDtcbihmdW5jdGlvbihvYmplY3RVdGlsMikge1xuICBvYmplY3RVdGlsMi5tZXJnZVNoYXBlcyA9IChmaXJzdCwgc2Vjb25kKSA9PiB7XG4gICAgcmV0dXJuIHtcbiAgICAgIC4uLmZpcnN0LFxuICAgICAgLi4uc2Vjb25kXG4gICAgfTtcbiAgfTtcbn0pKG9iamVjdFV0aWwgfHwgKG9iamVjdFV0aWwgPSB7fSkpO1xudmFyIEF1Z21lbnRGYWN0b3J5ID0gKGRlZikgPT4gKGF1Z21lbnRhdGlvbikgPT4ge1xuICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgLi4uZGVmLFxuICAgIHNoYXBlOiAoKSA9PiAoe1xuICAgICAgLi4uZGVmLnNoYXBlKCksXG4gICAgICAuLi5hdWdtZW50YXRpb25cbiAgICB9KVxuICB9KTtcbn07XG5mdW5jdGlvbiBkZWVwUGFydGlhbGlmeShzY2hlbWEpIHtcbiAgaWYgKHNjaGVtYSBpbnN0YW5jZW9mIFpvZE9iamVjdCkge1xuICAgIGNvbnN0IG5ld1NoYXBlID0ge307XG4gICAgZm9yIChjb25zdCBrZXkgaW4gc2NoZW1hLnNoYXBlKSB7XG4gICAgICBjb25zdCBmaWVsZFNjaGVtYSA9IHNjaGVtYS5zaGFwZVtrZXldO1xuICAgICAgbmV3U2hhcGVba2V5XSA9IFpvZE9wdGlvbmFsLmNyZWF0ZShkZWVwUGFydGlhbGlmeShmaWVsZFNjaGVtYSkpO1xuICAgIH1cbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi5zY2hlbWEuX2RlZixcbiAgICAgIHNoYXBlOiAoKSA9PiBuZXdTaGFwZVxuICAgIH0pO1xuICB9IGVsc2UgaWYgKHNjaGVtYSBpbnN0YW5jZW9mIFpvZEFycmF5KSB7XG4gICAgcmV0dXJuIFpvZEFycmF5LmNyZWF0ZShkZWVwUGFydGlhbGlmeShzY2hlbWEuZWxlbWVudCkpO1xuICB9IGVsc2UgaWYgKHNjaGVtYSBpbnN0YW5jZW9mIFpvZE9wdGlvbmFsKSB7XG4gICAgcmV0dXJuIFpvZE9wdGlvbmFsLmNyZWF0ZShkZWVwUGFydGlhbGlmeShzY2hlbWEudW53cmFwKCkpKTtcbiAgfSBlbHNlIGlmIChzY2hlbWEgaW5zdGFuY2VvZiBab2ROdWxsYWJsZSkge1xuICAgIHJldHVybiBab2ROdWxsYWJsZS5jcmVhdGUoZGVlcFBhcnRpYWxpZnkoc2NoZW1hLnVud3JhcCgpKSk7XG4gIH0gZWxzZSBpZiAoc2NoZW1hIGluc3RhbmNlb2YgWm9kVHVwbGUpIHtcbiAgICByZXR1cm4gWm9kVHVwbGUuY3JlYXRlKHNjaGVtYS5pdGVtcy5tYXAoKGl0ZW0pID0+IGRlZXBQYXJ0aWFsaWZ5KGl0ZW0pKSk7XG4gIH0gZWxzZSB7XG4gICAgcmV0dXJuIHNjaGVtYTtcbiAgfVxufVxudmFyIFpvZE9iamVjdCA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIGNvbnN0cnVjdG9yKCkge1xuICAgIHN1cGVyKC4uLmFyZ3VtZW50cyk7XG4gICAgdGhpcy5fY2FjaGVkID0gbnVsbDtcbiAgICB0aGlzLm5vbnN0cmljdCA9IHRoaXMucGFzc3Rocm91Z2g7XG4gICAgdGhpcy5hdWdtZW50ID0gQXVnbWVudEZhY3RvcnkodGhpcy5fZGVmKTtcbiAgICB0aGlzLmV4dGVuZCA9IEF1Z21lbnRGYWN0b3J5KHRoaXMuX2RlZik7XG4gIH1cbiAgX2dldENhY2hlZCgpIHtcbiAgICBpZiAodGhpcy5fY2FjaGVkICE9PSBudWxsKVxuICAgICAgcmV0dXJuIHRoaXMuX2NhY2hlZDtcbiAgICBjb25zdCBzaGFwZSA9IHRoaXMuX2RlZi5zaGFwZSgpO1xuICAgIGNvbnN0IGtleXMgPSB1dGlsLm9iamVjdEtleXMoc2hhcGUpO1xuICAgIHJldHVybiB0aGlzLl9jYWNoZWQgPSB7c2hhcGUsIGtleXN9O1xuICB9XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5vYmplY3QpIHtcbiAgICAgIGNvbnN0IGN0eDIgPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgyLCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm9iamVjdCxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eDIucGFyc2VkVHlwZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgY29uc3Qge3N0YXR1cywgY3R4fSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgY29uc3Qge3NoYXBlLCBrZXlzOiBzaGFwZUtleXN9ID0gdGhpcy5fZ2V0Q2FjaGVkKCk7XG4gICAgY29uc3QgZXh0cmFLZXlzID0gW107XG4gICAgZm9yIChjb25zdCBrZXkgaW4gY3R4LmRhdGEpIHtcbiAgICAgIGlmICghc2hhcGVLZXlzLmluY2x1ZGVzKGtleSkpIHtcbiAgICAgICAgZXh0cmFLZXlzLnB1c2goa2V5KTtcbiAgICAgIH1cbiAgICB9XG4gICAgY29uc3QgcGFpcnMgPSBbXTtcbiAgICBmb3IgKGNvbnN0IGtleSBvZiBzaGFwZUtleXMpIHtcbiAgICAgIGNvbnN0IGtleVZhbGlkYXRvciA9IHNoYXBlW2tleV07XG4gICAgICBjb25zdCB2YWx1ZSA9IGN0eC5kYXRhW2tleV07XG4gICAgICBwYWlycy5wdXNoKHtcbiAgICAgICAga2V5OiB7c3RhdHVzOiBcInZhbGlkXCIsIHZhbHVlOiBrZXl9LFxuICAgICAgICB2YWx1ZToga2V5VmFsaWRhdG9yLl9wYXJzZShuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgdmFsdWUsIGN0eC5wYXRoLCBrZXkpKSxcbiAgICAgICAgYWx3YXlzU2V0OiBrZXkgaW4gY3R4LmRhdGFcbiAgICAgIH0pO1xuICAgIH1cbiAgICBpZiAodGhpcy5fZGVmLmNhdGNoYWxsIGluc3RhbmNlb2YgWm9kTmV2ZXIpIHtcbiAgICAgIGNvbnN0IHVua25vd25LZXlzID0gdGhpcy5fZGVmLnVua25vd25LZXlzO1xuICAgICAgaWYgKHVua25vd25LZXlzID09PSBcInBhc3N0aHJvdWdoXCIpIHtcbiAgICAgICAgZm9yIChjb25zdCBrZXkgb2YgZXh0cmFLZXlzKSB7XG4gICAgICAgICAgcGFpcnMucHVzaCh7XG4gICAgICAgICAgICBrZXk6IHtzdGF0dXM6IFwidmFsaWRcIiwgdmFsdWU6IGtleX0sXG4gICAgICAgICAgICB2YWx1ZToge3N0YXR1czogXCJ2YWxpZFwiLCB2YWx1ZTogY3R4LmRhdGFba2V5XX1cbiAgICAgICAgICB9KTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmICh1bmtub3duS2V5cyA9PT0gXCJzdHJpY3RcIikge1xuICAgICAgICBpZiAoZXh0cmFLZXlzLmxlbmd0aCA+IDApIHtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS51bnJlY29nbml6ZWRfa2V5cyxcbiAgICAgICAgICAgIGtleXM6IGV4dHJhS2V5c1xuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKHVua25vd25LZXlzID09PSBcInN0cmlwXCIpXG4gICAgICAgIDtcbiAgICAgIGVsc2Uge1xuICAgICAgICB0aHJvdyBuZXcgRXJyb3IoYEludGVybmFsIFpvZE9iamVjdCBlcnJvcjogaW52YWxpZCB1bmtub3duS2V5cyB2YWx1ZS5gKTtcbiAgICAgIH1cbiAgICB9IGVsc2Uge1xuICAgICAgY29uc3QgY2F0Y2hhbGwgPSB0aGlzLl9kZWYuY2F0Y2hhbGw7XG4gICAgICBmb3IgKGNvbnN0IGtleSBvZiBleHRyYUtleXMpIHtcbiAgICAgICAgY29uc3QgdmFsdWUgPSBjdHguZGF0YVtrZXldO1xuICAgICAgICBwYWlycy5wdXNoKHtcbiAgICAgICAgICBrZXk6IHtzdGF0dXM6IFwidmFsaWRcIiwgdmFsdWU6IGtleX0sXG4gICAgICAgICAgdmFsdWU6IGNhdGNoYWxsLl9wYXJzZShuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgdmFsdWUsIGN0eC5wYXRoLCBrZXkpKSxcbiAgICAgICAgICBhbHdheXNTZXQ6IGtleSBpbiBjdHguZGF0YVxuICAgICAgICB9KTtcbiAgICAgIH1cbiAgICB9XG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBQcm9taXNlLnJlc29sdmUoKS50aGVuKGFzeW5jICgpID0+IHtcbiAgICAgICAgY29uc3Qgc3luY1BhaXJzID0gW107XG4gICAgICAgIGZvciAoY29uc3QgcGFpciBvZiBwYWlycykge1xuICAgICAgICAgIGNvbnN0IGtleSA9IGF3YWl0IHBhaXIua2V5O1xuICAgICAgICAgIHN5bmNQYWlycy5wdXNoKHtcbiAgICAgICAgICAgIGtleSxcbiAgICAgICAgICAgIHZhbHVlOiBhd2FpdCBwYWlyLnZhbHVlLFxuICAgICAgICAgICAgYWx3YXlzU2V0OiBwYWlyLmFsd2F5c1NldFxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiBzeW5jUGFpcnM7XG4gICAgICB9KS50aGVuKChzeW5jUGFpcnMpID0+IHtcbiAgICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlT2JqZWN0U3luYyhzdGF0dXMsIHN5bmNQYWlycyk7XG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlT2JqZWN0U3luYyhzdGF0dXMsIHBhaXJzKTtcbiAgICB9XG4gIH1cbiAgZ2V0IHNoYXBlKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuc2hhcGUoKTtcbiAgfVxuICBzdHJpY3QobWVzc2FnZSkge1xuICAgIGVycm9yVXRpbC5lcnJUb09iajtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICB1bmtub3duS2V5czogXCJzdHJpY3RcIixcbiAgICAgIC4uLm1lc3NhZ2UgIT09IHZvaWQgMCA/IHtcbiAgICAgICAgZXJyb3JNYXA6IChpc3N1ZSwgY3R4KSA9PiB7XG4gICAgICAgICAgdmFyIF9hLCBfYiwgX2MsIF9kO1xuICAgICAgICAgIGNvbnN0IGRlZmF1bHRFcnJvciA9IChfYyA9IChfYiA9IChfYSA9IHRoaXMuX2RlZikuZXJyb3JNYXApID09PSBudWxsIHx8IF9iID09PSB2b2lkIDAgPyB2b2lkIDAgOiBfYi5jYWxsKF9hLCBpc3N1ZSwgY3R4KS5tZXNzYWdlKSAhPT0gbnVsbCAmJiBfYyAhPT0gdm9pZCAwID8gX2MgOiBjdHguZGVmYXVsdEVycm9yO1xuICAgICAgICAgIGlmIChpc3N1ZS5jb2RlID09PSBcInVucmVjb2duaXplZF9rZXlzXCIpXG4gICAgICAgICAgICByZXR1cm4ge1xuICAgICAgICAgICAgICBtZXNzYWdlOiAoX2QgPSBlcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSkubWVzc2FnZSkgIT09IG51bGwgJiYgX2QgIT09IHZvaWQgMCA/IF9kIDogZGVmYXVsdEVycm9yXG4gICAgICAgICAgICB9O1xuICAgICAgICAgIHJldHVybiB7XG4gICAgICAgICAgICBtZXNzYWdlOiBkZWZhdWx0RXJyb3JcbiAgICAgICAgICB9O1xuICAgICAgICB9XG4gICAgICB9IDoge31cbiAgICB9KTtcbiAgfVxuICBzdHJpcCgpIHtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICB1bmtub3duS2V5czogXCJzdHJpcFwiXG4gICAgfSk7XG4gIH1cbiAgcGFzc3Rocm91Z2goKSB7XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgdW5rbm93bktleXM6IFwicGFzc3Rocm91Z2hcIlxuICAgIH0pO1xuICB9XG4gIHNldEtleShrZXksIHNjaGVtYSkge1xuICAgIHJldHVybiB0aGlzLmF1Z21lbnQoe1trZXldOiBzY2hlbWF9KTtcbiAgfVxuICBtZXJnZShtZXJnaW5nKSB7XG4gICAgY29uc3QgbWVyZ2VkID0gbmV3IFpvZE9iamVjdCh7XG4gICAgICB1bmtub3duS2V5czogbWVyZ2luZy5fZGVmLnVua25vd25LZXlzLFxuICAgICAgY2F0Y2hhbGw6IG1lcmdpbmcuX2RlZi5jYXRjaGFsbCxcbiAgICAgIHNoYXBlOiAoKSA9PiBvYmplY3RVdGlsLm1lcmdlU2hhcGVzKHRoaXMuX2RlZi5zaGFwZSgpLCBtZXJnaW5nLl9kZWYuc2hhcGUoKSksXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE9iamVjdFxuICAgIH0pO1xuICAgIHJldHVybiBtZXJnZWQ7XG4gIH1cbiAgY2F0Y2hhbGwoaW5kZXgpIHtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBjYXRjaGFsbDogaW5kZXhcbiAgICB9KTtcbiAgfVxuICBwaWNrKG1hc2spIHtcbiAgICBjb25zdCBzaGFwZSA9IHt9O1xuICAgIHV0aWwub2JqZWN0S2V5cyhtYXNrKS5tYXAoKGtleSkgPT4ge1xuICAgICAgaWYgKHRoaXMuc2hhcGVba2V5XSlcbiAgICAgICAgc2hhcGVba2V5XSA9IHRoaXMuc2hhcGVba2V5XTtcbiAgICB9KTtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBzaGFwZTogKCkgPT4gc2hhcGVcbiAgICB9KTtcbiAgfVxuICBvbWl0KG1hc2spIHtcbiAgICBjb25zdCBzaGFwZSA9IHt9O1xuICAgIHV0aWwub2JqZWN0S2V5cyh0aGlzLnNoYXBlKS5tYXAoKGtleSkgPT4ge1xuICAgICAgaWYgKHV0aWwub2JqZWN0S2V5cyhtYXNrKS5pbmRleE9mKGtleSkgPT09IC0xKSB7XG4gICAgICAgIHNoYXBlW2tleV0gPSB0aGlzLnNoYXBlW2tleV07XG4gICAgICB9XG4gICAgfSk7XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgc2hhcGU6ICgpID0+IHNoYXBlXG4gICAgfSk7XG4gIH1cbiAgZGVlcFBhcnRpYWwoKSB7XG4gICAgcmV0dXJuIGRlZXBQYXJ0aWFsaWZ5KHRoaXMpO1xuICB9XG4gIHBhcnRpYWwobWFzaykge1xuICAgIGNvbnN0IG5ld1NoYXBlID0ge307XG4gICAgaWYgKG1hc2spIHtcbiAgICAgIHV0aWwub2JqZWN0S2V5cyh0aGlzLnNoYXBlKS5tYXAoKGtleSkgPT4ge1xuICAgICAgICBpZiAodXRpbC5vYmplY3RLZXlzKG1hc2spLmluZGV4T2Yoa2V5KSA9PT0gLTEpIHtcbiAgICAgICAgICBuZXdTaGFwZVtrZXldID0gdGhpcy5zaGFwZVtrZXldO1xuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIG5ld1NoYXBlW2tleV0gPSB0aGlzLnNoYXBlW2tleV0ub3B0aW9uYWwoKTtcbiAgICAgICAgfVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgICAgc2hhcGU6ICgpID0+IG5ld1NoYXBlXG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgZm9yIChjb25zdCBrZXkgaW4gdGhpcy5zaGFwZSkge1xuICAgICAgICBjb25zdCBmaWVsZFNjaGVtYSA9IHRoaXMuc2hhcGVba2V5XTtcbiAgICAgICAgbmV3U2hhcGVba2V5XSA9IGZpZWxkU2NoZW1hLm9wdGlvbmFsKCk7XG4gICAgICB9XG4gICAgfVxuICAgIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIHNoYXBlOiAoKSA9PiBuZXdTaGFwZVxuICAgIH0pO1xuICB9XG4gIHJlcXVpcmVkKCkge1xuICAgIGNvbnN0IG5ld1NoYXBlID0ge307XG4gICAgZm9yIChjb25zdCBrZXkgaW4gdGhpcy5zaGFwZSkge1xuICAgICAgY29uc3QgZmllbGRTY2hlbWEgPSB0aGlzLnNoYXBlW2tleV07XG4gICAgICBsZXQgbmV3RmllbGQgPSBmaWVsZFNjaGVtYTtcbiAgICAgIHdoaWxlIChuZXdGaWVsZCBpbnN0YW5jZW9mIFpvZE9wdGlvbmFsKSB7XG4gICAgICAgIG5ld0ZpZWxkID0gbmV3RmllbGQuX2RlZi5pbm5lclR5cGU7XG4gICAgICB9XG4gICAgICBuZXdTaGFwZVtrZXldID0gbmV3RmllbGQ7XG4gICAgfVxuICAgIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIHNoYXBlOiAoKSA9PiBuZXdTaGFwZVxuICAgIH0pO1xuICB9XG4gIGtleW9mKCkge1xuICAgIHJldHVybiBjcmVhdGVab2RFbnVtKHV0aWwub2JqZWN0S2V5cyh0aGlzLnNoYXBlKSk7XG4gIH1cbn07XG5ab2RPYmplY3QuY3JlYXRlID0gKHNoYXBlLCBwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgIHNoYXBlOiAoKSA9PiBzaGFwZSxcbiAgICB1bmtub3duS2V5czogXCJzdHJpcFwiLFxuICAgIGNhdGNoYWxsOiBab2ROZXZlci5jcmVhdGUoKSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE9iamVjdCxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xuWm9kT2JqZWN0LnN0cmljdENyZWF0ZSA9IChzaGFwZSwgcGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICBzaGFwZTogKCkgPT4gc2hhcGUsXG4gICAgdW5rbm93bktleXM6IFwic3RyaWN0XCIsXG4gICAgY2F0Y2hhbGw6IFpvZE5ldmVyLmNyZWF0ZSgpLFxuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kT2JqZWN0LFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG5ab2RPYmplY3QubGF6eWNyZWF0ZSA9IChzaGFwZSwgcGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICBzaGFwZSxcbiAgICB1bmtub3duS2V5czogXCJzdHJpcFwiLFxuICAgIGNhdGNoYWxsOiBab2ROZXZlci5jcmVhdGUoKSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE9iamVjdCxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZFVuaW9uID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3Qge2N0eH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGNvbnN0IG9wdGlvbnMgPSB0aGlzLl9kZWYub3B0aW9ucztcbiAgICBmdW5jdGlvbiBoYW5kbGVSZXN1bHRzKHJlc3VsdHMpIHtcbiAgICAgIGZvciAoY29uc3QgcmVzdWx0IG9mIHJlc3VsdHMpIHtcbiAgICAgICAgaWYgKHJlc3VsdC5yZXN1bHQuc3RhdHVzID09PSBcInZhbGlkXCIpIHtcbiAgICAgICAgICByZXR1cm4gcmVzdWx0LnJlc3VsdDtcbiAgICAgICAgfVxuICAgICAgfVxuICAgICAgZm9yIChjb25zdCByZXN1bHQgb2YgcmVzdWx0cykge1xuICAgICAgICBpZiAocmVzdWx0LnJlc3VsdC5zdGF0dXMgPT09IFwiZGlydHlcIikge1xuICAgICAgICAgIGN0eC5jb21tb24uaXNzdWVzLnB1c2goLi4ucmVzdWx0LmN0eC5jb21tb24uaXNzdWVzKTtcbiAgICAgICAgICByZXR1cm4gcmVzdWx0LnJlc3VsdDtcbiAgICAgICAgfVxuICAgICAgfVxuICAgICAgY29uc3QgdW5pb25FcnJvcnMgPSByZXN1bHRzLm1hcCgocmVzdWx0KSA9PiBuZXcgWm9kRXJyb3IocmVzdWx0LmN0eC5jb21tb24uaXNzdWVzKSk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdW5pb24sXG4gICAgICAgIHVuaW9uRXJyb3JzXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgcmV0dXJuIFByb21pc2UuYWxsKG9wdGlvbnMubWFwKGFzeW5jIChvcHRpb24pID0+IHtcbiAgICAgICAgY29uc3QgY2hpbGRDdHggPSB7XG4gICAgICAgICAgLi4uY3R4LFxuICAgICAgICAgIGNvbW1vbjoge1xuICAgICAgICAgICAgLi4uY3R4LmNvbW1vbixcbiAgICAgICAgICAgIGlzc3VlczogW11cbiAgICAgICAgICB9LFxuICAgICAgICAgIHBhcmVudDogbnVsbFxuICAgICAgICB9O1xuICAgICAgICByZXR1cm4ge1xuICAgICAgICAgIHJlc3VsdDogYXdhaXQgb3B0aW9uLl9wYXJzZUFzeW5jKHtcbiAgICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgICBwYXJlbnQ6IGNoaWxkQ3R4XG4gICAgICAgICAgfSksXG4gICAgICAgICAgY3R4OiBjaGlsZEN0eFxuICAgICAgICB9O1xuICAgICAgfSkpLnRoZW4oaGFuZGxlUmVzdWx0cyk7XG4gICAgfSBlbHNlIHtcbiAgICAgIGxldCBkaXJ0eSA9IHZvaWQgMDtcbiAgICAgIGNvbnN0IGlzc3VlcyA9IFtdO1xuICAgICAgZm9yIChjb25zdCBvcHRpb24gb2Ygb3B0aW9ucykge1xuICAgICAgICBjb25zdCBjaGlsZEN0eCA9IHtcbiAgICAgICAgICAuLi5jdHgsXG4gICAgICAgICAgY29tbW9uOiB7XG4gICAgICAgICAgICAuLi5jdHguY29tbW9uLFxuICAgICAgICAgICAgaXNzdWVzOiBbXVxuICAgICAgICAgIH0sXG4gICAgICAgICAgcGFyZW50OiBudWxsXG4gICAgICAgIH07XG4gICAgICAgIGNvbnN0IHJlc3VsdCA9IG9wdGlvbi5fcGFyc2VTeW5jKHtcbiAgICAgICAgICBkYXRhOiBjdHguZGF0YSxcbiAgICAgICAgICBwYXRoOiBjdHgucGF0aCxcbiAgICAgICAgICBwYXJlbnQ6IGNoaWxkQ3R4XG4gICAgICAgIH0pO1xuICAgICAgICBpZiAocmVzdWx0LnN0YXR1cyA9PT0gXCJ2YWxpZFwiKSB7XG4gICAgICAgICAgcmV0dXJuIHJlc3VsdDtcbiAgICAgICAgfSBlbHNlIGlmIChyZXN1bHQuc3RhdHVzID09PSBcImRpcnR5XCIgJiYgIWRpcnR5KSB7XG4gICAgICAgICAgZGlydHkgPSB7cmVzdWx0LCBjdHg6IGNoaWxkQ3R4fTtcbiAgICAgICAgfVxuICAgICAgICBpZiAoY2hpbGRDdHguY29tbW9uLmlzc3Vlcy5sZW5ndGgpIHtcbiAgICAgICAgICBpc3N1ZXMucHVzaChjaGlsZEN0eC5jb21tb24uaXNzdWVzKTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgICAgaWYgKGRpcnR5KSB7XG4gICAgICAgIGN0eC5jb21tb24uaXNzdWVzLnB1c2goLi4uZGlydHkuY3R4LmNvbW1vbi5pc3N1ZXMpO1xuICAgICAgICByZXR1cm4gZGlydHkucmVzdWx0O1xuICAgICAgfVxuICAgICAgY29uc3QgdW5pb25FcnJvcnMgPSBpc3N1ZXMubWFwKChpc3N1ZXMyKSA9PiBuZXcgWm9kRXJyb3IoaXNzdWVzMikpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3VuaW9uLFxuICAgICAgICB1bmlvbkVycm9yc1xuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gIH1cbiAgZ2V0IG9wdGlvbnMoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5vcHRpb25zO1xuICB9XG59O1xuWm9kVW5pb24uY3JlYXRlID0gKHR5cGVzLCBwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RVbmlvbih7XG4gICAgb3B0aW9uczogdHlwZXMsXG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RVbmlvbixcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZERpc2NyaW1pbmF0ZWRVbmlvbiA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHtjdHh9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUub2JqZWN0KSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUub2JqZWN0LFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGNvbnN0IGRpc2NyaW1pbmF0b3IgPSB0aGlzLmRpc2NyaW1pbmF0b3I7XG4gICAgY29uc3QgZGlzY3JpbWluYXRvclZhbHVlID0gY3R4LmRhdGFbZGlzY3JpbWluYXRvcl07XG4gICAgY29uc3Qgb3B0aW9uID0gdGhpcy5vcHRpb25zLmdldChkaXNjcmltaW5hdG9yVmFsdWUpO1xuICAgIGlmICghb3B0aW9uKSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdW5pb25fZGlzY3JpbWluYXRvcixcbiAgICAgICAgb3B0aW9uczogdGhpcy52YWxpZERpc2NyaW1pbmF0b3JWYWx1ZXMsXG4gICAgICAgIHBhdGg6IFtkaXNjcmltaW5hdG9yXVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBvcHRpb24uX3BhcnNlQXN5bmMoe1xuICAgICAgICBkYXRhOiBjdHguZGF0YSxcbiAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgIHBhcmVudDogY3R4XG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIG9wdGlvbi5fcGFyc2VTeW5jKHtcbiAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBwYXJlbnQ6IGN0eFxuICAgICAgfSk7XG4gICAgfVxuICB9XG4gIGdldCBkaXNjcmltaW5hdG9yKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuZGlzY3JpbWluYXRvcjtcbiAgfVxuICBnZXQgdmFsaWREaXNjcmltaW5hdG9yVmFsdWVzKCkge1xuICAgIHJldHVybiBBcnJheS5mcm9tKHRoaXMub3B0aW9ucy5rZXlzKCkpO1xuICB9XG4gIGdldCBvcHRpb25zKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYub3B0aW9ucztcbiAgfVxuICBzdGF0aWMgY3JlYXRlKGRpc2NyaW1pbmF0b3IsIHR5cGVzLCBwYXJhbXMpIHtcbiAgICBjb25zdCBvcHRpb25zID0gLyogQF9fUFVSRV9fICovIG5ldyBNYXAoKTtcbiAgICB0cnkge1xuICAgICAgdHlwZXMuZm9yRWFjaCgodHlwZSkgPT4ge1xuICAgICAgICBjb25zdCBkaXNjcmltaW5hdG9yVmFsdWUgPSB0eXBlLnNoYXBlW2Rpc2NyaW1pbmF0b3JdLnZhbHVlO1xuICAgICAgICBvcHRpb25zLnNldChkaXNjcmltaW5hdG9yVmFsdWUsIHR5cGUpO1xuICAgICAgfSk7XG4gICAgfSBjYXRjaCAoZSkge1xuICAgICAgdGhyb3cgbmV3IEVycm9yKFwiVGhlIGRpc2NyaW1pbmF0b3IgdmFsdWUgY291bGQgbm90IGJlIGV4dHJhY3RlZCBmcm9tIGFsbCB0aGUgcHJvdmlkZWQgc2NoZW1hc1wiKTtcbiAgICB9XG4gICAgaWYgKG9wdGlvbnMuc2l6ZSAhPT0gdHlwZXMubGVuZ3RoKSB7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoXCJTb21lIG9mIHRoZSBkaXNjcmltaW5hdG9yIHZhbHVlcyBhcmUgbm90IHVuaXF1ZVwiKTtcbiAgICB9XG4gICAgcmV0dXJuIG5ldyBab2REaXNjcmltaW5hdGVkVW5pb24oe1xuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2REaXNjcmltaW5hdGVkVW5pb24sXG4gICAgICBkaXNjcmltaW5hdG9yLFxuICAgICAgb3B0aW9ucyxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICAgIH0pO1xuICB9XG59O1xuZnVuY3Rpb24gbWVyZ2VWYWx1ZXMoYSwgYikge1xuICBjb25zdCBhVHlwZSA9IGdldFBhcnNlZFR5cGUoYSk7XG4gIGNvbnN0IGJUeXBlID0gZ2V0UGFyc2VkVHlwZShiKTtcbiAgaWYgKGEgPT09IGIpIHtcbiAgICByZXR1cm4ge3ZhbGlkOiB0cnVlLCBkYXRhOiBhfTtcbiAgfSBlbHNlIGlmIChhVHlwZSA9PT0gWm9kUGFyc2VkVHlwZS5vYmplY3QgJiYgYlR5cGUgPT09IFpvZFBhcnNlZFR5cGUub2JqZWN0KSB7XG4gICAgY29uc3QgYktleXMgPSB1dGlsLm9iamVjdEtleXMoYik7XG4gICAgY29uc3Qgc2hhcmVkS2V5cyA9IHV0aWwub2JqZWN0S2V5cyhhKS5maWx0ZXIoKGtleSkgPT4gYktleXMuaW5kZXhPZihrZXkpICE9PSAtMSk7XG4gICAgY29uc3QgbmV3T2JqID0gey4uLmEsIC4uLmJ9O1xuICAgIGZvciAoY29uc3Qga2V5IG9mIHNoYXJlZEtleXMpIHtcbiAgICAgIGNvbnN0IHNoYXJlZFZhbHVlID0gbWVyZ2VWYWx1ZXMoYVtrZXldLCBiW2tleV0pO1xuICAgICAgaWYgKCFzaGFyZWRWYWx1ZS52YWxpZCkge1xuICAgICAgICByZXR1cm4ge3ZhbGlkOiBmYWxzZX07XG4gICAgICB9XG4gICAgICBuZXdPYmpba2V5XSA9IHNoYXJlZFZhbHVlLmRhdGE7XG4gICAgfVxuICAgIHJldHVybiB7dmFsaWQ6IHRydWUsIGRhdGE6IG5ld09ian07XG4gIH0gZWxzZSBpZiAoYVR5cGUgPT09IFpvZFBhcnNlZFR5cGUuYXJyYXkgJiYgYlR5cGUgPT09IFpvZFBhcnNlZFR5cGUuYXJyYXkpIHtcbiAgICBpZiAoYS5sZW5ndGggIT09IGIubGVuZ3RoKSB7XG4gICAgICByZXR1cm4ge3ZhbGlkOiBmYWxzZX07XG4gICAgfVxuICAgIGNvbnN0IG5ld0FycmF5ID0gW107XG4gICAgZm9yIChsZXQgaW5kZXggPSAwOyBpbmRleCA8IGEubGVuZ3RoOyBpbmRleCsrKSB7XG4gICAgICBjb25zdCBpdGVtQSA9IGFbaW5kZXhdO1xuICAgICAgY29uc3QgaXRlbUIgPSBiW2luZGV4XTtcbiAgICAgIGNvbnN0IHNoYXJlZFZhbHVlID0gbWVyZ2VWYWx1ZXMoaXRlbUEsIGl0ZW1CKTtcbiAgICAgIGlmICghc2hhcmVkVmFsdWUudmFsaWQpIHtcbiAgICAgICAgcmV0dXJuIHt2YWxpZDogZmFsc2V9O1xuICAgICAgfVxuICAgICAgbmV3QXJyYXkucHVzaChzaGFyZWRWYWx1ZS5kYXRhKTtcbiAgICB9XG4gICAgcmV0dXJuIHt2YWxpZDogdHJ1ZSwgZGF0YTogbmV3QXJyYXl9O1xuICB9IGVsc2UgaWYgKGFUeXBlID09PSBab2RQYXJzZWRUeXBlLmRhdGUgJiYgYlR5cGUgPT09IFpvZFBhcnNlZFR5cGUuZGF0ZSAmJiArYSA9PT0gK2IpIHtcbiAgICByZXR1cm4ge3ZhbGlkOiB0cnVlLCBkYXRhOiBhfTtcbiAgfSBlbHNlIHtcbiAgICByZXR1cm4ge3ZhbGlkOiBmYWxzZX07XG4gIH1cbn1cbnZhciBab2RJbnRlcnNlY3Rpb24gPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7c3RhdHVzLCBjdHh9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBjb25zdCBoYW5kbGVQYXJzZWQgPSAocGFyc2VkTGVmdCwgcGFyc2VkUmlnaHQpID0+IHtcbiAgICAgIGlmIChpc0Fib3J0ZWQocGFyc2VkTGVmdCkgfHwgaXNBYm9ydGVkKHBhcnNlZFJpZ2h0KSkge1xuICAgICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICAgIH1cbiAgICAgIGNvbnN0IG1lcmdlZCA9IG1lcmdlVmFsdWVzKHBhcnNlZExlZnQudmFsdWUsIHBhcnNlZFJpZ2h0LnZhbHVlKTtcbiAgICAgIGlmICghbWVyZ2VkLnZhbGlkKSB7XG4gICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX2ludGVyc2VjdGlvbl90eXBlc1xuICAgICAgICB9KTtcbiAgICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgICB9XG4gICAgICBpZiAoaXNEaXJ0eShwYXJzZWRMZWZ0KSB8fCBpc0RpcnR5KHBhcnNlZFJpZ2h0KSkge1xuICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgIH1cbiAgICAgIHJldHVybiB7c3RhdHVzOiBzdGF0dXMudmFsdWUsIHZhbHVlOiBtZXJnZWQuZGF0YX07XG4gICAgfTtcbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgcmV0dXJuIFByb21pc2UuYWxsKFtcbiAgICAgICAgdGhpcy5fZGVmLmxlZnQuX3BhcnNlQXN5bmMoe1xuICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgIHBhcmVudDogY3R4XG4gICAgICAgIH0pLFxuICAgICAgICB0aGlzLl9kZWYucmlnaHQuX3BhcnNlQXN5bmMoe1xuICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgIHBhcmVudDogY3R4XG4gICAgICAgIH0pXG4gICAgICBdKS50aGVuKChbbGVmdCwgcmlnaHRdKSA9PiBoYW5kbGVQYXJzZWQobGVmdCwgcmlnaHQpKTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIGhhbmRsZVBhcnNlZCh0aGlzLl9kZWYubGVmdC5fcGFyc2VTeW5jKHtcbiAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBwYXJlbnQ6IGN0eFxuICAgICAgfSksIHRoaXMuX2RlZi5yaWdodC5fcGFyc2VTeW5jKHtcbiAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBwYXJlbnQ6IGN0eFxuICAgICAgfSkpO1xuICAgIH1cbiAgfVxufTtcblpvZEludGVyc2VjdGlvbi5jcmVhdGUgPSAobGVmdCwgcmlnaHQsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZEludGVyc2VjdGlvbih7XG4gICAgbGVmdCxcbiAgICByaWdodCxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEludGVyc2VjdGlvbixcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZFR1cGxlID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3Qge3N0YXR1cywgY3R4fSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLmFycmF5KSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUuYXJyYXksXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgaWYgKGN0eC5kYXRhLmxlbmd0aCA8IHRoaXMuX2RlZi5pdGVtcy5sZW5ndGgpIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX3NtYWxsLFxuICAgICAgICBtaW5pbXVtOiB0aGlzLl9kZWYuaXRlbXMubGVuZ3RoLFxuICAgICAgICBpbmNsdXNpdmU6IHRydWUsXG4gICAgICAgIHR5cGU6IFwiYXJyYXlcIlxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgY29uc3QgcmVzdCA9IHRoaXMuX2RlZi5yZXN0O1xuICAgIGlmICghcmVzdCAmJiBjdHguZGF0YS5sZW5ndGggPiB0aGlzLl9kZWYuaXRlbXMubGVuZ3RoKSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19iaWcsXG4gICAgICAgIG1heGltdW06IHRoaXMuX2RlZi5pdGVtcy5sZW5ndGgsXG4gICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgdHlwZTogXCJhcnJheVwiXG4gICAgICB9KTtcbiAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgIH1cbiAgICBjb25zdCBpdGVtcyA9IGN0eC5kYXRhLm1hcCgoaXRlbSwgaXRlbUluZGV4KSA9PiB7XG4gICAgICBjb25zdCBzY2hlbWEgPSB0aGlzLl9kZWYuaXRlbXNbaXRlbUluZGV4XSB8fCB0aGlzLl9kZWYucmVzdDtcbiAgICAgIGlmICghc2NoZW1hKVxuICAgICAgICByZXR1cm4gbnVsbDtcbiAgICAgIHJldHVybiBzY2hlbWEuX3BhcnNlKG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCBpdGVtLCBjdHgucGF0aCwgaXRlbUluZGV4KSk7XG4gICAgfSkuZmlsdGVyKCh4KSA9PiAhIXgpO1xuICAgIGlmIChjdHguY29tbW9uLmFzeW5jKSB7XG4gICAgICByZXR1cm4gUHJvbWlzZS5hbGwoaXRlbXMpLnRoZW4oKHJlc3VsdHMpID0+IHtcbiAgICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlQXJyYXkoc3RhdHVzLCByZXN1bHRzKTtcbiAgICAgIH0pO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXR1cm4gUGFyc2VTdGF0dXMubWVyZ2VBcnJheShzdGF0dXMsIGl0ZW1zKTtcbiAgICB9XG4gIH1cbiAgZ2V0IGl0ZW1zKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuaXRlbXM7XG4gIH1cbiAgcmVzdChyZXN0KSB7XG4gICAgcmV0dXJuIG5ldyBab2RUdXBsZSh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICByZXN0XG4gICAgfSk7XG4gIH1cbn07XG5ab2RUdXBsZS5jcmVhdGUgPSAoc2NoZW1hcywgcGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kVHVwbGUoe1xuICAgIGl0ZW1zOiBzY2hlbWFzLFxuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kVHVwbGUsXG4gICAgcmVzdDogbnVsbCxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZFJlY29yZCA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIGdldCBrZXlTY2hlbWEoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5rZXlUeXBlO1xuICB9XG4gIGdldCB2YWx1ZVNjaGVtYSgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnZhbHVlVHlwZTtcbiAgfVxuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7c3RhdHVzLCBjdHh9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUub2JqZWN0KSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUub2JqZWN0LFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGNvbnN0IHBhaXJzID0gW107XG4gICAgY29uc3Qga2V5VHlwZSA9IHRoaXMuX2RlZi5rZXlUeXBlO1xuICAgIGNvbnN0IHZhbHVlVHlwZSA9IHRoaXMuX2RlZi52YWx1ZVR5cGU7XG4gICAgZm9yIChjb25zdCBrZXkgaW4gY3R4LmRhdGEpIHtcbiAgICAgIHBhaXJzLnB1c2goe1xuICAgICAgICBrZXk6IGtleVR5cGUuX3BhcnNlKG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCBrZXksIGN0eC5wYXRoLCBrZXkpKSxcbiAgICAgICAgdmFsdWU6IHZhbHVlVHlwZS5fcGFyc2UobmV3IFBhcnNlSW5wdXRMYXp5UGF0aChjdHgsIGN0eC5kYXRhW2tleV0sIGN0eC5wYXRoLCBrZXkpKVxuICAgICAgfSk7XG4gICAgfVxuICAgIGlmIChjdHguY29tbW9uLmFzeW5jKSB7XG4gICAgICByZXR1cm4gUGFyc2VTdGF0dXMubWVyZ2VPYmplY3RBc3luYyhzdGF0dXMsIHBhaXJzKTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlT2JqZWN0U3luYyhzdGF0dXMsIHBhaXJzKTtcbiAgICB9XG4gIH1cbiAgZ2V0IGVsZW1lbnQoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi52YWx1ZVR5cGU7XG4gIH1cbiAgc3RhdGljIGNyZWF0ZShmaXJzdCwgc2Vjb25kLCB0aGlyZCkge1xuICAgIGlmIChzZWNvbmQgaW5zdGFuY2VvZiBab2RUeXBlKSB7XG4gICAgICByZXR1cm4gbmV3IFpvZFJlY29yZCh7XG4gICAgICAgIGtleVR5cGU6IGZpcnN0LFxuICAgICAgICB2YWx1ZVR5cGU6IHNlY29uZCxcbiAgICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RSZWNvcmQsXG4gICAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXModGhpcmQpXG4gICAgICB9KTtcbiAgICB9XG4gICAgcmV0dXJuIG5ldyBab2RSZWNvcmQoe1xuICAgICAga2V5VHlwZTogWm9kU3RyaW5nLmNyZWF0ZSgpLFxuICAgICAgdmFsdWVUeXBlOiBmaXJzdCxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kUmVjb3JkLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhzZWNvbmQpXG4gICAgfSk7XG4gIH1cbn07XG52YXIgWm9kTWFwID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3Qge3N0YXR1cywgY3R4fSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLm1hcCkge1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm1hcCxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICBjb25zdCBrZXlUeXBlID0gdGhpcy5fZGVmLmtleVR5cGU7XG4gICAgY29uc3QgdmFsdWVUeXBlID0gdGhpcy5fZGVmLnZhbHVlVHlwZTtcbiAgICBjb25zdCBwYWlycyA9IFsuLi5jdHguZGF0YS5lbnRyaWVzKCldLm1hcCgoW2tleSwgdmFsdWVdLCBpbmRleCkgPT4ge1xuICAgICAgcmV0dXJuIHtcbiAgICAgICAga2V5OiBrZXlUeXBlLl9wYXJzZShuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwga2V5LCBjdHgucGF0aCwgW2luZGV4LCBcImtleVwiXSkpLFxuICAgICAgICB2YWx1ZTogdmFsdWVUeXBlLl9wYXJzZShuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgdmFsdWUsIGN0eC5wYXRoLCBbaW5kZXgsIFwidmFsdWVcIl0pKVxuICAgICAgfTtcbiAgICB9KTtcbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgY29uc3QgZmluYWxNYXAgPSAvKiBAX19QVVJFX18gKi8gbmV3IE1hcCgpO1xuICAgICAgcmV0dXJuIFByb21pc2UucmVzb2x2ZSgpLnRoZW4oYXN5bmMgKCkgPT4ge1xuICAgICAgICBmb3IgKGNvbnN0IHBhaXIgb2YgcGFpcnMpIHtcbiAgICAgICAgICBjb25zdCBrZXkgPSBhd2FpdCBwYWlyLmtleTtcbiAgICAgICAgICBjb25zdCB2YWx1ZSA9IGF3YWl0IHBhaXIudmFsdWU7XG4gICAgICAgICAgaWYgKGtleS5zdGF0dXMgPT09IFwiYWJvcnRlZFwiIHx8IHZhbHVlLnN0YXR1cyA9PT0gXCJhYm9ydGVkXCIpIHtcbiAgICAgICAgICAgIHJldHVybiBJTlZBTElEO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoa2V5LnN0YXR1cyA9PT0gXCJkaXJ0eVwiIHx8IHZhbHVlLnN0YXR1cyA9PT0gXCJkaXJ0eVwiKSB7XG4gICAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgICB9XG4gICAgICAgICAgZmluYWxNYXAuc2V0KGtleS52YWx1ZSwgdmFsdWUudmFsdWUpO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiB7c3RhdHVzOiBzdGF0dXMudmFsdWUsIHZhbHVlOiBmaW5hbE1hcH07XG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgY29uc3QgZmluYWxNYXAgPSAvKiBAX19QVVJFX18gKi8gbmV3IE1hcCgpO1xuICAgICAgZm9yIChjb25zdCBwYWlyIG9mIHBhaXJzKSB7XG4gICAgICAgIGNvbnN0IGtleSA9IHBhaXIua2V5O1xuICAgICAgICBjb25zdCB2YWx1ZSA9IHBhaXIudmFsdWU7XG4gICAgICAgIGlmIChrZXkuc3RhdHVzID09PSBcImFib3J0ZWRcIiB8fCB2YWx1ZS5zdGF0dXMgPT09IFwiYWJvcnRlZFwiKSB7XG4gICAgICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgICAgIH1cbiAgICAgICAgaWYgKGtleS5zdGF0dXMgPT09IFwiZGlydHlcIiB8fCB2YWx1ZS5zdGF0dXMgPT09IFwiZGlydHlcIikge1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICAgIGZpbmFsTWFwLnNldChrZXkudmFsdWUsIHZhbHVlLnZhbHVlKTtcbiAgICAgIH1cbiAgICAgIHJldHVybiB7c3RhdHVzOiBzdGF0dXMudmFsdWUsIHZhbHVlOiBmaW5hbE1hcH07XG4gICAgfVxuICB9XG59O1xuWm9kTWFwLmNyZWF0ZSA9IChrZXlUeXBlLCB2YWx1ZVR5cGUsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZE1hcCh7XG4gICAgdmFsdWVUeXBlLFxuICAgIGtleVR5cGUsXG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RNYXAsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2RTZXQgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7c3RhdHVzLCBjdHh9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuc2V0KSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUuc2V0LFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGNvbnN0IGRlZiA9IHRoaXMuX2RlZjtcbiAgICBpZiAoZGVmLm1pblNpemUgIT09IG51bGwpIHtcbiAgICAgIGlmIChjdHguZGF0YS5zaXplIDwgZGVmLm1pblNpemUudmFsdWUpIHtcbiAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19zbWFsbCxcbiAgICAgICAgICBtaW5pbXVtOiBkZWYubWluU2l6ZS52YWx1ZSxcbiAgICAgICAgICB0eXBlOiBcInNldFwiLFxuICAgICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgICBtZXNzYWdlOiBkZWYubWluU2l6ZS5tZXNzYWdlXG4gICAgICAgIH0pO1xuICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgIH1cbiAgICB9XG4gICAgaWYgKGRlZi5tYXhTaXplICE9PSBudWxsKSB7XG4gICAgICBpZiAoY3R4LmRhdGEuc2l6ZSA+IGRlZi5tYXhTaXplLnZhbHVlKSB7XG4gICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS50b29fYmlnLFxuICAgICAgICAgIG1heGltdW06IGRlZi5tYXhTaXplLnZhbHVlLFxuICAgICAgICAgIHR5cGU6IFwic2V0XCIsXG4gICAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICAgIG1lc3NhZ2U6IGRlZi5tYXhTaXplLm1lc3NhZ2VcbiAgICAgICAgfSk7XG4gICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgfVxuICAgIH1cbiAgICBjb25zdCB2YWx1ZVR5cGUgPSB0aGlzLl9kZWYudmFsdWVUeXBlO1xuICAgIGZ1bmN0aW9uIGZpbmFsaXplU2V0KGVsZW1lbnRzMikge1xuICAgICAgY29uc3QgcGFyc2VkU2V0ID0gLyogQF9fUFVSRV9fICovIG5ldyBTZXQoKTtcbiAgICAgIGZvciAoY29uc3QgZWxlbWVudCBvZiBlbGVtZW50czIpIHtcbiAgICAgICAgaWYgKGVsZW1lbnQuc3RhdHVzID09PSBcImFib3J0ZWRcIilcbiAgICAgICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICAgICAgaWYgKGVsZW1lbnQuc3RhdHVzID09PSBcImRpcnR5XCIpXG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIHBhcnNlZFNldC5hZGQoZWxlbWVudC52YWx1ZSk7XG4gICAgICB9XG4gICAgICByZXR1cm4ge3N0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogcGFyc2VkU2V0fTtcbiAgICB9XG4gICAgY29uc3QgZWxlbWVudHMgPSBbLi4uY3R4LmRhdGEudmFsdWVzKCldLm1hcCgoaXRlbSwgaSkgPT4gdmFsdWVUeXBlLl9wYXJzZShuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgaXRlbSwgY3R4LnBhdGgsIGkpKSk7XG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBQcm9taXNlLmFsbChlbGVtZW50cykudGhlbigoZWxlbWVudHMyKSA9PiBmaW5hbGl6ZVNldChlbGVtZW50czIpKTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIGZpbmFsaXplU2V0KGVsZW1lbnRzKTtcbiAgICB9XG4gIH1cbiAgbWluKG1pblNpemUsIG1lc3NhZ2UpIHtcbiAgICByZXR1cm4gbmV3IFpvZFNldCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBtaW5TaXplOiB7dmFsdWU6IG1pblNpemUsIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKX1cbiAgICB9KTtcbiAgfVxuICBtYXgobWF4U2l6ZSwgbWVzc2FnZSkge1xuICAgIHJldHVybiBuZXcgWm9kU2V0KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIG1heFNpemU6IHt2YWx1ZTogbWF4U2l6ZSwgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpfVxuICAgIH0pO1xuICB9XG4gIHNpemUoc2l6ZSwgbWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLm1pbihzaXplLCBtZXNzYWdlKS5tYXgoc2l6ZSwgbWVzc2FnZSk7XG4gIH1cbiAgbm9uZW1wdHkobWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLm1pbigxLCBtZXNzYWdlKTtcbiAgfVxufTtcblpvZFNldC5jcmVhdGUgPSAodmFsdWVUeXBlLCBwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RTZXQoe1xuICAgIHZhbHVlVHlwZSxcbiAgICBtaW5TaXplOiBudWxsLFxuICAgIG1heFNpemU6IG51bGwsXG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RTZXQsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2RGdW5jdGlvbiA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIGNvbnN0cnVjdG9yKCkge1xuICAgIHN1cGVyKC4uLmFyZ3VtZW50cyk7XG4gICAgdGhpcy52YWxpZGF0ZSA9IHRoaXMuaW1wbGVtZW50O1xuICB9XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHtjdHh9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuZnVuY3Rpb24pIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5mdW5jdGlvbixcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICBmdW5jdGlvbiBtYWtlQXJnc0lzc3VlKGFyZ3MsIGVycm9yKSB7XG4gICAgICByZXR1cm4gbWFrZUlzc3VlKHtcbiAgICAgICAgZGF0YTogYXJncyxcbiAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgIGVycm9yTWFwczogW1xuICAgICAgICAgIGN0eC5jb21tb24uY29udGV4dHVhbEVycm9yTWFwLFxuICAgICAgICAgIGN0eC5zY2hlbWFFcnJvck1hcCxcbiAgICAgICAgICBnZXRFcnJvck1hcCgpLFxuICAgICAgICAgIGRlZmF1bHRFcnJvck1hcFxuICAgICAgICBdLmZpbHRlcigoeCkgPT4gISF4KSxcbiAgICAgICAgaXNzdWVEYXRhOiB7XG4gICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfYXJndW1lbnRzLFxuICAgICAgICAgIGFyZ3VtZW50c0Vycm9yOiBlcnJvclxuICAgICAgICB9XG4gICAgICB9KTtcbiAgICB9XG4gICAgZnVuY3Rpb24gbWFrZVJldHVybnNJc3N1ZShyZXR1cm5zLCBlcnJvcikge1xuICAgICAgcmV0dXJuIG1ha2VJc3N1ZSh7XG4gICAgICAgIGRhdGE6IHJldHVybnMsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBlcnJvck1hcHM6IFtcbiAgICAgICAgICBjdHguY29tbW9uLmNvbnRleHR1YWxFcnJvck1hcCxcbiAgICAgICAgICBjdHguc2NoZW1hRXJyb3JNYXAsXG4gICAgICAgICAgZ2V0RXJyb3JNYXAoKSxcbiAgICAgICAgICBkZWZhdWx0RXJyb3JNYXBcbiAgICAgICAgXS5maWx0ZXIoKHgpID0+ICEheCksXG4gICAgICAgIGlzc3VlRGF0YToge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3JldHVybl90eXBlLFxuICAgICAgICAgIHJldHVyblR5cGVFcnJvcjogZXJyb3JcbiAgICAgICAgfVxuICAgICAgfSk7XG4gICAgfVxuICAgIGNvbnN0IHBhcmFtcyA9IHtlcnJvck1hcDogY3R4LmNvbW1vbi5jb250ZXh0dWFsRXJyb3JNYXB9O1xuICAgIGNvbnN0IGZuID0gY3R4LmRhdGE7XG4gICAgaWYgKHRoaXMuX2RlZi5yZXR1cm5zIGluc3RhbmNlb2YgWm9kUHJvbWlzZSkge1xuICAgICAgcmV0dXJuIE9LKGFzeW5jICguLi5hcmdzKSA9PiB7XG4gICAgICAgIGNvbnN0IGVycm9yID0gbmV3IFpvZEVycm9yKFtdKTtcbiAgICAgICAgY29uc3QgcGFyc2VkQXJncyA9IGF3YWl0IHRoaXMuX2RlZi5hcmdzLnBhcnNlQXN5bmMoYXJncywgcGFyYW1zKS5jYXRjaCgoZSkgPT4ge1xuICAgICAgICAgIGVycm9yLmFkZElzc3VlKG1ha2VBcmdzSXNzdWUoYXJncywgZSkpO1xuICAgICAgICAgIHRocm93IGVycm9yO1xuICAgICAgICB9KTtcbiAgICAgICAgY29uc3QgcmVzdWx0ID0gYXdhaXQgZm4oLi4ucGFyc2VkQXJncyk7XG4gICAgICAgIGNvbnN0IHBhcnNlZFJldHVybnMgPSBhd2FpdCB0aGlzLl9kZWYucmV0dXJucy5fZGVmLnR5cGUucGFyc2VBc3luYyhyZXN1bHQsIHBhcmFtcykuY2F0Y2goKGUpID0+IHtcbiAgICAgICAgICBlcnJvci5hZGRJc3N1ZShtYWtlUmV0dXJuc0lzc3VlKHJlc3VsdCwgZSkpO1xuICAgICAgICAgIHRocm93IGVycm9yO1xuICAgICAgICB9KTtcbiAgICAgICAgcmV0dXJuIHBhcnNlZFJldHVybnM7XG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIE9LKCguLi5hcmdzKSA9PiB7XG4gICAgICAgIGNvbnN0IHBhcnNlZEFyZ3MgPSB0aGlzLl9kZWYuYXJncy5zYWZlUGFyc2UoYXJncywgcGFyYW1zKTtcbiAgICAgICAgaWYgKCFwYXJzZWRBcmdzLnN1Y2Nlc3MpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgWm9kRXJyb3IoW21ha2VBcmdzSXNzdWUoYXJncywgcGFyc2VkQXJncy5lcnJvcildKTtcbiAgICAgICAgfVxuICAgICAgICBjb25zdCByZXN1bHQgPSBmbiguLi5wYXJzZWRBcmdzLmRhdGEpO1xuICAgICAgICBjb25zdCBwYXJzZWRSZXR1cm5zID0gdGhpcy5fZGVmLnJldHVybnMuc2FmZVBhcnNlKHJlc3VsdCwgcGFyYW1zKTtcbiAgICAgICAgaWYgKCFwYXJzZWRSZXR1cm5zLnN1Y2Nlc3MpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgWm9kRXJyb3IoW21ha2VSZXR1cm5zSXNzdWUocmVzdWx0LCBwYXJzZWRSZXR1cm5zLmVycm9yKV0pO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiBwYXJzZWRSZXR1cm5zLmRhdGE7XG4gICAgICB9KTtcbiAgICB9XG4gIH1cbiAgcGFyYW1ldGVycygpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLmFyZ3M7XG4gIH1cbiAgcmV0dXJuVHlwZSgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnJldHVybnM7XG4gIH1cbiAgYXJncyguLi5pdGVtcykge1xuICAgIHJldHVybiBuZXcgWm9kRnVuY3Rpb24oe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgYXJnczogWm9kVHVwbGUuY3JlYXRlKGl0ZW1zKS5yZXN0KFpvZFVua25vd24uY3JlYXRlKCkpXG4gICAgfSk7XG4gIH1cbiAgcmV0dXJucyhyZXR1cm5UeXBlKSB7XG4gICAgcmV0dXJuIG5ldyBab2RGdW5jdGlvbih7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICByZXR1cm5zOiByZXR1cm5UeXBlXG4gICAgfSk7XG4gIH1cbiAgaW1wbGVtZW50KGZ1bmMpIHtcbiAgICBjb25zdCB2YWxpZGF0ZWRGdW5jID0gdGhpcy5wYXJzZShmdW5jKTtcbiAgICByZXR1cm4gdmFsaWRhdGVkRnVuYztcbiAgfVxuICBzdHJpY3RJbXBsZW1lbnQoZnVuYykge1xuICAgIGNvbnN0IHZhbGlkYXRlZEZ1bmMgPSB0aGlzLnBhcnNlKGZ1bmMpO1xuICAgIHJldHVybiB2YWxpZGF0ZWRGdW5jO1xuICB9XG59O1xuWm9kRnVuY3Rpb24uY3JlYXRlID0gKGFyZ3MsIHJldHVybnMsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZEZ1bmN0aW9uKHtcbiAgICBhcmdzOiBhcmdzID8gYXJncy5yZXN0KFpvZFVua25vd24uY3JlYXRlKCkpIDogWm9kVHVwbGUuY3JlYXRlKFtdKS5yZXN0KFpvZFVua25vd24uY3JlYXRlKCkpLFxuICAgIHJldHVybnM6IHJldHVybnMgfHwgWm9kVW5rbm93bi5jcmVhdGUoKSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEZ1bmN0aW9uLFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG52YXIgWm9kTGF6eSA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIGdldCBzY2hlbWEoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5nZXR0ZXIoKTtcbiAgfVxuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7Y3R4fSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgY29uc3QgbGF6eVNjaGVtYSA9IHRoaXMuX2RlZi5nZXR0ZXIoKTtcbiAgICByZXR1cm4gbGF6eVNjaGVtYS5fcGFyc2Uoe2RhdGE6IGN0eC5kYXRhLCBwYXRoOiBjdHgucGF0aCwgcGFyZW50OiBjdHh9KTtcbiAgfVxufTtcblpvZExhenkuY3JlYXRlID0gKGdldHRlciwgcGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kTGF6eSh7XG4gICAgZ2V0dGVyLFxuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTGF6eSxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZExpdGVyYWwgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBpZiAoaW5wdXQuZGF0YSAhPT0gdGhpcy5fZGVmLnZhbHVlKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfbGl0ZXJhbCxcbiAgICAgICAgZXhwZWN0ZWQ6IHRoaXMuX2RlZi52YWx1ZVxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgcmV0dXJuIHtzdGF0dXM6IFwidmFsaWRcIiwgdmFsdWU6IGlucHV0LmRhdGF9O1xuICB9XG4gIGdldCB2YWx1ZSgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnZhbHVlO1xuICB9XG59O1xuWm9kTGl0ZXJhbC5jcmVhdGUgPSAodmFsdWUsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZExpdGVyYWwoe1xuICAgIHZhbHVlLFxuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTGl0ZXJhbCxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xuZnVuY3Rpb24gY3JlYXRlWm9kRW51bSh2YWx1ZXMsIHBhcmFtcykge1xuICByZXR1cm4gbmV3IFpvZEVudW0oe1xuICAgIHZhbHVlcyxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEVudW0sXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufVxudmFyIFpvZEVudW0gPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBpZiAodHlwZW9mIGlucHV0LmRhdGEgIT09IFwic3RyaW5nXCIpIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGNvbnN0IGV4cGVjdGVkVmFsdWVzID0gdGhpcy5fZGVmLnZhbHVlcztcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBleHBlY3RlZDogdXRpbC5qb2luVmFsdWVzKGV4cGVjdGVkVmFsdWVzKSxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICBpZiAodGhpcy5fZGVmLnZhbHVlcy5pbmRleE9mKGlucHV0LmRhdGEpID09PSAtMSkge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgY29uc3QgZXhwZWN0ZWRWYWx1ZXMgPSB0aGlzLl9kZWYudmFsdWVzO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIHJlY2VpdmVkOiBjdHguZGF0YSxcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfZW51bV92YWx1ZSxcbiAgICAgICAgb3B0aW9uczogZXhwZWN0ZWRWYWx1ZXNcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxuICBnZXQgb3B0aW9ucygpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnZhbHVlcztcbiAgfVxuICBnZXQgZW51bSgpIHtcbiAgICBjb25zdCBlbnVtVmFsdWVzID0ge307XG4gICAgZm9yIChjb25zdCB2YWwgb2YgdGhpcy5fZGVmLnZhbHVlcykge1xuICAgICAgZW51bVZhbHVlc1t2YWxdID0gdmFsO1xuICAgIH1cbiAgICByZXR1cm4gZW51bVZhbHVlcztcbiAgfVxuICBnZXQgVmFsdWVzKCkge1xuICAgIGNvbnN0IGVudW1WYWx1ZXMgPSB7fTtcbiAgICBmb3IgKGNvbnN0IHZhbCBvZiB0aGlzLl9kZWYudmFsdWVzKSB7XG4gICAgICBlbnVtVmFsdWVzW3ZhbF0gPSB2YWw7XG4gICAgfVxuICAgIHJldHVybiBlbnVtVmFsdWVzO1xuICB9XG4gIGdldCBFbnVtKCkge1xuICAgIGNvbnN0IGVudW1WYWx1ZXMgPSB7fTtcbiAgICBmb3IgKGNvbnN0IHZhbCBvZiB0aGlzLl9kZWYudmFsdWVzKSB7XG4gICAgICBlbnVtVmFsdWVzW3ZhbF0gPSB2YWw7XG4gICAgfVxuICAgIHJldHVybiBlbnVtVmFsdWVzO1xuICB9XG59O1xuWm9kRW51bS5jcmVhdGUgPSBjcmVhdGVab2RFbnVtO1xudmFyIFpvZE5hdGl2ZUVudW0gPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCBuYXRpdmVFbnVtVmFsdWVzID0gdXRpbC5nZXRWYWxpZEVudW1WYWx1ZXModGhpcy5fZGVmLnZhbHVlcyk7XG4gICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgIGlmIChjdHgucGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5zdHJpbmcgJiYgY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUubnVtYmVyKSB7XG4gICAgICBjb25zdCBleHBlY3RlZFZhbHVlcyA9IHV0aWwub2JqZWN0VmFsdWVzKG5hdGl2ZUVudW1WYWx1ZXMpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGV4cGVjdGVkOiB1dGlsLmpvaW5WYWx1ZXMoZXhwZWN0ZWRWYWx1ZXMpLFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGlmIChuYXRpdmVFbnVtVmFsdWVzLmluZGV4T2YoaW5wdXQuZGF0YSkgPT09IC0xKSB7XG4gICAgICBjb25zdCBleHBlY3RlZFZhbHVlcyA9IHV0aWwub2JqZWN0VmFsdWVzKG5hdGl2ZUVudW1WYWx1ZXMpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIHJlY2VpdmVkOiBjdHguZGF0YSxcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfZW51bV92YWx1ZSxcbiAgICAgICAgb3B0aW9uczogZXhwZWN0ZWRWYWx1ZXNcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxuICBnZXQgZW51bSgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnZhbHVlcztcbiAgfVxufTtcblpvZE5hdGl2ZUVudW0uY3JlYXRlID0gKHZhbHVlcywgcGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kTmF0aXZlRW51bSh7XG4gICAgdmFsdWVzLFxuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTmF0aXZlRW51bSxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZFByb21pc2UgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7Y3R4fSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLnByb21pc2UgJiYgY3R4LmNvbW1vbi5hc3luYyA9PT0gZmFsc2UpIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5wcm9taXNlLFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGVcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGNvbnN0IHByb21pc2lmaWVkID0gY3R4LnBhcnNlZFR5cGUgPT09IFpvZFBhcnNlZFR5cGUucHJvbWlzZSA/IGN0eC5kYXRhIDogUHJvbWlzZS5yZXNvbHZlKGN0eC5kYXRhKTtcbiAgICByZXR1cm4gT0socHJvbWlzaWZpZWQudGhlbigoZGF0YSkgPT4ge1xuICAgICAgcmV0dXJuIHRoaXMuX2RlZi50eXBlLnBhcnNlQXN5bmMoZGF0YSwge1xuICAgICAgICBwYXRoOiBjdHgucGF0aCxcbiAgICAgICAgZXJyb3JNYXA6IGN0eC5jb21tb24uY29udGV4dHVhbEVycm9yTWFwXG4gICAgICB9KTtcbiAgICB9KSk7XG4gIH1cbn07XG5ab2RQcm9taXNlLmNyZWF0ZSA9IChzY2hlbWEsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZFByb21pc2Uoe1xuICAgIHR5cGU6IHNjaGVtYSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFByb21pc2UsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2RFZmZlY3RzID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgaW5uZXJUeXBlKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuc2NoZW1hO1xuICB9XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHtzdGF0dXMsIGN0eH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGNvbnN0IGVmZmVjdCA9IHRoaXMuX2RlZi5lZmZlY3QgfHwgbnVsbDtcbiAgICBpZiAoZWZmZWN0LnR5cGUgPT09IFwicHJlcHJvY2Vzc1wiKSB7XG4gICAgICBjb25zdCBwcm9jZXNzZWQgPSBlZmZlY3QudHJhbnNmb3JtKGN0eC5kYXRhKTtcbiAgICAgIGlmIChjdHguY29tbW9uLmFzeW5jKSB7XG4gICAgICAgIHJldHVybiBQcm9taXNlLnJlc29sdmUocHJvY2Vzc2VkKS50aGVuKChwcm9jZXNzZWQyKSA9PiB7XG4gICAgICAgICAgcmV0dXJuIHRoaXMuX2RlZi5zY2hlbWEuX3BhcnNlQXN5bmMoe1xuICAgICAgICAgICAgZGF0YTogcHJvY2Vzc2VkMixcbiAgICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgICAgcGFyZW50OiBjdHhcbiAgICAgICAgICB9KTtcbiAgICAgICAgfSk7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gdGhpcy5fZGVmLnNjaGVtYS5fcGFyc2VTeW5jKHtcbiAgICAgICAgICBkYXRhOiBwcm9jZXNzZWQsXG4gICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgcGFyZW50OiBjdHhcbiAgICAgICAgfSk7XG4gICAgICB9XG4gICAgfVxuICAgIGNvbnN0IGNoZWNrQ3R4ID0ge1xuICAgICAgYWRkSXNzdWU6IChhcmcpID0+IHtcbiAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCBhcmcpO1xuICAgICAgICBpZiAoYXJnLmZhdGFsKSB7XG4gICAgICAgICAgc3RhdHVzLmFib3J0KCk7XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0sXG4gICAgICBnZXQgcGF0aCgpIHtcbiAgICAgICAgcmV0dXJuIGN0eC5wYXRoO1xuICAgICAgfVxuICAgIH07XG4gICAgY2hlY2tDdHguYWRkSXNzdWUgPSBjaGVja0N0eC5hZGRJc3N1ZS5iaW5kKGNoZWNrQ3R4KTtcbiAgICBpZiAoZWZmZWN0LnR5cGUgPT09IFwicmVmaW5lbWVudFwiKSB7XG4gICAgICBjb25zdCBleGVjdXRlUmVmaW5lbWVudCA9IChhY2MpID0+IHtcbiAgICAgICAgY29uc3QgcmVzdWx0ID0gZWZmZWN0LnJlZmluZW1lbnQoYWNjLCBjaGVja0N0eCk7XG4gICAgICAgIGlmIChjdHguY29tbW9uLmFzeW5jKSB7XG4gICAgICAgICAgcmV0dXJuIFByb21pc2UucmVzb2x2ZShyZXN1bHQpO1xuICAgICAgICB9XG4gICAgICAgIGlmIChyZXN1bHQgaW5zdGFuY2VvZiBQcm9taXNlKSB7XG4gICAgICAgICAgdGhyb3cgbmV3IEVycm9yKFwiQXN5bmMgcmVmaW5lbWVudCBlbmNvdW50ZXJlZCBkdXJpbmcgc3luY2hyb25vdXMgcGFyc2Ugb3BlcmF0aW9uLiBVc2UgLnBhcnNlQXN5bmMgaW5zdGVhZC5cIik7XG4gICAgICAgIH1cbiAgICAgICAgcmV0dXJuIGFjYztcbiAgICAgIH07XG4gICAgICBpZiAoY3R4LmNvbW1vbi5hc3luYyA9PT0gZmFsc2UpIHtcbiAgICAgICAgY29uc3QgaW5uZXIgPSB0aGlzLl9kZWYuc2NoZW1hLl9wYXJzZVN5bmMoe1xuICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgIHBhcmVudDogY3R4XG4gICAgICAgIH0pO1xuICAgICAgICBpZiAoaW5uZXIuc3RhdHVzID09PSBcImFib3J0ZWRcIilcbiAgICAgICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICAgICAgaWYgKGlubmVyLnN0YXR1cyA9PT0gXCJkaXJ0eVwiKVxuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICBleGVjdXRlUmVmaW5lbWVudChpbm5lci52YWx1ZSk7XG4gICAgICAgIHJldHVybiB7c3RhdHVzOiBzdGF0dXMudmFsdWUsIHZhbHVlOiBpbm5lci52YWx1ZX07XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gdGhpcy5fZGVmLnNjaGVtYS5fcGFyc2VBc3luYyh7ZGF0YTogY3R4LmRhdGEsIHBhdGg6IGN0eC5wYXRoLCBwYXJlbnQ6IGN0eH0pLnRoZW4oKGlubmVyKSA9PiB7XG4gICAgICAgICAgaWYgKGlubmVyLnN0YXR1cyA9PT0gXCJhYm9ydGVkXCIpXG4gICAgICAgICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICAgICAgICBpZiAoaW5uZXIuc3RhdHVzID09PSBcImRpcnR5XCIpXG4gICAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgICByZXR1cm4gZXhlY3V0ZVJlZmluZW1lbnQoaW5uZXIudmFsdWUpLnRoZW4oKCkgPT4ge1xuICAgICAgICAgICAgcmV0dXJuIHtzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IGlubmVyLnZhbHVlfTtcbiAgICAgICAgICB9KTtcbiAgICAgICAgfSk7XG4gICAgICB9XG4gICAgfVxuICAgIGlmIChlZmZlY3QudHlwZSA9PT0gXCJ0cmFuc2Zvcm1cIikge1xuICAgICAgaWYgKGN0eC5jb21tb24uYXN5bmMgPT09IGZhbHNlKSB7XG4gICAgICAgIGNvbnN0IGJhc2UgPSB0aGlzLl9kZWYuc2NoZW1hLl9wYXJzZVN5bmMoe1xuICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgIHBhcmVudDogY3R4XG4gICAgICAgIH0pO1xuICAgICAgICBpZiAoIWlzVmFsaWQoYmFzZSkpXG4gICAgICAgICAgcmV0dXJuIGJhc2U7XG4gICAgICAgIGNvbnN0IHJlc3VsdCA9IGVmZmVjdC50cmFuc2Zvcm0oYmFzZS52YWx1ZSwgY2hlY2tDdHgpO1xuICAgICAgICBpZiAocmVzdWx0IGluc3RhbmNlb2YgUHJvbWlzZSkge1xuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihgQXN5bmNocm9ub3VzIHRyYW5zZm9ybSBlbmNvdW50ZXJlZCBkdXJpbmcgc3luY2hyb25vdXMgcGFyc2Ugb3BlcmF0aW9uLiBVc2UgLnBhcnNlQXN5bmMgaW5zdGVhZC5gKTtcbiAgICAgICAgfVxuICAgICAgICByZXR1cm4ge3N0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogcmVzdWx0fTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJldHVybiB0aGlzLl9kZWYuc2NoZW1hLl9wYXJzZUFzeW5jKHtkYXRhOiBjdHguZGF0YSwgcGF0aDogY3R4LnBhdGgsIHBhcmVudDogY3R4fSkudGhlbigoYmFzZSkgPT4ge1xuICAgICAgICAgIGlmICghaXNWYWxpZChiYXNlKSlcbiAgICAgICAgICAgIHJldHVybiBiYXNlO1xuICAgICAgICAgIHJldHVybiBQcm9taXNlLnJlc29sdmUoZWZmZWN0LnRyYW5zZm9ybShiYXNlLnZhbHVlLCBjaGVja0N0eCkpLnRoZW4oKHJlc3VsdCkgPT4gKHtzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IHJlc3VsdH0pKTtcbiAgICAgICAgfSk7XG4gICAgICB9XG4gICAgfVxuICAgIHV0aWwuYXNzZXJ0TmV2ZXIoZWZmZWN0KTtcbiAgfVxufTtcblpvZEVmZmVjdHMuY3JlYXRlID0gKHNjaGVtYSwgZWZmZWN0LCBwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RFZmZlY3RzKHtcbiAgICBzY2hlbWEsXG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RFZmZlY3RzLFxuICAgIGVmZmVjdCxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xuWm9kRWZmZWN0cy5jcmVhdGVXaXRoUHJlcHJvY2VzcyA9IChwcmVwcm9jZXNzLCBzY2hlbWEsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZEVmZmVjdHMoe1xuICAgIHNjaGVtYSxcbiAgICBlZmZlY3Q6IHt0eXBlOiBcInByZXByb2Nlc3NcIiwgdHJhbnNmb3JtOiBwcmVwcm9jZXNzfSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEVmZmVjdHMsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2RPcHRpb25hbCA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSA9PT0gWm9kUGFyc2VkVHlwZS51bmRlZmluZWQpIHtcbiAgICAgIHJldHVybiBPSyh2b2lkIDApO1xuICAgIH1cbiAgICByZXR1cm4gdGhpcy5fZGVmLmlubmVyVHlwZS5fcGFyc2UoaW5wdXQpO1xuICB9XG4gIHVud3JhcCgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLmlubmVyVHlwZTtcbiAgfVxufTtcblpvZE9wdGlvbmFsLmNyZWF0ZSA9ICh0eXBlLCBwYXJhbXMpID0+IHtcbiAgcmV0dXJuIG5ldyBab2RPcHRpb25hbCh7XG4gICAgaW5uZXJUeXBlOiB0eXBlLFxuICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kT3B0aW9uYWwsXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBab2ROdWxsYWJsZSA9IGNsYXNzIGV4dGVuZHMgWm9kVHlwZSB7XG4gIF9wYXJzZShpbnB1dCkge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSA9PT0gWm9kUGFyc2VkVHlwZS5udWxsKSB7XG4gICAgICByZXR1cm4gT0sobnVsbCk7XG4gICAgfVxuICAgIHJldHVybiB0aGlzLl9kZWYuaW5uZXJUeXBlLl9wYXJzZShpbnB1dCk7XG4gIH1cbiAgdW53cmFwKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuaW5uZXJUeXBlO1xuICB9XG59O1xuWm9kTnVsbGFibGUuY3JlYXRlID0gKHR5cGUsIHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZE51bGxhYmxlKHtcbiAgICBpbm5lclR5cGU6IHR5cGUsXG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROdWxsYWJsZSxcbiAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcylcbiAgfSk7XG59O1xudmFyIFpvZERlZmF1bHQgPSBjbGFzcyBleHRlbmRzIFpvZFR5cGUge1xuICBfcGFyc2UoaW5wdXQpIHtcbiAgICBjb25zdCB7Y3R4fSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgbGV0IGRhdGEgPSBjdHguZGF0YTtcbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgPT09IFpvZFBhcnNlZFR5cGUudW5kZWZpbmVkKSB7XG4gICAgICBkYXRhID0gdGhpcy5fZGVmLmRlZmF1bHRWYWx1ZSgpO1xuICAgIH1cbiAgICByZXR1cm4gdGhpcy5fZGVmLmlubmVyVHlwZS5fcGFyc2Uoe1xuICAgICAgZGF0YSxcbiAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgcGFyZW50OiBjdHhcbiAgICB9KTtcbiAgfVxuICByZW1vdmVEZWZhdWx0KCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuaW5uZXJUeXBlO1xuICB9XG59O1xuWm9kRGVmYXVsdC5jcmVhdGUgPSAodHlwZSwgcGFyYW1zKSA9PiB7XG4gIHJldHVybiBuZXcgWm9kT3B0aW9uYWwoe1xuICAgIGlubmVyVHlwZTogdHlwZSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE9wdGlvbmFsLFxuICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKVxuICB9KTtcbn07XG52YXIgWm9kTmFOID0gY2xhc3MgZXh0ZW5kcyBab2RUeXBlIHtcbiAgX3BhcnNlKGlucHV0KSB7XG4gICAgY29uc3QgcGFyc2VkVHlwZSA9IHRoaXMuX2dldFR5cGUoaW5wdXQpO1xuICAgIGlmIChwYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLm5hbikge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm5hbixcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICByZXR1cm4ge3N0YXR1czogXCJ2YWxpZFwiLCB2YWx1ZTogaW5wdXQuZGF0YX07XG4gIH1cbn07XG5ab2ROYU4uY3JlYXRlID0gKHBhcmFtcykgPT4ge1xuICByZXR1cm4gbmV3IFpvZE5hTih7XG4gICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROYU4sXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpXG4gIH0pO1xufTtcbnZhciBjdXN0b20gPSAoY2hlY2ssIHBhcmFtcyA9IHt9LCBmYXRhbCkgPT4ge1xuICBpZiAoY2hlY2spXG4gICAgcmV0dXJuIFpvZEFueS5jcmVhdGUoKS5zdXBlclJlZmluZSgoZGF0YSwgY3R4KSA9PiB7XG4gICAgICBpZiAoIWNoZWNrKGRhdGEpKSB7XG4gICAgICAgIGNvbnN0IHAgPSB0eXBlb2YgcGFyYW1zID09PSBcImZ1bmN0aW9uXCIgPyBwYXJhbXMoZGF0YSkgOiBwYXJhbXM7XG4gICAgICAgIGNvbnN0IHAyID0gdHlwZW9mIHAgPT09IFwic3RyaW5nXCIgPyB7bWVzc2FnZTogcH0gOiBwO1xuICAgICAgICBjdHguYWRkSXNzdWUoe2NvZGU6IFwiY3VzdG9tXCIsIC4uLnAyLCBmYXRhbH0pO1xuICAgICAgfVxuICAgIH0pO1xuICByZXR1cm4gWm9kQW55LmNyZWF0ZSgpO1xufTtcbnZhciBsYXRlID0ge1xuICBvYmplY3Q6IFpvZE9iamVjdC5sYXp5Y3JlYXRlXG59O1xudmFyIFpvZEZpcnN0UGFydHlUeXBlS2luZDtcbihmdW5jdGlvbihab2RGaXJzdFBhcnR5VHlwZUtpbmQyKSB7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2RTdHJpbmdcIl0gPSBcIlpvZFN0cmluZ1wiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kTnVtYmVyXCJdID0gXCJab2ROdW1iZXJcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZE5hTlwiXSA9IFwiWm9kTmFOXCI7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2RCaWdJbnRcIl0gPSBcIlpvZEJpZ0ludFwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kQm9vbGVhblwiXSA9IFwiWm9kQm9vbGVhblwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kRGF0ZVwiXSA9IFwiWm9kRGF0ZVwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kVW5kZWZpbmVkXCJdID0gXCJab2RVbmRlZmluZWRcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZE51bGxcIl0gPSBcIlpvZE51bGxcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZEFueVwiXSA9IFwiWm9kQW55XCI7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2RVbmtub3duXCJdID0gXCJab2RVbmtub3duXCI7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2ROZXZlclwiXSA9IFwiWm9kTmV2ZXJcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZFZvaWRcIl0gPSBcIlpvZFZvaWRcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZEFycmF5XCJdID0gXCJab2RBcnJheVwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kT2JqZWN0XCJdID0gXCJab2RPYmplY3RcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZFVuaW9uXCJdID0gXCJab2RVbmlvblwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kRGlzY3JpbWluYXRlZFVuaW9uXCJdID0gXCJab2REaXNjcmltaW5hdGVkVW5pb25cIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZEludGVyc2VjdGlvblwiXSA9IFwiWm9kSW50ZXJzZWN0aW9uXCI7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2RUdXBsZVwiXSA9IFwiWm9kVHVwbGVcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZFJlY29yZFwiXSA9IFwiWm9kUmVjb3JkXCI7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2RNYXBcIl0gPSBcIlpvZE1hcFwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kU2V0XCJdID0gXCJab2RTZXRcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZEZ1bmN0aW9uXCJdID0gXCJab2RGdW5jdGlvblwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kTGF6eVwiXSA9IFwiWm9kTGF6eVwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kTGl0ZXJhbFwiXSA9IFwiWm9kTGl0ZXJhbFwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kRW51bVwiXSA9IFwiWm9kRW51bVwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kRWZmZWN0c1wiXSA9IFwiWm9kRWZmZWN0c1wiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kTmF0aXZlRW51bVwiXSA9IFwiWm9kTmF0aXZlRW51bVwiO1xuICBab2RGaXJzdFBhcnR5VHlwZUtpbmQyW1wiWm9kT3B0aW9uYWxcIl0gPSBcIlpvZE9wdGlvbmFsXCI7XG4gIFpvZEZpcnN0UGFydHlUeXBlS2luZDJbXCJab2ROdWxsYWJsZVwiXSA9IFwiWm9kTnVsbGFibGVcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZERlZmF1bHRcIl0gPSBcIlpvZERlZmF1bHRcIjtcbiAgWm9kRmlyc3RQYXJ0eVR5cGVLaW5kMltcIlpvZFByb21pc2VcIl0gPSBcIlpvZFByb21pc2VcIjtcbn0pKFpvZEZpcnN0UGFydHlUeXBlS2luZCB8fCAoWm9kRmlyc3RQYXJ0eVR5cGVLaW5kID0ge30pKTtcbnZhciBpbnN0YW5jZU9mVHlwZSA9IChjbHMsIHBhcmFtcyA9IHtcbiAgbWVzc2FnZTogYElucHV0IG5vdCBpbnN0YW5jZSBvZiAke2Nscy5uYW1lfWBcbn0pID0+IGN1c3RvbSgoZGF0YSkgPT4gZGF0YSBpbnN0YW5jZW9mIGNscywgcGFyYW1zLCB0cnVlKTtcbnZhciBzdHJpbmdUeXBlID0gWm9kU3RyaW5nLmNyZWF0ZTtcbnZhciBudW1iZXJUeXBlID0gWm9kTnVtYmVyLmNyZWF0ZTtcbnZhciBuYW5UeXBlID0gWm9kTmFOLmNyZWF0ZTtcbnZhciBiaWdJbnRUeXBlID0gWm9kQmlnSW50LmNyZWF0ZTtcbnZhciBib29sZWFuVHlwZSA9IFpvZEJvb2xlYW4uY3JlYXRlO1xudmFyIGRhdGVUeXBlID0gWm9kRGF0ZS5jcmVhdGU7XG52YXIgdW5kZWZpbmVkVHlwZSA9IFpvZFVuZGVmaW5lZC5jcmVhdGU7XG52YXIgbnVsbFR5cGUgPSBab2ROdWxsLmNyZWF0ZTtcbnZhciBhbnlUeXBlID0gWm9kQW55LmNyZWF0ZTtcbnZhciB1bmtub3duVHlwZSA9IFpvZFVua25vd24uY3JlYXRlO1xudmFyIG5ldmVyVHlwZSA9IFpvZE5ldmVyLmNyZWF0ZTtcbnZhciB2b2lkVHlwZSA9IFpvZFZvaWQuY3JlYXRlO1xudmFyIGFycmF5VHlwZSA9IFpvZEFycmF5LmNyZWF0ZTtcbnZhciBvYmplY3RUeXBlID0gWm9kT2JqZWN0LmNyZWF0ZTtcbnZhciBzdHJpY3RPYmplY3RUeXBlID0gWm9kT2JqZWN0LnN0cmljdENyZWF0ZTtcbnZhciB1bmlvblR5cGUgPSBab2RVbmlvbi5jcmVhdGU7XG52YXIgZGlzY3JpbWluYXRlZFVuaW9uVHlwZSA9IFpvZERpc2NyaW1pbmF0ZWRVbmlvbi5jcmVhdGU7XG52YXIgaW50ZXJzZWN0aW9uVHlwZSA9IFpvZEludGVyc2VjdGlvbi5jcmVhdGU7XG52YXIgdHVwbGVUeXBlID0gWm9kVHVwbGUuY3JlYXRlO1xudmFyIHJlY29yZFR5cGUgPSBab2RSZWNvcmQuY3JlYXRlO1xudmFyIG1hcFR5cGUgPSBab2RNYXAuY3JlYXRlO1xudmFyIHNldFR5cGUgPSBab2RTZXQuY3JlYXRlO1xudmFyIGZ1bmN0aW9uVHlwZSA9IFpvZEZ1bmN0aW9uLmNyZWF0ZTtcbnZhciBsYXp5VHlwZSA9IFpvZExhenkuY3JlYXRlO1xudmFyIGxpdGVyYWxUeXBlID0gWm9kTGl0ZXJhbC5jcmVhdGU7XG52YXIgZW51bVR5cGUgPSBab2RFbnVtLmNyZWF0ZTtcbnZhciBuYXRpdmVFbnVtVHlwZSA9IFpvZE5hdGl2ZUVudW0uY3JlYXRlO1xudmFyIHByb21pc2VUeXBlID0gWm9kUHJvbWlzZS5jcmVhdGU7XG52YXIgZWZmZWN0c1R5cGUgPSBab2RFZmZlY3RzLmNyZWF0ZTtcbnZhciBvcHRpb25hbFR5cGUgPSBab2RPcHRpb25hbC5jcmVhdGU7XG52YXIgbnVsbGFibGVUeXBlID0gWm9kTnVsbGFibGUuY3JlYXRlO1xudmFyIHByZXByb2Nlc3NUeXBlID0gWm9kRWZmZWN0cy5jcmVhdGVXaXRoUHJlcHJvY2VzcztcbnZhciBvc3RyaW5nID0gKCkgPT4gc3RyaW5nVHlwZSgpLm9wdGlvbmFsKCk7XG52YXIgb251bWJlciA9ICgpID0+IG51bWJlclR5cGUoKS5vcHRpb25hbCgpO1xudmFyIG9ib29sZWFuID0gKCkgPT4gYm9vbGVhblR5cGUoKS5vcHRpb25hbCgpO1xudmFyIG1vZCA9IC8qIEBfX1BVUkVfXyAqLyBPYmplY3QuZnJlZXplKHtcbiAgX19wcm90b19fOiBudWxsLFxuICBnZXRQYXJzZWRUeXBlLFxuICBab2RQYXJzZWRUeXBlLFxuICBtYWtlSXNzdWUsXG4gIEVNUFRZX1BBVEgsXG4gIGFkZElzc3VlVG9Db250ZXh0LFxuICBQYXJzZVN0YXR1cyxcbiAgSU5WQUxJRCxcbiAgRElSVFksXG4gIE9LLFxuICBpc0Fib3J0ZWQsXG4gIGlzRGlydHksXG4gIGlzVmFsaWQsXG4gIGlzQXN5bmMsXG4gIGpzb25TdHJpbmdpZnlSZXBsYWNlcixcbiAgWm9kVHlwZSxcbiAgWm9kU3RyaW5nLFxuICBab2ROdW1iZXIsXG4gIFpvZEJpZ0ludCxcbiAgWm9kQm9vbGVhbixcbiAgWm9kRGF0ZSxcbiAgWm9kVW5kZWZpbmVkLFxuICBab2ROdWxsLFxuICBab2RBbnksXG4gIFpvZFVua25vd24sXG4gIFpvZE5ldmVyLFxuICBab2RWb2lkLFxuICBab2RBcnJheSxcbiAgZ2V0IG9iamVjdFV0aWwoKSB7XG4gICAgcmV0dXJuIG9iamVjdFV0aWw7XG4gIH0sXG4gIFpvZE9iamVjdCxcbiAgWm9kVW5pb24sXG4gIFpvZERpc2NyaW1pbmF0ZWRVbmlvbixcbiAgWm9kSW50ZXJzZWN0aW9uLFxuICBab2RUdXBsZSxcbiAgWm9kUmVjb3JkLFxuICBab2RNYXAsXG4gIFpvZFNldCxcbiAgWm9kRnVuY3Rpb24sXG4gIFpvZExhenksXG4gIFpvZExpdGVyYWwsXG4gIFpvZEVudW0sXG4gIFpvZE5hdGl2ZUVudW0sXG4gIFpvZFByb21pc2UsXG4gIFpvZEVmZmVjdHMsXG4gIFpvZFRyYW5zZm9ybWVyOiBab2RFZmZlY3RzLFxuICBab2RPcHRpb25hbCxcbiAgWm9kTnVsbGFibGUsXG4gIFpvZERlZmF1bHQsXG4gIFpvZE5hTixcbiAgY3VzdG9tLFxuICBTY2hlbWE6IFpvZFR5cGUsXG4gIFpvZFNjaGVtYTogWm9kVHlwZSxcbiAgbGF0ZSxcbiAgZ2V0IFpvZEZpcnN0UGFydHlUeXBlS2luZCgpIHtcbiAgICByZXR1cm4gWm9kRmlyc3RQYXJ0eVR5cGVLaW5kO1xuICB9LFxuICBhbnk6IGFueVR5cGUsXG4gIGFycmF5OiBhcnJheVR5cGUsXG4gIGJpZ2ludDogYmlnSW50VHlwZSxcbiAgYm9vbGVhbjogYm9vbGVhblR5cGUsXG4gIGRhdGU6IGRhdGVUeXBlLFxuICBkaXNjcmltaW5hdGVkVW5pb246IGRpc2NyaW1pbmF0ZWRVbmlvblR5cGUsXG4gIGVmZmVjdDogZWZmZWN0c1R5cGUsXG4gIGVudW06IGVudW1UeXBlLFxuICBmdW5jdGlvbjogZnVuY3Rpb25UeXBlLFxuICBpbnN0YW5jZW9mOiBpbnN0YW5jZU9mVHlwZSxcbiAgaW50ZXJzZWN0aW9uOiBpbnRlcnNlY3Rpb25UeXBlLFxuICBsYXp5OiBsYXp5VHlwZSxcbiAgbGl0ZXJhbDogbGl0ZXJhbFR5cGUsXG4gIG1hcDogbWFwVHlwZSxcbiAgbmFuOiBuYW5UeXBlLFxuICBuYXRpdmVFbnVtOiBuYXRpdmVFbnVtVHlwZSxcbiAgbmV2ZXI6IG5ldmVyVHlwZSxcbiAgbnVsbDogbnVsbFR5cGUsXG4gIG51bGxhYmxlOiBudWxsYWJsZVR5cGUsXG4gIG51bWJlcjogbnVtYmVyVHlwZSxcbiAgb2JqZWN0OiBvYmplY3RUeXBlLFxuICBvYm9vbGVhbixcbiAgb251bWJlcixcbiAgb3B0aW9uYWw6IG9wdGlvbmFsVHlwZSxcbiAgb3N0cmluZyxcbiAgcHJlcHJvY2VzczogcHJlcHJvY2Vzc1R5cGUsXG4gIHByb21pc2U6IHByb21pc2VUeXBlLFxuICByZWNvcmQ6IHJlY29yZFR5cGUsXG4gIHNldDogc2V0VHlwZSxcbiAgc3RyaWN0T2JqZWN0OiBzdHJpY3RPYmplY3RUeXBlLFxuICBzdHJpbmc6IHN0cmluZ1R5cGUsXG4gIHRyYW5zZm9ybWVyOiBlZmZlY3RzVHlwZSxcbiAgdHVwbGU6IHR1cGxlVHlwZSxcbiAgdW5kZWZpbmVkOiB1bmRlZmluZWRUeXBlLFxuICB1bmlvbjogdW5pb25UeXBlLFxuICB1bmtub3duOiB1bmtub3duVHlwZSxcbiAgdm9pZDogdm9pZFR5cGUsXG4gIFpvZElzc3VlQ29kZSxcbiAgcXVvdGVsZXNzSnNvbixcbiAgWm9kRXJyb3IsXG4gIGRlZmF1bHRFcnJvck1hcCxcbiAgc2V0RXJyb3JNYXAsXG4gIGdldEVycm9yTWFwXG59KTtcbmZ1bmN0aW9uIGhhc0NvbW1hKG51bSkge1xuICByZXR1cm4gbnVtID09PSAwID8gXCJcIiA6IFwiLFwiO1xufVxuZnVuY3Rpb24gY2Fub25pZnkob2JqZWN0KSB7XG4gIGlmIChvYmplY3QgPT09IG51bGwgfHwgdHlwZW9mIG9iamVjdCA9PT0gXCJ1bmRlZmluZWRcIiB8fCB0eXBlb2Ygb2JqZWN0ID09PSBcImJvb2xlYW5cIiB8fCB0eXBlb2Ygb2JqZWN0ID09PSBcIm51bWJlclwiIHx8IHR5cGVvZiBvYmplY3QgPT09IFwic3RyaW5nXCIpIHtcbiAgICByZXR1cm4gSlNPTi5zdHJpbmdpZnkob2JqZWN0KTtcbiAgfVxuICBpZiAodHlwZW9mIG9iamVjdCA9PT0gXCJiaWdpbnRcIikge1xuICAgIHRocm93IG5ldyBUeXBlRXJyb3IoXCJCaWdJbnQgdmFsdWUgY2FuJ3QgYmUgc2VyaWFsaXplZCBpbiBKU09OXCIpO1xuICB9XG4gIGlmICh0eXBlb2Ygb2JqZWN0ID09PSBcImZ1bmN0aW9uXCIgfHwgdHlwZW9mIG9iamVjdCA9PT0gXCJzeW1ib2xcIikge1xuICAgIHJldHVybiBjYW5vbmlmeSh2b2lkIDApO1xuICB9XG4gIGlmIChvYmplY3QudG9KU09OIGluc3RhbmNlb2YgRnVuY3Rpb24pIHtcbiAgICByZXR1cm4gY2Fub25pZnkob2JqZWN0LnRvSlNPTigpKTtcbiAgfVxuICBpZiAoQXJyYXkuaXNBcnJheShvYmplY3QpKSB7XG4gICAgY29uc3QgdmFsdWVzMiA9IG9iamVjdC5yZWR1Y2UoKHQsIGN2LCBjaSkgPT4ge1xuICAgICAgY29uc3QgdmFsdWUgPSBjdiA9PT0gdm9pZCAwIHx8IHR5cGVvZiBjdiA9PT0gXCJzeW1ib2xcIiB8fCB0eXBlb2YgY3YgPT09IFwiZnVuY3Rpb25cIiA/IG51bGwgOiBjdjtcbiAgICAgIHJldHVybiBgJHt0fSR7aGFzQ29tbWEoY2kpfSR7Y2Fub25pZnkodmFsdWUpfWA7XG4gICAgfSwgXCJcIik7XG4gICAgcmV0dXJuIGBbJHt2YWx1ZXMyfV1gO1xuICB9XG4gIGNvbnN0IHZhbHVlcyA9IE9iamVjdC5rZXlzKG9iamVjdCkuc29ydCgpLnJlZHVjZSgodCwgY3YpID0+IHtcbiAgICBpZiAob2JqZWN0W2N2XSA9PT0gdm9pZCAwIHx8IHR5cGVvZiBvYmplY3RbY3ZdID09PSBcInN5bWJvbFwiIHx8IHR5cGVvZiBvYmplY3RbY3ZdID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgIHJldHVybiB0O1xuICAgIH1cbiAgICByZXR1cm4gYCR7dH0ke2hhc0NvbW1hKHQubGVuZ3RoKX0ke2Nhbm9uaWZ5KGN2KX06JHtjYW5vbmlmeShvYmplY3RbY3ZdKX1gO1xuICB9LCBcIlwiKTtcbiAgcmV0dXJuIGB7JHt2YWx1ZXN9fWA7XG59XG52YXIge2Jhc2U1OGNoZWNrfSA9IHJlcXVpcmVfbGliKCk7XG52YXIgSURfUFJFRklYID0gXCJ0cnVlc3RhbXBcIjtcbnZhciBJRF9TRVBBUkFUT1IgPSBcIi1cIjtcbnZhciBSRUdFWF9VTElEID0gL15bMDEyMzQ1Njc4OUFCQ0RFRkdISktNTlBRUlNUVldYWVpdezI2fSQvO1xudmFyIGlkQmluYXJ5U3RydWN0ID0gdXNlKHN0cnVjdCh7XG4gIHRlc3Q6IGJvb2xlYW4sXG4gIHRpbWVzdGFtcDogYmlnVWludDY0LFxuICB1bGlkOiBzdHJpbmcobGF0aW4xLCB1aW50OCksXG4gIHNpcGhhc2g6IGJ5dGVzKHVpbnQ4KVxufSkpO1xudmFyIFNpcEhhc2hLZXlTdHJ1Y3QgPSBtb2QuaW5zdGFuY2VvZihVaW50OEFycmF5KS5yZWZpbmUoKHZhbCkgPT4gdmFsLmxlbmd0aCA9PT0gOCwge1xuICBtZXNzYWdlOiBgU2lwSGFzaCBrZXkgbXVzdCBiZSAke2ltcG9ydF9oYWxmc2lwaGFzaC5LRVlfTEVOR1RIfSBieXRlc2Bcbn0pO1xudmFyIFNpcEhhc2hIYXNoU3RydWN0ID0gbW9kLmluc3RhbmNlb2YoVWludDhBcnJheSkucmVmaW5lKCh2YWx1ZSkgPT4ge1xuICByZXR1cm4gdmFsdWUubGVuZ3RoID09PSA0O1xufSwge1xuICBtZXNzYWdlOiBgU2lwSGFzaCBoYXNoIHNob3VsZCBiZSA0IGJ5dGVzYFxufSk7XG52YXIgUGF5bG9hZFN0cnVjdCA9IG1vZC5vYmplY3Qoe1xuICB0ZXN0OiBtb2QuYm9vbGVhbigpLmRlZmF1bHQoZmFsc2UpLFxuICB0aW1lc3RhbXA6IG1vZC5zdHJpbmcoKSxcbiAgdWxpZDogbW9kLnN0cmluZygpLnJlZ2V4KFJFR0VYX1VMSUQpXG59KTtcbmZ1bmN0aW9uIGhhc2hQYXlsb2FkKHBheWxvYWQsIGtleSkge1xuICBpZiAoIVBheWxvYWRTdHJ1Y3Quc2FmZVBhcnNlKHBheWxvYWQpLnN1Y2Nlc3MpIHtcbiAgICB0aHJvdyBuZXcgRXJyb3IoYGludmFsaWQgcGF5bG9hZGApO1xuICB9XG4gIGlmICghU2lwSGFzaEtleVN0cnVjdC5zYWZlUGFyc2Uoa2V5KS5zdWNjZXNzKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKGBTaXBIYXNoIGtleSBtdXN0IGJlICR7aW1wb3J0X2hhbGZzaXBoYXNoLktFWV9MRU5HVEh9IGJ5dGVzYCk7XG4gIH1cbiAgY29uc3QgY2Fub25pY2FsaXplZFBheWxvYWQgPSBjYW5vbmlmeShwYXlsb2FkKTtcbiAgaWYgKCFjYW5vbmljYWxpemVkUGF5bG9hZCkge1xuICAgIHRocm93IG5ldyBFcnJvcihgcGF5bG9hZCBjYW5vbmljYWxpemF0aW9uIGVycm9yYCk7XG4gIH1cbiAgY29uc3QgaGFzaCA9ICgwLCBpbXBvcnRfaGFsZnNpcGhhc2guaGFsZlNpcEhhc2gpKGtleSwgbmV3IFRleHRFbmNvZGVyKCkuZW5jb2RlKGNhbm9uaWNhbGl6ZWRQYXlsb2FkKSk7XG4gIGlmICghU2lwSGFzaEhhc2hTdHJ1Y3Quc2FmZVBhcnNlKGhhc2gpLnN1Y2Nlc3MpIHtcbiAgICB0aHJvdyBuZXcgRXJyb3IoYGludmFsaWQgU2lwSGFzaCBoYXNoIGdlbmVyYXRlZGApO1xuICB9XG4gIHJldHVybiBoYXNoO1xufVxuZnVuY3Rpb24gcGFyc2VJZChpZCkge1xuICBjb25zdCBzcGxpdElkID0gaWQuc3BsaXQoSURfU0VQQVJBVE9SKTtcbiAgaWYgKHNwbGl0SWQubGVuZ3RoICE9PSAxICYmIHNwbGl0SWQubGVuZ3RoICE9PSAyKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKGBzdHJpbmcgZm9ybWF0IGVycm9yYCk7XG4gIH1cbiAgY29uc3QgaWRTdHJpbmcgPSBzcGxpdElkLmxlbmd0aCA9PT0gMiA/IHNwbGl0SWRbMV0gOiBzcGxpdElkWzBdO1xuICBjb25zdCBkZWNvZGVkSWQgPSBiYXNlNThjaGVjayhpbXBvcnRfc2hhMjU2Lmhhc2gpLmRlY29kZShpZFN0cmluZyk7XG4gIGNvbnN0IGRlc2VyaWFsaXplZElkID0gaWRCaW5hcnlTdHJ1Y3QuZnJvbUJ5dGVzKGRlY29kZWRJZCk7XG4gIGNvbnN0IHtzaXBoYXNoLCAuLi5wYXlsb2FkfSA9IGRlc2VyaWFsaXplZElkO1xuICBjb25zdCBwYXlsb2FkV2l0aFN0cmluZ1RpbWVzdGFtcCA9IHtcbiAgICAuLi5wYXlsb2FkLFxuICAgIHRpbWVzdGFtcDogcGF5bG9hZC50aW1lc3RhbXAudG9TdHJpbmcoKVxuICB9O1xuICBpZiAoIVBheWxvYWRTdHJ1Y3Quc2FmZVBhcnNlKHBheWxvYWRXaXRoU3RyaW5nVGltZXN0YW1wKS5zdWNjZXNzKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKGBpbnZhbGlkIHBheWxvYWRgKTtcbiAgfVxuICByZXR1cm4gW1xuICAgIHBheWxvYWRXaXRoU3RyaW5nVGltZXN0YW1wLFxuICAgIHNpcGhhc2hcbiAgXTtcbn1cbnZhciBlbmNvZGUgPSAocGF5bG9hZCwga2V5KSA9PiB7XG4gIHRyeSB7XG4gICAgY29uc3QgaGFzaCA9IGhhc2hQYXlsb2FkKHBheWxvYWQsIGtleSk7XG4gICAgY29uc3QgaWRCeXRlcyA9IGlkQmluYXJ5U3RydWN0LnRvQnl0ZXMoe1xuICAgICAgc2lwaGFzaDogaGFzaCxcbiAgICAgIHRlc3Q6IHBheWxvYWQudGVzdCxcbiAgICAgIHRpbWVzdGFtcDogQmlnSW50KHBheWxvYWQudGltZXN0YW1wKSxcbiAgICAgIHVsaWQ6IHBheWxvYWQudWxpZFxuICAgIH0pO1xuICAgIGNvbnN0IGlkU3RyaW5nID0gYmFzZTU4Y2hlY2soaW1wb3J0X3NoYTI1Ni5oYXNoKS5lbmNvZGUoaWRCeXRlcyk7XG4gICAgcmV0dXJuIGAke0lEX1BSRUZJWH0ke0lEX1NFUEFSQVRPUn0ke2lkU3RyaW5nfWA7XG4gIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgaWYgKGVycm9yIGluc3RhbmNlb2YgbW9kLlpvZEVycm9yKSB7XG4gICAgICBjb25zdCBqb2luZWRJc3N1ZXMgPSBlcnJvci5pc3N1ZXMubWFwKChpc3N1ZSkgPT4ge1xuICAgICAgICByZXR1cm4gYCR7aXNzdWUuY29kZX0gOiBbJHtpc3N1ZS5wYXRoLmpvaW4oXCIsIFwiKX1dIDogJHtpc3N1ZS5tZXNzYWdlfWA7XG4gICAgICB9KS5qb2luKFwiOyBcIik7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoYEludmFsaWQgSUQ6ICAke2pvaW5lZElzc3Vlc31gKTtcbiAgICB9IGVsc2UgaWYgKGVycm9yIGluc3RhbmNlb2YgRXJyb3IpIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihgSW52YWxpZCBJRDogJHtlcnJvci5tZXNzYWdlfWApO1xuICAgIH0gZWxzZSB7XG4gICAgICB0aHJvdyBlcnJvcjtcbiAgICB9XG4gIH1cbn07XG52YXIgZGVjb2RlID0gKGlkLCBrZXkpID0+IHtcbiAgdHJ5IHtcbiAgICBjb25zdCBbcGF5bG9hZCwgc2lwaGFzaF0gPSBwYXJzZUlkKGlkKTtcbiAgICBjb25zdCBuZXdQYXlsb2FkSGFzaCA9IGhhc2hQYXlsb2FkKHBheWxvYWQsIGtleSk7XG4gICAgaWYgKCEoMCwgaW1wb3J0X2NvbnN0YW50X3RpbWUuZXF1YWwpKG5ld1BheWxvYWRIYXNoLCBzaXBoYXNoKSkge1xuICAgICAgdGhyb3cgbmV3IEVycm9yKGBTaXBIYXNoIHZlcmlmaWNhdGlvbiBlcnJvcmApO1xuICAgIH1cbiAgICByZXR1cm4gcGF5bG9hZDtcbiAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICBpZiAoZXJyb3IgaW5zdGFuY2VvZiBtb2QuWm9kRXJyb3IpIHtcbiAgICAgIGNvbnN0IGpvaW5lZElzc3VlcyA9IGVycm9yLmlzc3Vlcy5tYXAoKGlzc3VlKSA9PiB7XG4gICAgICAgIHJldHVybiBgJHtpc3N1ZS5jb2RlfSA6IFske2lzc3VlLnBhdGguam9pbihcIiwgXCIpfV0gOiAke2lzc3VlLm1lc3NhZ2V9YDtcbiAgICAgIH0pLmpvaW4oXCI7IFwiKTtcbiAgICAgIHRocm93IG5ldyBFcnJvcihgSW52YWxpZCBJRDogICR7am9pbmVkSXNzdWVzfWApO1xuICAgIH0gZWxzZSBpZiAoZXJyb3IgaW5zdGFuY2VvZiBFcnJvcikge1xuICAgICAgdGhyb3cgbmV3IEVycm9yKGBJbnZhbGlkIElEOiAke2Vycm9yLm1lc3NhZ2V9YCk7XG4gICAgfSBlbHNlIHtcbiAgICAgIHRocm93IGVycm9yO1xuICAgIH1cbiAgfVxufTtcbnZhciBkZWNvZGVVbnNhZmVseSA9IChpZCkgPT4ge1xuICB0cnkge1xuICAgIGNvbnN0IFtwYXlsb2FkXSA9IHBhcnNlSWQoaWQpO1xuICAgIHJldHVybiBwYXlsb2FkO1xuICB9IGNhdGNoIChlcnJvcikge1xuICAgIGlmIChlcnJvciBpbnN0YW5jZW9mIG1vZC5ab2RFcnJvcikge1xuICAgICAgY29uc3Qgam9pbmVkSXNzdWVzID0gZXJyb3IuaXNzdWVzLm1hcCgoaXNzdWUpID0+IHtcbiAgICAgICAgcmV0dXJuIGAke2lzc3VlLmNvZGV9IDogWyR7aXNzdWUucGF0aC5qb2luKFwiLCBcIil9XSA6ICR7aXNzdWUubWVzc2FnZX1gO1xuICAgICAgfSkuam9pbihcIjsgXCIpO1xuICAgICAgdGhyb3cgbmV3IEVycm9yKGBJbnZhbGlkIElEOiAgJHtqb2luZWRJc3N1ZXN9YCk7XG4gICAgfSBlbHNlIGlmIChlcnJvciBpbnN0YW5jZW9mIEVycm9yKSB7XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoYEludmFsaWQgSUQ6ICR7ZXJyb3IubWVzc2FnZX1gKTtcbiAgICB9IGVsc2Uge1xuICAgICAgdGhyb3cgZXJyb3I7XG4gICAgfVxuICB9XG59O1xudmFyIGlzVmFsaWQyID0gKGlkLCBrZXkpID0+IHtcbiAgdHJ5IHtcbiAgICBkZWNvZGUoaWQsIGtleSk7XG4gICAgcmV0dXJuIHRydWU7XG4gIH0gY2F0Y2ggKGVycm9yKSB7XG4gICAgcmV0dXJuIGZhbHNlO1xuICB9XG59O1xudmFyIGlzVmFsaWRVbnNhZmVseSA9IChpZCkgPT4ge1xuICB0cnkge1xuICAgIGRlY29kZVVuc2FmZWx5KGlkKTtcbiAgICByZXR1cm4gdHJ1ZTtcbiAgfSBjYXRjaCAoZXJyb3IpIHtcbiAgICByZXR1cm4gZmFsc2U7XG4gIH1cbn07XG4vKiEgc2N1cmUtYmFzZSAtIE1JVCBMaWNlbnNlIChjKSAyMDIyIFBhdWwgTWlsbGVyIChwYXVsbWlsbHIuY29tKSAqL1xuZXhwb3J0IHtQYXlsb2FkU3RydWN0LCBTaXBIYXNoSGFzaFN0cnVjdCwgU2lwSGFzaEtleVN0cnVjdCwgZGVjb2RlLCBkZWNvZGVVbnNhZmVseSwgZW5jb2RlLCBpc1ZhbGlkMiBhcyBpc1ZhbGlkLCBpc1ZhbGlkVW5zYWZlbHl9O1xuZXhwb3J0IGRlZmF1bHQgbnVsbDtcbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxJQUFJLFFBQVEsR0FBRyxNQUFNLENBQUMsTUFBTSxBQUFDO0FBQzdCLElBQUksU0FBUyxHQUFHLE1BQU0sQ0FBQyxjQUFjLEFBQUM7QUFDdEMsSUFBSSxnQkFBZ0IsR0FBRyxNQUFNLENBQUMsd0JBQXdCLEFBQUM7QUFDdkQsSUFBSSxpQkFBaUIsR0FBRyxNQUFNLENBQUMsbUJBQW1CLEFBQUM7QUFDbkQsSUFBSSxZQUFZLEdB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QUFDLEdBQUcsR0FBRyxHQUFHLENBQUMsR0FBRyxDQUFDLEFBQUM7b0JBQ3BCLENBQUMsQ0FBQyxDQUFDLENBQUMsR0FBRyxRQUFRLENBQUMsWUFBWSxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQztpQkFDcEM7Z0JBQ0QsSUFBSyxJQUFJLENBQUMsR0FBRyxFQUFFLEVBQUUsQ0FBQyxHQUFHLEVBQUUsRUFBRSxDQUFDLEVBQUUsQ0FBRTtvQkFDNUIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsQUFBQztvQkFDakIsSUFBSSxFQUFFLEdBQUcsQ0FBQyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsRUFBRSxDQUFDLEdBQUcsQ0FBQyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsRUFBRSxDQUFDLEdBQUcsQ0FBQyxLQUFLLEVBQUUsQUFBQztvQkFDMUUsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDLEdBQUcsRUFBRSxDQUFDLENBQUM7b0JBQ2QsSUFBSSxFQUFFLEdBQUcsQ0FBQyxDQUFDLEtBQUssQ0FBQyxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsRUFBRSxDQUFDLEdBQUcsQ0FBQyxLQUFLLENBQUMsQUFBQztvQkFDdkUsQ0FBQyxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUMsRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEdBQUcsQ0FBQyxDQUFDLENBQUMsR0FBRyxFQUFFLENBQUMsR0FBRyxDQUFDLENBQUMsQ0FBQztpQkFDbkQ7Z0JBQ0QsSUFBSyxJQUFJLENBQUMsR0FBRyxDQUFDLEVBQUUsQ0FBQyxHQUFHLEVBQUUsRUFBRSxDQUFDLEVBQUUsQ0FBRTtvQkFDM0IsSUFBSSxFQUFFLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDLElBQUksRUFBRSxHQUFHLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxLQUFLLEVBQUUsR0FBRyxDQUFDLElBQUksRUFBRSxHQUFHLEVBQUUsQ0FBQyxHQUFHLENBQUMsQ0FBQyxLQUFLLEVBQUUsR0FBRyxDQUFDLElBQUksRUFBRSxHQUFHLEVBQUUsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxHQUFHLENBQUMsQ0FBQyxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsR0FBRyxDQUFDLEFBQUM7b0JBQ3RKLElBQUksRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDLEtBQUssQ0FBQyxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsRUFBRSxDQUFDLEdBQUcsQ0FBQyxDQUFDLEtBQUssRUFBRSxHQUFHLENBQUMsSUFBSSxFQUFFLEdBQUcsRUFBRSxDQUFDLENBQUMsR0FBRyxDQUFDLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxHQUFHLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxDQUFDLEdBQUcsQ0FBQyxBQUFDO29CQUN6SCxDQUFDLEdBQUcsQ0FBQyxDQUFDO29CQUNOLENBQUMsR0FBRyxDQUFDLENBQUM7b0JBQ04sQ0FBQyxHQUFHLENBQUMsQ0FBQztvQkFDTixDQUFDLEdBQUcsQ0FBQyxHQUFHLEVBQUUsR0FBRyxDQUFDLENBQUM7b0JBQ2YsQ0FBQyxHQUFHLENBQUMsQ0FBQztvQkFDTixDQUFDLEdBQUcsQ0FBQyxDQUFDO29CQUNOLENBQUMsR0FBRyxDQUFDLENBQUM7b0JBQ04sQ0FBQyxHQUFHLEVBQUUsR0FBRyxFQUFFLEdBQUcsQ0FBQyxD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QUFDLENBQUM7Z0JBQ1IsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO2dCQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7Z0JBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7Z0JBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO2dCQUN4QixFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7Z0JBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztnQkFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztnQkFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7Z0JBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztnQkFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztnQkFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7Z0JBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztnQkFDeEIsRUFBRSxJQUFJLEVBQUUsQ0FBQztnQkFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7Z0JBQ3hCLEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztnQkFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDO2dCQUN2QixFQUFFLElBQUksRUFBRSxDQUFDO2dCQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztnQkFDeEIsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO2dCQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7Z0JBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7Z0JBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO2dCQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7Z0JBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7Z0JBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO2dCQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7Z0JBQ3hCLEVBQUUsSUFBSSxFQUFFLENBQUM7Z0JBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO2dCQUN4QixFQUFFLElBQUksQ0FBQyxDQUFDO2dCQUNSLEdBQUcsSUFBSSxDQUFDLENBQUM7Z0JBQ1QsR0FBRyxJQUFJLENBQUMsQ0FBQzthQUNWO1lBQ0QsT0FBUSxHQUFHO2dCQUNULEtBQUssQ0FBQztvQkFDSixHQUFHLElBQUksSUFBSSxDQUFDLEdBQUcsR0FBRyxDQUFDLENBQUMsSUFBSSxFQUFFLENBQUM7Z0JBQzdCLEtBQUssQ0FBQztvQkFDSixHQUFHLElBQUksSUFBSSxDQUFDLEdBQUcsR0FBRyxDQUFDLENBQUMsSUFBSSxDQUFDLENBQUM7Z0JBQzVCLEtBQUssQ0FBQztvQkFDSixHQUFHLElBQUksSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQ3BCO1lBQ0QsRUFBRSxJQUFJLEdBQUcsQ0FBQztZQUNWLEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7WUFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN4QixFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3hCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDeEIsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztZQUN2QixFQUFFLElBQUksRUFBRSxDQUFDO1lBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3hCLEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7WUFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7WUFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDeEIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN4QixFQUFFLElBQUksR0FBRyxDQUFDO1lBQ1YsRUFBRSxJQUFJLEdBQUcsQ0FBQztZQUNWLEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7WUFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN4QixFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3hCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDeEIsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztZQUN2QixFQUFFLElBQUksRUFBRSxDQUFDO1lBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3hCLEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7WUFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsQ0FBQyxDQUFDLENBQUM7WUFDdkIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN2QixFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDeEIsRUFBRSxJQUFJLEVBQUUsQ0FBQztZQUNULEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN4QixFQUFFLEdBQUcsS0FBSyxDQUFDLEdBQUcsQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDdkIsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLENBQUMsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsSUFBSSxFQUFFLENBQUM7WUFDVCxFQUFFLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLEVBQUUsRUFBRSxDQUFDLENBQUM7WUFDeEIsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztZQUN2QixFQUFFLElBQUksRUFBRSxDQUFDO1lBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztZQUN2QixFQUFFLElBQUksRUFBRSxDQUFDO1lBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3ZCLEVBQUUsR0FBRyxLQUFLLENBQUMsSUFBSSxDQUFDLEVBQUUsRUFBRSxFQUFFLENBQUMsQ0FBQztZQUN4QixFQUFFLElBQUksRUFBRSxDQUFDO1lBQ1QsRUFBRSxHQUFHLEtBQUssQ0FBQyxJQUFJLENBQUMsRUFBRSxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lB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LE1BQU0sQ0FBQyxDQUFDLEdBQUcsRUFBRSxJQUFJLEdBQUssS0FBSyxHQUFHLENBQUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxHQUFHLENBQUMsQ0FBQyxFQUFFLEVBQUUsQ0FBQyxJQUFJLENBQUMsRUFBRSxDQUFDLEdBQUcsR0FBSyxDQUFDLENBQUMsR0FBRyxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQUFBQztBQUNuSCxTQUFTLEdBQUcsQ0FBQyxFQUFDLEdBQUcsQ0FBQSxFQUFFLEdBQUcsQ0FBQSxFQUFDLEVBQUU7SUFDdkIsTUFBTSxHQUFHLEdBQUcsYUFBYSxFQUFFLEFBQUM7SUFDNUIsT0FBTztRQUNMLEdBQUc7UUFDSCxHQUFHO1FBQ0gsT0FBTyxFQUFFLENBQUMsSUFBSSxHQUFLLFVBQVUsQ0FBQyxHQUFHLEVBQUUsR0FBRyxFQUFFLElBQUksQ0FBQyxDQUFDLEtBQUssQ0FBQyxDQUFDLEVBQUUsR0FBRyxDQUFDLENBQUMsQ0FBQztRQUM3RCxhQUFhLEVBQUUsQ0FBQyxJQUFJLEdBQUssVUFBVSxDQUFDLEdBQUcsRUFBRSxHQUFHLEVBQUUsSUFBSSxDQUFDLENBQUMsUUFBUSxDQUFDLENBQUMsRUFBRSxHQUFHLENBQUMsQ0FBQyxDQUFDO1FBQ3RFLFNBQVMsRUFBRSxDQUFDLE1BQU0sR0FBSyxVQUFVLENBQUMsR0FBRyxFQUFFLEdBQUcsRUFBRSxNQUFNLENBQUM7S0FDcEQsQ0FBQztDQUNIO0FBQ0QsSUFBSSxLQUFLLEdBQUcsTUFBTSxDQUFDLENBQUMsR0FBRyxFQUFFLElBQUksR0FBSyxHQUFHLENBQUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxHQUFHLENBQUMsQ0FBQyxFQUFFLEVBQUUsSUFBSSxDQUFDLEVBQUUsQ0FBQyxHQUFHLEdBQUssR0FBRyxDQUFDLElBQUksQ0FBQyxRQUFRLENBQUMsR0FBRyxDQUFDLENBQUMsRUFBRSxDQUFDLENBQUMsQUFBQztBQUN6RyxJQUFJLFNBQVMsR0FBRyxNQUFNLENBQUMsQ0FBQyxHQUFHLEVB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