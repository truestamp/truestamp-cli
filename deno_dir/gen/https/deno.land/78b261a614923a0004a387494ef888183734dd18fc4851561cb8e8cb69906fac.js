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
    "warning",
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
    configurable: false,
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicHJvY2Vzcy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbInByb2Nlc3MudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBRUEsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUM3QyxPQUFPLFlBQVksTUFBTSxhQUFhLENBQUM7QUFDdkMsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBRTdDLE1BQU0sb0JBQW9CLEdBQUc7SUFDM0IsWUFBWTtJQUNaLFlBQVk7SUFDWixTQUFTO0lBQ1Qsa0JBQWtCO0lBQ2xCLGtCQUFrQjtJQUNsQixVQUFVO0lBQ1YsUUFBUTtJQUNSLFFBQVE7SUFDUixRQUFRO0lBQ1IsUUFBUTtJQUNSLFFBQVE7SUFDUixTQUFTO0lBQ1QsU0FBUztJQUNULFVBQVU7SUFDVixtQkFBbUI7SUFDbkIsMEJBQTBCO0lBQzFCLG9CQUFvQjtJQUNwQixTQUFTO0NBQ1YsQ0FBQztBQUdGLE1BQU0sQ0FBQyxNQUFNLElBQUksR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQztBQUlwQyxNQUFNLElBQUksR0FBRyxDQUFDLEVBQUUsRUFBRSxFQUFFLEVBQUUsR0FBRyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQUM7QUFFcEMsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLEVBQUUsR0FBRyxFQUFFO0lBQy9CLEdBQUc7UUFDRCxPQUFPLElBQUksQ0FBQyxRQUFRLEVBQUUsQ0FBQztJQUN6QixDQUFDO0NBQ0YsQ0FBQyxDQUFDO0FBRUgsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLEVBQUUsR0FBRyxFQUFFO0lBQy9CLEdBQUc7UUFDRCxPQUFPLFdBQVcsQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLENBQUM7SUFDdEMsQ0FBQztDQUNGLENBQUMsQ0FBQztBQUdILE1BQU0sQ0FBQyxNQUFNLEtBQUssR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDO0FBR2hDLE1BQU0sQ0FBQyxNQUFNLEdBQUcsR0FB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