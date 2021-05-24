import { unreachable } from "../testing/asserts.ts";
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
    constructor(a1, a2, a3) {
        super("ERR_INVALID_ARG_TYPE", `The "${a1}" argument must be of type ${typeof a2 === "string"
            ? a2.toLocaleLowerCase()
            : a2.map((x) => x.toLocaleLowerCase()).join(", ")}. Received ${typeof a3} (${a3})`);
    }
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2Vycm9ycy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIl9lcnJvcnMudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBdUJBLE9BQU8sRUFBRSxXQUFXLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQU1wRCxNQUFNLE9BQU8sb0JBQXFCLFNBQVEsS0FBSztJQUM3QyxJQUFJLENBQVM7SUFFYixZQUFZLElBQVksRUFBRSxJQUFZLEVBQUUsT0FBZTtRQUNyRCxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7UUFDZixJQUFJLENBQUMsSUFBSSxHQUFHLElBQUksQ0FBQztRQUNqQixJQUFJLENBQUMsSUFBSSxHQUFHLElBQUksQ0FBQztRQUdqQixJQUFJLENBQUMsS0FBSyxHQUFHLElBQUksQ0FBQyxLQUFLLElBQUksR0FBRyxJQUFJLEtBQUssSUFBSSxDQUFDLElBQUksSUFBSSxJQUFJLENBQUMsS0FBSyxDQUFDLEtBQUssQ0FBQyxFQUFFLENBQUMsRUFBRSxDQUFDO0lBQzdFLENBQUM7SUFFRCxRQUFRO1FBQ04sT0FBTyxHQUFHLElBQUksQ0FBQyxJQUFJLEtBQUssSUFBSSxDQUFDLElBQUksTUFBTSxJQUFJLENBQUMsT0FBTyxFQUFFLENBQUM7SUFDeEQsQ0FBQztDQUNGO0FBRUQsTUFBTSxPQUFPLFNBQVUsU0FBUSxvQkFBb0I7SUFDakQsWUFBWSxJQUFZLEVBQUUsT0FBZTtRQUN2QyxLQUFLLENBQUMsS0FBSyxDQUFDLFNBQVMsQ0FBQyxJQUFJLEVBQUUsSUFBSSxFQUFFLE9BQU8sQ0FBQyxDQUFDO0lBQzdDLENBQUM7Q0FDRjtBQUVELE1BQU0sT0FBTyxlQUFnQixTQUFRLG9CQUFvQjtJQUV2RCxZQUFZLElBQVksRUFBRSxPQUFlO1FBQ3ZDLEtBQUssQ0FBQyxXQUFXLENBQUMsU0FBUyxDQUFDLElBQUksRUFBRSxJQUFJLEVBQUUsT0FBTyxDQUFDLENBQUM7UUFDakQsTUFBTSxDQUFDLGNBQWMsQ0FBQyxJQUFJLEVBQUUsV0FBVyxDQUFDLFNBQVMs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