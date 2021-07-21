import { once } from "../_utils.ts";
import { destroyer as implDestroyer } from "./destroy.ts";
import eos from "./end_of_stream.ts";
import createReadableStreamAsyncIterator from "./async_iterator.ts";
import * as events from "../events.ts";
import PassThrough from "./passthrough.ts";
import { ERR_INVALID_ARG_TYPE, ERR_INVALID_CALLBACK, ERR_INVALID_RETURN_VALUE, ERR_MISSING_ARGS, ERR_STREAM_DESTROYED, } from "../_errors.ts";
function destroyer(stream, reading, writing, callback) {
    callback = once(callback);
    let finished = false;
    stream.on("close", () => {
        finished = true;
    });
    eos(stream, { readable: reading, writable: writing }, (err) => {
        finished = !err;
        const rState = stream?._readableState;
        if (err &&
            err.code === "ERR_STREAM_PREMATURE_CLOSE" &&
            reading &&
            (rState?.ended && !rState?.errored && !rState?.errorEmitted)) {
            stream
                .once("end", callback)
                .once("error", callback);
        }
        else {
            callback(err);
        }
    });
    return (err) => {
        if (finished)
            return;
        finished = true;
        implDestroyer(stream, err);
        callback(err || new ERR_STREAM_DESTROYED("pipe"));
    };
}
function popCallback(streams) {
    if (typeof streams[streams.length - 1] !== "function") {
        throw new ERR_INVALID_CALLBACK(streams[streams.length - 1]);
    }
    return streams.pop();
}
function isReadable(obj) {
    return !!(obj && typeof obj.pipe === "function");
}
function isWritable(obj) {
    return !!(obj && typeof obj.write === "function");
}
function isStream(obj) {
    return isReadable(obj) || isWritable(obj);
}
function isIterable(obj, isAsync) {
    if (!obj)
        return false;
    if (isAsync === true)
        return typeof obj[Symbol.asyncIterator] === "function";
    if (isAsync === false)
        return typeof obj[Symbol.iterator] === "function";
    return typeof obj[Symbol.asyncIterator] === "function" ||
        typeof obj[Symbol.iterator] === "function";
}
function makeAsyncIterable(val) {
    if (isIterable(val)) {
        return val;
    }
    else if (isReadable(val)) {
        return fromReadable(val);
    }
    throw new ERR_INVALID_ARG_TYPE("val", ["Readable", "Iterable", "AsyncIterable"], val);
}
async function* fromReadable(val) {
    yield* createReadableStreamAsyncIterator(val);
}
async function pump(iterable, writable, finish) {
    let error;
    try {
        for await (const chunk of iterable) {
            if (!writable.write(chunk)) {
                if (writable.destroyed)
                    return;
                await events.once(writable, "drain");
            }
        }
        writable.end();
    }
    catch (err) {
        error = err;
    }
    finally {
        finish(error);
    }
}
export default function pipeline(...args) {
    const callback = once(popCallback(args));
    let streams;
    if (args.length > 1) {
        streams = args;
    }
    else {
        throw new ERR_MISSING_ARGS("streams");
    }
    let error;
    let value;
    const destroys = [];
    let finishCount = 0;
    function finish(err) {
        const final = --finishCount === 0;
        if (err && (!error || error.code === "ERR_STREAM_PREMATURE_CLOSE")) {
            error = err;
        }
        if (!error && !final) {
            return;
        }
        while (destroys.length) {
            destroys.shift()(error);
        }
        if (final) {
            callback(error, value);
        }
    }
    let ret;
    for (let i = 0; i < streams.length; i++) {
        const stream = streams[i];
        const reading = i < streams.length - 1;
        const writing = i > 0;
        if (isStream(stream)) {
            finishCount++;
            destroys.push(destroyer(stream, reading, writing, finish));
        }
        if (i === 0) {
            if (typeof stream === "function") {
                ret = stream();
                if (!isIterable(ret)) {
                    throw new ERR_INVALID_RETURN_VALUE("Iterable, AsyncIterable or Stream", "source", ret);
                }
            }
            else if (isIterable(stream) || isReadable(stream)) {
                ret = stream;
            }
            else {
                throw new ERR_INVALID_ARG_TYPE("source", ["Stream", "Iterable", "AsyncIterable", "Function"], stream);
            }
        }
        else if (typeof stream === "function") {
            ret = makeAsyncIterable(ret);
            ret = stream(ret);
            if (reading) {
                if (!isIterable(ret, true)) {
                    throw new ERR_INVALID_RETURN_VALUE("AsyncIterable", `transform[${i - 1}]`, ret);
                }
            }
            else {
                const pt = new PassThrough({
                    objectMode: true,
                });
                if (ret instanceof Promise) {
                    ret
                        .then((val) => {
                        value = val;
                        pt.end(val);
                    }, (err) => {
                        pt.destroy(err);
                    });
                }
                else if (isIterable(ret, true)) {
                    finishCount++;
                    pump(ret, pt, finish);
                }
                else {
                    throw new ERR_INVALID_RETURN_VALUE("AsyncIterable or Promise", "destination", ret);
                }
                ret = pt;
                finishCount++;
                destroys.push(destroyer(ret, false, true, finish));
            }
        }
        else if (isStream(stream)) {
            if (isReadable(ret)) {
                ret.pipe(stream);
            }
            else {
                ret = makeAsyncIterable(ret);
                finishCount++;
                pump(ret, stream, finish);
            }
            ret = stream;
        }
        else {
            const name = reading ? `transform[${i - 1}]` : "destination";
            throw new ERR_INVALID_ARG_TYPE(name, ["Stream", "Function"], ret);
        }
    }
    return ret;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicGlwZWxpbmUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJwaXBlbGluZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsSUFBSSxFQUFFLE1BQU0sY0FBYyxDQUFDO0FBQ3BDLE9BQU8sRUFBRSxTQUFTLElBQUksYUFBYSxFQUFFLE1BQU0sY0FBYyxDQUFDO0FBQzFELE9BQU8sR0FBRyxNQUFNLG9CQUFvQixDQUFDO0FBQ3JDLE9BQU8saUNBQWlDLE1BQU0scUJBQXFCLENBQUM7QUFDcEUsT0FBTyxLQUFLLE1BQU0sTUFBTSxjQUFjLENBQUM7QUFDdkMsT0FBTyxXQUFXLE1BQU0sa0JBQWtCLENBQUM7QUFDM0MsT0FBTyxFQUNMLG9CQUFvQixFQUNwQixvQkFBb0IsRUFDcEIsd0JBQXdCLEVBQ3hCLGdCQUFnQixFQUNoQixvQkFBb0IsR0FFckIsTUFBTSxlQUFlLENBQUM7QUFxQ3ZCLFNBQVMsU0FBUyxDQUNoQixNQUFlLEVBQ2YsT0FBZ0IsRUFDaEIsT0FBZ0IsRUFDaEIsUUFBcUI7SUFFckIsUUFBUSxHQUFHLElBQUksQ0FBQyxRQUFRLENBQUMsQ0FBQztJQUUxQixJQUFJLFFBQVEsR0FBRyxLQUFLLENBQUM7SUFDckIsTUFBTSxDQUFDLEVBQUUsQ0FBQyxPQUFPLEVBQUUsR0FBRyxFQUFFO1FBQ3RCLFFBQVEsR0FBRyxJQUFJLENBQUM7SUFDbEIsQ0FBQyxDQUFDLENBQUM7SUFFSCxHQUFHLENBQUMsTUFBTSxFQUFFLEVBQUUsUUFBUSxFQUFFLE9BQU8sRUFBRSxRQUFRLEVBQUUsT0FBTyxFQUFFLEVBQUUsQ0FBQyxHQUFHLEVBQUUsRUFBRTtRQUM1RCxRQUFRLEdBQUcsQ0FBQyxHQUFHLENBQUM7UUFHaEIsTUFBTSxNQUFNLEdBQUksTUFBYyxFQUFFLGNBQWMsQ0FBQztRQUMvQyxJQUNFLEdBQUc7WUFDSCxHQUFHLENBQUMsSUFBSSxLQUFLLDRCQUE0QjtZQUN6QyxPQUFPO1lBQ1AsQ0FBQyxNQUFNLEVBQUUsS0FB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