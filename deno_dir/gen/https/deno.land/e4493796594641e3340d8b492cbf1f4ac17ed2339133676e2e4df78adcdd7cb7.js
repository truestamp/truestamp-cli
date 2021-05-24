import { notImplemented } from "./_utils.ts";
export const arch = Deno.build.arch;
export const chdir = Deno.chdir;
export const cwd = Deno.cwd;
export const exit = Deno.exit;
export const pid = Deno.pid;
export const platform = Deno.build.os === "windows" ? "win32" : Deno.build.os;
export const version = `v${Deno.version.deno}`;
export const versions = {
    node: Deno.version.deno,
    ...Deno.version,
};
export function nextTick(cb, ...args) {
    if (args) {
        queueMicrotask(() => cb.call(this, ...args));
    }
    else {
        queueMicrotask(cb);
    }
}
export const process = {
    arch,
    chdir,
    cwd,
    exit,
    pid,
    platform,
    version,
    versions,
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
    },
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
    },
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
    },
    on(_event, _callback) {
        notImplemented();
    },
    get argv() {
        return [Deno.execPath(), ...Deno.args];
    },
    get env() {
        return Deno.env.toObject();
    },
    nextTick,
};
export const argv = new Proxy(process.argv, {});
export const env = new Proxy(process.env, {});
export default process;
Object.defineProperty(process, Symbol.toStringTag, {
    enumerable: false,
    writable: true,
    configurable: false,
    value: "process",
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicHJvY2Vzcy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbInByb2Nlc3MudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLGNBQWMsRUFBRSxNQUFNLGFBQWEsQ0FBQztBQUc3QyxNQUFNLENBQUMsTUFBTSxJQUFJLEdBQUcsSUFBSSxDQUFDLEtBQUssQ0FBQyxJQUFJLENBQUM7QUFHcEMsTUFBTSxDQUFDLE1BQU0sS0FBSyxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUM7QUFHaEMsTUFBTSxDQUFDLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUM7QUFHNUIsTUFBTSxDQUFDLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUM7QUFHOUIsTUFBTSxDQUFDLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxHQUFHLENBQUM7QUFHNUIsTUFBTSxDQUFDLE1BQU0sUUFBUSxHQUFHLElBQUksQ0FBQyxLQUFLLENBQUMsRUFBRSxLQUFLLFNBQVMsQ0FBQyxDQUFDLENBQUMsT0FBTyxDQUFDLENBQUMsQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLEVBQUUsQ0FBQztBQUc5RSxNQUFNLENBQUMsTUFBTSxPQUFPLEdBQUcsSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRSxDQUFDO0FBRy9DLE1BQU0sQ0FBQyxNQUFNLFFBQVEsR0FBRztJQUN0QixJQUFJLEVBQUUsSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJO0lBQ3ZCLEdBQUcsSUFBSSxDQUFDLE9BQU87Q0FDaEIsQ0FBQztBQVNGLE1BQU0sVUFBVSxRQUFRLENBRXRCLEVBQXdCLEVBQ3hCLEdBQUcsSUFBTztJQUVWLElBQUksSUFBSSxFQUFFO1FBQ1IsY0FBYyxDQUFDLEdBQUcsRUFBRSxDQUFDLEVBQUUsQ0FBQyxJQUFJLENBQUMsSUFBSSxFQUFFLEdBQUcsSUFBSSxDQUFDLENBQUMsQ0FBQztLQUM5QztTQUFNO1FBQ0wsY0FBYyxDQUFDLEVBQUUsQ0FB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