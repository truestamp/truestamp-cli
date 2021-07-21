import { unreachable } from "../testing/asserts.ts";
import { inspect } from "./util.ts";
const classRegExp = /^([A-Z][a-z0-9]*)+$/;
const kTypes = [
    "string",
    "function",
    "number",
    "object",
    "Function",
    "Object",
    "boolean",
    "bigint",
    "symbol",
];
export class NodeErrorAbstraction extends Error {
    code;
    constructor(name, code, message) {
        super(message);
        this.code = code;
        this.name = name;
        this.stack = this.stack && `${name} [${this.code}]${this.stack.slice(20)}`;
    }
    toString() {
        return `${this.name} [${this.code}]: ${this.message}`;
    }
}
export class NodeError extends NodeErrorAbstraction {
    constructor(code, message) {
        super(Error.prototype.name, code, message);
    }
}
export class NodeSyntaxError extends NodeErrorAbstraction {
    constructor(code, message) {
        super(SyntaxError.prototype.name, code, message);
        Object.setPrototypeOf(this, SyntaxError.prototype);
    }
}
export class NodeRangeError extends NodeErrorAbstraction {
    constructor(code, message) {
        super(RangeError.prototype.name, code, message);
        Object.setPrototypeOf(this, RangeError.prototype);
    }
}
export class NodeTypeError extends NodeErrorAbstraction {
    constructor(code, message) {
        super(TypeError.prototype.name, code, message);
        Object.setPrototypeOf(this, TypeError.prototype);
    }
}
export class NodeURIError extends NodeErrorAbstraction {
    constructor(code, message) {
        super(URIError.prototype.name, code, message);
        Object.setPrototypeOf(this, URIError.prototype);
    }
}
export class ERR_INVALID_ARG_TYPE extends NodeTypeError {
    constructor(name, expected, actual) {
        expected = Array.isArray(expected) ? expected : [expected];
        let msg = "The ";
        if (name.endsWith(" argument")) {
            msg += `${name} `;
        }
        else {
            const type = name.includes(".") ? "property" : "argument";
            msg += `"${name}" ${type} `;
        }
        msg += "must be ";
        const types = [];
        const instances = [];
        const other = [];
        for (const value of expected) {
            if (kTypes.includes(value)) {
                types.push(value.toLocaleLowerCase());
            }
            else if (classRegExp.test(value)) {
                instances.push(value);
            }
            else {
                other.push(value);
            }
        }
        if (instances.length > 0) {
            const pos = types.indexOf("object");
            if (pos !== -1) {
                types.splice(pos, 1);
                instances.push("Object");
            }
        }
        if (types.length > 0) {
            if (types.length > 2) {
                const last = types.pop();
                msg += `one of type ${types.join(", ")}, or ${last}`;
            }
            else if (types.length === 2) {
                msg += `one of type ${types[0]} or ${types[1]}`;
            }
            else {
                msg += `of type ${types[0]}`;
            }
            if (instances.length > 0 || other.length > 0) {
                msg += " or ";
            }
        }
        if (instances.length > 0) {
            if (instances.length > 2) {
                const last = instances.pop();
                msg += `an instance of ${instances.join(", ")}, or ${last}`;
            }
            else {
                msg += `an instance of ${instances[0]}`;
                if (instances.length === 2) {
                    msg += ` or ${instances[1]}`;
                }
            }
            if (other.length > 0) {
                msg += " or ";
            }
        }
        if (other.length > 0) {
            if (other.length > 2) {
                const last = other.pop();
                msg += `one of ${other.join(", ")}, or ${last}`;
            }
            else if (other.length === 2) {
                msg += `one of ${other[0]} or ${other[1]}`;
            }
            else {
                if (other[0].toLowerCase() !== other[0]) {
                    msg += "an ";
                }
                msg += `${other[0]}`;
            }
        }
        super("ERR_INVALID_ARG_TYPE", `${msg}.${invalidArgTypeHelper(actual)}`);
    }
}
export class ERR_INVALID_ARG_VALUE extends NodeTypeError {
    constructor(name, value, reason) {
        super("ERR_INVALID_ARG_VALUE", `The argument '${name}' ${reason}. Received ${inspect(value)}`);
    }
}
function invalidArgTypeHelper(input) {
    if (input == null) {
        return ` Received ${input}`;
    }
    if (typeof input === "function" && input.name) {
        return ` Received function ${input.name}`;
    }
    if (typeof input === "object") {
        if (input.constructor && input.constructor.name) {
            return ` Received an instance of ${input.constructor.name}`;
        }
        return ` Received ${inspect(input, { depth: -1 })}`;
    }
    let inspected = inspect(input, { colors: false });
    if (inspected.length > 25) {
        inspected = `${inspected.slice(0, 25)}...`;
    }
    return ` Received type ${typeof input} (${inspected})`;
}
export class ERR_OUT_OF_RANGE extends RangeError {
    code = "ERR_OUT_OF_RANGE";
    constructor(str, range, received) {
        super(`The value of "${str}" is out of range. It must be ${range}. Received ${received}`);
        const { name } = this;
        this.name = `${name} [${this.code}]`;
        this.stack;
        this.name = name;
    }
}
export class ERR_AMBIGUOUS_ARGUMENT extends NodeTypeError {
    constructor(x, y) {
        super("ERR_AMBIGUOUS_ARGUMENT", `The "${x}" argument is ambiguous. ${y}`);
    }
}
export class ERR_ARG_NOT_ITERABLE extends NodeTypeError {
    constructor(x) {
        super("ERR_ARG_NOT_ITERABLE", `${x} must be iterable`);
    }
}
export class ERR_ASSERTION extends NodeError {
    constructor(x) {
        super("ERR_ASSERTION", `${x}`);
    }
}
export class ERR_ASYNC_CALLBACK extends NodeTypeError {
    constructor(x) {
        super("ERR_ASYNC_CALLBACK", `${x} must be a function`);
    }
}
export class ERR_ASYNC_TYPE extends NodeTypeError {
    constructor(x) {
        super("ERR_ASYNC_TYPE", `Invalid name for async "type": ${x}`);
    }
}
export class ERR_BROTLI_INVALID_PARAM extends NodeRangeError {
    constructor(x) {
        super("ERR_BROTLI_INVALID_PARAM", `${x} is not a valid Brotli parameter`);
    }
}
export class ERR_BUFFER_OUT_OF_BOUNDS extends NodeRangeError {
    constructor(name) {
        super("ERR_BUFFER_OUT_OF_BOUNDS", name
            ? `"${name}" is outside of buffer bounds`
            : "Attempt to access memory outside buffer bounds");
    }
}
export class ERR_BUFFER_TOO_LARGE extends NodeRangeError {
    constructor(x) {
        super("ERR_BUFFER_TOO_LARGE", `Cannot create a Buffer larger than ${x} bytes`);
    }
}
export class ERR_CANNOT_WATCH_SIGINT extends NodeError {
    constructor() {
        super("ERR_CANNOT_WATCH_SIGINT", "Cannot watch for SIGINT signals");
    }
}
export class ERR_CHILD_CLOSED_BEFORE_REPLY extends NodeError {
    constructor() {
        super("ERR_CHILD_CLOSED_BEFORE_REPLY", "Child closed before reply received");
    }
}
export class ERR_CHILD_PROCESS_IPC_REQUIRED extends NodeError {
    constructor(x) {
        super("ERR_CHILD_PROCESS_IPC_REQUIRED", `Forked processes must have an IPC channel, missing value 'ipc' in ${x}`);
    }
}
export class ERR_CHILD_PROCESS_STDIO_MAXBUFFER extends NodeRangeError {
    constructor(x) {
        super("ERR_CHILD_PROCESS_STDIO_MAXBUFFER", `${x} maxBuffer length exceeded`);
    }
}
export class ERR_CONSOLE_WRITABLE_STREAM extends NodeTypeError {
    constructor(x) {
        super("ERR_CONSOLE_WRITABLE_STREAM", `Console expects a writable stream instance for ${x}`);
    }
}
export class ERR_CONTEXT_NOT_INITIALIZED extends NodeError {
    constructor() {
        super("ERR_CONTEXT_NOT_INITIALIZED", "context used is not initialized");
    }
}
export class ERR_CPU_USAGE extends NodeError {
    constructor(x) {
        super("ERR_CPU_USAGE", `Unable to obtain cpu usage ${x}`);
    }
}
export class ERR_CRYPTO_CUSTOM_ENGINE_NOT_SUPPORTED extends NodeError {
    constructor() {
        super("ERR_CRYPTO_CUSTOM_ENGINE_NOT_SUPPORTED", "Custom engines not supported by this OpenSSL");
    }
}
export class ERR_CRYPTO_ECDH_INVALID_FORMAT extends NodeTypeError {
    constructor(x) {
        super("ERR_CRYPTO_ECDH_INVALID_FORMAT", `Invalid ECDH format: ${x}`);
    }
}
export class ERR_CRYPTO_ECDH_INVALID_PUBLIC_KEY extends NodeError {
    constructor() {
        super("ERR_CRYPTO_ECDH_INVALID_PUBLIC_KEY", "Public key is not valid for specified curve");
    }
}
export class ERR_CRYPTO_ENGINE_UNKNOWN extends NodeError {
    constructor(x) {
        super("ERR_CRYPTO_ENGINE_UNKNOWN", `Engine "${x}" was not found`);
    }
}
export class ERR_CRYPTO_FIPS_FORCED extends NodeError {
    constructor() {
        super("ERR_CRYPTO_FIPS_FORCED", "Cannot set FIPS mode, it was forced with --force-fips at startup.");
    }
}
export class ERR_CRYPTO_FIPS_UNAVAILABLE extends NodeError {
    constructor() {
        super("ERR_CRYPTO_FIPS_UNAVAILABLE", "Cannot set FIPS mode in a non-FIPS build.");
    }
}
export class ERR_CRYPTO_HASH_FINALIZED extends NodeError {
    constructor() {
        super("ERR_CRYPTO_HASH_FINALIZED", "Digest already called");
    }
}
export class ERR_CRYPTO_HASH_UPDATE_FAILED extends NodeError {
    constructor() {
        super("ERR_CRYPTO_HASH_UPDATE_FAILED", "Hash update failed");
    }
}
export class ERR_CRYPTO_INCOMPATIBLE_KEY extends NodeError {
    constructor(x, y) {
        super("ERR_CRYPTO_INCOMPATIBLE_KEY", `Incompatible ${x}: ${y}`);
    }
}
export class ERR_CRYPTO_INCOMPATIBLE_KEY_OPTIONS extends NodeError {
    constructor(x, y) {
        super("ERR_CRYPTO_INCOMPATIBLE_KEY_OPTIONS", `The selected key encoding ${x} ${y}.`);
    }
}
export class ERR_CRYPTO_INVALID_DIGEST extends NodeTypeError {
    constructor(x) {
        super("ERR_CRYPTO_INVALID_DIGEST", `Invalid digest: ${x}`);
    }
}
export class ERR_CRYPTO_INVALID_KEY_OBJECT_TYPE extends NodeTypeError {
    constructor(x, y) {
        super("ERR_CRYPTO_INVALID_KEY_OBJECT_TYPE", `Invalid key object type ${x}, expected ${y}.`);
    }
}
export class ERR_CRYPTO_INVALID_STATE extends NodeError {
    constructor(x) {
        super("ERR_CRYPTO_INVALID_STATE", `Invalid state for operation ${x}`);
    }
}
export class ERR_CRYPTO_PBKDF2_ERROR extends NodeError {
    constructor() {
        super("ERR_CRYPTO_PBKDF2_ERROR", "PBKDF2 error");
    }
}
export class ERR_CRYPTO_SCRYPT_INVALID_PARAMETER extends NodeError {
    constructor() {
        super("ERR_CRYPTO_SCRYPT_INVALID_PARAMETER", "Invalid scrypt parameter");
    }
}
export class ERR_CRYPTO_SCRYPT_NOT_SUPPORTED extends NodeError {
    constructor() {
        super("ERR_CRYPTO_SCRYPT_NOT_SUPPORTED", "Scrypt algorithm not supported");
    }
}
export class ERR_CRYPTO_SIGN_KEY_REQUIRED extends NodeError {
    constructor() {
        super("ERR_CRYPTO_SIGN_KEY_REQUIRED", "No key provided to sign");
    }
}
export class ERR_DIR_CLOSED extends NodeError {
    constructor() {
        super("ERR_DIR_CLOSED", "Directory handle was closed");
    }
}
export class ERR_DIR_CONCURRENT_OPERATION extends NodeError {
    constructor() {
        super("ERR_DIR_CONCURRENT_OPERATION", "Cannot do synchronous work on directory handle with concurrent asynchronous operations");
    }
}
export class ERR_DNS_SET_SERVERS_FAILED extends NodeError {
    constructor(x, y) {
        super("ERR_DNS_SET_SERVERS_FAILED", `c-ares failed to set servers: "${x}" [${y}]`);
    }
}
export class ERR_DOMAIN_CALLBACK_NOT_AVAILABLE extends NodeError {
    constructor() {
        super("ERR_DOMAIN_CALLBACK_NOT_AVAILABLE", "A callback was registered through " +
            "process.setUncaughtExceptionCaptureCallback(), which is mutually " +
            "exclusive with using the `domain` module");
    }
}
export class ERR_DOMAIN_CANNOT_SET_UNCAUGHT_EXCEPTION_CAPTURE extends NodeError {
    constructor() {
        super("ERR_DOMAIN_CANNOT_SET_UNCAUGHT_EXCEPTION_CAPTURE", "The `domain` module is in use, which is mutually exclusive with calling " +
            "process.setUncaughtExceptionCaptureCallback()");
    }
}
export class ERR_ENCODING_INVALID_ENCODED_DATA extends NodeErrorAbstraction {
    errno;
    constructor(encoding, ret) {
        super(TypeError.prototype.name, "ERR_ENCODING_INVALID_ENCODED_DATA", `The encoded data was not valid for encoding ${encoding}`);
        Object.setPrototypeOf(this, TypeError.prototype);
        this.errno = ret;
    }
}
const windows = [
    [-4093, ["E2BIG", "argument list too long"]],
    [-4092, ["EACCES", "permission denied"]],
    [-4091, ["EADDRINUSE", "address already in use"]],
    [-4090, ["EADDRNOTAVAIL", "address not available"]],
    [-4089, ["EAFNOSUPPORT", "address family not supported"]],
    [-4088, ["EAGAIN", "resource temporarily unavailable"]],
    [-3000, ["EAI_ADDRFAMILY", "address family not supported"]],
    [-3001, ["EAI_AGAIN", "temporary failure"]],
    [-3002, ["EAI_BADFLAGS", "bad ai_flags value"]],
    [-3013, ["EAI_BADHINTS", "invalid value for hints"]],
    [-3003, ["EAI_CANCELED", "request canceled"]],
    [-3004, ["EAI_FAIL", "permanent failure"]],
    [-3005, ["EAI_FAMILY", "ai_family not supported"]],
    [-3006, ["EAI_MEMORY", "out of memory"]],
    [-3007, ["EAI_NODATA", "no address"]],
    [-3008, ["EAI_NONAME", "unknown node or service"]],
    [-3009, ["EAI_OVERFLOW", "argument buffer overflow"]],
    [-3014, ["EAI_PROTOCOL", "resolved protocol is unknown"]],
    [-3010, ["EAI_SERVICE", "service not available for socket type"]],
    [-3011, ["EAI_SOCKTYPE", "socket type not supported"]],
    [-4084, ["EALREADY", "connection already in progress"]],
    [-4083, ["EBADF", "bad file descriptor"]],
    [-4082, ["EBUSY", "resource busy or locked"]],
    [-4081, ["ECANCELED", "operation canceled"]],
    [-4080, ["ECHARSET", "invalid Unicode character"]],
    [-4079, ["ECONNABORTED", "software caused connection abort"]],
    [-4078, ["ECONNREFUSED", "connection refused"]],
    [-4077, ["ECONNRESET", "connection reset by peer"]],
    [-4076, ["EDESTADDRREQ", "destination address required"]],
    [-4075, ["EEXIST", "file already exists"]],
    [-4074, ["EFAULT", "bad address in system call argument"]],
    [-4036, ["EFBIG", "file too large"]],
    [-4073, ["EHOSTUNREACH", "host is unreachable"]],
    [-4072, ["EINTR", "interrupted system call"]],
    [-4071, ["EINVAL", "invalid argument"]],
    [-4070, ["EIO", "i/o error"]],
    [-4069, ["EISCONN", "socket is already connected"]],
    [-4068, ["EISDIR", "illegal operation on a directory"]],
    [-4067, ["ELOOP", "too many symbolic links encountered"]],
    [-4066, ["EMFILE", "too many open files"]],
    [-4065, ["EMSGSIZE", "message too long"]],
    [-4064, ["ENAMETOOLONG", "name too long"]],
    [-4063, ["ENETDOWN", "network is down"]],
    [-4062, ["ENETUNREACH", "network is unreachable"]],
    [-4061, ["ENFILE", "file table overflow"]],
    [-4060, ["ENOBUFS", "no buffer space available"]],
    [-4059, ["ENODEV", "no such device"]],
    [-4058, ["ENOENT", "no such file or directory"]],
    [-4057, ["ENOMEM", "not enough memory"]],
    [-4056, ["ENONET", "machine is not on the network"]],
    [-4035, ["ENOPROTOOPT", "protocol not available"]],
    [-4055, ["ENOSPC", "no space left on device"]],
    [-4054, ["ENOSYS", "function not implemented"]],
    [-4053, ["ENOTCONN", "socket is not connected"]],
    [-4052, ["ENOTDIR", "not a directory"]],
    [-4051, ["ENOTEMPTY", "directory not empty"]],
    [-4050, ["ENOTSOCK", "socket operation on non-socket"]],
    [-4049, ["ENOTSUP", "operation not supported on socket"]],
    [-4048, ["EPERM", "operation not permitted"]],
    [-4047, ["EPIPE", "broken pipe"]],
    [-4046, ["EPROTO", "protocol error"]],
    [-4045, ["EPROTONOSUPPORT", "protocol not supported"]],
    [-4044, ["EPROTOTYPE", "protocol wrong type for socket"]],
    [-4034, ["ERANGE", "result too large"]],
    [-4043, ["EROFS", "read-only file system"]],
    [-4042, ["ESHUTDOWN", "cannot send after transport endpoint shutdown"]],
    [-4041, ["ESPIPE", "invalid seek"]],
    [-4040, ["ESRCH", "no such process"]],
    [-4039, ["ETIMEDOUT", "connection timed out"]],
    [-4038, ["ETXTBSY", "text file is busy"]],
    [-4037, ["EXDEV", "cross-device link not permitted"]],
    [-4094, ["UNKNOWN", "unknown error"]],
    [-4095, ["EOF", "end of file"]],
    [-4033, ["ENXIO", "no such device or address"]],
    [-4032, ["EMLINK", "too many links"]],
    [-4031, ["EHOSTDOWN", "host is down"]],
    [-4030, ["EREMOTEIO", "remote I/O error"]],
    [-4029, ["ENOTTY", "inappropriate ioctl for device"]],
    [-4028, ["EFTYPE", "inappropriate file type or format"]],
    [-4027, ["EILSEQ", "illegal byte sequence"]],
];
const darwin = [
    [-7, ["E2BIG", "argument list too long"]],
    [-13, ["EACCES", "permission denied"]],
    [-48, ["EADDRINUSE", "address already in use"]],
    [-49, ["EADDRNOTAVAIL", "address not available"]],
    [-47, ["EAFNOSUPPORT", "address family not supported"]],
    [-35, ["EAGAIN", "resource temporarily unavailable"]],
    [-3000, ["EAI_ADDRFAMILY", "address family not supported"]],
    [-3001, ["EAI_AGAIN", "temporary failure"]],
    [-3002, ["EAI_BADFLAGS", "bad ai_flags value"]],
    [-3013, ["EAI_BADHINTS", "invalid value for hints"]],
    [-3003, ["EAI_CANCELED", "request canceled"]],
    [-3004, ["EAI_FAIL", "permanent failure"]],
    [-3005, ["EAI_FAMILY", "ai_family not supported"]],
    [-3006, ["EAI_MEMORY", "out of memory"]],
    [-3007, ["EAI_NODATA", "no address"]],
    [-3008, ["EAI_NONAME", "unknown node or service"]],
    [-3009, ["EAI_OVERFLOW", "argument buffer overflow"]],
    [-3014, ["EAI_PROTOCOL", "resolved protocol is unknown"]],
    [-3010, ["EAI_SERVICE", "service not available for socket type"]],
    [-3011, ["EAI_SOCKTYPE", "socket type not supported"]],
    [-37, ["EALREADY", "connection already in progress"]],
    [-9, ["EBADF", "bad file descriptor"]],
    [-16, ["EBUSY", "resource busy or locked"]],
    [-89, ["ECANCELED", "operation canceled"]],
    [-4080, ["ECHARSET", "invalid Unicode character"]],
    [-53, ["ECONNABORTED", "software caused connection abort"]],
    [-61, ["ECONNREFUSED", "connection refused"]],
    [-54, ["ECONNRESET", "connection reset by peer"]],
    [-39, ["EDESTADDRREQ", "destination address required"]],
    [-17, ["EEXIST", "file already exists"]],
    [-14, ["EFAULT", "bad address in system call argument"]],
    [-27, ["EFBIG", "file too large"]],
    [-65, ["EHOSTUNREACH", "host is unreachable"]],
    [-4, ["EINTR", "interrupted system call"]],
    [-22, ["EINVAL", "invalid argument"]],
    [-5, ["EIO", "i/o error"]],
    [-56, ["EISCONN", "socket is already connected"]],
    [-21, ["EISDIR", "illegal operation on a directory"]],
    [-62, ["ELOOP", "too many symbolic links encountered"]],
    [-24, ["EMFILE", "too many open files"]],
    [-40, ["EMSGSIZE", "message too long"]],
    [-63, ["ENAMETOOLONG", "name too long"]],
    [-50, ["ENETDOWN", "network is down"]],
    [-51, ["ENETUNREACH", "network is unreachable"]],
    [-23, ["ENFILE", "file table overflow"]],
    [-55, ["ENOBUFS", "no buffer space available"]],
    [-19, ["ENODEV", "no such device"]],
    [-2, ["ENOENT", "no such file or directory"]],
    [-12, ["ENOMEM", "not enough memory"]],
    [-4056, ["ENONET", "machine is not on the network"]],
    [-42, ["ENOPROTOOPT", "protocol not available"]],
    [-28, ["ENOSPC", "no space left on device"]],
    [-78, ["ENOSYS", "function not implemented"]],
    [-57, ["ENOTCONN", "socket is not connected"]],
    [-20, ["ENOTDIR", "not a directory"]],
    [-66, ["ENOTEMPTY", "directory not empty"]],
    [-38, ["ENOTSOCK", "socket operation on non-socket"]],
    [-45, ["ENOTSUP", "operation not supported on socket"]],
    [-1, ["EPERM", "operation not permitted"]],
    [-32, ["EPIPE", "broken pipe"]],
    [-100, ["EPROTO", "protocol error"]],
    [-43, ["EPROTONOSUPPORT", "protocol not supported"]],
    [-41, ["EPROTOTYPE", "protocol wrong type for socket"]],
    [-34, ["ERANGE", "result too large"]],
    [-30, ["EROFS", "read-only file system"]],
    [-58, ["ESHUTDOWN", "cannot send after transport endpoint shutdown"]],
    [-29, ["ESPIPE", "invalid seek"]],
    [-3, ["ESRCH", "no such process"]],
    [-60, ["ETIMEDOUT", "connection timed out"]],
    [-26, ["ETXTBSY", "text file is busy"]],
    [-18, ["EXDEV", "cross-device link not permitted"]],
    [-4094, ["UNKNOWN", "unknown error"]],
    [-4095, ["EOF", "end of file"]],
    [-6, ["ENXIO", "no such device or address"]],
    [-31, ["EMLINK", "too many links"]],
    [-64, ["EHOSTDOWN", "host is down"]],
    [-4030, ["EREMOTEIO", "remote I/O error"]],
    [-25, ["ENOTTY", "inappropriate ioctl for device"]],
    [-79, ["EFTYPE", "inappropriate file type or format"]],
    [-92, ["EILSEQ", "illegal byte sequence"]],
];
const linux = [
    [-7, ["E2BIG", "argument list too long"]],
    [-13, ["EACCES", "permission denied"]],
    [-98, ["EADDRINUSE", "address already in use"]],
    [-99, ["EADDRNOTAVAIL", "address not available"]],
    [-97, ["EAFNOSUPPORT", "address family not supported"]],
    [-11, ["EAGAIN", "resource temporarily unavailable"]],
    [-3000, ["EAI_ADDRFAMILY", "address family not supported"]],
    [-3001, ["EAI_AGAIN", "temporary failure"]],
    [-3002, ["EAI_BADFLAGS", "bad ai_flags value"]],
    [-3013, ["EAI_BADHINTS", "invalid value for hints"]],
    [-3003, ["EAI_CANCELED", "request canceled"]],
    [-3004, ["EAI_FAIL", "permanent failure"]],
    [-3005, ["EAI_FAMILY", "ai_family not supported"]],
    [-3006, ["EAI_MEMORY", "out of memory"]],
    [-3007, ["EAI_NODATA", "no address"]],
    [-3008, ["EAI_NONAME", "unknown node or service"]],
    [-3009, ["EAI_OVERFLOW", "argument buffer overflow"]],
    [-3014, ["EAI_PROTOCOL", "resolved protocol is unknown"]],
    [-3010, ["EAI_SERVICE", "service not available for socket type"]],
    [-3011, ["EAI_SOCKTYPE", "socket type not supported"]],
    [-114, ["EALREADY", "connection already in progress"]],
    [-9, ["EBADF", "bad file descriptor"]],
    [-16, ["EBUSY", "resource busy or locked"]],
    [-125, ["ECANCELED", "operation canceled"]],
    [-4080, ["ECHARSET", "invalid Unicode character"]],
    [-103, ["ECONNABORTED", "software caused connection abort"]],
    [-111, ["ECONNREFUSED", "connection refused"]],
    [-104, ["ECONNRESET", "connection reset by peer"]],
    [-89, ["EDESTADDRREQ", "destination address required"]],
    [-17, ["EEXIST", "file already exists"]],
    [-14, ["EFAULT", "bad address in system call argument"]],
    [-27, ["EFBIG", "file too large"]],
    [-113, ["EHOSTUNREACH", "host is unreachable"]],
    [-4, ["EINTR", "interrupted system call"]],
    [-22, ["EINVAL", "invalid argument"]],
    [-5, ["EIO", "i/o error"]],
    [-106, ["EISCONN", "socket is already connected"]],
    [-21, ["EISDIR", "illegal operation on a directory"]],
    [-40, ["ELOOP", "too many symbolic links encountered"]],
    [-24, ["EMFILE", "too many open files"]],
    [-90, ["EMSGSIZE", "message too long"]],
    [-36, ["ENAMETOOLONG", "name too long"]],
    [-100, ["ENETDOWN", "network is down"]],
    [-101, ["ENETUNREACH", "network is unreachable"]],
    [-23, ["ENFILE", "file table overflow"]],
    [-105, ["ENOBUFS", "no buffer space available"]],
    [-19, ["ENODEV", "no such device"]],
    [-2, ["ENOENT", "no such file or directory"]],
    [-12, ["ENOMEM", "not enough memory"]],
    [-64, ["ENONET", "machine is not on the network"]],
    [-92, ["ENOPROTOOPT", "protocol not available"]],
    [-28, ["ENOSPC", "no space left on device"]],
    [-38, ["ENOSYS", "function not implemented"]],
    [-107, ["ENOTCONN", "socket is not connected"]],
    [-20, ["ENOTDIR", "not a directory"]],
    [-39, ["ENOTEMPTY", "directory not empty"]],
    [-88, ["ENOTSOCK", "socket operation on non-socket"]],
    [-95, ["ENOTSUP", "operation not supported on socket"]],
    [-1, ["EPERM", "operation not permitted"]],
    [-32, ["EPIPE", "broken pipe"]],
    [-71, ["EPROTO", "protocol error"]],
    [-93, ["EPROTONOSUPPORT", "protocol not supported"]],
    [-91, ["EPROTOTYPE", "protocol wrong type for socket"]],
    [-34, ["ERANGE", "result too large"]],
    [-30, ["EROFS", "read-only file system"]],
    [-108, ["ESHUTDOWN", "cannot send after transport endpoint shutdown"]],
    [-29, ["ESPIPE", "invalid seek"]],
    [-3, ["ESRCH", "no such process"]],
    [-110, ["ETIMEDOUT", "connection timed out"]],
    [-26, ["ETXTBSY", "text file is busy"]],
    [-18, ["EXDEV", "cross-device link not permitted"]],
    [-4094, ["UNKNOWN", "unknown error"]],
    [-4095, ["EOF", "end of file"]],
    [-6, ["ENXIO", "no such device or address"]],
    [-31, ["EMLINK", "too many links"]],
    [-112, ["EHOSTDOWN", "host is down"]],
    [-121, ["EREMOTEIO", "remote I/O error"]],
    [-25, ["ENOTTY", "inappropriate ioctl for device"]],
    [-4028, ["EFTYPE", "inappropriate file type or format"]],
    [-84, ["EILSEQ", "illegal byte sequence"]],
];
const { os } = Deno.build;
export const errorMap = new Map(os === "windows"
    ? windows
    : os === "darwin"
        ? darwin
        : os === "linux"
            ? linux
            : unreachable());
export class ERR_ENCODING_NOT_SUPPORTED extends NodeRangeError {
    constructor(x) {
        super("ERR_ENCODING_NOT_SUPPORTED", `The "${x}" encoding is not supported`);
    }
}
export class ERR_EVAL_ESM_CANNOT_PRINT extends NodeError {
    constructor() {
        super("ERR_EVAL_ESM_CANNOT_PRINT", `--print cannot be used with ESM input`);
    }
}
export class ERR_EVENT_RECURSION extends NodeError {
    constructor(x) {
        super("ERR_EVENT_RECURSION", `The event "${x}" is already being dispatched`);
    }
}
export class ERR_FEATURE_UNAVAILABLE_ON_PLATFORM extends NodeTypeError {
    constructor(x) {
        super("ERR_FEATURE_UNAVAILABLE_ON_PLATFORM", `The feature ${x} is unavailable on the current platform, which is being used to run Node.js`);
    }
}
export class ERR_FS_FILE_TOO_LARGE extends NodeRangeError {
    constructor(x) {
        super("ERR_FS_FILE_TOO_LARGE", `File size (${x}) is greater than 2 GB`);
    }
}
export class ERR_FS_INVALID_SYMLINK_TYPE extends NodeError {
    constructor(x) {
        super("ERR_FS_INVALID_SYMLINK_TYPE", `Symlink type must be one of "dir", "file", or "junction". Received "${x}"`);
    }
}
export class ERR_HTTP2_ALTSVC_INVALID_ORIGIN extends NodeTypeError {
    constructor() {
        super("ERR_HTTP2_ALTSVC_INVALID_ORIGIN", `HTTP/2 ALTSVC frames require a valid origin`);
    }
}
export class ERR_HTTP2_ALTSVC_LENGTH extends NodeTypeError {
    constructor() {
        super("ERR_HTTP2_ALTSVC_LENGTH", `HTTP/2 ALTSVC frames are limited to 16382 bytes`);
    }
}
export class ERR_HTTP2_CONNECT_AUTHORITY extends NodeError {
    constructor() {
        super("ERR_HTTP2_CONNECT_AUTHORITY", `:authority header is required for CONNECT requests`);
    }
}
export class ERR_HTTP2_CONNECT_PATH extends NodeError {
    constructor() {
        super("ERR_HTTP2_CONNECT_PATH", `The :path header is forbidden for CONNECT requests`);
    }
}
export class ERR_HTTP2_CONNECT_SCHEME extends NodeError {
    constructor() {
        super("ERR_HTTP2_CONNECT_SCHEME", `The :scheme header is forbidden for CONNECT requests`);
    }
}
export class ERR_HTTP2_GOAWAY_SESSION extends NodeError {
    constructor() {
        super("ERR_HTTP2_GOAWAY_SESSION", `New streams cannot be created after receiving a GOAWAY`);
    }
}
export class ERR_HTTP2_HEADERS_AFTER_RESPOND extends NodeError {
    constructor() {
        super("ERR_HTTP2_HEADERS_AFTER_RESPOND", `Cannot specify additional headers after response initiated`);
    }
}
export class ERR_HTTP2_HEADERS_SENT extends NodeError {
    constructor() {
        super("ERR_HTTP2_HEADERS_SENT", `Response has already been initiated.`);
    }
}
export class ERR_HTTP2_HEADER_SINGLE_VALUE extends NodeTypeError {
    constructor(x) {
        super("ERR_HTTP2_HEADER_SINGLE_VALUE", `Header field "${x}" must only have a single value`);
    }
}
export class ERR_HTTP2_INFO_STATUS_NOT_ALLOWED extends NodeRangeError {
    constructor() {
        super("ERR_HTTP2_INFO_STATUS_NOT_ALLOWED", `Informational status codes cannot be used`);
    }
}
export class ERR_HTTP2_INVALID_CONNECTION_HEADERS extends NodeTypeError {
    constructor(x) {
        super("ERR_HTTP2_INVALID_CONNECTION_HEADERS", `HTTP/1 Connection specific headers are forbidden: "${x}"`);
    }
}
export class ERR_HTTP2_INVALID_HEADER_VALUE extends NodeTypeError {
    constructor(x, y) {
        super("ERR_HTTP2_INVALID_HEADER_VALUE", `Invalid value "${x}" for header "${y}"`);
    }
}
export class ERR_HTTP2_INVALID_INFO_STATUS extends NodeRangeError {
    constructor(x) {
        super("ERR_HTTP2_INVALID_INFO_STATUS", `Invalid informational status code: ${x}`);
    }
}
export class ERR_HTTP2_INVALID_ORIGIN extends NodeTypeError {
    constructor() {
        super("ERR_HTTP2_INVALID_ORIGIN", `HTTP/2 ORIGIN frames require a valid origin`);
    }
}
export class ERR_HTTP2_INVALID_PACKED_SETTINGS_LENGTH extends NodeRangeError {
    constructor() {
        super("ERR_HTTP2_INVALID_PACKED_SETTINGS_LENGTH", `Packed settings length must be a multiple of six`);
    }
}
export class ERR_HTTP2_INVALID_PSEUDOHEADER extends NodeTypeError {
    constructor(x) {
        super("ERR_HTTP2_INVALID_PSEUDOHEADER", `"${x}" is an invalid pseudoheader or is used incorrectly`);
    }
}
export class ERR_HTTP2_INVALID_SESSION extends NodeError {
    constructor() {
        super("ERR_HTTP2_INVALID_SESSION", `The session has been destroyed`);
    }
}
export class ERR_HTTP2_INVALID_STREAM extends NodeError {
    constructor() {
        super("ERR_HTTP2_INVALID_STREAM", `The stream has been destroyed`);
    }
}
export class ERR_HTTP2_MAX_PENDING_SETTINGS_ACK extends NodeError {
    constructor() {
        super("ERR_HTTP2_MAX_PENDING_SETTINGS_ACK", `Maximum number of pending settings acknowledgements`);
    }
}
export class ERR_HTTP2_NESTED_PUSH extends NodeError {
    constructor() {
        super("ERR_HTTP2_NESTED_PUSH", `A push stream cannot initiate another push stream.`);
    }
}
export class ERR_HTTP2_NO_SOCKET_MANIPULATION extends NodeError {
    constructor() {
        super("ERR_HTTP2_NO_SOCKET_MANIPULATION", `HTTP/2 sockets should not be directly manipulated (e.g. read and written)`);
    }
}
export class ERR_HTTP2_ORIGIN_LENGTH extends NodeTypeError {
    constructor() {
        super("ERR_HTTP2_ORIGIN_LENGTH", `HTTP/2 ORIGIN frames are limited to 16382 bytes`);
    }
}
export class ERR_HTTP2_OUT_OF_STREAMS extends NodeError {
    constructor() {
        super("ERR_HTTP2_OUT_OF_STREAMS", `No stream ID is available because maximum stream ID has been reached`);
    }
}
export class ERR_HTTP2_PAYLOAD_FORBIDDEN extends NodeError {
    constructor(x) {
        super("ERR_HTTP2_PAYLOAD_FORBIDDEN", `Responses with ${x} status must not have a payload`);
    }
}
export class ERR_HTTP2_PING_CANCEL extends NodeError {
    constructor() {
        super("ERR_HTTP2_PING_CANCEL", `HTTP2 ping cancelled`);
    }
}
export class ERR_HTTP2_PING_LENGTH extends NodeRangeError {
    constructor() {
        super("ERR_HTTP2_PING_LENGTH", `HTTP2 ping payload must be 8 bytes`);
    }
}
export class ERR_HTTP2_PSEUDOHEADER_NOT_ALLOWED extends NodeTypeError {
    constructor() {
        super("ERR_HTTP2_PSEUDOHEADER_NOT_ALLOWED", `Cannot set HTTP/2 pseudo-headers`);
    }
}
export class ERR_HTTP2_PUSH_DISABLED extends NodeError {
    constructor() {
        super("ERR_HTTP2_PUSH_DISABLED", `HTTP/2 client has disabled push streams`);
    }
}
export class ERR_HTTP2_SEND_FILE extends NodeError {
    constructor() {
        super("ERR_HTTP2_SEND_FILE", `Directories cannot be sent`);
    }
}
export class ERR_HTTP2_SEND_FILE_NOSEEK extends NodeError {
    constructor() {
        super("ERR_HTTP2_SEND_FILE_NOSEEK", `Offset or length can only be specified for regular files`);
    }
}
export class ERR_HTTP2_SESSION_ERROR extends NodeError {
    constructor(x) {
        super("ERR_HTTP2_SESSION_ERROR", `Session closed with error code ${x}`);
    }
}
export class ERR_HTTP2_SETTINGS_CANCEL extends NodeError {
    constructor() {
        super("ERR_HTTP2_SETTINGS_CANCEL", `HTTP2 session settings canceled`);
    }
}
export class ERR_HTTP2_SOCKET_BOUND extends NodeError {
    constructor() {
        super("ERR_HTTP2_SOCKET_BOUND", `The socket is already bound to an Http2Session`);
    }
}
export class ERR_HTTP2_SOCKET_UNBOUND extends NodeError {
    constructor() {
        super("ERR_HTTP2_SOCKET_UNBOUND", `The socket has been disconnected from the Http2Session`);
    }
}
export class ERR_HTTP2_STATUS_101 extends NodeError {
    constructor() {
        super("ERR_HTTP2_STATUS_101", `HTTP status code 101 (Switching Protocols) is forbidden in HTTP/2`);
    }
}
export class ERR_HTTP2_STATUS_INVALID extends NodeRangeError {
    constructor(x) {
        super("ERR_HTTP2_STATUS_INVALID", `Invalid status code: ${x}`);
    }
}
export class ERR_HTTP2_STREAM_ERROR extends NodeError {
    constructor(x) {
        super("ERR_HTTP2_STREAM_ERROR", `Stream closed with error code ${x}`);
    }
}
export class ERR_HTTP2_STREAM_SELF_DEPENDENCY extends NodeError {
    constructor() {
        super("ERR_HTTP2_STREAM_SELF_DEPENDENCY", `A stream cannot depend on itself`);
    }
}
export class ERR_HTTP2_TRAILERS_ALREADY_SENT extends NodeError {
    constructor() {
        super("ERR_HTTP2_TRAILERS_ALREADY_SENT", `Trailing headers have already been sent`);
    }
}
export class ERR_HTTP2_TRAILERS_NOT_READY extends NodeError {
    constructor() {
        super("ERR_HTTP2_TRAILERS_NOT_READY", `Trailing headers cannot be sent until after the wantTrailers event is emitted`);
    }
}
export class ERR_HTTP2_UNSUPPORTED_PROTOCOL extends NodeError {
    constructor(x) {
        super("ERR_HTTP2_UNSUPPORTED_PROTOCOL", `protocol "${x}" is unsupported.`);
    }
}
export class ERR_HTTP_HEADERS_SENT extends NodeError {
    constructor(x) {
        super("ERR_HTTP_HEADERS_SENT", `Cannot ${x} headers after they are sent to the client`);
    }
}
export class ERR_HTTP_INVALID_HEADER_VALUE extends NodeTypeError {
    constructor(x, y) {
        super("ERR_HTTP_INVALID_HEADER_VALUE", `Invalid value "${x}" for header "${y}"`);
    }
}
export class ERR_HTTP_INVALID_STATUS_CODE extends NodeRangeError {
    constructor(x) {
        super("ERR_HTTP_INVALID_STATUS_CODE", `Invalid status code: ${x}`);
    }
}
export class ERR_HTTP_SOCKET_ENCODING extends NodeError {
    constructor() {
        super("ERR_HTTP_SOCKET_ENCODING", `Changing the socket encoding is not allowed per RFC7230 Section 3.`);
    }
}
export class ERR_HTTP_TRAILER_INVALID extends NodeError {
    constructor() {
        super("ERR_HTTP_TRAILER_INVALID", `Trailers are invalid with this transfer encoding`);
    }
}
export class ERR_INCOMPATIBLE_OPTION_PAIR extends NodeTypeError {
    constructor(x, y) {
        super("ERR_INCOMPATIBLE_OPTION_PAIR", `Option "${x}" cannot be used in combination with option "${y}"`);
    }
}
export class ERR_INPUT_TYPE_NOT_ALLOWED extends NodeError {
    constructor() {
        super("ERR_INPUT_TYPE_NOT_ALLOWED", `--input-type can only be used with string input via --eval, --print, or STDIN`);
    }
}
export class ERR_INSPECTOR_ALREADY_ACTIVATED extends NodeError {
    constructor() {
        super("ERR_INSPECTOR_ALREADY_ACTIVATED", `Inspector is already activated. Close it with inspector.close() before activating it again.`);
    }
}
export class ERR_INSPECTOR_ALREADY_CONNECTED extends NodeError {
    constructor(x) {
        super("ERR_INSPECTOR_ALREADY_CONNECTED", `${x} is already connected`);
    }
}
export class ERR_INSPECTOR_CLOSED extends NodeError {
    constructor() {
        super("ERR_INSPECTOR_CLOSED", `Session was closed`);
    }
}
export class ERR_INSPECTOR_COMMAND extends NodeError {
    constructor(x, y) {
        super("ERR_INSPECTOR_COMMAND", `Inspector error ${x}: ${y}`);
    }
}
export class ERR_INSPECTOR_NOT_ACTIVE extends NodeError {
    constructor() {
        super("ERR_INSPECTOR_NOT_ACTIVE", `Inspector is not active`);
    }
}
export class ERR_INSPECTOR_NOT_AVAILABLE extends NodeError {
    constructor() {
        super("ERR_INSPECTOR_NOT_AVAILABLE", `Inspector is not available`);
    }
}
export class ERR_INSPECTOR_NOT_CONNECTED extends NodeError {
    constructor() {
        super("ERR_INSPECTOR_NOT_CONNECTED", `Session is not connected`);
    }
}
export class ERR_INSPECTOR_NOT_WORKER extends NodeError {
    constructor() {
        super("ERR_INSPECTOR_NOT_WORKER", `Current thread is not a worker`);
    }
}
export class ERR_INVALID_ASYNC_ID extends NodeRangeError {
    constructor(x, y) {
        super("ERR_INVALID_ASYNC_ID", `Invalid ${x} value: ${y}`);
    }
}
export class ERR_INVALID_BUFFER_SIZE extends NodeRangeError {
    constructor(x) {
        super("ERR_INVALID_BUFFER_SIZE", `Buffer size must be a multiple of ${x}`);
    }
}
export class ERR_INVALID_CALLBACK extends NodeTypeError {
    constructor(object) {
        super("ERR_INVALID_CALLBACK", `Callback must be a function. Received ${JSON.stringify(object)}`);
    }
}
export class ERR_INVALID_CURSOR_POS extends NodeTypeError {
    constructor() {
        super("ERR_INVALID_CURSOR_POS", `Cannot set cursor row without setting its column`);
    }
}
export class ERR_INVALID_FD extends NodeRangeError {
    constructor(x) {
        super("ERR_INVALID_FD", `"fd" must be a positive integer: ${x}`);
    }
}
export class ERR_INVALID_FD_TYPE extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_FD_TYPE", `Unsupported fd type: ${x}`);
    }
}
export class ERR_INVALID_FILE_URL_HOST extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_FILE_URL_HOST", `File URL host must be "localhost" or empty on ${x}`);
    }
}
export class ERR_INVALID_FILE_URL_PATH extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_FILE_URL_PATH", `File URL path ${x}`);
    }
}
export class ERR_INVALID_HANDLE_TYPE extends NodeTypeError {
    constructor() {
        super("ERR_INVALID_HANDLE_TYPE", `This handle type cannot be sent`);
    }
}
export class ERR_INVALID_HTTP_TOKEN extends NodeTypeError {
    constructor(x, y) {
        super("ERR_INVALID_HTTP_TOKEN", `${x} must be a valid HTTP token ["${y}"]`);
    }
}
export class ERR_INVALID_IP_ADDRESS extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_IP_ADDRESS", `Invalid IP address: ${x}`);
    }
}
export class ERR_INVALID_OPT_VALUE_ENCODING extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_OPT_VALUE_ENCODING", `The value "${x}" is invalid for option "encoding"`);
    }
}
export class ERR_INVALID_PERFORMANCE_MARK extends NodeError {
    constructor(x) {
        super("ERR_INVALID_PERFORMANCE_MARK", `The "${x}" performance mark has not been set`);
    }
}
export class ERR_INVALID_PROTOCOL extends NodeTypeError {
    constructor(x, y) {
        super("ERR_INVALID_PROTOCOL", `Protocol "${x}" not supported. Expected "${y}"`);
    }
}
export class ERR_INVALID_REPL_EVAL_CONFIG extends NodeTypeError {
    constructor() {
        super("ERR_INVALID_REPL_EVAL_CONFIG", `Cannot specify both "breakEvalOnSigint" and "eval" for REPL`);
    }
}
export class ERR_INVALID_REPL_INPUT extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_REPL_INPUT", `${x}`);
    }
}
export class ERR_INVALID_SYNC_FORK_INPUT extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_SYNC_FORK_INPUT", `Asynchronous forks do not support Buffer, TypedArray, DataView or string input: ${x}`);
    }
}
export class ERR_INVALID_THIS extends NodeTypeError {
    constructor(x) {
        super("ERR_INVALID_THIS", `Value of "this" must be of type ${x}`);
    }
}
export class ERR_INVALID_TUPLE extends NodeTypeError {
    constructor(x, y) {
        super("ERR_INVALID_TUPLE", `${x} must be an iterable ${y} tuple`);
    }
}
export class ERR_INVALID_URI extends NodeURIError {
    constructor() {
        super("ERR_INVALID_URI", `URI malformed`);
    }
}
export class ERR_IPC_CHANNEL_CLOSED extends NodeError {
    constructor() {
        super("ERR_IPC_CHANNEL_CLOSED", `Channel closed`);
    }
}
export class ERR_IPC_DISCONNECTED extends NodeError {
    constructor() {
        super("ERR_IPC_DISCONNECTED", `IPC channel is already disconnected`);
    }
}
export class ERR_IPC_ONE_PIPE extends NodeError {
    constructor() {
        super("ERR_IPC_ONE_PIPE", `Child process can have only one IPC pipe`);
    }
}
export class ERR_IPC_SYNC_FORK extends NodeError {
    constructor() {
        super("ERR_IPC_SYNC_FORK", `IPC cannot be used with synchronous forks`);
    }
}
export class ERR_MANIFEST_DEPENDENCY_MISSING extends NodeError {
    constructor(x, y) {
        super("ERR_MANIFEST_DEPENDENCY_MISSING", `Manifest resource ${x} does not list ${y} as a dependency specifier`);
    }
}
export class ERR_MANIFEST_INTEGRITY_MISMATCH extends NodeSyntaxError {
    constructor(x) {
        super("ERR_MANIFEST_INTEGRITY_MISMATCH", `Manifest resource ${x} has multiple entries but integrity lists do not match`);
    }
}
export class ERR_MANIFEST_INVALID_RESOURCE_FIELD extends NodeTypeError {
    constructor(x, y) {
        super("ERR_MANIFEST_INVALID_RESOURCE_FIELD", `Manifest resource ${x} has invalid property value for ${y}`);
    }
}
export class ERR_MANIFEST_TDZ extends NodeError {
    constructor() {
        super("ERR_MANIFEST_TDZ", `Manifest initialization has not yet run`);
    }
}
export class ERR_MANIFEST_UNKNOWN_ONERROR extends NodeSyntaxError {
    constructor(x) {
        super("ERR_MANIFEST_UNKNOWN_ONERROR", `Manifest specified unknown error behavior "${x}".`);
    }
}
export class ERR_METHOD_NOT_IMPLEMENTED extends NodeError {
    constructor(x) {
        super("ERR_METHOD_NOT_IMPLEMENTED", `The ${x} method is not implemented`);
    }
}
export class ERR_MISSING_ARGS extends NodeTypeError {
    constructor(...args) {
        args = args.map((a) => `"${a}"`);
        let msg = "The ";
        switch (args.length) {
            case 1:
                msg += `${args[0]} argument`;
                break;
            case 2:
                msg += `${args[0]} and ${args[1]} arguments`;
                break;
            default:
                msg += args.slice(0, args.length - 1).join(", ");
                msg += `, and ${args[args.length - 1]} arguments`;
                break;
        }
        super("ERR_MISSING_ARGS", `${msg} must be specified`);
    }
}
export class ERR_MISSING_OPTION extends NodeTypeError {
    constructor(x) {
        super("ERR_MISSING_OPTION", `${x} is required`);
    }
}
export class ERR_MULTIPLE_CALLBACK extends NodeError {
    constructor() {
        super("ERR_MULTIPLE_CALLBACK", `Callback called multiple times`);
    }
}
export class ERR_NAPI_CONS_FUNCTION extends NodeTypeError {
    constructor() {
        super("ERR_NAPI_CONS_FUNCTION", `Constructor must be a function`);
    }
}
export class ERR_NAPI_INVALID_DATAVIEW_ARGS extends NodeRangeError {
    constructor() {
        super("ERR_NAPI_INVALID_DATAVIEW_ARGS", `byte_offset + byte_length should be less than or equal to the size in bytes of the array passed in`);
    }
}
export class ERR_NAPI_INVALID_TYPEDARRAY_ALIGNMENT extends NodeRangeError {
    constructor(x, y) {
        super("ERR_NAPI_INVALID_TYPEDARRAY_ALIGNMENT", `start offset of ${x} should be a multiple of ${y}`);
    }
}
export class ERR_NAPI_INVALID_TYPEDARRAY_LENGTH extends NodeRangeError {
    constructor() {
        super("ERR_NAPI_INVALID_TYPEDARRAY_LENGTH", `Invalid typed array length`);
    }
}
export class ERR_NO_CRYPTO extends NodeError {
    constructor() {
        super("ERR_NO_CRYPTO", `Node.js is not compiled with OpenSSL crypto support`);
    }
}
export class ERR_NO_ICU extends NodeTypeError {
    constructor(x) {
        super("ERR_NO_ICU", `${x} is not supported on Node.js compiled without ICU`);
    }
}
export class ERR_QUICCLIENTSESSION_FAILED extends NodeError {
    constructor(x) {
        super("ERR_QUICCLIENTSESSION_FAILED", `Failed to create a new QuicClientSession: ${x}`);
    }
}
export class ERR_QUICCLIENTSESSION_FAILED_SETSOCKET extends NodeError {
    constructor() {
        super("ERR_QUICCLIENTSESSION_FAILED_SETSOCKET", `Failed to set the QuicSocket`);
    }
}
export class ERR_QUICSESSION_DESTROYED extends NodeError {
    constructor(x) {
        super("ERR_QUICSESSION_DESTROYED", `Cannot call ${x} after a QuicSession has been destroyed`);
    }
}
export class ERR_QUICSESSION_INVALID_DCID extends NodeError {
    constructor(x) {
        super("ERR_QUICSESSION_INVALID_DCID", `Invalid DCID value: ${x}`);
    }
}
export class ERR_QUICSESSION_UPDATEKEY extends NodeError {
    constructor() {
        super("ERR_QUICSESSION_UPDATEKEY", `Unable to update QuicSession keys`);
    }
}
export class ERR_QUICSOCKET_DESTROYED extends NodeError {
    constructor(x) {
        super("ERR_QUICSOCKET_DESTROYED", `Cannot call ${x} after a QuicSocket has been destroyed`);
    }
}
export class ERR_QUICSOCKET_INVALID_STATELESS_RESET_SECRET_LENGTH extends NodeError {
    constructor() {
        super("ERR_QUICSOCKET_INVALID_STATELESS_RESET_SECRET_LENGTH", `The stateResetToken must be exactly 16-bytes in length`);
    }
}
export class ERR_QUICSOCKET_LISTENING extends NodeError {
    constructor() {
        super("ERR_QUICSOCKET_LISTENING", `This QuicSocket is already listening`);
    }
}
export class ERR_QUICSOCKET_UNBOUND extends NodeError {
    constructor(x) {
        super("ERR_QUICSOCKET_UNBOUND", `Cannot call ${x} before a QuicSocket has been bound`);
    }
}
export class ERR_QUICSTREAM_DESTROYED extends NodeError {
    constructor(x) {
        super("ERR_QUICSTREAM_DESTROYED", `Cannot call ${x} after a QuicStream has been destroyed`);
    }
}
export class ERR_QUICSTREAM_INVALID_PUSH extends NodeError {
    constructor() {
        super("ERR_QUICSTREAM_INVALID_PUSH", `Push streams are only supported on client-initiated, bidirectional streams`);
    }
}
export class ERR_QUICSTREAM_OPEN_FAILED extends NodeError {
    constructor() {
        super("ERR_QUICSTREAM_OPEN_FAILED", `Opening a new QuicStream failed`);
    }
}
export class ERR_QUICSTREAM_UNSUPPORTED_PUSH extends NodeError {
    constructor() {
        super("ERR_QUICSTREAM_UNSUPPORTED_PUSH", `Push streams are not supported on this QuicSession`);
    }
}
export class ERR_QUIC_TLS13_REQUIRED extends NodeError {
    constructor() {
        super("ERR_QUIC_TLS13_REQUIRED", `QUIC requires TLS version 1.3`);
    }
}
export class ERR_SCRIPT_EXECUTION_INTERRUPTED extends NodeError {
    constructor() {
        super("ERR_SCRIPT_EXECUTION_INTERRUPTED", "Script execution was interrupted by `SIGINT`");
    }
}
export class ERR_SERVER_ALREADY_LISTEN extends NodeError {
    constructor() {
        super("ERR_SERVER_ALREADY_LISTEN", `Listen method has been called more than once without closing.`);
    }
}
export class ERR_SERVER_NOT_RUNNING extends NodeError {
    constructor() {
        super("ERR_SERVER_NOT_RUNNING", `Server is not running.`);
    }
}
export class ERR_SOCKET_ALREADY_BOUND extends NodeError {
    constructor() {
        super("ERR_SOCKET_ALREADY_BOUND", `Socket is already bound`);
    }
}
export class ERR_SOCKET_BAD_BUFFER_SIZE extends NodeTypeError {
    constructor() {
        super("ERR_SOCKET_BAD_BUFFER_SIZE", `Buffer size must be a positive integer`);
    }
}
export class ERR_SOCKET_BAD_TYPE extends NodeTypeError {
    constructor() {
        super("ERR_SOCKET_BAD_TYPE", `Bad socket type specified. Valid types are: udp4, udp6`);
    }
}
export class ERR_SOCKET_CLOSED extends NodeError {
    constructor() {
        super("ERR_SOCKET_CLOSED", `Socket is closed`);
    }
}
export class ERR_SOCKET_DGRAM_IS_CONNECTED extends NodeError {
    constructor() {
        super("ERR_SOCKET_DGRAM_IS_CONNECTED", `Already connected`);
    }
}
export class ERR_SOCKET_DGRAM_NOT_CONNECTED extends NodeError {
    constructor() {
        super("ERR_SOCKET_DGRAM_NOT_CONNECTED", `Not connected`);
    }
}
export class ERR_SOCKET_DGRAM_NOT_RUNNING extends NodeError {
    constructor() {
        super("ERR_SOCKET_DGRAM_NOT_RUNNING", `Not running`);
    }
}
export class ERR_SRI_PARSE extends NodeSyntaxError {
    constructor(name, char, position) {
        super("ERR_SRI_PARSE", `Subresource Integrity string ${name} had an unexpected ${char} at position ${position}`);
    }
}
export class ERR_STREAM_ALREADY_FINISHED extends NodeError {
    constructor(x) {
        super("ERR_STREAM_ALREADY_FINISHED", `Cannot call ${x} after a stream was finished`);
    }
}
export class ERR_STREAM_CANNOT_PIPE extends NodeError {
    constructor() {
        super("ERR_STREAM_CANNOT_PIPE", `Cannot pipe, not readable`);
    }
}
export class ERR_STREAM_DESTROYED extends NodeError {
    constructor(x) {
        super("ERR_STREAM_DESTROYED", `Cannot call ${x} after a stream was destroyed`);
    }
}
export class ERR_STREAM_NULL_VALUES extends NodeTypeError {
    constructor() {
        super("ERR_STREAM_NULL_VALUES", `May not write null values to stream`);
    }
}
export class ERR_STREAM_PREMATURE_CLOSE extends NodeError {
    constructor() {
        super("ERR_STREAM_PREMATURE_CLOSE", `Premature close`);
    }
}
export class ERR_STREAM_PUSH_AFTER_EOF extends NodeError {
    constructor() {
        super("ERR_STREAM_PUSH_AFTER_EOF", `stream.push() after EOF`);
    }
}
export class ERR_STREAM_UNSHIFT_AFTER_END_EVENT extends NodeError {
    constructor() {
        super("ERR_STREAM_UNSHIFT_AFTER_END_EVENT", `stream.unshift() after end event`);
    }
}
export class ERR_STREAM_WRAP extends NodeError {
    constructor() {
        super("ERR_STREAM_WRAP", `Stream has StringDecoder set or is in objectMode`);
    }
}
export class ERR_STREAM_WRITE_AFTER_END extends NodeError {
    constructor() {
        super("ERR_STREAM_WRITE_AFTER_END", `write after end`);
    }
}
export class ERR_SYNTHETIC extends NodeError {
    constructor() {
        super("ERR_SYNTHETIC", `JavaScript Callstack`);
    }
}
export class ERR_TLS_DH_PARAM_SIZE extends NodeError {
    constructor(x) {
        super("ERR_TLS_DH_PARAM_SIZE", `DH parameter size ${x} is less than 2048`);
    }
}
export class ERR_TLS_HANDSHAKE_TIMEOUT extends NodeError {
    constructor() {
        super("ERR_TLS_HANDSHAKE_TIMEOUT", `TLS handshake timeout`);
    }
}
export class ERR_TLS_INVALID_CONTEXT extends NodeTypeError {
    constructor(x) {
        super("ERR_TLS_INVALID_CONTEXT", `${x} must be a SecureContext`);
    }
}
export class ERR_TLS_INVALID_STATE extends NodeError {
    constructor() {
        super("ERR_TLS_INVALID_STATE", `TLS socket connection must be securely established`);
    }
}
export class ERR_TLS_INVALID_PROTOCOL_VERSION extends NodeTypeError {
    constructor(protocol, x) {
        super("ERR_TLS_INVALID_PROTOCOL_VERSION", `${protocol} is not a valid ${x} TLS protocol version`);
    }
}
export class ERR_TLS_PROTOCOL_VERSION_CONFLICT extends NodeTypeError {
    constructor(prevProtocol, protocol) {
        super("ERR_TLS_PROTOCOL_VERSION_CONFLICT", `TLS protocol version ${prevProtocol} conflicts with secureProtocol ${protocol}`);
    }
}
export class ERR_TLS_RENEGOTIATION_DISABLED extends NodeError {
    constructor() {
        super("ERR_TLS_RENEGOTIATION_DISABLED", `TLS session renegotiation disabled for this socket`);
    }
}
export class ERR_TLS_REQUIRED_SERVER_NAME extends NodeError {
    constructor() {
        super("ERR_TLS_REQUIRED_SERVER_NAME", `"servername" is required parameter for Server.addContext`);
    }
}
export class ERR_TLS_SESSION_ATTACK extends NodeError {
    constructor() {
        super("ERR_TLS_SESSION_ATTACK", `TLS session renegotiation attack detected`);
    }
}
export class ERR_TLS_SNI_FROM_SERVER extends NodeError {
    constructor() {
        super("ERR_TLS_SNI_FROM_SERVER", `Cannot issue SNI from a TLS server-side socket`);
    }
}
export class ERR_TRACE_EVENTS_CATEGORY_REQUIRED extends NodeTypeError {
    constructor() {
        super("ERR_TRACE_EVENTS_CATEGORY_REQUIRED", `At least one category is required`);
    }
}
export class ERR_TRACE_EVENTS_UNAVAILABLE extends NodeError {
    constructor() {
        super("ERR_TRACE_EVENTS_UNAVAILABLE", `Trace events are unavailable`);
    }
}
export class ERR_UNAVAILABLE_DURING_EXIT extends NodeError {
    constructor() {
        super("ERR_UNAVAILABLE_DURING_EXIT", `Cannot call function in process exit handler`);
    }
}
export class ERR_UNCAUGHT_EXCEPTION_CAPTURE_ALREADY_SET extends NodeError {
    constructor() {
        super("ERR_UNCAUGHT_EXCEPTION_CAPTURE_ALREADY_SET", "`process.setupUncaughtExceptionCapture()` was called while a capture callback was already active");
    }
}
export class ERR_UNESCAPED_CHARACTERS extends NodeTypeError {
    constructor(x) {
        super("ERR_UNESCAPED_CHARACTERS", `${x} contains unescaped characters`);
    }
}
export class ERR_UNKNOWN_BUILTIN_MODULE extends NodeError {
    constructor(x) {
        super("ERR_UNKNOWN_BUILTIN_MODULE", `No such built-in module: ${x}`);
    }
}
export class ERR_UNKNOWN_CREDENTIAL extends NodeError {
    constructor(x, y) {
        super("ERR_UNKNOWN_CREDENTIAL", `${x} identifier does not exist: ${y}`);
    }
}
export class ERR_UNKNOWN_ENCODING extends NodeTypeError {
    constructor(x) {
        super("ERR_UNKNOWN_ENCODING", `Unknown encoding: ${x}`);
    }
}
export class ERR_UNKNOWN_FILE_EXTENSION extends NodeTypeError {
    constructor(x, y) {
        super("ERR_UNKNOWN_FILE_EXTENSION", `Unknown file extension "${x}" for ${y}`);
    }
}
export class ERR_UNKNOWN_MODULE_FORMAT extends NodeRangeError {
    constructor(x) {
        super("ERR_UNKNOWN_MODULE_FORMAT", `Unknown module format: ${x}`);
    }
}
export class ERR_UNKNOWN_SIGNAL extends NodeTypeError {
    constructor(x) {
        super("ERR_UNKNOWN_SIGNAL", `Unknown signal: ${x}`);
    }
}
export class ERR_UNSUPPORTED_DIR_IMPORT extends NodeError {
    constructor(x, y) {
        super("ERR_UNSUPPORTED_DIR_IMPORT", `Directory import '${x}' is not supported resolving ES modules, imported from ${y}`);
    }
}
export class ERR_UNSUPPORTED_ESM_URL_SCHEME extends NodeError {
    constructor() {
        super("ERR_UNSUPPORTED_ESM_URL_SCHEME", `Only file and data URLs are supported by the default ESM loader`);
    }
}
export class ERR_V8BREAKITERATOR extends NodeError {
    constructor() {
        super("ERR_V8BREAKITERATOR", `Full ICU data not installed. See https://github.com/nodejs/node/wiki/Intl`);
    }
}
export class ERR_VALID_PERFORMANCE_ENTRY_TYPE extends NodeError {
    constructor() {
        super("ERR_VALID_PERFORMANCE_ENTRY_TYPE", `At least one valid performance entry type is required`);
    }
}
export class ERR_VM_DYNAMIC_IMPORT_CALLBACK_MISSING extends NodeTypeError {
    constructor() {
        super("ERR_VM_DYNAMIC_IMPORT_CALLBACK_MISSING", `A dynamic import callback was not specified.`);
    }
}
export class ERR_VM_MODULE_ALREADY_LINKED extends NodeError {
    constructor() {
        super("ERR_VM_MODULE_ALREADY_LINKED", `Module has already been linked`);
    }
}
export class ERR_VM_MODULE_CANNOT_CREATE_CACHED_DATA extends NodeError {
    constructor() {
        super("ERR_VM_MODULE_CANNOT_CREATE_CACHED_DATA", `Cached data cannot be created for a module which has been evaluated`);
    }
}
export class ERR_VM_MODULE_DIFFERENT_CONTEXT extends NodeError {
    constructor() {
        super("ERR_VM_MODULE_DIFFERENT_CONTEXT", `Linked modules must use the same context`);
    }
}
export class ERR_VM_MODULE_LINKING_ERRORED extends NodeError {
    constructor() {
        super("ERR_VM_MODULE_LINKING_ERRORED", `Linking has already failed for the provided module`);
    }
}
export class ERR_VM_MODULE_NOT_MODULE extends NodeError {
    constructor() {
        super("ERR_VM_MODULE_NOT_MODULE", `Provided module is not an instance of Module`);
    }
}
export class ERR_VM_MODULE_STATUS extends NodeError {
    constructor(x) {
        super("ERR_VM_MODULE_STATUS", `Module status ${x}`);
    }
}
export class ERR_WASI_ALREADY_STARTED extends NodeError {
    constructor() {
        super("ERR_WASI_ALREADY_STARTED", `WASI instance has already started`);
    }
}
export class ERR_WORKER_INIT_FAILED extends NodeError {
    constructor(x) {
        super("ERR_WORKER_INIT_FAILED", `Worker initialization failure: ${x}`);
    }
}
export class ERR_WORKER_NOT_RUNNING extends NodeError {
    constructor() {
        super("ERR_WORKER_NOT_RUNNING", `Worker instance not running`);
    }
}
export class ERR_WORKER_OUT_OF_MEMORY extends NodeError {
    constructor(x) {
        super("ERR_WORKER_OUT_OF_MEMORY", `Worker terminated due to reaching memory limit: ${x}`);
    }
}
export class ERR_WORKER_UNSERIALIZABLE_ERROR extends NodeError {
    constructor() {
        super("ERR_WORKER_UNSERIALIZABLE_ERROR", `Serializing an uncaught exception failed`);
    }
}
export class ERR_WORKER_UNSUPPORTED_EXTENSION extends NodeTypeError {
    constructor(x) {
        super("ERR_WORKER_UNSUPPORTED_EXTENSION", `The worker script extension must be ".js", ".mjs", or ".cjs". Received "${x}"`);
    }
}
export class ERR_WORKER_UNSUPPORTED_OPERATION extends NodeTypeError {
    constructor(x) {
        super("ERR_WORKER_UNSUPPORTED_OPERATION", `${x} is not supported in workers`);
    }
}
export class ERR_ZLIB_INITIALIZATION_FAILED extends NodeError {
    constructor() {
        super("ERR_ZLIB_INITIALIZATION_FAILED", `Initialization failed`);
    }
}
export class ERR_FALSY_VALUE_REJECTION extends NodeError {
    reason;
    constructor(reason) {
        super("ERR_FALSY_VALUE_REJECTION", "Promise was rejected with falsy value");
        this.reason = reason;
    }
}
export class ERR_HTTP2_INVALID_SETTING_VALUE extends NodeRangeError {
    actual;
    min;
    max;
    constructor(name, actual, min, max) {
        super("ERR_HTTP2_INVALID_SETTING_VALUE", `Invalid value for setting "${name}": ${actual}`);
        this.actual = actual;
        if (min !== undefined) {
            this.min = min;
            this.max = max;
        }
    }
}
export class ERR_HTTP2_STREAM_CANCEL extends NodeError {
    cause;
    constructor(error) {
        super("ERR_HTTP2_STREAM_CANCEL", typeof error.message === "string"
            ? `The pending stream has been canceled (caused by: ${error.message})`
            : "The pending stream has been canceled");
        if (error) {
            this.cause = error;
        }
    }
}
export class ERR_INVALID_ADDRESS_FAMILY extends NodeRangeError {
    host;
    port;
    constructor(addressType, host, port) {
        super("ERR_INVALID_ADDRESS_FAMILY", `Invalid address family: ${addressType} ${host}:${port}`);
        this.host = host;
        this.port = port;
    }
}
export class ERR_INVALID_CHAR extends NodeTypeError {
    constructor(name, field) {
        super("ERR_INVALID_CHAR", field
            ? `Invalid character in ${name}`
            : `Invalid character in ${name} ["${field}"]`);
    }
}
export class ERR_INVALID_OPT_VALUE extends NodeTypeError {
    constructor(name, value) {
        super("ERR_INVALID_OPT_VALUE", `The value "${value}" is invalid for option "${name}"`);
    }
}
export class ERR_INVALID_RETURN_PROPERTY extends NodeTypeError {
    constructor(input, name, prop, value) {
        super("ERR_INVALID_RETURN_PROPERTY", `Expected a valid ${input} to be returned for the "${prop}" from the "${name}" function but got ${value}.`);
    }
}
function buildReturnPropertyType(value) {
    if (value && value.constructor && value.constructor.name) {
        return `instance of ${value.constructor.name}`;
    }
    else {
        return `type ${typeof value}`;
    }
}
export class ERR_INVALID_RETURN_PROPERTY_VALUE extends NodeTypeError {
    constructor(input, name, prop, value) {
        super("ERR_INVALID_RETURN_PROPERTY_VALUE", `Expected ${input} to be returned for the "${prop}" from the "${name}" function but got ${buildReturnPropertyType(value)}.`);
    }
}
export class ERR_INVALID_RETURN_VALUE extends NodeTypeError {
    constructor(input, name, value) {
        super("ERR_INVALID_RETURN_VALUE", `Expected ${input} to be returned from the "${name}" function but got ${buildReturnPropertyType(value)}.`);
    }
}
export class ERR_INVALID_URL extends NodeTypeError {
    input;
    constructor(input) {
        super("ERR_INVALID_URL", `Invalid URL: ${input}`);
        this.input = input;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Vycm9ycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBc0JBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUNwRCxPQUFPLEVBQUUsT0FBTyxFQUFFLE1BQU0sV0FBVyxDQUFDO0FBS3BDLE1BQU0sV0FBVyxHQUFHLHFCQUFxQixDQUFDO0FBTTFDLE1BQU0sTUFBTSxHQUFHO0lBQ2IsUUFBUTtJQUNSLFVBQVU7SUFDVixRQUFRO0lBQ1IsUUFBUTtJQUVSLFVBQVU7SUFDVixRQUFRO0lBQ1IsU0FBUztJQUNULFFBQVE7SUFDUixRQUFRO0NBQ1QsQ0FBQztBQU1GLE1BQU0sT0FBTyxvQkFBcUIsU0FBUSxLQUFLO0lBQzdDLElBQUksQ0FBUztJQUViLFlBQVksSUFBWSxFQUFFLElBQVksRUFBRSxPQUFlO1FBQ3JELEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQztRQUNmLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBQ2pCLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxDQUFDO1FBR2pCLElBQUksQ0FBQyxLQUFLLEdBQUcsSUFBSSxDQUFDLEtBQUssSUFBSSxHQUFHLElBQUksS0FBSyxJQUFJLENBQUMsSUFBSSxJQUFJLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyxDQUFDLEVBQUUsQ0FBQyxFQUFFLENBQUM7SUFDN0UsQ0FBQztJQUVELFFBQVE7UUFDTixPQUFPLEdBQUcsSUFBSSxDQUFDLElBQUksS0FBSyxJQUFJLENBQUMsSUFBSSxNQUFNLElBQUksQ0FBQyxPQUFPLEVBQUUsQ0FBQztJQUN4RCxDQUFDO0NBQ0Y7QUFFRCxNQUFNLE9BQU8sU0FBVSxTQUFRLG9CQUFvQjtJQUNqRCxZQUFZLElBQVksRUFBRSxPQUFlO1FBQ3ZDLEtBQUssQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDLElBQUksRUFBRSxJQUFJLEVBQUUsT0FBTyxDQUFDLENBQUM7SUFD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