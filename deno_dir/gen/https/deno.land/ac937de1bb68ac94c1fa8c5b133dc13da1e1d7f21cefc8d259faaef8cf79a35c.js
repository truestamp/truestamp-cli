import { captureRejectionSymbol } from "../events.ts";
import Stream from "./stream.ts";
import BufferList from "./buffer_list.ts";
import { ERR_INVALID_OPT_VALUE, ERR_METHOD_NOT_IMPLEMENTED, } from "../_errors.ts";
import { StringDecoder } from "../string_decoder.ts";
import createReadableStreamAsyncIterator from "./async_iterator.ts";
import streamFrom from "./from.ts";
import { kDestroy, kPaused } from "./symbols.ts";
import { _destroy, computeNewHighWaterMark, emitReadable, endReadable, errorOrDestroy, fromList, howMuchToRead, nReadingNextTick, pipeOnDrain, prependListener, readableAddChunk, resume, updateReadableListening, } from "./readable_internal.ts";
import { errorOrDestroy as errorOrDestroyWritable } from "./writable_internal.ts";
import Duplex, { errorOrDestroy as errorOrDestroyDuplex } from "./duplex.ts";
export class ReadableState {
    [kPaused] = null;
    awaitDrainWriters = null;
    buffer = new BufferList();
    closed = false;
    closeEmitted = false;
    constructed;
    decoder = null;
    destroyed = false;
    emittedReadable = false;
    encoding = null;
    ended = false;
    endEmitted = false;
    errored = null;
    errorEmitted = false;
    flowing = null;
    highWaterMark;
    length = 0;
    multiAwaitDrain = false;
    needReadable = false;
    objectMode;
    pipes = [];
    readable = true;
    readableListening = false;
    reading = false;
    readingMore = false;
    resumeScheduled = false;
    sync = true;
    emitClose;
    autoDestroy;
    defaultEncoding;
    constructor(options) {
        this.objectMode = !!options?.objectMode;
        this.highWaterMark = options?.highWaterMark ??
            (this.objectMode ? 16 : 16 * 1024);
        if (Number.isInteger(this.highWaterMark) && this.highWaterMark >= 0) {
            this.highWaterMark = Math.floor(this.highWaterMark);
        }
        else {
            throw new ERR_INVALID_OPT_VALUE("highWaterMark", this.highWaterMark);
        }
        this.emitClose = options?.emitClose ?? true;
        this.autoDestroy = options?.autoDestroy ?? true;
        this.defaultEncoding = options?.defaultEncoding || "utf8";
        if (options?.encoding) {
            this.decoder = new StringDecoder(options.encoding);
            this.encoding = options.encoding;
        }
        this.constructed = true;
    }
}
class Readable extends Stream {
    _readableState;
    constructor(options) {
        super();
        if (options) {
            if (typeof options.read === "function") {
                this._read = options.read;
            }
            if (typeof options.destroy === "function") {
                this._destroy = options.destroy;
            }
        }
        this._readableState = new ReadableState(options);
    }
    static from(iterable, opts) {
        return streamFrom(iterable, opts);
    }
    static ReadableState = ReadableState;
    static _fromList = fromList;
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
                endReadable(this);
            }
            else {
                emitReadable(this);
            }
            return null;
        }
        n = howMuchToRead(n, state);
        if (n === 0 && state.ended) {
            if (state.length === 0) {
                endReadable(this);
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
                endReadable(this);
            }
        }
        if (ret !== null) {
            this.emit("data", ret);
        }
        return ret;
    }
    _read(_size) {
        throw new ERR_METHOD_NOT_IMPLEMENTED("_read()");
    }
    pipe(dest, pipeOpts) {
        const src = this;
        const state = this._readableState;
        if (state.pipes.length === 1) {
            if (!state.multiAwaitDrain) {
                state.multiAwaitDrain = true;
                state.awaitDrainWriters = new Set(state.awaitDrainWriters ? [state.awaitDrainWriters] : []);
            }
        }
        state.pipes.push(dest);
        const doEnd = (!pipeOpts || pipeOpts.end !== false);
        const endFn = doEnd ? onend : unpipe;
        if (state.endEmitted) {
            queueMicrotask(endFn);
        }
        else {
            this.once("end", endFn);
        }
        dest.on("unpipe", onunpipe);
        function onunpipe(readable, unpipeInfo) {
            if (readable === src) {
                if (unpipeInfo && unpipeInfo.hasUnpiped === false) {
                    unpipeInfo.hasUnpiped = true;
                    cleanup();
                }
            }
        }
        function onend() {
            dest.end();
        }
        let ondrain;
        let cleanedUp = false;
        function cleanup() {
            dest.removeListener("close", onclose);
            dest.removeListener("finish", onfinish);
            if (ondrain) {
                dest.removeListener("drain", ondrain);
            }
            dest.removeListener("error", onerror);
            dest.removeListener("unpipe", onunpipe);
            src.removeListener("end", onend);
            src.removeListener("end", unpipe);
            src.removeListener("data", ondata);
            cleanedUp = true;
            if (ondrain && state.awaitDrainWriters &&
                (!dest._writableState || dest._writableState.needDrain)) {
                ondrain();
            }
        }
        this.on("data", ondata);
        function ondata(chunk) {
            const ret = dest.write(chunk);
            if (ret === false) {
                if (!cleanedUp) {
                    if (state.pipes.length === 1 && state.pipes[0] === dest) {
                        state.awaitDrainWriters = dest;
                        state.multiAwaitDrain = false;
                    }
                    else if (state.pipes.length > 1 && state.pipes.includes(dest)) {
                        state.awaitDrainWriters.add(dest);
                    }
                    src.pause();
                }
                if (!ondrain) {
                    ondrain = pipeOnDrain(src, dest);
                    dest.on("drain", ondrain);
                }
            }
        }
        function onerror(er) {
            unpipe();
            dest.removeListener("error", onerror);
            if (dest.listenerCount("error") === 0) {
                const s = dest._writableState || dest._readableState;
                if (s && !s.errorEmitted) {
                    if (dest instanceof Duplex) {
                        errorOrDestroyDuplex(dest, er);
                    }
                    else {
                        errorOrDestroyWritable(dest, er);
                    }
                }
                else {
                    dest.emit("error", er);
                }
            }
        }
        prependListener(dest, "error", onerror);
        function onclose() {
            dest.removeListener("finish", onfinish);
            unpipe();
        }
        dest.once("close", onclose);
        function onfinish() {
            dest.removeListener("close", onclose);
            unpipe();
        }
        dest.once("finish", onfinish);
        function unpipe() {
            src.unpipe(dest);
        }
        dest.emit("pipe", this);
        if (!state.flowing) {
            this.resume();
        }
        return dest;
    }
    isPaused() {
        return this._readableState[kPaused] === true ||
            this._readableState.flowing === false;
    }
    setEncoding(enc) {
        const decoder = new StringDecoder(enc);
        this._readableState.decoder = decoder;
        this._readableState.encoding = this._readableState.decoder
            .encoding;
        const buffer = this._readableState.buffer;
        let content = "";
        for (const data of buffer) {
            content += decoder.write(data);
        }
        buffer.clear();
        if (content !== "") {
            buffer.push(content);
        }
        this._readableState.length = content.length;
        return this;
    }
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
    removeListener(ev, fn) {
        const res = super.removeListener.call(this, ev, fn);
        if (ev === "readable") {
            queueMicrotask(() => updateReadableListening(this));
        }
        return res;
    }
    off = this.removeListener;
    destroy(err, cb) {
        const r = this._readableState;
        if (r.destroyed) {
            if (typeof cb === "function") {
                cb();
            }
            return this;
        }
        if (err) {
            err.stack;
            if (!r.errored) {
                r.errored = err;
            }
        }
        r.destroyed = true;
        if (!r.constructed) {
            this.once(kDestroy, (er) => {
                _destroy(this, err || er, cb);
            });
        }
        else {
            _destroy(this, err, cb);
        }
        return this;
    }
    _undestroy() {
        const r = this._readableState;
        r.constructed = true;
        r.closed = false;
        r.closeEmitted = false;
        r.destroyed = false;
        r.errored = null;
        r.errorEmitted = false;
        r.reading = false;
        r.ended = false;
        r.endEmitted = false;
    }
    _destroy(error, callback) {
        callback(error);
    }
    [captureRejectionSymbol](err) {
        this.destroy(err);
    }
    push(chunk, encoding) {
        return readableAddChunk(this, chunk, encoding, false);
    }
    unshift(chunk, encoding) {
        return readableAddChunk(this, chunk, encoding, true);
    }
    unpipe(dest) {
        const state = this._readableState;
        const unpipeInfo = { hasUnpiped: false };
        if (state.pipes.length === 0) {
            return this;
        }
        if (!dest) {
            const dests = state.pipes;
            state.pipes = [];
            this.pause();
            for (const dest of dests) {
                dest.emit("unpipe", this, { hasUnpiped: false });
            }
            return this;
        }
        const index = state.pipes.indexOf(dest);
        if (index === -1) {
            return this;
        }
        state.pipes.splice(index, 1);
        if (state.pipes.length === 0) {
            this.pause();
        }
        dest.emit("unpipe", this, unpipeInfo);
        return this;
    }
    removeAllListeners(ev) {
        const res = super.removeAllListeners(ev);
        if (ev === "readable" || ev === undefined) {
            queueMicrotask(() => updateReadableListening(this));
        }
        return res;
    }
    resume() {
        const state = this._readableState;
        if (!state.flowing) {
            state.flowing = !state.readableListening;
            resume(this, state);
        }
        state[kPaused] = false;
        return this;
    }
    pause() {
        if (this._readableState.flowing !== false) {
            this._readableState.flowing = false;
            this.emit("pause");
        }
        this._readableState[kPaused] = true;
        return this;
    }
    wrap(stream) {
        const state = this._readableState;
        let paused = false;
        stream.on("end", () => {
            if (state.decoder && !state.ended) {
                const chunk = state.decoder.end();
                if (chunk && chunk.length) {
                    this.push(chunk);
                }
            }
            this.push(null);
        });
        stream.on("data", (chunk) => {
            if (state.decoder) {
                chunk = state.decoder.write(chunk);
            }
            if (state.objectMode && (chunk === null || chunk === undefined)) {
                return;
            }
            else if (!state.objectMode && (!chunk || !chunk.length)) {
                return;
            }
            const ret = this.push(chunk);
            if (!ret) {
                paused = true;
                stream.pause();
            }
        });
        for (const i in stream) {
            if (this[i] === undefined && typeof stream[i] === "function") {
                this[i] = function methodWrap(method) {
                    return function methodWrapReturnFunction() {
                        return stream[method].apply(stream);
                    };
                }(i);
            }
        }
        stream.on("error", (err) => {
            errorOrDestroy(this, err);
        });
        stream.on("close", () => {
            this.emit("close");
        });
        stream.on("destroy", () => {
            this.emit("destroy");
        });
        stream.on("pause", () => {
            this.emit("pause");
        });
        stream.on("resume", () => {
            this.emit("resume");
        });
        this._read = () => {
            if (paused) {
                paused = false;
                stream.resume();
            }
        };
        return this;
    }
    [Symbol.asyncIterator]() {
        return createReadableStreamAsyncIterator(this);
    }
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
    get destroyed() {
        if (this._readableState === undefined) {
            return false;
        }
        return this._readableState.destroyed;
    }
    set destroyed(value) {
        if (!this._readableState) {
            return;
        }
        this._readableState.destroyed = value;
    }
    get readableEnded() {
        return this._readableState ? this._readableState.endEmitted : false;
    }
}
Object.defineProperties(Readable, {
    _readableState: { enumerable: false },
    destroyed: { enumerable: false },
    readableBuffer: { enumerable: false },
    readableEncoding: { enumerable: false },
    readableEnded: { enumerable: false },
    readableFlowing: { enumerable: false },
    readableHighWaterMark: { enumerable: false },
    readableLength: { enumerable: false },
    readableObjectMode: { enumerable: false },
});
export default Readable;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicmVhZGFibGUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJyZWFkYWJsZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsc0JBQXNCLEVBQUUsTUFBTSxjQUFjLENBQUM7QUFDdEQsT0FBTyxNQUFNLE1BQU0sYUFBYSxDQUFDO0FBRWpDLE9BQU8sVUFBVSxNQUFNLGtCQUFrQixDQUFDO0FBQzFDLE9BQU8sRUFDTCxxQkFBcUIsRUFDckIsMEJBQTBCLEdBQzNCLE1BQU0sZUFBZSxDQUFDO0FBRXZCLE9BQU8sRUFBRSxhQUFhLEVBQUUsTUFBTSxzQkFBc0IsQ0FBQztBQUNyRCxPQUFPLGlDQUFpQyxNQUFNLHFCQUFxQixDQUFDO0FBQ3BFLE9BQU8sVUFBVSxNQUFNLFdBQVcsQ0FBQztBQUNuQyxPQUFPLEVBQUUsUUFBUSxFQUFFLE9BQU8sRUFBRSxNQUFNLGNBQWMsQ0FBQztBQUNqRCxPQUFPLEVBQ0wsUUFBUSxFQUNSLHVCQUF1QixFQUN2QixZQUFZLEVBQ1osV0FBVyxFQUNYLGNBQWMsRUFDZCxRQUFRLEVBQ1IsYUFBYSxFQUNiLGdCQUFnQixFQUNoQixXQUFXLEVBQ1gsZUFBZSxFQUNmLGdCQUFnQixFQUNoQixNQUFNLEVBQ04sdUJBQXVCLEdBQ3hCLE1BQU0sd0JBQXdCLENBQUM7QUFFaEMsT0FBTyxFQUFFLGNBQWMsSUFBSSxzQkFBc0IsRUFBRSxNQUFNLHdCQUF3QixDQUFDO0FBQ2xGLE9BQU8sTUFBTSxFQUFFLEVBQUUsY0FBYyxJQUFJLG9CQUFvQixFQUFFLE1BQU0sYUFBYSxDQUFDO0FBaUI3RSxNQUFNLE9BQU8sYUFBYTtJQUN4QixDQUFDLE9BQU8sQ0FBQyxHQUFtQixJQUFJLENBQUM7SUFDakMsaUJBQWlCLEdBQXNELElBQUksQ0FBQztJQUM1RSxNQUFNLEdBQUcsSUFBSSxVQUFVLEVBQUUsQ0FBQztJQUMxQixNQUFNLEdBQUcsS0FBSyxDQUFDO0lBQ2YsWUFBWSxHQUFHLEtBQUssQ0FBQztJQUNyQixXQUFX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