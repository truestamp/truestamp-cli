import { addChunk, maybeReadMore, onEofChunk } from "./readable_internal.ts";
import { afterWrite, afterWriteTick, clearBuffer, errorBuffer, kOnFinished, needFinish, prefinish, } from "./writable_internal.ts";
import { Buffer } from "../buffer.ts";
import { ERR_MULTIPLE_CALLBACK, ERR_STREAM_PUSH_AFTER_EOF, ERR_STREAM_UNSHIFT_AFTER_END_EVENT, } from "../_errors.ts";
export function endDuplex(stream) {
    const state = stream._readableState;
    if (!state.endEmitted) {
        state.ended = true;
        queueMicrotask(() => endReadableNT(state, stream));
    }
}
function endReadableNT(state, stream) {
    if (!state.errorEmitted && !state.closeEmitted &&
        !state.endEmitted && state.length === 0) {
        state.endEmitted = true;
        stream.emit("end");
        if (stream.writable && stream.allowHalfOpen === false) {
            queueMicrotask(() => endWritableNT(state, stream));
        }
        else if (state.autoDestroy) {
            const wState = stream._writableState;
            const autoDestroy = !wState || (wState.autoDestroy &&
                (wState.finished || wState.writable === false));
            if (autoDestroy) {
                stream.destroy();
            }
        }
    }
}
function endWritableNT(_state, stream) {
    const writable = stream.writable &&
        !stream.writableEnded &&
        !stream.destroyed;
    if (writable) {
        stream.end();
    }
}
export function errorOrDestroy(stream, err, sync = false) {
    const r = stream._readableState;
    const w = stream._writableState;
    if (w.destroyed || r.destroyed) {
        return this;
    }
    if (r.autoDestroy || w.autoDestroy) {
        stream.destroy(err);
    }
    else if (err) {
        err.stack;
        if (w && !w.errored) {
            w.errored = err;
        }
        if (r && !r.errored) {
            r.errored = err;
        }
        if (sync) {
            queueMicrotask(() => {
                if (w.errorEmitted || r.errorEmitted) {
                    return;
                }
                w.errorEmitted = true;
                r.errorEmitted = true;
                stream.emit("error", err);
            });
        }
        else {
            if (w.errorEmitted || r.errorEmitted) {
                return;
            }
            w.errorEmitted = true;
            r.errorEmitted = true;
            stream.emit("error", err);
        }
    }
}
function finish(stream, state) {
    state.pendingcb--;
    if (state.errorEmitted || state.closeEmitted) {
        return;
    }
    state.finished = true;
    for (const callback of state[kOnFinished].splice(0)) {
        callback();
    }
    stream.emit("finish");
    if (state.autoDestroy) {
        stream.destroy();
    }
}
export function finishMaybe(stream, state, sync) {
    if (needFinish(state)) {
        prefinish(stream, state);
        if (state.pendingcb === 0 && needFinish(state)) {
            state.pendingcb++;
            if (sync) {
                queueMicrotask(() => finish(stream, state));
            }
            else {
                finish(stream, state);
            }
        }
    }
}
export function onwrite(stream, er) {
    const state = stream._writableState;
    const sync = state.sync;
    const cb = state.writecb;
    if (typeof cb !== "function") {
        errorOrDestroy(stream, new ERR_MULTIPLE_CALLBACK());
        return;
    }
    state.writing = false;
    state.writecb = null;
    state.length -= state.writelen;
    state.writelen = 0;
    if (er) {
        er.stack;
        if (!state.errored) {
            state.errored = er;
        }
        if (stream._readableState && !stream._readableState.errored) {
            stream._readableState.errored = er;
        }
        if (sync) {
            queueMicrotask(() => onwriteError(stream, state, er, cb));
        }
        else {
            onwriteError(stream, state, er, cb);
        }
    }
    else {
        if (state.buffered.length > state.bufferedIndex) {
            clearBuffer(stream, state);
        }
        if (sync) {
            if (state.afterWriteTickInfo !== null &&
                state.afterWriteTickInfo.cb === cb) {
                state.afterWriteTickInfo.count++;
            }
            else {
                state.afterWriteTickInfo = {
                    count: 1,
                    cb: cb,
                    stream: stream,
                    state,
                };
                queueMicrotask(() => afterWriteTick(state.afterWriteTickInfo));
            }
        }
        else {
            afterWrite(stream, state, 1, cb);
        }
    }
}
function onwriteError(stream, state, er, cb) {
    --state.pendingcb;
    cb(er);
    errorBuffer(state);
    errorOrDestroy(stream, er);
}
export function readableAddChunk(stream, chunk, encoding = undefined, addToFront) {
    const state = stream._readableState;
    let usedEncoding = encoding;
    let err;
    if (!state.objectMode) {
        if (typeof chunk === "string") {
            usedEncoding = encoding || state.defaultEncoding;
            if (state.encoding !== usedEncoding) {
                if (addToFront && state.encoding) {
                    chunk = Buffer.from(chunk, usedEncoding).toString(state.encoding);
                }
                else {
                    chunk = Buffer.from(chunk, usedEncoding);
                    usedEncoding = "";
                }
            }
        }
        else if (chunk instanceof Uint8Array) {
            chunk = Buffer.from(chunk);
        }
    }
    if (err) {
        errorOrDestroy(stream, err);
    }
    else if (chunk === null) {
        state.reading = false;
        onEofChunk(stream, state);
    }
    else if (state.objectMode || (chunk.length > 0)) {
        if (addToFront) {
            if (state.endEmitted) {
                errorOrDestroy(stream, new ERR_STREAM_UNSHIFT_AFTER_END_EVENT());
            }
            else {
                addChunk(stream, state, chunk, true);
            }
        }
        else if (state.ended) {
            errorOrDestroy(stream, new ERR_STREAM_PUSH_AFTER_EOF());
        }
        else if (state.destroyed || state.errored) {
            return false;
        }
        else {
            state.reading = false;
            if (state.decoder && !usedEncoding) {
                chunk = state.decoder.write(Buffer.from(chunk));
                if (state.objectMode || chunk.length !== 0) {
                    addChunk(stream, state, chunk, false);
                }
                else {
                    maybeReadMore(stream, state);
                }
            }
            else {
                addChunk(stream, state, chunk, false);
            }
        }
    }
    else if (!addToFront) {
        state.reading = false;
        maybeReadMore(stream, state);
    }
    return !state.ended &&
        (state.length < state.highWaterMark || state.length === 0);
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZHVwbGV4X2ludGVybmFsLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZHVwbGV4X2ludGVybmFsLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxRQUFRLEVBQUUsYUFBYSxFQUFFLFVBQVUsRUFBRSxNQUFNLHdCQUF3QixDQUFDO0FBRzdFLE9BQU8sRUFDTCxVQUFVLEVBRVYsY0FBYyxFQUNkLFdBQVcsRUFDWCxXQUFXLEVBQ1gsV0FBVyxFQUNYLFVBQVUsRUFDVixTQUFTLEdBQ1YsTUFBTSx3QkFBd0IsQ0FBQztBQUNoQyxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRXRDLE9BQU8sRUFDTCxxQkFBcUIsRUFDckIseUJBQXlCLEVBQ3pCLGtDQUFrQyxHQUNuQyxNQUFNLGVBQWUsQ0FBQztBQUV2QixNQUFNLFVBQVUsU0FBUyxDQUFDLE1BQWM7SUFDdEMsTUFBTSxLQUFLLEdBQUcsTUFBTSxDQUFDLGNBQWMsQ0FBQztJQUVwQyxJQUFJLENBQUMsS0FBSyxDQUFDLFVBQVUsRUFBRTtRQUNyQixLQUFLLENBQUMsS0FBSyxHQUFHLElBQUksQ0FBQztRQUNuQixjQUFjLENBQUMsR0FBRyxFQUFFLENBQUMsYUFBYSxDQUFDLEtBQUssRUFBRSxNQUFNLENBQUMsQ0FBQyxDQUFDO0tBQ3BEO0FBQ0gsQ0FBQztBQUVELFNBQVMsYUFBYSxDQUFDLEtBQW9CLEVBQUUsTUFBYztJQUV6RCxJQUNFLENBQUMsS0FBSyxDQUFDLFlBQVksSUFBSSxDQUFDLEtBQUssQ0FBQyxZQUFZO1FBQzFDLENBQUMsS0FBSyxDQUFDLFVBQVUsSUFBSSxLQUFLLENBQUMsTUFBTSxLQUFLLENBQUMsRUFDdkM7UUFDQSxLQUFLLENBQUMsVUFBVSxHQUFHLElBQUksQ0FBQztRQUN4QixNQUFNLENBQUMsSUFBSSxDQUFDLEtBQUssQ0FBQyxDQUFDO1FBRW5CLElBQUksTUFBTSxDQUFDLFFBQVEsSUFBSSxNQUFNLENBQUMsYUFBYSxLQUFL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