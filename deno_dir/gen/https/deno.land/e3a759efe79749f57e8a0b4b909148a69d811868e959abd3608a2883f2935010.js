import { notImplemented } from "./_utils.ts";
import EventEmitter from "./events.ts";
import { fromFileUrl } from "../path/mod.ts";
const notImplementedEvents = [
    "beforeExit",
    "disconnect",
    "message",
    "multipleResolves",
    "rejectionHandled",
    "SIGBREAK",
    "SIGBUS",
    "SIGFPE",
    "SIGHUP",
    "SIGILL",
    "SIGINT",
    "SIGSEGV",
    "SIGTERM",
    "SIGWINCH",
    "uncaughtException",
    "uncaughtExceptionMonitor",
    "unhandledRejection",
];
export const arch = Deno.build.arch;
const argv = ["", "", ...Deno.args];
Object.defineProperty(argv, "0", {
    get() {
        return Deno.execPath();
    },
});
Object.defineProperty(argv, "1", {
    get() {
        return fromFileUrl(Deno.mainModule);
    },
});
export const chdir = Deno.chdir;
export const cwd = Deno.cwd;
const _env = {};
Object.defineProperty(_env, Deno.customInspect, {
    enumerable: false,
    configurable: true,
    get: function () {
        return Deno.env.toObject();
    },
});
export const env = new Proxy(_env, {
    get(target, prop) {
        if (prop === Deno.customInspect) {
            return target[Deno.customInspect];
        }
        return Deno.env.get(String(prop));
    },
    ownKeys() {
        return Reflect.ownKeys(Deno.env.toObject());
    },
    getOwnPropertyDescriptor() {
        return { enumerable: true, configurable: true };
    },
    set(_target, prop, value) {
        Deno.env.set(String(prop), String(value));
        return value;
    },
});
export const exit = Deno.exit;
export function nextTick(cb, ...args) {
    if (args) {
        queueMicrotask(() => cb.call(this, ...args));
    }
    else {
        queueMicrotask(cb);
    }
}
export const pid = Deno.pid;
export const platform = Deno.build.os === "windows" ? "win32" : Deno.build.os;
export const version = `v${Deno.version.deno}`;
export const versions = {
    node: Deno.version.deno,
    ...Deno.version,
};
class Process extends EventEmitter {
    constructor() {
        super();
        window.addEventListener("unload", () => {
            super.emit("exit", 0);
        });
    }
    arch = arch;
    argv = argv;
    chdir = chdir;
    cwd = cwd;
    exit = exit;
    env = env;
    nextTick = nextTick;
    on(event, listener) {
        if (notImplementedEvents.includes(event)) {
            notImplemented();
        }
        super.on(event, listener);
        return this;
    }
    pid = pid;
    platform = platform;
    removeAllListeners(_event) {
        notImplemented();
    }
    removeListener(event, listener) {
        if (notImplementedEvents.includes(event)) {
            notImplemented();
        }
        super.removeListener("exit", listener);
        return this;
    }
    hrtime(time) {
        const milli = performance.now();
        const sec = Math.floor(milli / 1000);
        const nano = Math.floor(milli * 1_000_000 - sec * 1_000_000_000);
        if (!time) {
            return [sec, nano];
        }
        const [prevSec, prevNano] = time;
        return [sec - prevSec, nano - prevNano];
    }
    get stderr() {
        return {
            fd: Deno.stderr.rid,
            get isTTY() {
                return Deno.isatty(this.fd);
            },
            pipe(_destination, _options) {
                notImplemented();
            },
            write(_chunk, _callback) {
                notImplemented();
            },
            on(_event, _callback) {
                notImplemented();
            },
        };
    }
    get stdin() {
        return {
            fd: Deno.stdin.rid,
            get isTTY() {
                return Deno.isatty(this.fd);
            },
            read(_size) {
                notImplemented();
            },
            on(_event, _callback) {
                notImplemented();
            },
        };
    }
    get stdout() {
        return {
            fd: Deno.stdout.rid,
            get isTTY() {
                return Deno.isatty(this.fd);
            },
            pipe(_destination, _options) {
                notImplemented();
            },
            write(_chunk, _callback) {
                notImplemented();
            },
            on(_event, _callback) {
                notImplemented();
            },
        };
    }
    version = version;
    versions = versions;
}
const process = new Process();
Object.defineProperty(process, Symbol.toStringTag, {
    enumerable: false,
    writable: true,
    configurable: false,
    value: "process",
});
export const removeListener = process.removeListener;
export const removeAllListeners = process.removeAllListeners;
export const stderr = process.stderr;
export const stdin = process.stdin;
export const stdout = process.stdout;
export default process;
export { process };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicHJvY2Vzcy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbInByb2Nlc3MudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBRUEsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUM3QyxPQUFPLFlBQVksTUFBTSxhQUFhLENBQUM7QUFDdkMsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBRTdDLE1BQU0sb0JBQW9CLEdBQUc7SUFDM0IsWUFBWTtJQUNaLFlBQVk7SUFDWixTQUFTO0lBQ1Qsa0JBQWtCO0lBQ2xCLGtCQUFrQjtJQUNsQixVQUFVO0lBQ1YsUUFBUTtJQUNSLFFBQVE7SUFDUixRQUFRO0lBQ1IsUUFBUTtJQUNSLFFBQVE7SUFDUixTQUFTO0lBQ1QsU0FBUztJQUNULFVBQVU7SUFDVixtQkFBbUI7SUFDbkIsMEJBQTBCO0lBQzFCLG9CQUFvQjtDQUNyQixDQUFDO0FBR0YsTUFBTSxDQUFDLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDO0FBSXBDLE1BQU0sSUFBSSxHQUFHLENBQUMsRUFBRSxFQUFFLEVBQUUsRUFBRSxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztBQUVwQyxNQUFNLENBQUMsY0FBYyxDQUFDLElBQUksRUFBRSxHQUFHLEVBQUU7SUFDL0IsR0FBRztRQUNELE9BQU8sSUFBSSxDQUFDLFFBQVEsRUFBRSxDQUFDO0lBQ3pCLENBQUM7Q0FDRixDQUFDLENBQUM7QUFFSCxNQUFNLENBQUMsY0FBYyxDQUFDLElBQUksRUFBRSxHQUFHLEVBQUU7SUFDL0IsR0FBRztRQUNELE9BQU8sV0FBVyxDQUFDLElBQUksQ0FBQyxVQUFVLENBQUMsQ0FBQztJQUN0QyxDQUFDO0NBQ0YsQ0FBQyxDQUFDO0FBR0gsTUFBTSxDQUFDLE1BQU0sS0FBSyxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUM7QUFHaEMsTUFBTSxDQUFDLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FB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