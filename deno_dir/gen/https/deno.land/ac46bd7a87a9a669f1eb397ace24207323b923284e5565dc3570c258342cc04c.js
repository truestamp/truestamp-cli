import { validateIntegerRange } from "./_utils.ts";
import { assert } from "../_util/assert.ts";
function createIterResult(value, done) {
    return { value, done };
}
export let defaultMaxListeners = 10;
export class EventEmitter {
    static captureRejectionSymbol = Symbol.for("nodejs.rejection");
    static errorMonitor = Symbol("events.errorMonitor");
    static get defaultMaxListeners() {
        return defaultMaxListeners;
    }
    static set defaultMaxListeners(value) {
        defaultMaxListeners = value;
    }
    maxListeners;
    _events;
    constructor() {
        this._events = new Map();
    }
    _addListener(eventName, listener, prepend) {
        this.emit("newListener", eventName, listener);
        if (this._events.has(eventName)) {
            const listeners = this._events.get(eventName);
            if (prepend) {
                listeners.unshift(listener);
            }
            else {
                listeners.push(listener);
            }
        }
        else {
            this._events.set(eventName, [listener]);
        }
        const max = this.getMaxListeners();
        if (max > 0 && this.listenerCount(eventName) > max) {
            const warning = new Error(`Possible EventEmitter memory leak detected.
         ${this.listenerCount(eventName)} ${eventName.toString()} listeners.
         Use emitter.setMaxListeners() to increase limit`);
            warning.name = "MaxListenersExceededWarning";
            console.warn(warning);
        }
        return this;
    }
    addListener(eventName, listener) {
        return this._addListener(eventName, listener, false);
    }
    emit(eventName, ...args) {
        if (this._events.has(eventName)) {
            if (eventName === "error" &&
                this._events.get(EventEmitter.errorMonitor)) {
                this.emit(EventEmitter.errorMonitor, ...args);
            }
            const listeners = this._events.get(eventName).slice();
            for (const listener of listeners) {
                try {
                    listener.apply(this, args);
                }
                catch (err) {
                    this.emit("error", err);
                }
            }
            return true;
        }
        else if (eventName === "error") {
            if (this._events.get(EventEmitter.errorMonitor)) {
                this.emit(EventEmitter.errorMonitor, ...args);
            }
            const errMsg = args.length > 0 ? args[0] : Error("Unhandled error.");
            throw errMsg;
        }
        return false;
    }
    eventNames() {
        return Array.from(this._events.keys());
    }
    getMaxListeners() {
        return this.maxListeners || EventEmitter.defaultMaxListeners;
    }
    listenerCount(eventName) {
        if (this._events.has(eventName)) {
            return this._events.get(eventName).length;
        }
        else {
            return 0;
        }
    }
    static listenerCount(emitter, eventName) {
        return emitter.listenerCount(eventName);
    }
    _listeners(target, eventName, unwrap) {
        if (!target._events.has(eventName)) {
            return [];
        }
        const eventListeners = target._events.get(eventName);
        return unwrap
            ? this.unwrapListeners(eventListeners)
            : eventListeners.slice(0);
    }
    unwrapListeners(arr) {
        const unwrappedListeners = new Array(arr.length);
        for (let i = 0; i < arr.length; i++) {
            unwrappedListeners[i] = arr[i]["listener"] || arr[i];
        }
        return unwrappedListeners;
    }
    listeners(eventName) {
        return this._listeners(this, eventName, true);
    }
    rawListeners(eventName) {
        return this._listeners(this, eventName, false);
    }
    off(eventName, listener) {
        return this.removeListener(eventName, listener);
    }
    on(eventName, listener) {
        return this._addListener(eventName, listener, false);
    }
    once(eventName, listener) {
        const wrapped = this.onceWrap(eventName, listener);
        this.on(eventName, wrapped);
        return this;
    }
    onceWrap(eventName, listener) {
        const wrapper = function (...args) {
            this.context.removeListener(this.eventName, this.rawListener);
            this.listener.apply(this.context, args);
        };
        const wrapperContext = {
            eventName: eventName,
            listener: listener,
            rawListener: wrapper,
            context: this,
        };
        const wrapped = wrapper.bind(wrapperContext);
        wrapperContext.rawListener = wrapped;
        wrapped.listener = listener;
        return wrapped;
    }
    prependListener(eventName, listener) {
        return this._addListener(eventName, listener, true);
    }
    prependOnceListener(eventName, listener) {
        const wrapped = this.onceWrap(eventName, listener);
        this.prependListener(eventName, wrapped);
        return this;
    }
    removeAllListeners(eventName) {
        if (this._events === undefined) {
            return this;
        }
        if (eventName) {
            if (this._events.has(eventName)) {
                const listeners = this._events.get(eventName).slice();
                this._events.delete(eventName);
                for (const listener of listeners) {
                    this.emit("removeListener", eventName, listener);
                }
            }
        }
        else {
            const eventList = this.eventNames();
            eventList.map((value) => {
                this.removeAllListeners(value);
            });
        }
        return this;
    }
    removeListener(eventName, listener) {
        if (this._events.has(eventName)) {
            const arr = this._events.get(eventName);
            assert(arr);
            let listenerIndex = -1;
            for (let i = arr.length - 1; i >= 0; i--) {
                if (arr[i] == listener ||
                    (arr[i] && arr[i]["listener"] == listener)) {
                    listenerIndex = i;
                    break;
                }
            }
            if (listenerIndex >= 0) {
                arr.splice(listenerIndex, 1);
                this.emit("removeListener", eventName, listener);
                if (arr.length === 0) {
                    this._events.delete(eventName);
                }
            }
        }
        return this;
    }
    setMaxListeners(n) {
        if (n !== Infinity) {
            if (n === 0) {
                n = Infinity;
            }
            else {
                validateIntegerRange(n, "maxListeners", 0);
            }
        }
        this.maxListeners = n;
        return this;
    }
    static once(emitter, name) {
        return new Promise((resolve, reject) => {
            if (emitter instanceof EventTarget) {
                emitter.addEventListener(name, (...args) => {
                    resolve(args);
                }, { once: true, passive: false, capture: false });
                return;
            }
            else if (emitter instanceof EventEmitter) {
                const eventListener = (...args) => {
                    if (errorListener !== undefined) {
                        emitter.removeListener("error", errorListener);
                    }
                    resolve(args);
                };
                let errorListener;
                if (name !== "error") {
                    errorListener = (err) => {
                        emitter.removeListener(name, eventListener);
                        reject(err);
                    };
                    emitter.once("error", errorListener);
                }
                emitter.once(name, eventListener);
                return;
            }
        });
    }
    static on(emitter, event) {
        const unconsumedEventValues = [];
        const unconsumedPromises = [];
        let error = null;
        let finished = false;
        const iterator = {
            next() {
                const value = unconsumedEventValues.shift();
                if (value) {
                    return Promise.resolve(createIterResult(value, false));
                }
                if (error) {
                    const p = Promise.reject(error);
                    error = null;
                    return p;
                }
                if (finished) {
                    return Promise.resolve(createIterResult(undefined, true));
                }
                return new Promise(function (resolve, reject) {
                    unconsumedPromises.push({ resolve, reject });
                });
            },
            return() {
                emitter.removeListener(event, eventHandler);
                emitter.removeListener("error", errorHandler);
                finished = true;
                for (const promise of unconsumedPromises) {
                    promise.resolve(createIterResult(undefined, true));
                }
                return Promise.resolve(createIterResult(undefined, true));
            },
            throw(err) {
                error = err;
                emitter.removeListener(event, eventHandler);
                emitter.removeListener("error", errorHandler);
            },
            [Symbol.asyncIterator]() {
                return this;
            },
        };
        emitter.on(event, eventHandler);
        emitter.on("error", errorHandler);
        return iterator;
        function eventHandler(...args) {
            const promise = unconsumedPromises.shift();
            if (promise) {
                promise.resolve(createIterResult(args, false));
            }
            else {
                unconsumedEventValues.push(args);
            }
        }
        function errorHandler(err) {
            finished = true;
            const toError = unconsumedPromises.shift();
            if (toError) {
                toError.reject(err);
            }
            else {
                error = err;
            }
            iterator.return();
        }
    }
}
export default Object.assign(EventEmitter, { EventEmitter });
export const captureRejectionSymbol = EventEmitter.captureRejectionSymbol;
export const errorMonitor = EventEmitter.errorMonitor;
export const listenerCount = EventEmitter.listenerCount;
export const on = EventEmitter.on;
export const once = EventEmitter.once;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXZlbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZXZlbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQXVCQSxPQUFPLEVBQUUsb0JBQW9CLEVBQUUsTUFBTSxhQUFhLENBQUM7QUFDbkQsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBVTVDLFNBQVMsZ0JBQWdCLENBQUMsS0FBVSxFQUFFLElBQWE7SUFDakQsT0FBTyxFQUFFLEtBQUssRUFBRSxJQUFJLEVBQUUsQ0FBQztBQUN6QixDQUFDO0FBWUQsTUFBTSxDQUFDLElBQUksbUJBQW1CLEdBQUcsRUFBRSxDQUFDO0FBS3BDLE1BQU0sT0FBTyxZQUFZO0lBQ2hCLE1BQU0sQ0FBQyxzQkFBc0IsR0FBRyxNQUFNLENBQUMsR0FBRyxDQUFDLGtCQUFrQixDQUFDLENBQUM7SUFDL0QsTUFBTSxDQUFDLFlBQVksR0FBRyxNQUFNLENBQUMscUJBQXFCLENBQUMsQ0FBQztJQUNwRCxNQUFNLEtBQUssbUJBQW1CO1FBQ25DLE9BQU8sbUJBQW1CLENBQUM7SUFDN0IsQ0FBQztJQUNNLE1BQU0sS0FBSyxtQkFBbUIsQ0FBQyxLQUFhO1FBQ2pELG1CQUFtQixHQUFHLEtBQUssQ0FBQztJQUM5QixDQUFDO0lBRU8sWUFBWSxDQUFxQjtJQUNqQyxPQUFPLENBR2I7SUFFRjtRQUNFLElBQUksQ0FBQyxPQUFPLEdBQUcsSUFBSSxHQUFHLEVBQUUsQ0FBQztJQUMzQixDQUFDO0lBRU8sWUFBWSxDQUNsQixTQUEwQixFQUMxQixRQUEyQyxFQUMzQyxPQUFnQjtRQUVoQixJQUFJLENBQUMsSUFBSSxDQUFDLGFBQWEsRUFBRSxTQUFTLEVBQUUsUUFBUSxDQUFDLENBQUM7UUFDOUMsSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxTQUFTLENBQUMsRUFBRTtZQUMvQixNQUFNLFNBQVMsR0FBRyxJQUFJLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxTQUFTLENBRTNDLENB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