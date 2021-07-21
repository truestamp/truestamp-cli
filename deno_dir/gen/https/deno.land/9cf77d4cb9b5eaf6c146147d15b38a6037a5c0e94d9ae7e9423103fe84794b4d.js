import { assert } from "../_util/assert.ts";
import { ERR_INVALID_ARG_TYPE, ERR_OUT_OF_RANGE } from "./_errors.ts";
import { inspect } from "./util.ts";
function createIterResult(value, done) {
    return { value, done };
}
export let defaultMaxListeners = 10;
function validateMaxListeners(n, name) {
    if (!Number.isInteger(n) || n < 0) {
        throw new ERR_OUT_OF_RANGE(name, "a non-negative number", inspect(n));
    }
}
export class EventEmitter {
    static captureRejectionSymbol = Symbol.for("nodejs.rejection");
    static errorMonitor = Symbol("events.errorMonitor");
    static get defaultMaxListeners() {
        return defaultMaxListeners;
    }
    static set defaultMaxListeners(value) {
        validateMaxListeners(value, "defaultMaxListeners");
        defaultMaxListeners = value;
    }
    maxListeners;
    _events;
    static _alreadyWarnedEvents;
    constructor() {
        this._events = new Map();
    }
    _addListener(eventName, listener, prepend) {
        this.checkListenerArgument(listener);
        this.emit("newListener", eventName, this.unwrapListener(listener));
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
            const warning = new MaxListenersExceededWarning(this, eventName);
            this.warnIfNeeded(eventName, warning);
        }
        return this;
    }
    addListener(eventName, listener) {
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
        return this.maxListeners == null
            ? EventEmitter.defaultMaxListeners
            : this.maxListeners;
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
        if (!target._events?.has(eventName)) {
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
            unwrappedListeners[i] = this.unwrapListener(arr[i]);
        }
        return unwrappedListeners;
    }
    unwrapListener(listener) {
        return listener["listener"] ?? listener;
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
        this.checkListenerArgument(listener);
        const wrapper = function (...args) {
            if (this.isCalled) {
                return;
            }
            this.context.removeListener(this.eventName, this.rawListener);
            this.isCalled = true;
            return this.listener.apply(this.context, args);
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
                const listeners = this._events.get(eventName).slice().reverse();
                for (const listener of listeners) {
                    this.removeListener(eventName, this.unwrapListener(listener));
                }
            }
        }
        else {
            const eventList = this.eventNames();
            eventList.forEach((value) => {
                this.removeAllListeners(value);
            });
        }
        return this;
    }
    removeListener(eventName, listener) {
        this.checkListenerArgument(listener);
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
            validateMaxListeners(n, "n");
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
    checkListenerArgument(listener) {
        if (typeof listener !== "function") {
            throw new ERR_INVALID_ARG_TYPE("listener", "function", listener);
        }
    }
    warnIfNeeded(eventName, warning) {
        EventEmitter._alreadyWarnedEvents ||= new Set();
        if (EventEmitter._alreadyWarnedEvents.has(eventName)) {
            return;
        }
        EventEmitter._alreadyWarnedEvents.add(eventName);
        console.warn(warning);
        const maybeProcess = globalThis.process;
        if (maybeProcess instanceof EventEmitter) {
            maybeProcess.emit("warning", warning);
        }
    }
}
EventEmitter.prototype.addListener = EventEmitter.prototype.on;
class MaxListenersExceededWarning extends Error {
    emitter;
    type;
    count;
    constructor(emitter, type) {
        const listenerCount = emitter.listenerCount(type);
        const message = "Possible EventEmitter memory leak detected. " +
            `${listenerCount} ${type == null ? "null" : type.toString()} listeners added to [${emitter.constructor.name}]. ` +
            " Use emitter.setMaxListeners() to increase limit";
        super(message);
        this.emitter = emitter;
        this.type = type;
        this.count = listenerCount;
        this.name = "MaxListenersExceededWarning";
    }
}
export default Object.assign(EventEmitter, { EventEmitter });
export const captureRejectionSymbol = EventEmitter.captureRejectionSymbol;
export const errorMonitor = EventEmitter.errorMonitor;
export const listenerCount = EventEmitter.listenerCount;
export const on = EventEmitter.on;
export const once = EventEmitter.once;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZXZlbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZXZlbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQXVCQSxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDNUMsT0FBTyxFQUFFLG9CQUFvQixFQUFFLGdCQUFnQixFQUFFLE1BQU0sY0FBYyxDQUFDO0FBQ3RFLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFVcEMsU0FBUyxnQkFBZ0IsQ0FBQyxLQUFVLEVBQUUsSUFBYTtJQUNqRCxPQUFPLEVBQUUsS0FBSyxFQUFFLElBQUksRUFBRSxDQUFDO0FBQ3pCLENBQUM7QUFZRCxNQUFNLENBQUMsSUFBSSxtQkFBbUIsR0FBRyxFQUFFLENBQUM7QUFDcEMsU0FBUyxvQkFBb0IsQ0FBQyxDQUFTLEVBQUUsSUFBWTtJQUNuRCxJQUFJLENBQUMsTUFBTSxDQUFDLFNBQVMsQ0FBQyxDQUFDLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxFQUFFO1FBQ2pDLE1BQU0sSUFBSSxnQkFBZ0IsQ0FBQyxJQUFJLEVBQUUsdUJBQXVCLEVBQUUsT0FBTyxDQUFDLENBQUMsQ0FBQyxDQUFDLENBQUM7S0FDdkU7QUFDSCxDQUFDO0FBS0QsTUFBTSxPQUFPLFlBQVk7SUFDaEIsTUFBTSxDQUFDLHNCQUFzQixHQUFHLE1BQU0sQ0FBQyxHQUFHLENBQUMsa0JBQWtCLENBQUMsQ0FBQztJQUMvRCxNQUFNLENBQUMsWUFBWSxHQUFHLE1BQU0sQ0FBQyxxQkFBcUIsQ0FBQyxDQUFDO0lBQ3BELE1BQU0sS0FBSyxtQkFBbUI7UUFDbkMsT0FBTyxtQkFBbUIsQ0FBQztJQUM3QixDQUFDO0lBQ00sTUFBTSxLQUFLLG1CQUFtQixDQUFDLEtBQWE7UUFDakQsb0JBQW9CLENBQUMsS0FBSyxFQUFFLHFCQUFxQixDQUFDLENBQUM7UUFDbkQsbUJBQW1CLEdBQUcsS0FBSyxDQUFDO0lBQzlCLENBQUM7SUFFTyxZQUFZLENBQXFCO0lB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