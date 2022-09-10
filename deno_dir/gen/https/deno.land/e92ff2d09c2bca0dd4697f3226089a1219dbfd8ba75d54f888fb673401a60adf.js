// Copyright 2018-2022 the Deno authors. All rights reserved. MIT license.
// Copyright the Browserify authors. MIT License.
// Ported from https://github.com/browserify/path-browserify/
// This module is browser compatible.
import { CHAR_BACKWARD_SLASH, CHAR_COLON, CHAR_DOT, CHAR_QUESTION_MARK } from "./_constants.ts";
import { _format, assertPath, encodeWhitespace, isPathSeparator, isWindowsDeviceRoot, normalizeString } from "./_util.ts";
import { assert } from "../_util/assert.ts";
export const sep = "\\";
export const delimiter = ";";
/**
 * Resolves path segments into a `path`
 * @param pathSegments to process to path
 */ export function resolve(...pathSegments) {
    let resolvedDevice = "";
    let resolvedTail = "";
    let resolvedAbsolute = false;
    for(let i = pathSegments.length - 1; i >= -1; i--){
        let path;
        // deno-lint-ignore no-explicit-any
        const { Deno  } = globalThis;
        if (i >= 0) {
            path = pathSegments[i];
        } else if (!resolvedDevice) {
            if (typeof Deno?.cwd !== "function") {
                throw new TypeError("Resolved a drive-letter-less path without a CWD.");
            }
            path = Deno.cwd();
        } else {
            if (typeof Deno?.env?.get !== "function" || typeof Deno?.cwd !== "function") {
                throw new TypeError("Resolved a relative path without a CWD.");
            }
            path = Deno.cwd();
            // Verify that a cwd was found and that it actually points
            // to our drive. If not, default to the drive's root.
            if (path === undefined || path.slice(0, 3).toLowerCase() !== `${resolvedDevice.toLowerCase()}\\`) {
                path = `${resolvedDevice}\\`;
            }
        }
        assertPath(path);
        const len = path.length;
        // Skip empty entries
        if (len === 0) continue;
        let rootEnd = 0;
        let device = "";
        let isAbsolute = false;
        const code = path.charCodeAt(0);
        // Try to match a root
        if (len > 1) {
            if (isPathSeparator(code)) {
                // Possible UNC root
                // If we started with a separator, we know we at least have an
                // absolute path of some kind (UNC or otherwise)
                isAbsolute = true;
                if (isPathSeparator(path.charCodeAt(1))) {
                    // Matched double path separator at beginning
                    let j = 2;
                    let last = j;
                    // Match 1 or more non-path separators
                    for(; j < len; ++j){
                        if (isPathSeparator(path.charCodeAt(j))) break;
                    }
                    if (j < len && j !== last) {
                        const firstPart = path.slice(last, j);
                        // Matched!
                        last = j;
                        // Match 1 or more path separators
                        for(; j < len; ++j){
                            if (!isPathSeparator(path.charCodeAt(j))) break;
                        }
                        if (j < len && j !== last) {
                            // Matched!
                            last = j;
                            // Match 1 or more non-path separators
                            for(; j < len; ++j){
                                if (isPathSeparator(path.charCodeAt(j))) break;
                            }
                            if (j === len) {
                                // We matched a UNC root only
                                device = `\\\\${firstPart}\\${path.slice(last)}`;
                                rootEnd = j;
                            } else if (j !== last) {
                                // We matched a UNC root with leftovers
                                device = `\\\\${firstPart}\\${path.slice(last, j)}`;
                                rootEnd = j;
                            }
                        }
                    }
                } else {
                    rootEnd = 1;
                }
            } else if (isWindowsDeviceRoot(code)) {
                // Possible device root
                if (path.charCodeAt(1) === CHAR_COLON) {
                    device = path.slice(0, 2);
                    rootEnd = 2;
                    if (len > 2) {
                        if (isPathSeparator(path.charCodeAt(2))) {
                            // Treat separator following drive name as an absolute path
                            // indicator
                            isAbsolute = true;
                            rootEnd = 3;
                        }
                    }
                }
            }
        } else if (isPathSeparator(code)) {
            // `path` contains just a path separator
            rootEnd = 1;
            isAbsolute = true;
        }
        if (device.length > 0 && resolvedDevice.length > 0 && device.toLowerCase() !== resolvedDevice.toLowerCase()) {
            continue;
        }
        if (resolvedDevice.length === 0 && device.length > 0) {
            resolvedDevice = device;
        }
        if (!resolvedAbsolute) {
            resolvedTail = `${path.slice(rootEnd)}\\${resolvedTail}`;
            resolvedAbsolute = isAbsolute;
        }
        if (resolvedAbsolute && resolvedDevice.length > 0) break;
    }
    // At this point the path should be resolved to a full absolute path,
    // but handle relative paths to be safe (might happen when process.cwd()
    // fails)
    // Normalize the tail path
    resolvedTail = normalizeString(resolvedTail, !resolvedAbsolute, "\\", isPathSeparator);
    return resolvedDevice + (resolvedAbsolute ? "\\" : "") + resolvedTail || ".";
}
/**
 * Normalizes a `path`
 * @param path to normalize
 */ export function normalize(path) {
    assertPath(path);
    const len = path.length;
    if (len === 0) return ".";
    let rootEnd = 0;
    let device;
    let isAbsolute = false;
    const code = path.charCodeAt(0);
    // Try to match a root
    if (len > 1) {
        if (isPathSeparator(code)) {
            // Possible UNC root
            // If we started with a separator, we know we at least have an absolute
            // path of some kind (UNC or otherwise)
            isAbsolute = true;
            if (isPathSeparator(path.charCodeAt(1))) {
                // Matched double path separator at beginning
                let j = 2;
                let last = j;
                // Match 1 or more non-path separators
                for(; j < len; ++j){
                    if (isPathSeparator(path.charCodeAt(j))) break;
                }
                if (j < len && j !== last) {
                    const firstPart = path.slice(last, j);
                    // Matched!
                    last = j;
                    // Match 1 or more path separators
                    for(; j < len; ++j){
                        if (!isPathSeparator(path.charCodeAt(j))) break;
                    }
                    if (j < len && j !== last) {
                        // Matched!
                        last = j;
                        // Match 1 or more non-path separators
                        for(; j < len; ++j){
                            if (isPathSeparator(path.charCodeAt(j))) break;
                        }
                        if (j === len) {
                            // We matched a UNC root only
                            // Return the normalized version of the UNC root since there
                            // is nothing left to process
                            return `\\\\${firstPart}\\${path.slice(last)}\\`;
                        } else if (j !== last) {
                            // We matched a UNC root with leftovers
                            device = `\\\\${firstPart}\\${path.slice(last, j)}`;
                            rootEnd = j;
                        }
                    }
                }
            } else {
                rootEnd = 1;
            }
        } else if (isWindowsDeviceRoot(code)) {
            // Possible device root
            if (path.charCodeAt(1) === CHAR_COLON) {
                device = path.slice(0, 2);
                rootEnd = 2;
                if (len > 2) {
                    if (isPathSeparator(path.charCodeAt(2))) {
                        // Treat separator following drive name as an absolute path
                        // indicator
                        isAbsolute = true;
                        rootEnd = 3;
                    }
                }
            }
        }
    } else if (isPathSeparator(code)) {
        // `path` contains just a path separator, exit early to avoid unnecessary
        // work
        return "\\";
    }
    let tail;
    if (rootEnd < len) {
        tail = normalizeString(path.slice(rootEnd), !isAbsolute, "\\", isPathSeparator);
    } else {
        tail = "";
    }
    if (tail.length === 0 && !isAbsolute) tail = ".";
    if (tail.length > 0 && isPathSeparator(path.charCodeAt(len - 1))) {
        tail += "\\";
    }
    if (device === undefined) {
        if (isAbsolute) {
            if (tail.length > 0) return `\\${tail}`;
            else return "\\";
        } else if (tail.length > 0) {
            return tail;
        } else {
            return "";
        }
    } else if (isAbsolute) {
        if (tail.length > 0) return `${device}\\${tail}`;
        else return `${device}\\`;
    } else if (tail.length > 0) {
        return device + tail;
    } else {
        return device;
    }
}
/**
 * Verifies whether path is absolute
 * @param path to verify
 */ export function isAbsolute(path) {
    assertPath(path);
    const len = path.length;
    if (len === 0) return false;
    const code = path.charCodeAt(0);
    if (isPathSeparator(code)) {
        return true;
    } else if (isWindowsDeviceRoot(code)) {
        // Possible device root
        if (len > 2 && path.charCodeAt(1) === CHAR_COLON) {
            if (isPathSeparator(path.charCodeAt(2))) return true;
        }
    }
    return false;
}
/**
 * Join all given a sequence of `paths`,then normalizes the resulting path.
 * @param paths to be joined and normalized
 */ export function join(...paths) {
    const pathsCount = paths.length;
    if (pathsCount === 0) return ".";
    let joined;
    let firstPart = null;
    for(let i = 0; i < pathsCount; ++i){
        const path = paths[i];
        assertPath(path);
        if (path.length > 0) {
            if (joined === undefined) joined = firstPart = path;
            else joined += `\\${path}`;
        }
    }
    if (joined === undefined) return ".";
    // Make sure that the joined path doesn't start with two slashes, because
    // normalize() will mistake it for an UNC path then.
    //
    // This step is skipped when it is very clear that the user actually
    // intended to point at an UNC path. This is assumed when the first
    // non-empty string arguments starts with exactly two slashes followed by
    // at least one more non-slash character.
    //
    // Note that for normalize() to treat a path as an UNC path it needs to
    // have at least 2 components, so we don't filter for that here.
    // This means that the user can use join to construct UNC paths from
    // a server name and a share name; for example:
    //   path.join('//server', 'share') -> '\\\\server\\share\\')
    let needsReplace = true;
    let slashCount = 0;
    assert(firstPart != null);
    if (isPathSeparator(firstPart.charCodeAt(0))) {
        ++slashCount;
        const firstLen = firstPart.length;
        if (firstLen > 1) {
            if (isPathSeparator(firstPart.charCodeAt(1))) {
                ++slashCount;
                if (firstLen > 2) {
                    if (isPathSeparator(firstPart.charCodeAt(2))) ++slashCount;
                    else {
                        // We matched a UNC path in the first part
                        needsReplace = false;
                    }
                }
            }
        }
    }
    if (needsReplace) {
        // Find any more consecutive slashes we need to replace
        for(; slashCount < joined.length; ++slashCount){
            if (!isPathSeparator(joined.charCodeAt(slashCount))) break;
        }
        // Replace the slashes if needed
        if (slashCount >= 2) joined = `\\${joined.slice(slashCount)}`;
    }
    return normalize(joined);
}
/**
 * It will solve the relative path from `from` to `to`, for instance:
 *  from = 'C:\\orandea\\test\\aaa'
 *  to = 'C:\\orandea\\impl\\bbb'
 * The output of the function should be: '..\\..\\impl\\bbb'
 * @param from relative path
 * @param to relative path
 */ export function relative(from, to) {
    assertPath(from);
    assertPath(to);
    if (from === to) return "";
    const fromOrig = resolve(from);
    const toOrig = resolve(to);
    if (fromOrig === toOrig) return "";
    from = fromOrig.toLowerCase();
    to = toOrig.toLowerCase();
    if (from === to) return "";
    // Trim any leading backslashes
    let fromStart = 0;
    let fromEnd = from.length;
    for(; fromStart < fromEnd; ++fromStart){
        if (from.charCodeAt(fromStart) !== CHAR_BACKWARD_SLASH) break;
    }
    // Trim trailing backslashes (applicable to UNC paths only)
    for(; fromEnd - 1 > fromStart; --fromEnd){
        if (from.charCodeAt(fromEnd - 1) !== CHAR_BACKWARD_SLASH) break;
    }
    const fromLen = fromEnd - fromStart;
    // Trim any leading backslashes
    let toStart = 0;
    let toEnd = to.length;
    for(; toStart < toEnd; ++toStart){
        if (to.charCodeAt(toStart) !== CHAR_BACKWARD_SLASH) break;
    }
    // Trim trailing backslashes (applicable to UNC paths only)
    for(; toEnd - 1 > toStart; --toEnd){
        if (to.charCodeAt(toEnd - 1) !== CHAR_BACKWARD_SLASH) break;
    }
    const toLen = toEnd - toStart;
    // Compare paths to find the longest common path from root
    const length = fromLen < toLen ? fromLen : toLen;
    let lastCommonSep = -1;
    let i = 0;
    for(; i <= length; ++i){
        if (i === length) {
            if (toLen > length) {
                if (to.charCodeAt(toStart + i) === CHAR_BACKWARD_SLASH) {
                    // We get here if `from` is the exact base path for `to`.
                    // For example: from='C:\\foo\\bar'; to='C:\\foo\\bar\\baz'
                    return toOrig.slice(toStart + i + 1);
                } else if (i === 2) {
                    // We get here if `from` is the device root.
                    // For example: from='C:\\'; to='C:\\foo'
                    return toOrig.slice(toStart + i);
                }
            }
            if (fromLen > length) {
                if (from.charCodeAt(fromStart + i) === CHAR_BACKWARD_SLASH) {
                    // We get here if `to` is the exact base path for `from`.
                    // For example: from='C:\\foo\\bar'; to='C:\\foo'
                    lastCommonSep = i;
                } else if (i === 2) {
                    // We get here if `to` is the device root.
                    // For example: from='C:\\foo\\bar'; to='C:\\'
                    lastCommonSep = 3;
                }
            }
            break;
        }
        const fromCode = from.charCodeAt(fromStart + i);
        const toCode = to.charCodeAt(toStart + i);
        if (fromCode !== toCode) break;
        else if (fromCode === CHAR_BACKWARD_SLASH) lastCommonSep = i;
    }
    // We found a mismatch before the first common path separator was seen, so
    // return the original `to`.
    if (i !== length && lastCommonSep === -1) {
        return toOrig;
    }
    let out = "";
    if (lastCommonSep === -1) lastCommonSep = 0;
    // Generate the relative path based on the path difference between `to` and
    // `from`
    for(i = fromStart + lastCommonSep + 1; i <= fromEnd; ++i){
        if (i === fromEnd || from.charCodeAt(i) === CHAR_BACKWARD_SLASH) {
            if (out.length === 0) out += "..";
            else out += "\\..";
        }
    }
    // Lastly, append the rest of the destination (`to`) path that comes after
    // the common path parts
    if (out.length > 0) {
        return out + toOrig.slice(toStart + lastCommonSep, toEnd);
    } else {
        toStart += lastCommonSep;
        if (toOrig.charCodeAt(toStart) === CHAR_BACKWARD_SLASH) ++toStart;
        return toOrig.slice(toStart, toEnd);
    }
}
/**
 * Resolves path to a namespace path
 * @param path to resolve to namespace
 */ export function toNamespacedPath(path) {
    // Note: this will *probably* throw somewhere.
    if (typeof path !== "string") return path;
    if (path.length === 0) return "";
    const resolvedPath = resolve(path);
    if (resolvedPath.length >= 3) {
        if (resolvedPath.charCodeAt(0) === CHAR_BACKWARD_SLASH) {
            // Possible UNC root
            if (resolvedPath.charCodeAt(1) === CHAR_BACKWARD_SLASH) {
                const code = resolvedPath.charCodeAt(2);
                if (code !== CHAR_QUESTION_MARK && code !== CHAR_DOT) {
                    // Matched non-long UNC root, convert the path to a long UNC path
                    return `\\\\?\\UNC\\${resolvedPath.slice(2)}`;
                }
            }
        } else if (isWindowsDeviceRoot(resolvedPath.charCodeAt(0))) {
            // Possible device root
            if (resolvedPath.charCodeAt(1) === CHAR_COLON && resolvedPath.charCodeAt(2) === CHAR_BACKWARD_SLASH) {
                // Matched device root, convert the path to a long UNC path
                return `\\\\?\\${resolvedPath}`;
            }
        }
    }
    return path;
}
/**
 * Return the directory name of a `path`.
 * @param path to determine name for
 */ export function dirname(path) {
    assertPath(path);
    const len = path.length;
    if (len === 0) return ".";
    let rootEnd = -1;
    let end = -1;
    let matchedSlash = true;
    let offset = 0;
    const code = path.charCodeAt(0);
    // Try to match a root
    if (len > 1) {
        if (isPathSeparator(code)) {
            // Possible UNC root
            rootEnd = offset = 1;
            if (isPathSeparator(path.charCodeAt(1))) {
                // Matched double path separator at beginning
                let j = 2;
                let last = j;
                // Match 1 or more non-path separators
                for(; j < len; ++j){
                    if (isPathSeparator(path.charCodeAt(j))) break;
                }
                if (j < len && j !== last) {
                    // Matched!
                    last = j;
                    // Match 1 or more path separators
                    for(; j < len; ++j){
                        if (!isPathSeparator(path.charCodeAt(j))) break;
                    }
                    if (j < len && j !== last) {
                        // Matched!
                        last = j;
                        // Match 1 or more non-path separators
                        for(; j < len; ++j){
                            if (isPathSeparator(path.charCodeAt(j))) break;
                        }
                        if (j === len) {
                            // We matched a UNC root only
                            return path;
                        }
                        if (j !== last) {
                            // We matched a UNC root with leftovers
                            // Offset by 1 to include the separator after the UNC root to
                            // treat it as a "normal root" on top of a (UNC) root
                            rootEnd = offset = j + 1;
                        }
                    }
                }
            }
        } else if (isWindowsDeviceRoot(code)) {
            // Possible device root
            if (path.charCodeAt(1) === CHAR_COLON) {
                rootEnd = offset = 2;
                if (len > 2) {
                    if (isPathSeparator(path.charCodeAt(2))) rootEnd = offset = 3;
                }
            }
        }
    } else if (isPathSeparator(code)) {
        // `path` contains just a path separator, exit early to avoid
        // unnecessary work
        return path;
    }
    for(let i = len - 1; i >= offset; --i){
        if (isPathSeparator(path.charCodeAt(i))) {
            if (!matchedSlash) {
                end = i;
                break;
            }
        } else {
            // We saw the first non-path separator
            matchedSlash = false;
        }
    }
    if (end === -1) {
        if (rootEnd === -1) return ".";
        else end = rootEnd;
    }
    return path.slice(0, end);
}
/**
 * Return the last portion of a `path`. Trailing directory separators are ignored.
 * @param path to process
 * @param ext of path directory
 */ export function basename(path, ext = "") {
    if (ext !== undefined && typeof ext !== "string") {
        throw new TypeError('"ext" argument must be a string');
    }
    assertPath(path);
    let start = 0;
    let end = -1;
    let matchedSlash = true;
    let i;
    // Check for a drive letter prefix so as not to mistake the following
    // path separator as an extra separator at the end of the path that can be
    // disregarded
    if (path.length >= 2) {
        const drive = path.charCodeAt(0);
        if (isWindowsDeviceRoot(drive)) {
            if (path.charCodeAt(1) === CHAR_COLON) start = 2;
        }
    }
    if (ext !== undefined && ext.length > 0 && ext.length <= path.length) {
        if (ext.length === path.length && ext === path) return "";
        let extIdx = ext.length - 1;
        let firstNonSlashEnd = -1;
        for(i = path.length - 1; i >= start; --i){
            const code = path.charCodeAt(i);
            if (isPathSeparator(code)) {
                // If we reached a path separator that was not part of a set of path
                // separators at the end of the string, stop now
                if (!matchedSlash) {
                    start = i + 1;
                    break;
                }
            } else {
                if (firstNonSlashEnd === -1) {
                    // We saw the first non-path separator, remember this index in case
                    // we need it if the extension ends up not matching
                    matchedSlash = false;
                    firstNonSlashEnd = i + 1;
                }
                if (extIdx >= 0) {
                    // Try to match the explicit extension
                    if (code === ext.charCodeAt(extIdx)) {
                        if (--extIdx === -1) {
                            // We matched the extension, so mark this as the end of our path
                            // component
                            end = i;
                        }
                    } else {
                        // Extension does not match, so our result is the entire path
                        // component
                        extIdx = -1;
                        end = firstNonSlashEnd;
                    }
                }
            }
        }
        if (start === end) end = firstNonSlashEnd;
        else if (end === -1) end = path.length;
        return path.slice(start, end);
    } else {
        for(i = path.length - 1; i >= start; --i){
            if (isPathSeparator(path.charCodeAt(i))) {
                // If we reached a path separator that was not part of a set of path
                // separators at the end of the string, stop now
                if (!matchedSlash) {
                    start = i + 1;
                    break;
                }
            } else if (end === -1) {
                // We saw the first non-path separator, mark this as the end of our
                // path component
                matchedSlash = false;
                end = i + 1;
            }
        }
        if (end === -1) return "";
        return path.slice(start, end);
    }
}
/**
 * Return the extension of the `path` with leading period.
 * @param path with extension
 * @returns extension (ex. for `file.ts` returns `.ts`)
 */ export function extname(path) {
    assertPath(path);
    let start = 0;
    let startDot = -1;
    let startPart = 0;
    let end = -1;
    let matchedSlash = true;
    // Track the state of characters (if any) we see before our first dot and
    // after any path separator we find
    let preDotState = 0;
    // Check for a drive letter prefix so as not to mistake the following
    // path separator as an extra separator at the end of the path that can be
    // disregarded
    if (path.length >= 2 && path.charCodeAt(1) === CHAR_COLON && isWindowsDeviceRoot(path.charCodeAt(0))) {
        start = startPart = 2;
    }
    for(let i = path.length - 1; i >= start; --i){
        const code = path.charCodeAt(i);
        if (isPathSeparator(code)) {
            // If we reached a path separator that was not part of a set of path
            // separators at the end of the string, stop now
            if (!matchedSlash) {
                startPart = i + 1;
                break;
            }
            continue;
        }
        if (end === -1) {
            // We saw the first non-path separator, mark this as the end of our
            // extension
            matchedSlash = false;
            end = i + 1;
        }
        if (code === CHAR_DOT) {
            // If this is our first dot, mark it as the start of our extension
            if (startDot === -1) startDot = i;
            else if (preDotState !== 1) preDotState = 1;
        } else if (startDot !== -1) {
            // We saw a non-dot and non-path separator before our dot, so we should
            // have a good chance at having a non-empty extension
            preDotState = -1;
        }
    }
    if (startDot === -1 || end === -1 || // We saw a non-dot character immediately before the dot
    preDotState === 0 || // The (right-most) trimmed path component is exactly '..'
    (preDotState === 1 && startDot === end - 1 && startDot === startPart + 1)) {
        return "";
    }
    return path.slice(startDot, end);
}
/**
 * Generate a path from `FormatInputPathObject` object.
 * @param pathObject with path
 */ export function format(pathObject) {
    if (pathObject === null || typeof pathObject !== "object") {
        throw new TypeError(`The "pathObject" argument must be of type Object. Received type ${typeof pathObject}`);
    }
    return _format("\\", pathObject);
}
/**
 * Return a `ParsedPath` object of the `path`.
 * @param path to process
 */ export function parse(path) {
    assertPath(path);
    const ret = {
        root: "",
        dir: "",
        base: "",
        ext: "",
        name: ""
    };
    const len = path.length;
    if (len === 0) return ret;
    let rootEnd = 0;
    let code = path.charCodeAt(0);
    // Try to match a root
    if (len > 1) {
        if (isPathSeparator(code)) {
            // Possible UNC root
            rootEnd = 1;
            if (isPathSeparator(path.charCodeAt(1))) {
                // Matched double path separator at beginning
                let j = 2;
                let last = j;
                // Match 1 or more non-path separators
                for(; j < len; ++j){
                    if (isPathSeparator(path.charCodeAt(j))) break;
                }
                if (j < len && j !== last) {
                    // Matched!
                    last = j;
                    // Match 1 or more path separators
                    for(; j < len; ++j){
                        if (!isPathSeparator(path.charCodeAt(j))) break;
                    }
                    if (j < len && j !== last) {
                        // Matched!
                        last = j;
                        // Match 1 or more non-path separators
                        for(; j < len; ++j){
                            if (isPathSeparator(path.charCodeAt(j))) break;
                        }
                        if (j === len) {
                            // We matched a UNC root only
                            rootEnd = j;
                        } else if (j !== last) {
                            // We matched a UNC root with leftovers
                            rootEnd = j + 1;
                        }
                    }
                }
            }
        } else if (isWindowsDeviceRoot(code)) {
            // Possible device root
            if (path.charCodeAt(1) === CHAR_COLON) {
                rootEnd = 2;
                if (len > 2) {
                    if (isPathSeparator(path.charCodeAt(2))) {
                        if (len === 3) {
                            // `path` contains just a drive root, exit early to avoid
                            // unnecessary work
                            ret.root = ret.dir = path;
                            return ret;
                        }
                        rootEnd = 3;
                    }
                } else {
                    // `path` contains just a drive root, exit early to avoid
                    // unnecessary work
                    ret.root = ret.dir = path;
                    return ret;
                }
            }
        }
    } else if (isPathSeparator(code)) {
        // `path` contains just a path separator, exit early to avoid
        // unnecessary work
        ret.root = ret.dir = path;
        return ret;
    }
    if (rootEnd > 0) ret.root = path.slice(0, rootEnd);
    let startDot = -1;
    let startPart = rootEnd;
    let end = -1;
    let matchedSlash = true;
    let i = path.length - 1;
    // Track the state of characters (if any) we see before our first dot and
    // after any path separator we find
    let preDotState = 0;
    // Get non-dir info
    for(; i >= rootEnd; --i){
        code = path.charCodeAt(i);
        if (isPathSeparator(code)) {
            // If we reached a path separator that was not part of a set of path
            // separators at the end of the string, stop now
            if (!matchedSlash) {
                startPart = i + 1;
                break;
            }
            continue;
        }
        if (end === -1) {
            // We saw the first non-path separator, mark this as the end of our
            // extension
            matchedSlash = false;
            end = i + 1;
        }
        if (code === CHAR_DOT) {
            // If this is our first dot, mark it as the start of our extension
            if (startDot === -1) startDot = i;
            else if (preDotState !== 1) preDotState = 1;
        } else if (startDot !== -1) {
            // We saw a non-dot and non-path separator before our dot, so we should
            // have a good chance at having a non-empty extension
            preDotState = -1;
        }
    }
    if (startDot === -1 || end === -1 || // We saw a non-dot character immediately before the dot
    preDotState === 0 || // The (right-most) trimmed path component is exactly '..'
    (preDotState === 1 && startDot === end - 1 && startDot === startPart + 1)) {
        if (end !== -1) {
            ret.base = ret.name = path.slice(startPart, end);
        }
    } else {
        ret.name = path.slice(startPart, startDot);
        ret.base = path.slice(startPart, end);
        ret.ext = path.slice(startDot, end);
    }
    // If the directory is the root, use the entire root as the `dir` including
    // the trailing slash if any (`C:\abc` -> `C:\`). Otherwise, strip out the
    // trailing slash (`C:\abc\def` -> `C:\abc`).
    if (startPart > 0 && startPart !== rootEnd) {
        ret.dir = path.slice(0, startPart - 1);
    } else ret.dir = ret.root;
    return ret;
}
/**
 * Converts a file URL to a path string.
 *
 * ```ts
 *      import { fromFileUrl } from "./win32.ts";
 *      fromFileUrl("file:///home/foo"); // "\\home\\foo"
 *      fromFileUrl("file:///C:/Users/foo"); // "C:\\Users\\foo"
 *      fromFileUrl("file://localhost/home/foo"); // "\\\\localhost\\home\\foo"
 * ```
 * @param url of a file URL
 */ export function fromFileUrl(url) {
    url = url instanceof URL ? url : new URL(url);
    if (url.protocol != "file:") {
        throw new TypeError("Must be a file URL.");
    }
    let path = decodeURIComponent(url.pathname.replace(/\//g, "\\").replace(/%(?![0-9A-Fa-f]{2})/g, "%25")).replace(/^\\*([A-Za-z]:)(\\|$)/, "$1\\");
    if (url.hostname != "") {
        // Note: The `URL` implementation guarantees that the drive letter and
        // hostname are mutually exclusive. Otherwise it would not have been valid
        // to append the hostname and path like this.
        path = `\\\\${url.hostname}${path}`;
    }
    return path;
}
/**
 * Converts a path string to a file URL.
 *
 * ```ts
 *      import { toFileUrl } from "./win32.ts";
 *      toFileUrl("\\home\\foo"); // new URL("file:///home/foo")
 *      toFileUrl("C:\\Users\\foo"); // new URL("file:///C:/Users/foo")
 *      toFileUrl("\\\\127.0.0.1\\home\\foo"); // new URL("file://127.0.0.1/home/foo")
 * ```
 * @param path to convert to file URL
 */ export function toFileUrl(path) {
    if (!isAbsolute(path)) {
        throw new TypeError("Must be an absolute path.");
    }
    const [, hostname, pathname] = path.match(/^(?:[/\\]{2}([^/\\]+)(?=[/\\](?:[^/\\]|$)))?(.*)/);
    const url = new URL("file:///");
    url.pathname = encodeWhitespace(pathname.replace(/%/g, "%25"));
    if (hostname != null && hostname != "localhost") {
        url.hostname = hostname;
        if (!url.hostname) {
            throw new TypeError("Invalid hostname.");
        }
    }
    return url;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3N0ZEAwLjE0NC4wL3BhdGgvd2luMzIudHMiXSwic291cmNlc0NvbnRlbnQiOlsiLy8gQ29weXJpZ2h0IDIwMTgtMjAyMiB0aGUgRGVubyBhdXRob3JzLiBBbGwgcmlnaHRzIHJlc2VydmVkLiBNSVQgbGljZW5zZS5cbi8vIENvcHlyaWdodCB0aGUgQnJvd3NlcmlmeSBhdXRob3JzLiBNSVQgTGljZW5zZS5cbi8vIFBvcnRlZCBmcm9tIGh0dHBzOi8vZ2l0aHViLmNvbS9icm93c2VyaWZ5L3BhdGgtYnJvd3NlcmlmeS9cbi8vIFRoaXMgbW9kdWxlIGlzIGJyb3dzZXIgY29tcGF0aWJsZS5cblxuaW1wb3J0IHR5cGUgeyBGb3JtYXRJbnB1dFBhdGhPYmplY3QsIFBhcnNlZFBhdGggfSBmcm9tIFwiLi9faW50ZXJmYWNlLnRzXCI7XG5pbXBvcnQge1xuICBDSEFSX0JBQ0tXQVJEX1NMQVNILFxuICBDSEFSX0NPTE9OLFxuICBDSEFSX0RPVCxcbiAgQ0hBUl9RVUVTVElPTl9NQVJLLFxufSBmcm9tIFwiLi9fY29uc3RhbnRzLnRzXCI7XG5cbmltcG9ydCB7XG4gIF9mb3JtYXQsXG4gIGFzc2VydFBhdGgsXG4gIGVuY29kZVdoaXRlc3BhY2UsXG4gIGlzUGF0aFNlcGFyYXRvcixcbiAgaXNXaW5kb3dzRGV2aWNlUm9vdCxcbiAgbm9ybWFsaXplU3RyaW5nLFxufSBmcm9tIFwiLi9fdXRpbC50c1wiO1xuaW1wb3J0IHsgYXNzZXJ0IH0gZnJvbSBcIi4uL191dGlsL2Fzc2VydC50c1wiO1xuXG5leHBvcnQgY29uc3Qgc2VwID0gXCJcXFxcXCI7XG5leHBvcnQgY29uc3QgZGVsaW1pdGVyID0gXCI7XCI7XG5cbi8qKlxuICogUmVzb2x2ZXMgcGF0aCBzZWdtZW50cyBpbnRvIGEgYHBhdGhgXG4gKiBAcGFyYW0gcGF0aFNlZ21lbnRzIHRvIHByb2Nlc3MgdG8gcGF0aFxuICovXG5leHBvcnQgZnVuY3Rpb24gcmVzb2x2ZSguLi5wYXRoU2VnbWVudHM6IHN0cmluZ1tdKTogc3RyaW5nIHtcbiAgbGV0IHJlc29sdmVkRGV2aWNlID0gXCJcIjtcbiAgbGV0IHJlc29sdmVkVGFpbCA9IFwiXCI7XG4gIGxldCByZXNvbHZlZEFic29sdXRlID0gZmFsc2U7XG5cbiAgZm9yIChsZXQgaSA9IHBhdGhTZWdtZW50cy5sZW5ndGggLSAxOyBpID49IC0xOyBpLS0pIHtcbiAgICBsZXQgcGF0aDogc3RyaW5nO1xuICAgIC8vIGRlbm8tbGludC1pZ25vcmUgbm8tZXhwbGljaXQtYW55XG4gICAgY29uc3QgeyBEZW5vIH0gPSBnbG9iYWxUaGlzIGFzIGFueTtcbiAgICBpZiAoaSA+PSAwKSB7XG4gICAgICBwYXRoID0gcGF0aFNlZ21lbnRzW2ldO1xuICAgIH0gZWxzZSBpZiAoIXJlc29sdmVkRGV2aWNlKSB7XG4gICAgICBpZiAodHlwZW9mIERlbm8/LmN3ZCAhPT0gXCJmdW5jdGlvblwiKSB7XG4gICAgICAgIHRocm93IG5ldyBUeXBlRXJyb3IoXCJSZXNvbHZlZCBhIGRyaXZlLWxldHRlci1sZXNzIHBhdGggd2l0aG91dCBhIENXRC5cIik7XG4gICAgICB9XG4gICAgICBwYXRoID0gRGVuby5jd2QoKTtcbiAgICB9IGVsc2Uge1xuICAgICAgaWYgKFxuICAgICAgICB0eXBlb2YgRGVubz8uZW52Py5nZXQgIT09IFwiZnVuY3Rpb25cIiB8fCB0eXBlb2YgRGVubz8uY3dkICE9PSBcImZ1bmN0aW9uXCJcbiAgICAgICkge1xuICAgICAgICB0aHJvdyBuZXcgVHlwZUVycm9yKFwiUmVzb2x2ZWQgYSByZWxhdGl2ZSBwYXRoIHdpdGhvdXQgYSBDV0QuXCIpO1xuICAgICAgfVxuICAgICAgcGF0aCA9IERlbm8uY3dkKCk7XG5cbiAgICAgIC8vIFZlcmlmeSB0aGF0IGEgY3dkIHdhcyBmb3VuZCBhbmQgdGhhdCBpdCBhY3R1YWxseSBwb2ludHNcbiAgICAgIC8vIHRvIG91ciBkcml2ZS4gSWYgbm90LCBkZWZhdWx0IHRvIHRoZSBkcml2ZSdzIHJvb3QuXG4gICAgICBpZiAoXG4gICAgICAgIHBhdGggPT09IHVuZGVmaW5lZCB8fFxuICAgICAgICBwYXRoLnNsaWNlKDAsIDMpLnRvTG93ZXJDYXNlKCkgIT09IGAke3Jlc29sdmVkRGV2aWNlLnRvTG93ZXJDYXNlKCl9XFxcXGBcbiAgICAgICkge1xuICAgICAgICBwYXRoID0gYCR7cmVzb2x2ZWREZXZpY2V9XFxcXGA7XG4gICAgICB9XG4gICAgfVxuXG4gICAgYXNzZXJ0UGF0aChwYXRoKTtcblxuICAgIGNvbnN0IGxlbiA9IHBhdGgubGVuZ3RoO1xuXG4gICAgLy8gU2tpcCBlbXB0eSBlbnRyaWVzXG4gICAgaWYgKGxlbiA9PT0gMCkgY29udGludWU7XG5cbiAgICBsZXQgcm9vdEVuZCA9IDA7XG4gICAgbGV0IGRldmljZSA9IFwiXCI7XG4gICAgbGV0IGlzQWJzb2x1dGUgPSBmYWxzZTtcbiAgICBjb25zdCBjb2RlID0gcGF0aC5jaGFyQ29kZUF0KDApO1xuXG4gICAgLy8gVHJ5IHRvIG1hdGNoIGEgcm9vdFxuICAgIGlmIChsZW4gPiAxKSB7XG4gICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgICAgIC8vIFBvc3NpYmxlIFVOQyByb290XG5cbiAgICAgICAgLy8gSWYgd2Ugc3RhcnRlZCB3aXRoIGEgc2VwYXJhdG9yLCB3ZSBrbm93IHdlIGF0IGxlYXN0IGhhdmUgYW5cbiAgICAgICAgLy8gYWJzb2x1dGUgcGF0aCBvZiBzb21lIGtpbmQgKFVOQyBvciBvdGhlcndpc2UpXG4gICAgICAgIGlzQWJzb2x1dGUgPSB0cnVlO1xuXG4gICAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KDEpKSkge1xuICAgICAgICAgIC8vIE1hdGNoZWQgZG91YmxlIHBhdGggc2VwYXJhdG9yIGF0IGJlZ2lubmluZ1xuICAgICAgICAgIGxldCBqID0gMjtcbiAgICAgICAgICBsZXQgbGFzdCA9IGo7XG4gICAgICAgICAgLy8gTWF0Y2ggMSBvciBtb3JlIG5vbi1wYXRoIHNlcGFyYXRvcnNcbiAgICAgICAgICBmb3IgKDsgaiA8IGxlbjsgKytqKSB7XG4gICAgICAgICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdChqKSkpIGJyZWFrO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoaiA8IGxlbiAmJiBqICE9PSBsYXN0KSB7XG4gICAgICAgICAgICBjb25zdCBmaXJzdFBhcnQgPSBwYXRoLnNsaWNlKGxhc3QsIGopO1xuICAgICAgICAgICAgLy8gTWF0Y2hlZCFcbiAgICAgICAgICAgIGxhc3QgPSBqO1xuICAgICAgICAgICAgLy8gTWF0Y2ggMSBvciBtb3JlIHBhdGggc2VwYXJhdG9yc1xuICAgICAgICAgICAgZm9yICg7IGogPCBsZW47ICsraikge1xuICAgICAgICAgICAgICBpZiAoIWlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoaikpKSBicmVhaztcbiAgICAgICAgICAgIH1cbiAgICAgICAgICAgIGlmIChqIDwgbGVuICYmIGogIT09IGxhc3QpIHtcbiAgICAgICAgICAgICAgLy8gTWF0Y2hlZCFcbiAgICAgICAgICAgICAgbGFzdCA9IGo7XG4gICAgICAgICAgICAgIC8vIE1hdGNoIDEgb3IgbW9yZSBub24tcGF0aCBzZXBhcmF0b3JzXG4gICAgICAgICAgICAgIGZvciAoOyBqIDwgbGVuOyArK2opIHtcbiAgICAgICAgICAgICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdChqKSkpIGJyZWFrO1xuICAgICAgICAgICAgICB9XG4gICAgICAgICAgICAgIGlmIChqID09PSBsZW4pIHtcbiAgICAgICAgICAgICAgICAvLyBXZSBtYXRjaGVkIGEgVU5DIHJvb3Qgb25seVxuICAgICAgICAgICAgICAgIGRldmljZSA9IGBcXFxcXFxcXCR7Zmlyc3RQYXJ0fVxcXFwke3BhdGguc2xpY2UobGFzdCl9YDtcbiAgICAgICAgICAgICAgICByb290RW5kID0gajtcbiAgICAgICAgICAgICAgfSBlbHNlIGlmIChqICE9PSBsYXN0KSB7XG4gICAgICAgICAgICAgICAgLy8gV2UgbWF0Y2hlZCBhIFVOQyByb290IHdpdGggbGVmdG92ZXJzXG5cbiAgICAgICAgICAgICAgICBkZXZpY2UgPSBgXFxcXFxcXFwke2ZpcnN0UGFydH1cXFxcJHtwYXRoLnNsaWNlKGxhc3QsIGopfWA7XG4gICAgICAgICAgICAgICAgcm9vdEVuZCA9IGo7XG4gICAgICAgICAgICAgIH1cbiAgICAgICAgICAgIH1cbiAgICAgICAgICB9XG4gICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgcm9vdEVuZCA9IDE7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoaXNXaW5kb3dzRGV2aWNlUm9vdChjb2RlKSkge1xuICAgICAgICAvLyBQb3NzaWJsZSBkZXZpY2Ugcm9vdFxuXG4gICAgICAgIGlmIChwYXRoLmNoYXJDb2RlQXQoMSkgPT09IENIQVJfQ09MT04pIHtcbiAgICAgICAgICBkZXZpY2UgPSBwYXRoLnNsaWNlKDAsIDIpO1xuICAgICAgICAgIHJvb3RFbmQgPSAyO1xuICAgICAgICAgIGlmIChsZW4gPiAyKSB7XG4gICAgICAgICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdCgyKSkpIHtcbiAgICAgICAgICAgICAgLy8gVHJlYXQgc2VwYXJhdG9yIGZvbGxvd2luZyBkcml2ZSBuYW1lIGFzIGFuIGFic29sdXRlIHBhdGhcbiAgICAgICAgICAgICAgLy8gaW5kaWNhdG9yXG4gICAgICAgICAgICAgIGlzQWJzb2x1dGUgPSB0cnVlO1xuICAgICAgICAgICAgICByb290RW5kID0gMztcbiAgICAgICAgICAgIH1cbiAgICAgICAgICB9XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9IGVsc2UgaWYgKGlzUGF0aFNlcGFyYXRvcihjb2RlKSkge1xuICAgICAgLy8gYHBhdGhgIGNvbnRhaW5zIGp1c3QgYSBwYXRoIHNlcGFyYXRvclxuICAgICAgcm9vdEVuZCA9IDE7XG4gICAgICBpc0Fic29sdXRlID0gdHJ1ZTtcbiAgICB9XG5cbiAgICBpZiAoXG4gICAgICBkZXZpY2UubGVuZ3RoID4gMCAmJlxuICAgICAgcmVzb2x2ZWREZXZpY2UubGVuZ3RoID4gMCAmJlxuICAgICAgZGV2aWNlLnRvTG93ZXJDYXNlKCkgIT09IHJlc29sdmVkRGV2aWNlLnRvTG93ZXJDYXNlKClcbiAgICApIHtcbiAgICAgIC8vIFRoaXMgcGF0aCBwb2ludHMgdG8gYW5vdGhlciBkZXZpY2Ugc28gaXQgaXMgbm90IGFwcGxpY2FibGVcbiAgICAgIGNvbnRpbnVlO1xuICAgIH1cblxuICAgIGlmIChyZXNvbHZlZERldmljZS5sZW5ndGggPT09IDAgJiYgZGV2aWNlLmxlbmd0aCA+IDApIHtcbiAgICAgIHJlc29sdmVkRGV2aWNlID0gZGV2aWNlO1xuICAgIH1cbiAgICBpZiAoIXJlc29sdmVkQWJzb2x1dGUpIHtcbiAgICAgIHJlc29sdmVkVGFpbCA9IGAke3BhdGguc2xpY2Uocm9vdEVuZCl9XFxcXCR7cmVzb2x2ZWRUYWlsfWA7XG4gICAgICByZXNvbHZlZEFic29sdXRlID0gaXNBYnNvbHV0ZTtcbiAgICB9XG5cbiAgICBpZiAocmVzb2x2ZWRBYnNvbHV0ZSAmJiByZXNvbHZlZERldmljZS5sZW5ndGggPiAwKSBicmVhaztcbiAgfVxuXG4gIC8vIEF0IHRoaXMgcG9pbnQgdGhlIHBhdGggc2hvdWxkIGJlIHJlc29sdmVkIHRvIGEgZnVsbCBhYnNvbHV0ZSBwYXRoLFxuICAvLyBidXQgaGFuZGxlIHJlbGF0aXZlIHBhdGhzIHRvIGJlIHNhZmUgKG1pZ2h0IGhhcHBlbiB3aGVuIHByb2Nlc3MuY3dkKClcbiAgLy8gZmFpbHMpXG5cbiAgLy8gTm9ybWFsaXplIHRoZSB0YWlsIHBhdGhcbiAgcmVzb2x2ZWRUYWlsID0gbm9ybWFsaXplU3RyaW5nKFxuICAgIHJlc29sdmVkVGFpbCxcbiAgICAhcmVzb2x2ZWRBYnNvbHV0ZSxcbiAgICBcIlxcXFxcIixcbiAgICBpc1BhdGhTZXBhcmF0b3IsXG4gICk7XG5cbiAgcmV0dXJuIHJlc29sdmVkRGV2aWNlICsgKHJlc29sdmVkQWJzb2x1dGUgPyBcIlxcXFxcIiA6IFwiXCIpICsgcmVzb2x2ZWRUYWlsIHx8IFwiLlwiO1xufVxuXG4vKipcbiAqIE5vcm1hbGl6ZXMgYSBgcGF0aGBcbiAqIEBwYXJhbSBwYXRoIHRvIG5vcm1hbGl6ZVxuICovXG5leHBvcnQgZnVuY3Rpb24gbm9ybWFsaXplKHBhdGg6IHN0cmluZyk6IHN0cmluZyB7XG4gIGFzc2VydFBhdGgocGF0aCk7XG4gIGNvbnN0IGxlbiA9IHBhdGgubGVuZ3RoO1xuICBpZiAobGVuID09PSAwKSByZXR1cm4gXCIuXCI7XG4gIGxldCByb290RW5kID0gMDtcbiAgbGV0IGRldmljZTogc3RyaW5nIHwgdW5kZWZpbmVkO1xuICBsZXQgaXNBYnNvbHV0ZSA9IGZhbHNlO1xuICBjb25zdCBjb2RlID0gcGF0aC5jaGFyQ29kZUF0KDApO1xuXG4gIC8vIFRyeSB0byBtYXRjaCBhIHJvb3RcbiAgaWYgKGxlbiA+IDEpIHtcbiAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgICAvLyBQb3NzaWJsZSBVTkMgcm9vdFxuXG4gICAgICAvLyBJZiB3ZSBzdGFydGVkIHdpdGggYSBzZXBhcmF0b3IsIHdlIGtub3cgd2UgYXQgbGVhc3QgaGF2ZSBhbiBhYnNvbHV0ZVxuICAgICAgLy8gcGF0aCBvZiBzb21lIGtpbmQgKFVOQyBvciBvdGhlcndpc2UpXG4gICAgICBpc0Fic29sdXRlID0gdHJ1ZTtcblxuICAgICAgaWYgKGlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoMSkpKSB7XG4gICAgICAgIC8vIE1hdGNoZWQgZG91YmxlIHBhdGggc2VwYXJhdG9yIGF0IGJlZ2lubmluZ1xuICAgICAgICBsZXQgaiA9IDI7XG4gICAgICAgIGxldCBsYXN0ID0gajtcbiAgICAgICAgLy8gTWF0Y2ggMSBvciBtb3JlIG5vbi1wYXRoIHNlcGFyYXRvcnNcbiAgICAgICAgZm9yICg7IGogPCBsZW47ICsraikge1xuICAgICAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KGopKSkgYnJlYWs7XG4gICAgICAgIH1cbiAgICAgICAgaWYgKGogPCBsZW4gJiYgaiAhPT0gbGFzdCkge1xuICAgICAgICAgIGNvbnN0IGZpcnN0UGFydCA9IHBhdGguc2xpY2UobGFzdCwgaik7XG4gICAgICAgICAgLy8gTWF0Y2hlZCFcbiAgICAgICAgICBsYXN0ID0gajtcbiAgICAgICAgICAvLyBNYXRjaCAxIG9yIG1vcmUgcGF0aCBzZXBhcmF0b3JzXG4gICAgICAgICAgZm9yICg7IGogPCBsZW47ICsraikge1xuICAgICAgICAgICAgaWYgKCFpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KGopKSkgYnJlYWs7XG4gICAgICAgICAgfVxuICAgICAgICAgIGlmIChqIDwgbGVuICYmIGogIT09IGxhc3QpIHtcbiAgICAgICAgICAgIC8vIE1hdGNoZWQhXG4gICAgICAgICAgICBsYXN0ID0gajtcbiAgICAgICAgICAgIC8vIE1hdGNoIDEgb3IgbW9yZSBub24tcGF0aCBzZXBhcmF0b3JzXG4gICAgICAgICAgICBmb3IgKDsgaiA8IGxlbjsgKytqKSB7XG4gICAgICAgICAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KGopKSkgYnJlYWs7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgICBpZiAoaiA9PT0gbGVuKSB7XG4gICAgICAgICAgICAgIC8vIFdlIG1hdGNoZWQgYSBVTkMgcm9vdCBvbmx5XG4gICAgICAgICAgICAgIC8vIFJldHVybiB0aGUgbm9ybWFsaXplZCB2ZXJzaW9uIG9mIHRoZSBVTkMgcm9vdCBzaW5jZSB0aGVyZVxuICAgICAgICAgICAgICAvLyBpcyBub3RoaW5nIGxlZnQgdG8gcHJvY2Vzc1xuXG4gICAgICAgICAgICAgIHJldHVybiBgXFxcXFxcXFwke2ZpcnN0UGFydH1cXFxcJHtwYXRoLnNsaWNlKGxhc3QpfVxcXFxgO1xuICAgICAgICAgICAgfSBlbHNlIGlmIChqICE9PSBsYXN0KSB7XG4gICAgICAgICAgICAgIC8vIFdlIG1hdGNoZWQgYSBVTkMgcm9vdCB3aXRoIGxlZnRvdmVyc1xuXG4gICAgICAgICAgICAgIGRldmljZSA9IGBcXFxcXFxcXCR7Zmlyc3RQYXJ0fVxcXFwke3BhdGguc2xpY2UobGFzdCwgail9YDtcbiAgICAgICAgICAgICAgcm9vdEVuZCA9IGo7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByb290RW5kID0gMTtcbiAgICAgIH1cbiAgICB9IGVsc2UgaWYgKGlzV2luZG93c0RldmljZVJvb3QoY29kZSkpIHtcbiAgICAgIC8vIFBvc3NpYmxlIGRldmljZSByb290XG5cbiAgICAgIGlmIChwYXRoLmNoYXJDb2RlQXQoMSkgPT09IENIQVJfQ09MT04pIHtcbiAgICAgICAgZGV2aWNlID0gcGF0aC5zbGljZSgwLCAyKTtcbiAgICAgICAgcm9vdEVuZCA9IDI7XG4gICAgICAgIGlmIChsZW4gPiAyKSB7XG4gICAgICAgICAgaWYgKGlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoMikpKSB7XG4gICAgICAgICAgICAvLyBUcmVhdCBzZXBhcmF0b3IgZm9sbG93aW5nIGRyaXZlIG5hbWUgYXMgYW4gYWJzb2x1dGUgcGF0aFxuICAgICAgICAgICAgLy8gaW5kaWNhdG9yXG4gICAgICAgICAgICBpc0Fic29sdXRlID0gdHJ1ZTtcbiAgICAgICAgICAgIHJvb3RFbmQgPSAzO1xuICAgICAgICAgIH1cbiAgICAgICAgfVxuICAgICAgfVxuICAgIH1cbiAgfSBlbHNlIGlmIChpc1BhdGhTZXBhcmF0b3IoY29kZSkpIHtcbiAgICAvLyBgcGF0aGAgY29udGFpbnMganVzdCBhIHBhdGggc2VwYXJhdG9yLCBleGl0IGVhcmx5IHRvIGF2b2lkIHVubmVjZXNzYXJ5XG4gICAgLy8gd29ya1xuICAgIHJldHVybiBcIlxcXFxcIjtcbiAgfVxuXG4gIGxldCB0YWlsOiBzdHJpbmc7XG4gIGlmIChyb290RW5kIDwgbGVuKSB7XG4gICAgdGFpbCA9IG5vcm1hbGl6ZVN0cmluZyhcbiAgICAgIHBhdGguc2xpY2Uocm9vdEVuZCksXG4gICAgICAhaXNBYnNvbHV0ZSxcbiAgICAgIFwiXFxcXFwiLFxuICAgICAgaXNQYXRoU2VwYXJhdG9yLFxuICAgICk7XG4gIH0gZWxzZSB7XG4gICAgdGFpbCA9IFwiXCI7XG4gIH1cbiAgaWYgKHRhaWwubGVuZ3RoID09PSAwICYmICFpc0Fic29sdXRlKSB0YWlsID0gXCIuXCI7XG4gIGlmICh0YWlsLmxlbmd0aCA+IDAgJiYgaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdChsZW4gLSAxKSkpIHtcbiAgICB0YWlsICs9IFwiXFxcXFwiO1xuICB9XG4gIGlmIChkZXZpY2UgPT09IHVuZGVmaW5lZCkge1xuICAgIGlmIChpc0Fic29sdXRlKSB7XG4gICAgICBpZiAodGFpbC5sZW5ndGggPiAwKSByZXR1cm4gYFxcXFwke3RhaWx9YDtcbiAgICAgIGVsc2UgcmV0dXJuIFwiXFxcXFwiO1xuICAgIH0gZWxzZSBpZiAodGFpbC5sZW5ndGggPiAwKSB7XG4gICAgICByZXR1cm4gdGFpbDtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIFwiXCI7XG4gICAgfVxuICB9IGVsc2UgaWYgKGlzQWJzb2x1dGUpIHtcbiAgICBpZiAodGFpbC5sZW5ndGggPiAwKSByZXR1cm4gYCR7ZGV2aWNlfVxcXFwke3RhaWx9YDtcbiAgICBlbHNlIHJldHVybiBgJHtkZXZpY2V9XFxcXGA7XG4gIH0gZWxzZSBpZiAodGFpbC5sZW5ndGggPiAwKSB7XG4gICAgcmV0dXJuIGRldmljZSArIHRhaWw7XG4gIH0gZWxzZSB7XG4gICAgcmV0dXJuIGRldmljZTtcbiAgfVxufVxuXG4vKipcbiAqIFZlcmlmaWVzIHdoZXRoZXIgcGF0aCBpcyBhYnNvbHV0ZVxuICogQHBhcmFtIHBhdGggdG8gdmVyaWZ5XG4gKi9cbmV4cG9ydCBmdW5jdGlvbiBpc0Fic29sdXRlKHBhdGg6IHN0cmluZyk6IGJvb2xlYW4ge1xuICBhc3NlcnRQYXRoKHBhdGgpO1xuICBjb25zdCBsZW4gPSBwYXRoLmxlbmd0aDtcbiAgaWYgKGxlbiA9PT0gMCkgcmV0dXJuIGZhbHNlO1xuXG4gIGNvbnN0IGNvZGUgPSBwYXRoLmNoYXJDb2RlQXQoMCk7XG4gIGlmIChpc1BhdGhTZXBhcmF0b3IoY29kZSkpIHtcbiAgICByZXR1cm4gdHJ1ZTtcbiAgfSBlbHNlIGlmIChpc1dpbmRvd3NEZXZpY2VSb290KGNvZGUpKSB7XG4gICAgLy8gUG9zc2libGUgZGV2aWNlIHJvb3RcblxuICAgIGlmIChsZW4gPiAyICYmIHBhdGguY2hhckNvZGVBdCgxKSA9PT0gQ0hBUl9DT0xPTikge1xuICAgICAgaWYgKGlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoMikpKSByZXR1cm4gdHJ1ZTtcbiAgICB9XG4gIH1cbiAgcmV0dXJuIGZhbHNlO1xufVxuXG4vKipcbiAqIEpvaW4gYWxsIGdpdmVuIGEgc2VxdWVuY2Ugb2YgYHBhdGhzYCx0aGVuIG5vcm1hbGl6ZXMgdGhlIHJlc3VsdGluZyBwYXRoLlxuICogQHBhcmFtIHBhdGhzIHRvIGJlIGpvaW5lZCBhbmQgbm9ybWFsaXplZFxuICovXG5leHBvcnQgZnVuY3Rpb24gam9pbiguLi5wYXRoczogc3RyaW5nW10pOiBzdHJpbmcge1xuICBjb25zdCBwYXRoc0NvdW50ID0gcGF0aHMubGVuZ3RoO1xuICBpZiAocGF0aHNDb3VudCA9PT0gMCkgcmV0dXJuIFwiLlwiO1xuXG4gIGxldCBqb2luZWQ6IHN0cmluZyB8IHVuZGVmaW5lZDtcbiAgbGV0IGZpcnN0UGFydDogc3RyaW5nIHwgbnVsbCA9IG51bGw7XG4gIGZvciAobGV0IGkgPSAwOyBpIDwgcGF0aHNDb3VudDsgKytpKSB7XG4gICAgY29uc3QgcGF0aCA9IHBhdGhzW2ldO1xuICAgIGFzc2VydFBhdGgocGF0aCk7XG4gICAgaWYgKHBhdGgubGVuZ3RoID4gMCkge1xuICAgICAgaWYgKGpvaW5lZCA9PT0gdW5kZWZpbmVkKSBqb2luZWQgPSBmaXJzdFBhcnQgPSBwYXRoO1xuICAgICAgZWxzZSBqb2luZWQgKz0gYFxcXFwke3BhdGh9YDtcbiAgICB9XG4gIH1cblxuICBpZiAoam9pbmVkID09PSB1bmRlZmluZWQpIHJldHVybiBcIi5cIjtcblxuICAvLyBNYWtlIHN1cmUgdGhhdCB0aGUgam9pbmVkIHBhdGggZG9lc24ndCBzdGFydCB3aXRoIHR3byBzbGFzaGVzLCBiZWNhdXNlXG4gIC8vIG5vcm1hbGl6ZSgpIHdpbGwgbWlzdGFrZSBpdCBmb3IgYW4gVU5DIHBhdGggdGhlbi5cbiAgLy9cbiAgLy8gVGhpcyBzdGVwIGlzIHNraXBwZWQgd2hlbiBpdCBpcyB2ZXJ5IGNsZWFyIHRoYXQgdGhlIHVzZXIgYWN0dWFsbHlcbiAgLy8gaW50ZW5kZWQgdG8gcG9pbnQgYXQgYW4gVU5DIHBhdGguIFRoaXMgaXMgYXNzdW1lZCB3aGVuIHRoZSBmaXJzdFxuICAvLyBub24tZW1wdHkgc3RyaW5nIGFyZ3VtZW50cyBzdGFydHMgd2l0aCBleGFjdGx5IHR3byBzbGFzaGVzIGZvbGxvd2VkIGJ5XG4gIC8vIGF0IGxlYXN0IG9uZSBtb3JlIG5vbi1zbGFzaCBjaGFyYWN0ZXIuXG4gIC8vXG4gIC8vIE5vdGUgdGhhdCBmb3Igbm9ybWFsaXplKCkgdG8gdHJlYXQgYSBwYXRoIGFzIGFuIFVOQyBwYXRoIGl0IG5lZWRzIHRvXG4gIC8vIGhhdmUgYXQgbGVhc3QgMiBjb21wb25lbnRzLCBzbyB3ZSBkb24ndCBmaWx0ZXIgZm9yIHRoYXQgaGVyZS5cbiAgLy8gVGhpcyBtZWFucyB0aGF0IHRoZSB1c2VyIGNhbiB1c2Ugam9pbiB0byBjb25zdHJ1Y3QgVU5DIHBhdGhzIGZyb21cbiAgLy8gYSBzZXJ2ZXIgbmFtZSBhbmQgYSBzaGFyZSBuYW1lOyBmb3IgZXhhbXBsZTpcbiAgLy8gICBwYXRoLmpvaW4oJy8vc2VydmVyJywgJ3NoYXJlJykgLT4gJ1xcXFxcXFxcc2VydmVyXFxcXHNoYXJlXFxcXCcpXG4gIGxldCBuZWVkc1JlcGxhY2UgPSB0cnVlO1xuICBsZXQgc2xhc2hDb3VudCA9IDA7XG4gIGFzc2VydChmaXJzdFBhcnQgIT0gbnVsbCk7XG4gIGlmIChpc1BhdGhTZXBhcmF0b3IoZmlyc3RQYXJ0LmNoYXJDb2RlQXQoMCkpKSB7XG4gICAgKytzbGFzaENvdW50O1xuICAgIGNvbnN0IGZpcnN0TGVuID0gZmlyc3RQYXJ0Lmxlbmd0aDtcbiAgICBpZiAoZmlyc3RMZW4gPiAxKSB7XG4gICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGZpcnN0UGFydC5jaGFyQ29kZUF0KDEpKSkge1xuICAgICAgICArK3NsYXNoQ291bnQ7XG4gICAgICAgIGlmIChmaXJzdExlbiA+IDIpIHtcbiAgICAgICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGZpcnN0UGFydC5jaGFyQ29kZUF0KDIpKSkgKytzbGFzaENvdW50O1xuICAgICAgICAgIGVsc2Uge1xuICAgICAgICAgICAgLy8gV2UgbWF0Y2hlZCBhIFVOQyBwYXRoIGluIHRoZSBmaXJzdCBwYXJ0XG4gICAgICAgICAgICBuZWVkc1JlcGxhY2UgPSBmYWxzZTtcbiAgICAgICAgICB9XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG4gIH1cbiAgaWYgKG5lZWRzUmVwbGFjZSkge1xuICAgIC8vIEZpbmQgYW55IG1vcmUgY29uc2VjdXRpdmUgc2xhc2hlcyB3ZSBuZWVkIHRvIHJlcGxhY2VcbiAgICBmb3IgKDsgc2xhc2hDb3VudCA8IGpvaW5lZC5sZW5ndGg7ICsrc2xhc2hDb3VudCkge1xuICAgICAgaWYgKCFpc1BhdGhTZXBhcmF0b3Ioam9pbmVkLmNoYXJDb2RlQXQoc2xhc2hDb3VudCkpKSBicmVhaztcbiAgICB9XG5cbiAgICAvLyBSZXBsYWNlIHRoZSBzbGFzaGVzIGlmIG5lZWRlZFxuICAgIGlmIChzbGFzaENvdW50ID49IDIpIGpvaW5lZCA9IGBcXFxcJHtqb2luZWQuc2xpY2Uoc2xhc2hDb3VudCl9YDtcbiAgfVxuXG4gIHJldHVybiBub3JtYWxpemUoam9pbmVkKTtcbn1cblxuLyoqXG4gKiBJdCB3aWxsIHNvbHZlIHRoZSByZWxhdGl2ZSBwYXRoIGZyb20gYGZyb21gIHRvIGB0b2AsIGZvciBpbnN0YW5jZTpcbiAqICBmcm9tID0gJ0M6XFxcXG9yYW5kZWFcXFxcdGVzdFxcXFxhYWEnXG4gKiAgdG8gPSAnQzpcXFxcb3JhbmRlYVxcXFxpbXBsXFxcXGJiYidcbiAqIFRoZSBvdXRwdXQgb2YgdGhlIGZ1bmN0aW9uIHNob3VsZCBiZTogJy4uXFxcXC4uXFxcXGltcGxcXFxcYmJiJ1xuICogQHBhcmFtIGZyb20gcmVsYXRpdmUgcGF0aFxuICogQHBhcmFtIHRvIHJlbGF0aXZlIHBhdGhcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIHJlbGF0aXZlKGZyb206IHN0cmluZywgdG86IHN0cmluZyk6IHN0cmluZyB7XG4gIGFzc2VydFBhdGgoZnJvbSk7XG4gIGFzc2VydFBhdGgodG8pO1xuXG4gIGlmIChmcm9tID09PSB0bykgcmV0dXJuIFwiXCI7XG5cbiAgY29uc3QgZnJvbU9yaWcgPSByZXNvbHZlKGZyb20pO1xuICBjb25zdCB0b09yaWcgPSByZXNvbHZlKHRvKTtcblxuICBpZiAoZnJvbU9yaWcgPT09IHRvT3JpZykgcmV0dXJuIFwiXCI7XG5cbiAgZnJvbSA9IGZyb21PcmlnLnRvTG93ZXJDYXNlKCk7XG4gIHRvID0gdG9PcmlnLnRvTG93ZXJDYXNlKCk7XG5cbiAgaWYgKGZyb20gPT09IHRvKSByZXR1cm4gXCJcIjtcblxuICAvLyBUcmltIGFueSBsZWFkaW5nIGJhY2tzbGFzaGVzXG4gIGxldCBmcm9tU3RhcnQgPSAwO1xuICBsZXQgZnJvbUVuZCA9IGZyb20ubGVuZ3RoO1xuICBmb3IgKDsgZnJvbVN0YXJ0IDwgZnJvbUVuZDsgKytmcm9tU3RhcnQpIHtcbiAgICBpZiAoZnJvbS5jaGFyQ29kZUF0KGZyb21TdGFydCkgIT09IENIQVJfQkFDS1dBUkRfU0xBU0gpIGJyZWFrO1xuICB9XG4gIC8vIFRyaW0gdHJhaWxpbmcgYmFja3NsYXNoZXMgKGFwcGxpY2FibGUgdG8gVU5DIHBhdGhzIG9ubHkpXG4gIGZvciAoOyBmcm9tRW5kIC0gMSA+IGZyb21TdGFydDsgLS1mcm9tRW5kKSB7XG4gICAgaWYgKGZyb20uY2hhckNvZGVBdChmcm9tRW5kIC0gMSkgIT09IENIQVJfQkFDS1dBUkRfU0xBU0gpIGJyZWFrO1xuICB9XG4gIGNvbnN0IGZyb21MZW4gPSBmcm9tRW5kIC0gZnJvbVN0YXJ0O1xuXG4gIC8vIFRyaW0gYW55IGxlYWRpbmcgYmFja3NsYXNoZXNcbiAgbGV0IHRvU3RhcnQgPSAwO1xuICBsZXQgdG9FbmQgPSB0by5sZW5ndGg7XG4gIGZvciAoOyB0b1N0YXJ0IDwgdG9FbmQ7ICsrdG9TdGFydCkge1xuICAgIGlmICh0by5jaGFyQ29kZUF0KHRvU3RhcnQpICE9PSBDSEFSX0JBQ0tXQVJEX1NMQVNIKSBicmVhaztcbiAgfVxuICAvLyBUcmltIHRyYWlsaW5nIGJhY2tzbGFzaGVzIChhcHBsaWNhYmxlIHRvIFVOQyBwYXRocyBvbmx5KVxuICBmb3IgKDsgdG9FbmQgLSAxID4gdG9TdGFydDsgLS10b0VuZCkge1xuICAgIGlmICh0by5jaGFyQ29kZUF0KHRvRW5kIC0gMSkgIT09IENIQVJfQkFDS1dBUkRfU0xBU0gpIGJyZWFrO1xuICB9XG4gIGNvbnN0IHRvTGVuID0gdG9FbmQgLSB0b1N0YXJ0O1xuXG4gIC8vIENvbXBhcmUgcGF0aHMgdG8gZmluZCB0aGUgbG9uZ2VzdCBjb21tb24gcGF0aCBmcm9tIHJvb3RcbiAgY29uc3QgbGVuZ3RoID0gZnJvbUxlbiA8IHRvTGVuID8gZnJvbUxlbiA6IHRvTGVuO1xuICBsZXQgbGFzdENvbW1vblNlcCA9IC0xO1xuICBsZXQgaSA9IDA7XG4gIGZvciAoOyBpIDw9IGxlbmd0aDsgKytpKSB7XG4gICAgaWYgKGkgPT09IGxlbmd0aCkge1xuICAgICAgaWYgKHRvTGVuID4gbGVuZ3RoKSB7XG4gICAgICAgIGlmICh0by5jaGFyQ29kZUF0KHRvU3RhcnQgKyBpKSA9PT0gQ0hBUl9CQUNLV0FSRF9TTEFTSCkge1xuICAgICAgICAgIC8vIFdlIGdldCBoZXJlIGlmIGBmcm9tYCBpcyB0aGUgZXhhY3QgYmFzZSBwYXRoIGZvciBgdG9gLlxuICAgICAgICAgIC8vIEZvciBleGFtcGxlOiBmcm9tPSdDOlxcXFxmb29cXFxcYmFyJzsgdG89J0M6XFxcXGZvb1xcXFxiYXJcXFxcYmF6J1xuICAgICAgICAgIHJldHVybiB0b09yaWcuc2xpY2UodG9TdGFydCArIGkgKyAxKTtcbiAgICAgICAgfSBlbHNlIGlmIChpID09PSAyKSB7XG4gICAgICAgICAgLy8gV2UgZ2V0IGhlcmUgaWYgYGZyb21gIGlzIHRoZSBkZXZpY2Ugcm9vdC5cbiAgICAgICAgICAvLyBGb3IgZXhhbXBsZTogZnJvbT0nQzpcXFxcJzsgdG89J0M6XFxcXGZvbydcbiAgICAgICAgICByZXR1cm4gdG9PcmlnLnNsaWNlKHRvU3RhcnQgKyBpKTtcbiAgICAgICAgfVxuICAgICAgfVxuICAgICAgaWYgKGZyb21MZW4gPiBsZW5ndGgpIHtcbiAgICAgICAgaWYgKGZyb20uY2hhckNvZGVBdChmcm9tU3RhcnQgKyBpKSA9PT0gQ0hBUl9CQUNLV0FSRF9TTEFTSCkge1xuICAgICAgICAgIC8vIFdlIGdldCBoZXJlIGlmIGB0b2AgaXMgdGhlIGV4YWN0IGJhc2UgcGF0aCBmb3IgYGZyb21gLlxuICAgICAgICAgIC8vIEZvciBleGFtcGxlOiBmcm9tPSdDOlxcXFxmb29cXFxcYmFyJzsgdG89J0M6XFxcXGZvbydcbiAgICAgICAgICBsYXN0Q29tbW9uU2VwID0gaTtcbiAgICAgICAgfSBlbHNlIGlmIChpID09PSAyKSB7XG4gICAgICAgICAgLy8gV2UgZ2V0IGhlcmUgaWYgYHRvYCBpcyB0aGUgZGV2aWNlIHJvb3QuXG4gICAgICAgICAgLy8gRm9yIGV4YW1wbGU6IGZyb209J0M6XFxcXGZvb1xcXFxiYXInOyB0bz0nQzpcXFxcJ1xuICAgICAgICAgIGxhc3RDb21tb25TZXAgPSAzO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgICBicmVhaztcbiAgICB9XG4gICAgY29uc3QgZnJvbUNvZGUgPSBmcm9tLmNoYXJDb2RlQXQoZnJvbVN0YXJ0ICsgaSk7XG4gICAgY29uc3QgdG9Db2RlID0gdG8uY2hhckNvZGVBdCh0b1N0YXJ0ICsgaSk7XG4gICAgaWYgKGZyb21Db2RlICE9PSB0b0NvZGUpIGJyZWFrO1xuICAgIGVsc2UgaWYgKGZyb21Db2RlID09PSBDSEFSX0JBQ0tXQVJEX1NMQVNIKSBsYXN0Q29tbW9uU2VwID0gaTtcbiAgfVxuXG4gIC8vIFdlIGZvdW5kIGEgbWlzbWF0Y2ggYmVmb3JlIHRoZSBmaXJzdCBjb21tb24gcGF0aCBzZXBhcmF0b3Igd2FzIHNlZW4sIHNvXG4gIC8vIHJldHVybiB0aGUgb3JpZ2luYWwgYHRvYC5cbiAgaWYgKGkgIT09IGxlbmd0aCAmJiBsYXN0Q29tbW9uU2VwID09PSAtMSkge1xuICAgIHJldHVybiB0b09yaWc7XG4gIH1cblxuICBsZXQgb3V0ID0gXCJcIjtcbiAgaWYgKGxhc3RDb21tb25TZXAgPT09IC0xKSBsYXN0Q29tbW9uU2VwID0gMDtcbiAgLy8gR2VuZXJhdGUgdGhlIHJlbGF0aXZlIHBhdGggYmFzZWQgb24gdGhlIHBhdGggZGlmZmVyZW5jZSBiZXR3ZWVuIGB0b2AgYW5kXG4gIC8vIGBmcm9tYFxuICBmb3IgKGkgPSBmcm9tU3RhcnQgKyBsYXN0Q29tbW9uU2VwICsgMTsgaSA8PSBmcm9tRW5kOyArK2kpIHtcbiAgICBpZiAoaSA9PT0gZnJvbUVuZCB8fCBmcm9tLmNoYXJDb2RlQXQoaSkgPT09IENIQVJfQkFDS1dBUkRfU0xBU0gpIHtcbiAgICAgIGlmIChvdXQubGVuZ3RoID09PSAwKSBvdXQgKz0gXCIuLlwiO1xuICAgICAgZWxzZSBvdXQgKz0gXCJcXFxcLi5cIjtcbiAgICB9XG4gIH1cblxuICAvLyBMYXN0bHksIGFwcGVuZCB0aGUgcmVzdCBvZiB0aGUgZGVzdGluYXRpb24gKGB0b2ApIHBhdGggdGhhdCBjb21lcyBhZnRlclxuICAvLyB0aGUgY29tbW9uIHBhdGggcGFydHNcbiAgaWYgKG91dC5sZW5ndGggPiAwKSB7XG4gICAgcmV0dXJuIG91dCArIHRvT3JpZy5zbGljZSh0b1N0YXJ0ICsgbGFzdENvbW1vblNlcCwgdG9FbmQpO1xuICB9IGVsc2Uge1xuICAgIHRvU3RhcnQgKz0gbGFzdENvbW1vblNlcDtcbiAgICBpZiAodG9PcmlnLmNoYXJDb2RlQXQodG9TdGFydCkgPT09IENIQVJfQkFDS1dBUkRfU0xBU0gpICsrdG9TdGFydDtcbiAgICByZXR1cm4gdG9PcmlnLnNsaWNlKHRvU3RhcnQsIHRvRW5kKTtcbiAgfVxufVxuXG4vKipcbiAqIFJlc29sdmVzIHBhdGggdG8gYSBuYW1lc3BhY2UgcGF0aFxuICogQHBhcmFtIHBhdGggdG8gcmVzb2x2ZSB0byBuYW1lc3BhY2VcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIHRvTmFtZXNwYWNlZFBhdGgocGF0aDogc3RyaW5nKTogc3RyaW5nIHtcbiAgLy8gTm90ZTogdGhpcyB3aWxsICpwcm9iYWJseSogdGhyb3cgc29tZXdoZXJlLlxuICBpZiAodHlwZW9mIHBhdGggIT09IFwic3RyaW5nXCIpIHJldHVybiBwYXRoO1xuICBpZiAocGF0aC5sZW5ndGggPT09IDApIHJldHVybiBcIlwiO1xuXG4gIGNvbnN0IHJlc29sdmVkUGF0aCA9IHJlc29sdmUocGF0aCk7XG5cbiAgaWYgKHJlc29sdmVkUGF0aC5sZW5ndGggPj0gMykge1xuICAgIGlmIChyZXNvbHZlZFBhdGguY2hhckNvZGVBdCgwKSA9PT0gQ0hBUl9CQUNLV0FSRF9TTEFTSCkge1xuICAgICAgLy8gUG9zc2libGUgVU5DIHJvb3RcblxuICAgICAgaWYgKHJlc29sdmVkUGF0aC5jaGFyQ29kZUF0KDEpID09PSBDSEFSX0JBQ0tXQVJEX1NMQVNIKSB7XG4gICAgICAgIGNvbnN0IGNvZGUgPSByZXNvbHZlZFBhdGguY2hhckNvZGVBdCgyKTtcbiAgICAgICAgaWYgKGNvZGUgIT09IENIQVJfUVVFU1RJT05fTUFSSyAmJiBjb2RlICE9PSBDSEFSX0RPVCkge1xuICAgICAgICAgIC8vIE1hdGNoZWQgbm9uLWxvbmcgVU5DIHJvb3QsIGNvbnZlcnQgdGhlIHBhdGggdG8gYSBsb25nIFVOQyBwYXRoXG4gICAgICAgICAgcmV0dXJuIGBcXFxcXFxcXD9cXFxcVU5DXFxcXCR7cmVzb2x2ZWRQYXRoLnNsaWNlKDIpfWA7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9IGVsc2UgaWYgKGlzV2luZG93c0RldmljZVJvb3QocmVzb2x2ZWRQYXRoLmNoYXJDb2RlQXQoMCkpKSB7XG4gICAgICAvLyBQb3NzaWJsZSBkZXZpY2Ugcm9vdFxuXG4gICAgICBpZiAoXG4gICAgICAgIHJlc29sdmVkUGF0aC5jaGFyQ29kZUF0KDEpID09PSBDSEFSX0NPTE9OICYmXG4gICAgICAgIHJlc29sdmVkUGF0aC5jaGFyQ29kZUF0KDIpID09PSBDSEFSX0JBQ0tXQVJEX1NMQVNIXG4gICAgICApIHtcbiAgICAgICAgLy8gTWF0Y2hlZCBkZXZpY2Ugcm9vdCwgY29udmVydCB0aGUgcGF0aCB0byBhIGxvbmcgVU5DIHBhdGhcbiAgICAgICAgcmV0dXJuIGBcXFxcXFxcXD9cXFxcJHtyZXNvbHZlZFBhdGh9YDtcbiAgICAgIH1cbiAgICB9XG4gIH1cblxuICByZXR1cm4gcGF0aDtcbn1cblxuLyoqXG4gKiBSZXR1cm4gdGhlIGRpcmVjdG9yeSBuYW1lIG9mIGEgYHBhdGhgLlxuICogQHBhcmFtIHBhdGggdG8gZGV0ZXJtaW5lIG5hbWUgZm9yXG4gKi9cbmV4cG9ydCBmdW5jdGlvbiBkaXJuYW1lKHBhdGg6IHN0cmluZyk6IHN0cmluZyB7XG4gIGFzc2VydFBhdGgocGF0aCk7XG4gIGNvbnN0IGxlbiA9IHBhdGgubGVuZ3RoO1xuICBpZiAobGVuID09PSAwKSByZXR1cm4gXCIuXCI7XG4gIGxldCByb290RW5kID0gLTE7XG4gIGxldCBlbmQgPSAtMTtcbiAgbGV0IG1hdGNoZWRTbGFzaCA9IHRydWU7XG4gIGxldCBvZmZzZXQgPSAwO1xuICBjb25zdCBjb2RlID0gcGF0aC5jaGFyQ29kZUF0KDApO1xuXG4gIC8vIFRyeSB0byBtYXRjaCBhIHJvb3RcbiAgaWYgKGxlbiA+IDEpIHtcbiAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgICAvLyBQb3NzaWJsZSBVTkMgcm9vdFxuXG4gICAgICByb290RW5kID0gb2Zmc2V0ID0gMTtcblxuICAgICAgaWYgKGlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoMSkpKSB7XG4gICAgICAgIC8vIE1hdGNoZWQgZG91YmxlIHBhdGggc2VwYXJhdG9yIGF0IGJlZ2lubmluZ1xuICAgICAgICBsZXQgaiA9IDI7XG4gICAgICAgIGxldCBsYXN0ID0gajtcbiAgICAgICAgLy8gTWF0Y2ggMSBvciBtb3JlIG5vbi1wYXRoIHNlcGFyYXRvcnNcbiAgICAgICAgZm9yICg7IGogPCBsZW47ICsraikge1xuICAgICAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KGopKSkgYnJlYWs7XG4gICAgICAgIH1cbiAgICAgICAgaWYgKGogPCBsZW4gJiYgaiAhPT0gbGFzdCkge1xuICAgICAgICAgIC8vIE1hdGNoZWQhXG4gICAgICAgICAgbGFzdCA9IGo7XG4gICAgICAgICAgLy8gTWF0Y2ggMSBvciBtb3JlIHBhdGggc2VwYXJhdG9yc1xuICAgICAgICAgIGZvciAoOyBqIDwgbGVuOyArK2opIHtcbiAgICAgICAgICAgIGlmICghaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdChqKSkpIGJyZWFrO1xuICAgICAgICAgIH1cbiAgICAgICAgICBpZiAoaiA8IGxlbiAmJiBqICE9PSBsYXN0KSB7XG4gICAgICAgICAgICAvLyBNYXRjaGVkIVxuICAgICAgICAgICAgbGFzdCA9IGo7XG4gICAgICAgICAgICAvLyBNYXRjaCAxIG9yIG1vcmUgbm9uLXBhdGggc2VwYXJhdG9yc1xuICAgICAgICAgICAgZm9yICg7IGogPCBsZW47ICsraikge1xuICAgICAgICAgICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdChqKSkpIGJyZWFrO1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgaWYgKGogPT09IGxlbikge1xuICAgICAgICAgICAgICAvLyBXZSBtYXRjaGVkIGEgVU5DIHJvb3Qgb25seVxuICAgICAgICAgICAgICByZXR1cm4gcGF0aDtcbiAgICAgICAgICAgIH1cbiAgICAgICAgICAgIGlmIChqICE9PSBsYXN0KSB7XG4gICAgICAgICAgICAgIC8vIFdlIG1hdGNoZWQgYSBVTkMgcm9vdCB3aXRoIGxlZnRvdmVyc1xuXG4gICAgICAgICAgICAgIC8vIE9mZnNldCBieSAxIHRvIGluY2x1ZGUgdGhlIHNlcGFyYXRvciBhZnRlciB0aGUgVU5DIHJvb3QgdG9cbiAgICAgICAgICAgICAgLy8gdHJlYXQgaXQgYXMgYSBcIm5vcm1hbCByb290XCIgb24gdG9wIG9mIGEgKFVOQykgcm9vdFxuICAgICAgICAgICAgICByb290RW5kID0gb2Zmc2V0ID0gaiArIDE7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICB9XG4gICAgfSBlbHNlIGlmIChpc1dpbmRvd3NEZXZpY2VSb290KGNvZGUpKSB7XG4gICAgICAvLyBQb3NzaWJsZSBkZXZpY2Ugcm9vdFxuXG4gICAgICBpZiAocGF0aC5jaGFyQ29kZUF0KDEpID09PSBDSEFSX0NPTE9OKSB7XG4gICAgICAgIHJvb3RFbmQgPSBvZmZzZXQgPSAyO1xuICAgICAgICBpZiAobGVuID4gMikge1xuICAgICAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KDIpKSkgcm9vdEVuZCA9IG9mZnNldCA9IDM7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG4gIH0gZWxzZSBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgLy8gYHBhdGhgIGNvbnRhaW5zIGp1c3QgYSBwYXRoIHNlcGFyYXRvciwgZXhpdCBlYXJseSB0byBhdm9pZFxuICAgIC8vIHVubmVjZXNzYXJ5IHdvcmtcbiAgICByZXR1cm4gcGF0aDtcbiAgfVxuXG4gIGZvciAobGV0IGkgPSBsZW4gLSAxOyBpID49IG9mZnNldDsgLS1pKSB7XG4gICAgaWYgKGlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoaSkpKSB7XG4gICAgICBpZiAoIW1hdGNoZWRTbGFzaCkge1xuICAgICAgICBlbmQgPSBpO1xuICAgICAgICBicmVhaztcbiAgICAgIH1cbiAgICB9IGVsc2Uge1xuICAgICAgLy8gV2Ugc2F3IHRoZSBmaXJzdCBub24tcGF0aCBzZXBhcmF0b3JcbiAgICAgIG1hdGNoZWRTbGFzaCA9IGZhbHNlO1xuICAgIH1cbiAgfVxuXG4gIGlmIChlbmQgPT09IC0xKSB7XG4gICAgaWYgKHJvb3RFbmQgPT09IC0xKSByZXR1cm4gXCIuXCI7XG4gICAgZWxzZSBlbmQgPSByb290RW5kO1xuICB9XG4gIHJldHVybiBwYXRoLnNsaWNlKDAsIGVuZCk7XG59XG5cbi8qKlxuICogUmV0dXJuIHRoZSBsYXN0IHBvcnRpb24gb2YgYSBgcGF0aGAuIFRyYWlsaW5nIGRpcmVjdG9yeSBzZXBhcmF0b3JzIGFyZSBpZ25vcmVkLlxuICogQHBhcmFtIHBhdGggdG8gcHJvY2Vzc1xuICogQHBhcmFtIGV4dCBvZiBwYXRoIGRpcmVjdG9yeVxuICovXG5leHBvcnQgZnVuY3Rpb24gYmFzZW5hbWUocGF0aDogc3RyaW5nLCBleHQgPSBcIlwiKTogc3RyaW5nIHtcbiAgaWYgKGV4dCAhPT0gdW5kZWZpbmVkICYmIHR5cGVvZiBleHQgIT09IFwic3RyaW5nXCIpIHtcbiAgICB0aHJvdyBuZXcgVHlwZUVycm9yKCdcImV4dFwiIGFyZ3VtZW50IG11c3QgYmUgYSBzdHJpbmcnKTtcbiAgfVxuXG4gIGFzc2VydFBhdGgocGF0aCk7XG5cbiAgbGV0IHN0YXJ0ID0gMDtcbiAgbGV0IGVuZCA9IC0xO1xuICBsZXQgbWF0Y2hlZFNsYXNoID0gdHJ1ZTtcbiAgbGV0IGk6IG51bWJlcjtcblxuICAvLyBDaGVjayBmb3IgYSBkcml2ZSBsZXR0ZXIgcHJlZml4IHNvIGFzIG5vdCB0byBtaXN0YWtlIHRoZSBmb2xsb3dpbmdcbiAgLy8gcGF0aCBzZXBhcmF0b3IgYXMgYW4gZXh0cmEgc2VwYXJhdG9yIGF0IHRoZSBlbmQgb2YgdGhlIHBhdGggdGhhdCBjYW4gYmVcbiAgLy8gZGlzcmVnYXJkZWRcbiAgaWYgKHBhdGgubGVuZ3RoID49IDIpIHtcbiAgICBjb25zdCBkcml2ZSA9IHBhdGguY2hhckNvZGVBdCgwKTtcbiAgICBpZiAoaXNXaW5kb3dzRGV2aWNlUm9vdChkcml2ZSkpIHtcbiAgICAgIGlmIChwYXRoLmNoYXJDb2RlQXQoMSkgPT09IENIQVJfQ09MT04pIHN0YXJ0ID0gMjtcbiAgICB9XG4gIH1cblxuICBpZiAoZXh0ICE9PSB1bmRlZmluZWQgJiYgZXh0Lmxlbmd0aCA+IDAgJiYgZXh0Lmxlbmd0aCA8PSBwYXRoLmxlbmd0aCkge1xuICAgIGlmIChleHQubGVuZ3RoID09PSBwYXRoLmxlbmd0aCAmJiBleHQgPT09IHBhdGgpIHJldHVybiBcIlwiO1xuICAgIGxldCBleHRJZHggPSBleHQubGVuZ3RoIC0gMTtcbiAgICBsZXQgZmlyc3ROb25TbGFzaEVuZCA9IC0xO1xuICAgIGZvciAoaSA9IHBhdGgubGVuZ3RoIC0gMTsgaSA+PSBzdGFydDsgLS1pKSB7XG4gICAgICBjb25zdCBjb2RlID0gcGF0aC5jaGFyQ29kZUF0KGkpO1xuICAgICAgaWYgKGlzUGF0aFNlcGFyYXRvcihjb2RlKSkge1xuICAgICAgICAvLyBJZiB3ZSByZWFjaGVkIGEgcGF0aCBzZXBhcmF0b3IgdGhhdCB3YXMgbm90IHBhcnQgb2YgYSBzZXQgb2YgcGF0aFxuICAgICAgICAvLyBzZXBhcmF0b3JzIGF0IHRoZSBlbmQgb2YgdGhlIHN0cmluZywgc3RvcCBub3dcbiAgICAgICAgaWYgKCFtYXRjaGVkU2xhc2gpIHtcbiAgICAgICAgICBzdGFydCA9IGkgKyAxO1xuICAgICAgICAgIGJyZWFrO1xuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICBpZiAoZmlyc3ROb25TbGFzaEVuZCA9PT0gLTEpIHtcbiAgICAgICAgICAvLyBXZSBzYXcgdGhlIGZpcnN0IG5vbi1wYXRoIHNlcGFyYXRvciwgcmVtZW1iZXIgdGhpcyBpbmRleCBpbiBjYXNlXG4gICAgICAgICAgLy8gd2UgbmVlZCBpdCBpZiB0aGUgZXh0ZW5zaW9uIGVuZHMgdXAgbm90IG1hdGNoaW5nXG4gICAgICAgICAgbWF0Y2hlZFNsYXNoID0gZmFsc2U7XG4gICAgICAgICAgZmlyc3ROb25TbGFzaEVuZCA9IGkgKyAxO1xuICAgICAgICB9XG4gICAgICAgIGlmIChleHRJZHggPj0gMCkge1xuICAgICAgICAgIC8vIFRyeSB0byBtYXRjaCB0aGUgZXhwbGljaXQgZXh0ZW5zaW9uXG4gICAgICAgICAgaWYgKGNvZGUgPT09IGV4dC5jaGFyQ29kZUF0KGV4dElkeCkpIHtcbiAgICAgICAgICAgIGlmICgtLWV4dElkeCA9PT0gLTEpIHtcbiAgICAgICAgICAgICAgLy8gV2UgbWF0Y2hlZCB0aGUgZXh0ZW5zaW9uLCBzbyBtYXJrIHRoaXMgYXMgdGhlIGVuZCBvZiBvdXIgcGF0aFxuICAgICAgICAgICAgICAvLyBjb21wb25lbnRcbiAgICAgICAgICAgICAgZW5kID0gaTtcbiAgICAgICAgICAgIH1cbiAgICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgICAgLy8gRXh0ZW5zaW9uIGRvZXMgbm90IG1hdGNoLCBzbyBvdXIgcmVzdWx0IGlzIHRoZSBlbnRpcmUgcGF0aFxuICAgICAgICAgICAgLy8gY29tcG9uZW50XG4gICAgICAgICAgICBleHRJZHggPSAtMTtcbiAgICAgICAgICAgIGVuZCA9IGZpcnN0Tm9uU2xhc2hFbmQ7XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKHN0YXJ0ID09PSBlbmQpIGVuZCA9IGZpcnN0Tm9uU2xhc2hFbmQ7XG4gICAgZWxzZSBpZiAoZW5kID09PSAtMSkgZW5kID0gcGF0aC5sZW5ndGg7XG4gICAgcmV0dXJuIHBhdGguc2xpY2Uoc3RhcnQsIGVuZCk7XG4gIH0gZWxzZSB7XG4gICAgZm9yIChpID0gcGF0aC5sZW5ndGggLSAxOyBpID49IHN0YXJ0OyAtLWkpIHtcbiAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KGkpKSkge1xuICAgICAgICAvLyBJZiB3ZSByZWFjaGVkIGEgcGF0aCBzZXBhcmF0b3IgdGhhdCB3YXMgbm90IHBhcnQgb2YgYSBzZXQgb2YgcGF0aFxuICAgICAgICAvLyBzZXBhcmF0b3JzIGF0IHRoZSBlbmQgb2YgdGhlIHN0cmluZywgc3RvcCBub3dcbiAgICAgICAgaWYgKCFtYXRjaGVkU2xhc2gpIHtcbiAgICAgICAgICBzdGFydCA9IGkgKyAxO1xuICAgICAgICAgIGJyZWFrO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGVuZCA9PT0gLTEpIHtcbiAgICAgICAgLy8gV2Ugc2F3IHRoZSBmaXJzdCBub24tcGF0aCBzZXBhcmF0b3IsIG1hcmsgdGhpcyBhcyB0aGUgZW5kIG9mIG91clxuICAgICAgICAvLyBwYXRoIGNvbXBvbmVudFxuICAgICAgICBtYXRjaGVkU2xhc2ggPSBmYWxzZTtcbiAgICAgICAgZW5kID0gaSArIDE7XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKGVuZCA9PT0gLTEpIHJldHVybiBcIlwiO1xuICAgIHJldHVybiBwYXRoLnNsaWNlKHN0YXJ0LCBlbmQpO1xuICB9XG59XG5cbi8qKlxuICogUmV0dXJuIHRoZSBleHRlbnNpb24gb2YgdGhlIGBwYXRoYCB3aXRoIGxlYWRpbmcgcGVyaW9kLlxuICogQHBhcmFtIHBhdGggd2l0aCBleHRlbnNpb25cbiAqIEByZXR1cm5zIGV4dGVuc2lvbiAoZXguIGZvciBgZmlsZS50c2AgcmV0dXJucyBgLnRzYClcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIGV4dG5hbWUocGF0aDogc3RyaW5nKTogc3RyaW5nIHtcbiAgYXNzZXJ0UGF0aChwYXRoKTtcbiAgbGV0IHN0YXJ0ID0gMDtcbiAgbGV0IHN0YXJ0RG90ID0gLTE7XG4gIGxldCBzdGFydFBhcnQgPSAwO1xuICBsZXQgZW5kID0gLTE7XG4gIGxldCBtYXRjaGVkU2xhc2ggPSB0cnVlO1xuICAvLyBUcmFjayB0aGUgc3RhdGUgb2YgY2hhcmFjdGVycyAoaWYgYW55KSB3ZSBzZWUgYmVmb3JlIG91ciBmaXJzdCBkb3QgYW5kXG4gIC8vIGFmdGVyIGFueSBwYXRoIHNlcGFyYXRvciB3ZSBmaW5kXG4gIGxldCBwcmVEb3RTdGF0ZSA9IDA7XG5cbiAgLy8gQ2hlY2sgZm9yIGEgZHJpdmUgbGV0dGVyIHByZWZpeCBzbyBhcyBub3QgdG8gbWlzdGFrZSB0aGUgZm9sbG93aW5nXG4gIC8vIHBhdGggc2VwYXJhdG9yIGFzIGFuIGV4dHJhIHNlcGFyYXRvciBhdCB0aGUgZW5kIG9mIHRoZSBwYXRoIHRoYXQgY2FuIGJlXG4gIC8vIGRpc3JlZ2FyZGVkXG5cbiAgaWYgKFxuICAgIHBhdGgubGVuZ3RoID49IDIgJiZcbiAgICBwYXRoLmNoYXJDb2RlQXQoMSkgPT09IENIQVJfQ09MT04gJiZcbiAgICBpc1dpbmRvd3NEZXZpY2VSb290KHBhdGguY2hhckNvZGVBdCgwKSlcbiAgKSB7XG4gICAgc3RhcnQgPSBzdGFydFBhcnQgPSAyO1xuICB9XG5cbiAgZm9yIChsZXQgaSA9IHBhdGgubGVuZ3RoIC0gMTsgaSA+PSBzdGFydDsgLS1pKSB7XG4gICAgY29uc3QgY29kZSA9IHBhdGguY2hhckNvZGVBdChpKTtcbiAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgICAvLyBJZiB3ZSByZWFjaGVkIGEgcGF0aCBzZXBhcmF0b3IgdGhhdCB3YXMgbm90IHBhcnQgb2YgYSBzZXQgb2YgcGF0aFxuICAgICAgLy8gc2VwYXJhdG9ycyBhdCB0aGUgZW5kIG9mIHRoZSBzdHJpbmcsIHN0b3Agbm93XG4gICAgICBpZiAoIW1hdGNoZWRTbGFzaCkge1xuICAgICAgICBzdGFydFBhcnQgPSBpICsgMTtcbiAgICAgICAgYnJlYWs7XG4gICAgICB9XG4gICAgICBjb250aW51ZTtcbiAgICB9XG4gICAgaWYgKGVuZCA9PT0gLTEpIHtcbiAgICAgIC8vIFdlIHNhdyB0aGUgZmlyc3Qgbm9uLXBhdGggc2VwYXJhdG9yLCBtYXJrIHRoaXMgYXMgdGhlIGVuZCBvZiBvdXJcbiAgICAgIC8vIGV4dGVuc2lvblxuICAgICAgbWF0Y2hlZFNsYXNoID0gZmFsc2U7XG4gICAgICBlbmQgPSBpICsgMTtcbiAgICB9XG4gICAgaWYgKGNvZGUgPT09IENIQVJfRE9UKSB7XG4gICAgICAvLyBJZiB0aGlzIGlzIG91ciBmaXJzdCBkb3QsIG1hcmsgaXQgYXMgdGhlIHN0YXJ0IG9mIG91ciBleHRlbnNpb25cbiAgICAgIGlmIChzdGFydERvdCA9PT0gLTEpIHN0YXJ0RG90ID0gaTtcbiAgICAgIGVsc2UgaWYgKHByZURvdFN0YXRlICE9PSAxKSBwcmVEb3RTdGF0ZSA9IDE7XG4gICAgfSBlbHNlIGlmIChzdGFydERvdCAhPT0gLTEpIHtcbiAgICAgIC8vIFdlIHNhdyBhIG5vbi1kb3QgYW5kIG5vbi1wYXRoIHNlcGFyYXRvciBiZWZvcmUgb3VyIGRvdCwgc28gd2Ugc2hvdWxkXG4gICAgICAvLyBoYXZlIGEgZ29vZCBjaGFuY2UgYXQgaGF2aW5nIGEgbm9uLWVtcHR5IGV4dGVuc2lvblxuICAgICAgcHJlRG90U3RhdGUgPSAtMTtcbiAgICB9XG4gIH1cblxuICBpZiAoXG4gICAgc3RhcnREb3QgPT09IC0xIHx8XG4gICAgZW5kID09PSAtMSB8fFxuICAgIC8vIFdlIHNhdyBhIG5vbi1kb3QgY2hhcmFjdGVyIGltbWVkaWF0ZWx5IGJlZm9yZSB0aGUgZG90XG4gICAgcHJlRG90U3RhdGUgPT09IDAgfHxcbiAgICAvLyBUaGUgKHJpZ2h0LW1vc3QpIHRyaW1tZWQgcGF0aCBjb21wb25lbnQgaXMgZXhhY3RseSAnLi4nXG4gICAgKHByZURvdFN0YXRlID09PSAxICYmIHN0YXJ0RG90ID09PSBlbmQgLSAxICYmIHN0YXJ0RG90ID09PSBzdGFydFBhcnQgKyAxKVxuICApIHtcbiAgICByZXR1cm4gXCJcIjtcbiAgfVxuICByZXR1cm4gcGF0aC5zbGljZShzdGFydERvdCwgZW5kKTtcbn1cblxuLyoqXG4gKiBHZW5lcmF0ZSBhIHBhdGggZnJvbSBgRm9ybWF0SW5wdXRQYXRoT2JqZWN0YCBvYmplY3QuXG4gKiBAcGFyYW0gcGF0aE9iamVjdCB3aXRoIHBhdGhcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIGZvcm1hdChwYXRoT2JqZWN0OiBGb3JtYXRJbnB1dFBhdGhPYmplY3QpOiBzdHJpbmcge1xuICBpZiAocGF0aE9iamVjdCA9PT0gbnVsbCB8fCB0eXBlb2YgcGF0aE9iamVjdCAhPT0gXCJvYmplY3RcIikge1xuICAgIHRocm93IG5ldyBUeXBlRXJyb3IoXG4gICAgICBgVGhlIFwicGF0aE9iamVjdFwiIGFyZ3VtZW50IG11c3QgYmUgb2YgdHlwZSBPYmplY3QuIFJlY2VpdmVkIHR5cGUgJHt0eXBlb2YgcGF0aE9iamVjdH1gLFxuICAgICk7XG4gIH1cbiAgcmV0dXJuIF9mb3JtYXQoXCJcXFxcXCIsIHBhdGhPYmplY3QpO1xufVxuXG4vKipcbiAqIFJldHVybiBhIGBQYXJzZWRQYXRoYCBvYmplY3Qgb2YgdGhlIGBwYXRoYC5cbiAqIEBwYXJhbSBwYXRoIHRvIHByb2Nlc3NcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIHBhcnNlKHBhdGg6IHN0cmluZyk6IFBhcnNlZFBhdGgge1xuICBhc3NlcnRQYXRoKHBhdGgpO1xuXG4gIGNvbnN0IHJldDogUGFyc2VkUGF0aCA9IHsgcm9vdDogXCJcIiwgZGlyOiBcIlwiLCBiYXNlOiBcIlwiLCBleHQ6IFwiXCIsIG5hbWU6IFwiXCIgfTtcblxuICBjb25zdCBsZW4gPSBwYXRoLmxlbmd0aDtcbiAgaWYgKGxlbiA9PT0gMCkgcmV0dXJuIHJldDtcblxuICBsZXQgcm9vdEVuZCA9IDA7XG4gIGxldCBjb2RlID0gcGF0aC5jaGFyQ29kZUF0KDApO1xuXG4gIC8vIFRyeSB0byBtYXRjaCBhIHJvb3RcbiAgaWYgKGxlbiA+IDEpIHtcbiAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgICAvLyBQb3NzaWJsZSBVTkMgcm9vdFxuXG4gICAgICByb290RW5kID0gMTtcbiAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KDEpKSkge1xuICAgICAgICAvLyBNYXRjaGVkIGRvdWJsZSBwYXRoIHNlcGFyYXRvciBhdCBiZWdpbm5pbmdcbiAgICAgICAgbGV0IGogPSAyO1xuICAgICAgICBsZXQgbGFzdCA9IGo7XG4gICAgICAgIC8vIE1hdGNoIDEgb3IgbW9yZSBub24tcGF0aCBzZXBhcmF0b3JzXG4gICAgICAgIGZvciAoOyBqIDwgbGVuOyArK2opIHtcbiAgICAgICAgICBpZiAoaXNQYXRoU2VwYXJhdG9yKHBhdGguY2hhckNvZGVBdChqKSkpIGJyZWFrO1xuICAgICAgICB9XG4gICAgICAgIGlmIChqIDwgbGVuICYmIGogIT09IGxhc3QpIHtcbiAgICAgICAgICAvLyBNYXRjaGVkIVxuICAgICAgICAgIGxhc3QgPSBqO1xuICAgICAgICAgIC8vIE1hdGNoIDEgb3IgbW9yZSBwYXRoIHNlcGFyYXRvcnNcbiAgICAgICAgICBmb3IgKDsgaiA8IGxlbjsgKytqKSB7XG4gICAgICAgICAgICBpZiAoIWlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoaikpKSBicmVhaztcbiAgICAgICAgICB9XG4gICAgICAgICAgaWYgKGogPCBsZW4gJiYgaiAhPT0gbGFzdCkge1xuICAgICAgICAgICAgLy8gTWF0Y2hlZCFcbiAgICAgICAgICAgIGxhc3QgPSBqO1xuICAgICAgICAgICAgLy8gTWF0Y2ggMSBvciBtb3JlIG5vbi1wYXRoIHNlcGFyYXRvcnNcbiAgICAgICAgICAgIGZvciAoOyBqIDwgbGVuOyArK2opIHtcbiAgICAgICAgICAgICAgaWYgKGlzUGF0aFNlcGFyYXRvcihwYXRoLmNoYXJDb2RlQXQoaikpKSBicmVhaztcbiAgICAgICAgICAgIH1cbiAgICAgICAgICAgIGlmIChqID09PSBsZW4pIHtcbiAgICAgICAgICAgICAgLy8gV2UgbWF0Y2hlZCBhIFVOQyByb290IG9ubHlcblxuICAgICAgICAgICAgICByb290RW5kID0gajtcbiAgICAgICAgICAgIH0gZWxzZSBpZiAoaiAhPT0gbGFzdCkge1xuICAgICAgICAgICAgICAvLyBXZSBtYXRjaGVkIGEgVU5DIHJvb3Qgd2l0aCBsZWZ0b3ZlcnNcblxuICAgICAgICAgICAgICByb290RW5kID0gaiArIDE7XG4gICAgICAgICAgICB9XG4gICAgICAgICAgfVxuICAgICAgICB9XG4gICAgICB9XG4gICAgfSBlbHNlIGlmIChpc1dpbmRvd3NEZXZpY2VSb290KGNvZGUpKSB7XG4gICAgICAvLyBQb3NzaWJsZSBkZXZpY2Ugcm9vdFxuXG4gICAgICBpZiAocGF0aC5jaGFyQ29kZUF0KDEpID09PSBDSEFSX0NPTE9OKSB7XG4gICAgICAgIHJvb3RFbmQgPSAyO1xuICAgICAgICBpZiAobGVuID4gMikge1xuICAgICAgICAgIGlmIChpc1BhdGhTZXBhcmF0b3IocGF0aC5jaGFyQ29kZUF0KDIpKSkge1xuICAgICAgICAgICAgaWYgKGxlbiA9PT0gMykge1xuICAgICAgICAgICAgICAvLyBgcGF0aGAgY29udGFpbnMganVzdCBhIGRyaXZlIHJvb3QsIGV4aXQgZWFybHkgdG8gYXZvaWRcbiAgICAgICAgICAgICAgLy8gdW5uZWNlc3Nhcnkgd29ya1xuICAgICAgICAgICAgICByZXQucm9vdCA9IHJldC5kaXIgPSBwYXRoO1xuICAgICAgICAgICAgICByZXR1cm4gcmV0O1xuICAgICAgICAgICAgfVxuICAgICAgICAgICAgcm9vdEVuZCA9IDM7XG4gICAgICAgICAgfVxuICAgICAgICB9IGVsc2Uge1xuICAgICAgICAgIC8vIGBwYXRoYCBjb250YWlucyBqdXN0IGEgZHJpdmUgcm9vdCwgZXhpdCBlYXJseSB0byBhdm9pZFxuICAgICAgICAgIC8vIHVubmVjZXNzYXJ5IHdvcmtcbiAgICAgICAgICByZXQucm9vdCA9IHJldC5kaXIgPSBwYXRoO1xuICAgICAgICAgIHJldHVybiByZXQ7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG4gIH0gZWxzZSBpZiAoaXNQYXRoU2VwYXJhdG9yKGNvZGUpKSB7XG4gICAgLy8gYHBhdGhgIGNvbnRhaW5zIGp1c3QgYSBwYXRoIHNlcGFyYXRvciwgZXhpdCBlYXJseSB0byBhdm9pZFxuICAgIC8vIHVubmVjZXNzYXJ5IHdvcmtcbiAgICByZXQucm9vdCA9IHJldC5kaXIgPSBwYXRoO1xuICAgIHJldHVybiByZXQ7XG4gIH1cblxuICBpZiAocm9vdEVuZCA+IDApIHJldC5yb290ID0gcGF0aC5zbGljZSgwLCByb290RW5kKTtcblxuICBsZXQgc3RhcnREb3QgPSAtMTtcbiAgbGV0IHN0YXJ0UGFydCA9IHJvb3RFbmQ7XG4gIGxldCBlbmQgPSAtMTtcbiAgbGV0IG1hdGNoZWRTbGFzaCA9IHRydWU7XG4gIGxldCBpID0gcGF0aC5sZW5ndGggLSAxO1xuXG4gIC8vIFRyYWNrIHRoZSBzdGF0ZSBvZiBjaGFyYWN0ZXJzIChpZiBhbnkpIHdlIHNlZSBiZWZvcmUgb3VyIGZpcnN0IGRvdCBhbmRcbiAgLy8gYWZ0ZXIgYW55IHBhdGggc2VwYXJhdG9yIHdlIGZpbmRcbiAgbGV0IHByZURvdFN0YXRlID0gMDtcblxuICAvLyBHZXQgbm9uLWRpciBpbmZvXG4gIGZvciAoOyBpID49IHJvb3RFbmQ7IC0taSkge1xuICAgIGNvZGUgPSBwYXRoLmNoYXJDb2RlQXQoaSk7XG4gICAgaWYgKGlzUGF0aFNlcGFyYXRvcihjb2RlKSkge1xuICAgICAgLy8gSWYgd2UgcmVhY2hlZCBhIHBhdGggc2VwYXJhdG9yIHRoYXQgd2FzIG5vdCBwYXJ0IG9mIGEgc2V0IG9mIHBhdGhcbiAgICAgIC8vIHNlcGFyYXRvcnMgYXQgdGhlIGVuZCBvZiB0aGUgc3RyaW5nLCBzdG9wIG5vd1xuICAgICAgaWYgKCFtYXRjaGVkU2xhc2gpIHtcbiAgICAgICAgc3RhcnRQYXJ0ID0gaSArIDE7XG4gICAgICAgIGJyZWFrO1xuICAgICAgfVxuICAgICAgY29udGludWU7XG4gICAgfVxuICAgIGlmIChlbmQgPT09IC0xKSB7XG4gICAgICAvLyBXZSBzYXcgdGhlIGZpcnN0IG5vbi1wYXRoIHNlcGFyYXRvciwgbWFyayB0aGlzIGFzIHRoZSBlbmQgb2Ygb3VyXG4gICAgICAvLyBleHRlbnNpb25cbiAgICAgIG1hdGNoZWRTbGFzaCA9IGZhbHNlO1xuICAgICAgZW5kID0gaSArIDE7XG4gICAgfVxuICAgIGlmIChjb2RlID09PSBDSEFSX0RPVCkge1xuICAgICAgLy8gSWYgdGhpcyBpcyBvdXIgZmlyc3QgZG90LCBtYXJrIGl0IGFzIHRoZSBzdGFydCBvZiBvdXIgZXh0ZW5zaW9uXG4gICAgICBpZiAoc3RhcnREb3QgPT09IC0xKSBzdGFydERvdCA9IGk7XG4gICAgICBlbHNlIGlmIChwcmVEb3RTdGF0ZSAhPT0gMSkgcHJlRG90U3RhdGUgPSAxO1xuICAgIH0gZWxzZSBpZiAoc3RhcnREb3QgIT09IC0xKSB7XG4gICAgICAvLyBXZSBzYXcgYSBub24tZG90IGFuZCBub24tcGF0aCBzZXBhcmF0b3IgYmVmb3JlIG91ciBkb3QsIHNvIHdlIHNob3VsZFxuICAgICAgLy8gaGF2ZSBhIGdvb2QgY2hhbmNlIGF0IGhhdmluZyBhIG5vbi1lbXB0eSBleHRlbnNpb25cbiAgICAgIHByZURvdFN0YXRlID0gLTE7XG4gICAgfVxuICB9XG5cbiAgaWYgKFxuICAgIHN0YXJ0RG90ID09PSAtMSB8fFxuICAgIGVuZCA9PT0gLTEgfHxcbiAgICAvLyBXZSBzYXcgYSBub24tZG90IGNoYXJhY3RlciBpbW1lZGlhdGVseSBiZWZvcmUgdGhlIGRvdFxuICAgIHByZURvdFN0YXRlID09PSAwIHx8XG4gICAgLy8gVGhlIChyaWdodC1tb3N0KSB0cmltbWVkIHBhdGggY29tcG9uZW50IGlzIGV4YWN0bHkgJy4uJ1xuICAgIChwcmVEb3RTdGF0ZSA9PT0gMSAmJiBzdGFydERvdCA9PT0gZW5kIC0gMSAmJiBzdGFydERvdCA9PT0gc3RhcnRQYXJ0ICsgMSlcbiAgKSB7XG4gICAgaWYgKGVuZCAhPT0gLTEpIHtcbiAgICAgIHJldC5iYXNlID0gcmV0Lm5hbWUgPSBwYXRoLnNsaWNlKHN0YXJ0UGFydCwgZW5kKTtcbiAgICB9XG4gIH0gZWxzZSB7XG4gICAgcmV0Lm5hbWUgPSBwYXRoLnNsaWNlKHN0YXJ0UGFydCwgc3RhcnREb3QpO1xuICAgIHJldC5iYXNlID0gcGF0aC5zbGljZShzdGFydFBhcnQsIGVuZCk7XG4gICAgcmV0LmV4dCA9IHBhdGguc2xpY2Uoc3RhcnREb3QsIGVuZCk7XG4gIH1cblxuICAvLyBJZiB0aGUgZGlyZWN0b3J5IGlzIHRoZSByb290LCB1c2UgdGhlIGVudGlyZSByb290IGFzIHRoZSBgZGlyYCBpbmNsdWRpbmdcbiAgLy8gdGhlIHRyYWlsaW5nIHNsYXNoIGlmIGFueSAoYEM6XFxhYmNgIC0+IGBDOlxcYCkuIE90aGVyd2lzZSwgc3RyaXAgb3V0IHRoZVxuICAvLyB0cmFpbGluZyBzbGFzaCAoYEM6XFxhYmNcXGRlZmAgLT4gYEM6XFxhYmNgKS5cbiAgaWYgKHN0YXJ0UGFydCA+IDAgJiYgc3RhcnRQYXJ0ICE9PSByb290RW5kKSB7XG4gICAgcmV0LmRpciA9IHBhdGguc2xpY2UoMCwgc3RhcnRQYXJ0IC0gMSk7XG4gIH0gZWxzZSByZXQuZGlyID0gcmV0LnJvb3Q7XG5cbiAgcmV0dXJuIHJldDtcbn1cblxuLyoqXG4gKiBDb252ZXJ0cyBhIGZpbGUgVVJMIHRvIGEgcGF0aCBzdHJpbmcuXG4gKlxuICogYGBgdHNcbiAqICAgICAgaW1wb3J0IHsgZnJvbUZpbGVVcmwgfSBmcm9tIFwiLi93aW4zMi50c1wiO1xuICogICAgICBmcm9tRmlsZVVybChcImZpbGU6Ly8vaG9tZS9mb29cIik7IC8vIFwiXFxcXGhvbWVcXFxcZm9vXCJcbiAqICAgICAgZnJvbUZpbGVVcmwoXCJmaWxlOi8vL0M6L1VzZXJzL2Zvb1wiKTsgLy8gXCJDOlxcXFxVc2Vyc1xcXFxmb29cIlxuICogICAgICBmcm9tRmlsZVVybChcImZpbGU6Ly9sb2NhbGhvc3QvaG9tZS9mb29cIik7IC8vIFwiXFxcXFxcXFxsb2NhbGhvc3RcXFxcaG9tZVxcXFxmb29cIlxuICogYGBgXG4gKiBAcGFyYW0gdXJsIG9mIGEgZmlsZSBVUkxcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIGZyb21GaWxlVXJsKHVybDogc3RyaW5nIHwgVVJMKTogc3RyaW5nIHtcbiAgdXJsID0gdXJsIGluc3RhbmNlb2YgVVJMID8gdXJsIDogbmV3IFVSTCh1cmwpO1xuICBpZiAodXJsLnByb3RvY29sICE9IFwiZmlsZTpcIikge1xuICAgIHRocm93IG5ldyBUeXBlRXJyb3IoXCJNdXN0IGJlIGEgZmlsZSBVUkwuXCIpO1xuICB9XG4gIGxldCBwYXRoID0gZGVjb2RlVVJJQ29tcG9uZW50KFxuICAgIHVybC5wYXRobmFtZS5yZXBsYWNlKC9cXC8vZywgXCJcXFxcXCIpLnJlcGxhY2UoLyUoPyFbMC05QS1GYS1mXXsyfSkvZywgXCIlMjVcIiksXG4gICkucmVwbGFjZSgvXlxcXFwqKFtBLVphLXpdOikoXFxcXHwkKS8sIFwiJDFcXFxcXCIpO1xuICBpZiAodXJsLmhvc3RuYW1lICE9IFwiXCIpIHtcbiAgICAvLyBOb3RlOiBUaGUgYFVSTGAgaW1wbGVtZW50YXRpb24gZ3VhcmFudGVlcyB0aGF0IHRoZSBkcml2ZSBsZXR0ZXIgYW5kXG4gICAgLy8gaG9zdG5hbWUgYXJlIG11dHVhbGx5IGV4Y2x1c2l2ZS4gT3RoZXJ3aXNlIGl0IHdvdWxkIG5vdCBoYXZlIGJlZW4gdmFsaWRcbiAgICAvLyB0byBhcHBlbmQgdGhlIGhvc3RuYW1lIGFuZCBwYXRoIGxpa2UgdGhpcy5cbiAgICBwYXRoID0gYFxcXFxcXFxcJHt1cmwuaG9zdG5hbWV9JHtwYXRofWA7XG4gIH1cbiAgcmV0dXJuIHBhdGg7XG59XG5cbi8qKlxuICogQ29udmVydHMgYSBwYXRoIHN0cmluZyB0byBhIGZpbGUgVVJMLlxuICpcbiAqIGBgYHRzXG4gKiAgICAgIGltcG9ydCB7IHRvRmlsZVVybCB9IGZyb20gXCIuL3dpbjMyLnRzXCI7XG4gKiAgICAgIHRvRmlsZVVybChcIlxcXFxob21lXFxcXGZvb1wiKTsgLy8gbmV3IFVSTChcImZpbGU6Ly8vaG9tZS9mb29cIilcbiAqICAgICAgdG9GaWxlVXJsKFwiQzpcXFxcVXNlcnNcXFxcZm9vXCIpOyAvLyBuZXcgVVJMKFwiZmlsZTovLy9DOi9Vc2Vycy9mb29cIilcbiAqICAgICAgdG9GaWxlVXJsKFwiXFxcXFxcXFwxMjcuMC4wLjFcXFxcaG9tZVxcXFxmb29cIik7IC8vIG5ldyBVUkwoXCJmaWxlOi8vMTI3LjAuMC4xL2hvbWUvZm9vXCIpXG4gKiBgYGBcbiAqIEBwYXJhbSBwYXRoIHRvIGNvbnZlcnQgdG8gZmlsZSBVUkxcbiAqL1xuZXhwb3J0IGZ1bmN0aW9uIHRvRmlsZVVybChwYXRoOiBzdHJpbmcpOiBVUkwge1xuICBpZiAoIWlzQWJzb2x1dGUocGF0aCkpIHtcbiAgICB0aHJvdyBuZXcgVHlwZUVycm9yKFwiTXVzdCBiZSBhbiBhYnNvbHV0ZSBwYXRoLlwiKTtcbiAgfVxuICBjb25zdCBbLCBob3N0bmFtZSwgcGF0aG5hbWVdID0gcGF0aC5tYXRjaChcbiAgICAvXig/OlsvXFxcXF17Mn0oW14vXFxcXF0rKSg/PVsvXFxcXF0oPzpbXi9cXFxcXXwkKSkpPyguKikvLFxuICApITtcbiAgY29uc3QgdXJsID0gbmV3IFVSTChcImZpbGU6Ly8vXCIpO1xuICB1cmwucGF0aG5hbWUgPSBlbmNvZGVXaGl0ZXNwYWNlKHBhdGhuYW1lLnJlcGxhY2UoLyUvZywgXCIlMjVcIikpO1xuICBpZiAoaG9zdG5hbWUgIT0gbnVsbCAmJiBob3N0bmFtZSAhPSBcImxvY2FsaG9zdFwiKSB7XG4gICAgdXJsLmhvc3RuYW1lID0gaG9zdG5hbWU7XG4gICAgaWYgKCF1cmwuaG9zdG5hbWUpIHtcbiAgICAgIHRocm93IG5ldyBUeXBlRXJyb3IoXCJJbnZhbGlkIGhvc3RuYW1lLlwiKTtcbiAgICB9XG4gIH1cbiAgcmV0dXJuIHVybDtcbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSwwRUFBMEU7QUFDMUUsaURBQWlEO0FBQ2pELDZEQUE2RDtBQUM3RCxxQ0FBcUM7QUFHckMsU0FDRSxtQkFBbUIsRUFDbkIsVUFBVSxFQUNWLFFBQVEsRUFDUixrQkFBa0IsUUFDYixpQkFBaUIsQ0FBQztBQUV6QixTQUNFLE9BQU8sRUFDUCxVQUFVLEVBQ1YsZ0JBQWdCLEVB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