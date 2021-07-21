import { captureRejectionSymbol } from "../events.ts";
import Readable, { ReadableState } from "./readable.ts";
import Stream from "./stream.ts";
import Writable, { WritableState } from "./writable.ts";
import { Buffer } from "../buffer.ts";
import { ERR_STREAM_ALREADY_FINISHED, ERR_STREAM_DESTROYED, ERR_UNKNOWN_ENCODING, } from "../_errors.ts";
import createReadableStreamAsyncIterator from "./async_iterator.ts";
import { computeNewHighWaterMark, emitReadable, fromList, howMuchToRead, nReadingNextTick, updateReadableListening, } from "./readable_internal.ts";
import { kOnFinished } from "./writable_internal.ts";
import { endDuplex, finishMaybe, onwrite, readableAddChunk, } from "./duplex_internal.ts";
export { errorOrDestroy } from "./duplex_internal.ts";
class Duplex extends Stream {
    allowHalfOpen = true;
    _final;
    _readableState;
    _writableState;
    _writev;
    constructor(options) {
        super();
        if (options) {
            if (options.allowHalfOpen === false) {
                this.allowHalfOpen = false;
            }
            if (typeof options.destroy === "function") {
                this._destroy = options.destroy;
            }
            if (typeof options.final === "function") {
                this._final = options.final;
            }
            if (typeof options.read === "function") {
                this._read = options.read;
            }
            if (options.readable === false) {
                this.readable = false;
            }
            if (options.writable === false) {
                this.writable = false;
            }
            if (typeof options.write === "function") {
                this._write = options.write;
            }
            if (typeof options.writev === "function") {
                this._writev = options.writev;
            }
        }
        const readableOptions = {
            autoDestroy: options?.autoDestroy,
            defaultEncoding: options?.defaultEncoding,
            destroy: options?.destroy,
            emitClose: options?.emitClose,
            encoding: options?.encoding,
            highWaterMark: options?.highWaterMark ?? options?.readableHighWaterMark,
            objectMode: options?.objectMode ?? options?.readableObjectMode,
            read: options?.read,
        };
        const writableOptions = {
            autoDestroy: options?.autoDestroy,
            decodeStrings: options?.decodeStrings,
            defaultEncoding: options?.defaultEncoding,
            destroy: options?.destroy,
            emitClose: options?.emitClose,
            final: options?.final,
            highWaterMark: options?.highWaterMark ?? options?.writableHighWaterMark,
            objectMode: options?.objectMode ?? options?.writableObjectMode,
            write: options?.write,
            writev: options?.writev,
        };
        this._readableState = new ReadableState(readableOptions);
        this._writableState = new WritableState(writableOptions, this);
        this._writableState.onwrite = onwrite.bind(undefined, this);
    }
    [captureRejectionSymbol](err) {
        this.destroy(err);
    }
    [Symbol.asyncIterator]() {
        return createReadableStreamAsyncIterator(this);
    }
    _destroy(error, callback) {
        callback(error);
    }
    _read = Readable.prototype._read;
    _undestroy = Readable.prototype._undestroy;
    destroy(err, cb) {
        const r = this._readableState;
        const w = this._writableState;
        if (w.destroyed || r.destroyed) {
            if (typeof cb === "function") {
                cb();
            }
            return this;
        }
        if (err) {
            err.stack;
            if (!w.errored) {
                w.errored = err;
            }
            if (!r.errored) {
                r.errored = err;
            }
        }
        w.destroyed = true;
        r.destroyed = true;
        this._destroy(err || null, (err) => {
            if (err) {
                err.stack;
                if (!w.errored) {
                    w.errored = err;
                }
                if (!r.errored) {
                    r.errored = err;
                }
            }
            w.closed = true;
            r.closed = true;
            if (typeof cb === "function") {
                cb(err);
            }
            if (err) {
                queueMicrotask(() => {
                    const r = this._readableState;
                    const w = this._writableState;
                    if (!w.errorEmitted && !r.errorEmitted) {
                        w.errorEmitted = true;
                        r.errorEmitted = true;
                        this.emit("error", err);
                    }
                    r.closeEmitted = true;
                    if (w.emitClose || r.emitClose) {
                        this.emit("close");
                    }
                });
            }
            else {
                queueMicrotask(() => {
                    const r = this._readableState;
                    const w = this._writableState;
                    r.closeEmitted = true;
                    if (w.emitClose || r.emitClose) {
                        this.emit("close");
                    }
                });
            }
        });
        return this;
    }
    isPaused = Readable.prototype.isPaused;
    off = this.removeListener;
    on(ev, fn) {
        const res = super.on.call(this, ev, fn);
        const state = this._readableState;
        if (ev === "data") {
            state.readableListening = this.listenerCount("readable") > 0;
            if (state.flowing !== false) {
                this.resume();
            }
        }
        else if (ev === "readable") {
            if (!state.endEmitted && !state.readableListening) {
                state.readableListening = state.needReadable = true;
                state.flowing = false;
                state.emittedReadable = false;
                if (state.length) {
                    emitReadable(this);
                }
                else if (!state.reading) {
                    queueMicrotask(() => nReadingNextTick(this));
                }
            }
        }
        return res;
    }
    pause = Readable.prototype.pause;
    pipe = Readable.prototype.pipe;
    push(chunk, encoding) {
        return readableAddChunk(this, chunk, encoding, false);
    }
    read(n) {
        if (n === undefined) {
            n = NaN;
        }
        const state = this._readableState;
        const nOrig = n;
        if (n > state.highWaterMark) {
            state.highWaterMark = computeNewHighWaterMark(n);
        }
        if (n !== 0) {
            state.emittedReadable = false;
        }
        if (n === 0 &&
            state.needReadable &&
            ((state.highWaterMark !== 0
                ? state.length >= state.highWaterMark
                : state.length > 0) ||
                state.ended)) {
            if (state.length === 0 && state.ended) {
                endDuplex(this);
            }
            else {
                emitReadable(this);
            }
            return null;
        }
        n = howMuchToRead(n, state);
        if (n === 0 && state.ended) {
            if (state.length === 0) {
                endDuplex(this);
            }
            return null;
        }
        let doRead = state.needReadable;
        if (state.length === 0 || state.length - n < state.highWaterMark) {
            doRead = true;
        }
        if (state.ended || state.reading || state.destroyed || state.errored ||
            !state.constructed) {
            doRead = false;
        }
        else if (doRead) {
            state.reading = true;
            state.sync = true;
            if (state.length === 0) {
                state.needReadable = true;
            }
            this._read();
            state.sync = false;
            if (!state.reading) {
                n = howMuchToRead(nOrig, state);
            }
        }
        let ret;
        if (n > 0) {
            ret = fromList(n, state);
        }
        else {
            ret = null;
        }
        if (ret === null) {
            state.needReadable = state.length <= state.highWaterMark;
            n = 0;
        }
        else {
            state.length -= n;
            if (state.multiAwaitDrain) {
                state.awaitDrainWriters.clear();
            }
            else {
                state.awaitDrainWriters = null;
            }
        }
        if (state.length === 0) {
            if (!state.ended) {
                state.needReadable = true;
            }
            if (nOrig !== n && state.ended) {
                endDuplex(this);
            }
        }
        if (ret !== null) {
            this.emit("data", ret);
        }
        return ret;
    }
    removeAllListeners(ev) {
        const res = super.removeAllListeners(ev);
        if (ev === "readable" || ev === undefined) {
            queueMicrotask(() => updateReadableListening(this));
        }
        return res;
    }
    removeListener(ev, fn) {
        const res = super.removeListener.call(this, ev, fn);
        if (ev === "readable") {
            queueMicrotask(() => updateReadableListening(this));
        }
        return res;
    }
    resume = Readable.prototype.resume;
    setEncoding = Readable.prototype.setEncoding;
    unshift(chunk, encoding) {
        return readableAddChunk(this, chunk, encoding, true);
    }
    unpipe = Readable.prototype.unpipe;
    wrap = Readable.prototype.wrap;
    get readable() {
        return this._readableState?.readable &&
            !this._readableState?.destroyed &&
            !this._readableState?.errorEmitted &&
            !this._readableState?.endEmitted;
    }
    set readable(val) {
        if (this._readableState) {
            this._readableState.readable = val;
        }
    }
    get readableHighWaterMark() {
        return this._readableState.highWaterMark;
    }
    get readableBuffer() {
        return this._readableState && this._readableState.buffer;
    }
    get readableFlowing() {
        return this._readableState.flowing;
    }
    set readableFlowing(state) {
        if (this._readableState) {
            this._readableState.flowing = state;
        }
    }
    get readableLength() {
        return this._readableState.length;
    }
    get readableObjectMode() {
        return this._readableState ? this._readableState.objectMode : false;
    }
    get readableEncoding() {
        return this._readableState ? this._readableState.encoding : null;
    }
    get readableEnded() {
        return this._readableState ? this._readableState.endEmitted : false;
    }
    _write = Writable.prototype._write;
    write = Writable.prototype.write;
    cork = Writable.prototype.cork;
    uncork = Writable.prototype.uncork;
    setDefaultEncoding(encoding) {
        if (typeof encoding === "string") {
            encoding = encoding.toLowerCase();
        }
        if (!Buffer.isEncoding(encoding)) {
            throw new ERR_UNKNOWN_ENCODING(encoding);
        }
        this._writableState.defaultEncoding = encoding;
        return this;
    }
    end(x, y, z) {
        const state = this._writableState;
        let chunk;
        let encoding;
        let cb;
        if (typeof x === "function") {
            chunk = null;
            encoding = null;
            cb = x;
        }
        else if (typeof y === "function") {
            chunk = x;
            encoding = null;
            cb = y;
        }
        else {
            chunk = x;
            encoding = y;
            cb = z;
        }
        if (chunk !== null && chunk !== undefined) {
            this.write(chunk, encoding);
        }
        if (state.corked) {
            state.corked = 1;
            this.uncork();
        }
        let err;
        if (!state.errored && !state.ending) {
            state.ending = true;
            finishMaybe(this, state, true);
            state.ended = true;
        }
        else if (state.finished) {
            err = new ERR_STREAM_ALREADY_FINISHED("end");
        }
        else if (state.destroyed) {
            err = new ERR_STREAM_DESTROYED("end");
        }
        if (typeof cb === "function") {
            if (err || state.finished) {
                queueMicrotask(() => {
                    cb(err);
                });
            }
            else {
                state[kOnFinished].push(cb);
            }
        }
        return this;
    }
    get destroyed() {
        if (this._readableState === undefined ||
            this._writableState === undefined) {
            return false;
        }
        return this._readableState.destroyed && this._writableState.destroyed;
    }
    set destroyed(value) {
        if (this._readableState && this._writableState) {
            this._readableState.destroyed = value;
            this._writableState.destroyed = value;
        }
    }
    get writable() {
        const w = this._writableState;
        return !w.destroyed && !w.errored && !w.ending && !w.ended;
    }
    set writable(val) {
        if (this._writableState) {
            this._writableState.writable = !!val;
        }
    }
    get writableFinished() {
        return this._writableState ? this._writableState.finished : false;
    }
    get writableObjectMode() {
        return this._writableState ? this._writableState.objectMode : false;
    }
    get writableBuffer() {
        return this._writableState && this._writableState.getBuffer();
    }
    get writableEnded() {
        return this._writableState ? this._writableState.ending : false;
    }
    get writableHighWaterMark() {
        return this._writableState && this._writableState.highWaterMark;
    }
    get writableCorked() {
        return this._writableState ? this._writableState.corked : 0;
    }
    get writableLength() {
        return this._writableState && this._writableState.length;
    }
}
export default Duplex;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZHVwbGV4LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZHVwbGV4LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUNBLE9BQU8sRUFBRSxzQkFBc0IsRUFBRSxNQUFNLGNBQWMsQ0FBQztBQUN0RCxPQUFPLFFBQVEsRUFBRSxFQUFFLGFBQWEsRUFBRSxNQUFNLGVBQWUsQ0FBQztBQUN4RCxPQUFPLE1BQU0sTUFBTSxhQUFhLENBQUM7QUFDakMsT0FBTyxRQUFRLEVBQUUsRUFBRSxhQUFhLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFDeEQsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLGNBQWMsQ0FBQztBQUN0QyxPQUFPLEVBQ0wsMkJBQTJCLEVBQzNCLG9CQUFvQixFQUNwQixvQkFBb0IsR0FDckIsTUFBTSxlQUFlLENBQUM7QUFFdkIsT0FBTyxpQ0FBaUMsTUFBTSxxQkFBcUIsQ0FBQztBQUVwRSxPQUFPLEVBRUwsdUJBQXVCLEVBQ3ZCLFlBQVksRUFDWixRQUFRLEVBQ1IsYUFBYSxFQUNiLGdCQUFnQixFQUNoQix1QkFBdUIsR0FDeEIsTUFBTSx3QkFBd0IsQ0FBQztBQUNoQyxPQUFPLEVBQUUsV0FBVyxFQUFVLE1BQU0sd0JBQXdCLENBQUM7QUFDN0QsT0FBTyxFQUNMLFNBQVMsRUFDVCxXQUFXLEVBQ1gsT0FBTyxFQUNQLGdCQUFnQixHQUNqQixNQUFNLHNCQUFzQixDQUFDO0FBQzlCLE9BQU8sRUFBRSxjQUFjLEVBQUUsTUFBTSxzQkFBc0IsQ0FBQztBQXlDdEQsTUFBTSxNQUFPLFNBQVEsTUFBTTtJQUN6QixhQUFhLEdBQUcsSUFBSSxDQUFDO0lBQ3JCLE1BQU0sQ0FFSTtJQUNWLGNBQWMsQ0FBZ0I7SUFDOUIsY0FBYyxDQUFnQjtJQUM5QixPQUFPLENBQWlCO0lBRXhCLFlBQVksT0FBdUI7UUFDakMsS0FBSyxFQUFFLENBQUM7UUFFUixJQUFJLE9BQU8sRUFBRTtZQUNYLElBQUksT0FBTyxDQUFDLGFBQWEsS0FBSyxLQUFLLEVBQUU7Z0JB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