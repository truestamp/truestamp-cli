import finished from "./end_of_stream.ts";
import Readable from "./readable.ts";
import { destroyer } from "./destroy.ts";
const kLastResolve = Symbol("lastResolve");
const kLastReject = Symbol("lastReject");
const kError = Symbol("error");
const kEnded = Symbol("ended");
const kLastPromise = Symbol("lastPromise");
const kHandlePromise = Symbol("handlePromise");
const kStream = Symbol("stream");
function initIteratorSymbols(o, symbols) {
    const properties = {};
    for (const sym in symbols) {
        properties[sym] = {
            configurable: false,
            enumerable: false,
            writable: true,
        };
    }
    Object.defineProperties(o, properties);
}
function createIterResult(value, done) {
    return { value, done };
}
function readAndResolve(iter) {
    const resolve = iter[kLastResolve];
    if (resolve !== null) {
        const data = iter[kStream].read();
        if (data !== null) {
            iter[kLastPromise] = null;
            iter[kLastResolve] = null;
            iter[kLastReject] = null;
            resolve(createIterResult(data, false));
        }
    }
}
function onReadable(iter) {
    queueMicrotask(() => readAndResolve(iter));
}
function wrapForNext(lastPromise, iter) {
    return (resolve, reject) => {
        lastPromise.then(() => {
            if (iter[kEnded]) {
                resolve(createIterResult(undefined, true));
                return;
            }
            iter[kHandlePromise](resolve, reject);
        }, reject);
    };
}
function finish(self, err) {
    return new Promise((resolve, reject) => {
        const stream = self[kStream];
        finished(stream, (err) => {
            if (err && err.code !== "ERR_STREAM_PREMATURE_CLOSE") {
                reject(err);
            }
            else {
                resolve(createIterResult(undefined, true));
            }
        });
        destroyer(stream, err);
    });
}
const AsyncIteratorPrototype = Object.getPrototypeOf(Object.getPrototypeOf(async function* () { }).prototype);
export class ReadableStreamAsyncIterator {
    [kEnded];
    [kError] = null;
    [kHandlePromise] = (resolve, reject) => {
        const data = this[kStream].read();
        if (data) {
            this[kLastPromise] = null;
            this[kLastResolve] = null;
            this[kLastReject] = null;
            resolve(createIterResult(data, false));
        }
        else {
            this[kLastResolve] = resolve;
            this[kLastReject] = reject;
        }
    };
    [kLastPromise];
    [kLastReject] = null;
    [kLastResolve] = null;
    [kStream];
    [Symbol.asyncIterator] = AsyncIteratorPrototype[Symbol.asyncIterator];
    constructor(stream) {
        this[kEnded] = stream.readableEnded || stream._readableState.endEmitted;
        this[kStream] = stream;
        initIteratorSymbols(this, [
            kEnded,
            kError,
            kHandlePromise,
            kLastPromise,
            kLastReject,
            kLastResolve,
            kStream,
        ]);
    }
    get stream() {
        return this[kStream];
    }
    next() {
        const error = this[kError];
        if (error !== null) {
            return Promise.reject(error);
        }
        if (this[kEnded]) {
            return Promise.resolve(createIterResult(undefined, true));
        }
        if (this[kStream].destroyed) {
            return new Promise((resolve, reject) => {
                if (this[kError]) {
                    reject(this[kError]);
                }
                else if (this[kEnded]) {
                    resolve(createIterResult(undefined, true));
                }
                else {
                    finished(this[kStream], (err) => {
                        if (err && err.code !== "ERR_STREAM_PREMATURE_CLOSE") {
                            reject(err);
                        }
                        else {
                            resolve(createIterResult(undefined, true));
                        }
                    });
                }
            });
        }
        const lastPromise = this[kLastPromise];
        let promise;
        if (lastPromise) {
            promise = new Promise(wrapForNext(lastPromise, this));
        }
        else {
            const data = this[kStream].read();
            if (data !== null) {
                return Promise.resolve(createIterResult(data, false));
            }
            promise = new Promise(this[kHandlePromise]);
        }
        this[kLastPromise] = promise;
        return promise;
    }
    return() {
        return finish(this);
    }
    throw(err) {
        return finish(this, err);
    }
}
const createReadableStreamAsyncIterator = (stream) => {
    if (typeof stream.read !== "function") {
        const src = stream;
        stream = new Readable({ objectMode: true }).wrap(src);
        finished(stream, (err) => destroyer(src, err));
    }
    const iterator = new ReadableStreamAsyncIterator(stream);
    iterator[kLastPromise] = null;
    finished(stream, { writable: false }, (err) => {
        if (err && err.code !== "ERR_STREAM_PREMATURE_CLOSE") {
            const reject = iterator[kLastReject];
            if (reject !== null) {
                iterator[kLastPromise] = null;
                iterator[kLastResolve] = null;
                iterator[kLastReject] = null;
                reject(err);
            }
            iterator[kError] = err;
            return;
        }
        const resolve = iterator[kLastResolve];
        if (resolve !== null) {
            iterator[kLastPromise] = null;
            iterator[kLastResolve] = null;
            iterator[kLastReject] = null;
            resolve(createIterResult(undefined, true));
        }
        iterator[kEnded] = true;
    });
    stream.on("readable", onReadable.bind(null, iterator));
    return iterator;
};
export default createReadableStreamAsyncIterator;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYXN5bmNfaXRlcmF0b3IuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJhc3luY19pdGVyYXRvci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFFQSxPQUFPLFFBQVEsTUFBTSxvQkFBb0IsQ0FBQztBQUMxQyxPQUFPLFFBQVEsTUFBTSxlQUFlLENBQUM7QUFFckMsT0FBTyxFQUFFLFNBQVMsRUFBRSxNQUFNLGNBQWMsQ0FBQztBQUV6QyxNQUFNLFlBQVksR0FBRyxNQUFNLENBQUMsYUFBYSxDQUFDLENBQUM7QUFDM0MsTUFBTSxXQUFXLEdBQUcsTUFBTSxDQUFDLFlBQVksQ0FBQyxDQUFDO0FBQ3pDLE1BQU0sTUFBTSxHQUFHLE1BQU0sQ0FBQyxPQUFPLENBQUMsQ0FBQztBQUMvQixNQUFNLE1BQU0sR0FBRyxNQUFNLENBQUMsT0FBTyxDQUFDLENBQUM7QUFDL0IsTUFBTSxZQUFZLEdBQUcsTUFBTSxDQUFDLGFBQWEsQ0FBQyxDQUFDO0FBQzNDLE1BQU0sY0FBYyxHQUFHLE1BQU0sQ0FBQyxlQUFlLENBQUMsQ0FBQztBQUMvQyxNQUFNLE9BQU8sR0FBRyxNQUFNLENBQUMsUUFBUSxDQUFDLENBQUM7QUFTakMsU0FBUyxtQkFBbUIsQ0FDMUIsQ0FBOEIsRUFDOUIsT0FBaUI7SUFFakIsTUFBTSxVQUFVLEdBQTBCLEVBQUUsQ0FBQztJQUM3QyxLQUFLLE1BQU0sR0FBRyxJQUFJLE9BQU8sRUFBRTtRQUN6QixVQUFVLENBQUMsR0FBRyxDQUFDLEdBQUc7WUFDaEIsWUFBWSxFQUFFLEtBQUs7WUFDbkIsVUFBVSxFQUFFLEtBQUs7WUFDakIsUUFBUSxFQUFFLElBQUk7U0FDZixDQUFDO0tBQ0g7SUFDRCxNQUFNLENBQUMsZ0JBQWdCLENBQUMsQ0FBQyxFQUFFLFVBQVUsQ0FBQyxDQUFDO0FBQ3pDLENBQUM7QUFFRCxTQUFTLGdCQUFnQixDQUN2QixLQUFtQixFQUNuQixJQUFhO0lBRWIsT0FBTyxFQUFFLEtBQUssRUFBRSxJQUFJLEVBQUUsQ0FBQztB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