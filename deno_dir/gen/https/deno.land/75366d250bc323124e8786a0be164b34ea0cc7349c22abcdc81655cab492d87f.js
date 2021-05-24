import { assert } from "../_util/assert.ts";
import { basename, join, normalize } from "../path/mod.ts";
export function _createWalkEntrySync(path) {
    path = normalize(path);
    const name = basename(path);
    const info = Deno.statSync(path);
    return {
        path,
        name,
        isFile: info.isFile,
        isDirectory: info.isDirectory,
        isSymlink: info.isSymlink,
    };
}
export async function _createWalkEntry(path) {
    path = normalize(path);
    const name = basename(path);
    const info = await Deno.stat(path);
    return {
        path,
        name,
        isFile: info.isFile,
        isDirectory: info.isDirectory,
        isSymlink: info.isSymlink,
    };
}
function include(path, exts, match, skip) {
    if (exts && !exts.some((ext) => path.endsWith(ext))) {
        return false;
    }
    if (match && !match.some((pattern) => !!path.match(pattern))) {
        return false;
    }
    if (skip && skip.some((pattern) => !!path.match(pattern))) {
        return false;
    }
    return true;
}
export async function* walk(root, { maxDepth = Infinity, includeFiles = true, includeDirs = true, followSymlinks = false, exts = undefined, match = undefined, skip = undefined, } = {}) {
    if (maxDepth < 0) {
        return;
    }
    if (includeDirs && include(root, exts, match, skip)) {
        yield await _createWalkEntry(root);
    }
    if (maxDepth < 1 || !include(root, undefined, undefined, skip)) {
        return;
    }
    for await (const entry of Deno.readDir(root)) {
        assert(entry.name != null);
        let path = join(root, entry.name);
        if (entry.isSymlink) {
            if (followSymlinks) {
                path = await Deno.realPath(path);
            }
            else {
                continue;
            }
        }
        if (entry.isFile) {
            if (includeFiles && include(path, exts, match, skip)) {
                yield { path, ...entry };
            }
        }
        else {
            yield* walk(path, {
                maxDepth: maxDepth - 1,
                includeFiles,
                includeDirs,
                followSymlinks,
                exts,
                match,
                skip,
            });
        }
    }
}
export function* walkSync(root, { maxDepth = Infinity, includeFiles = true, includeDirs = true, followSymlinks = false, exts = undefined, match = undefined, skip = undefined, } = {}) {
    if (maxDepth < 0) {
        return;
    }
    if (includeDirs && include(root, exts, match, skip)) {
        yield _createWalkEntrySync(root);
    }
    if (maxDepth < 1 || !include(root, undefined, undefined, skip)) {
        return;
    }
    for (const entry of Deno.readDirSync(root)) {
        assert(entry.name != null);
        let path = join(root, entry.name);
        if (entry.isSymlink) {
            if (followSymlinks) {
                path = Deno.realPathSync(path);
            }
            else {
                continue;
            }
        }
        if (entry.isFile) {
            if (includeFiles && include(path, exts, match, skip)) {
                yield { path, ...entry };
            }
        }
        else {
            yield* walkSync(path, {
                maxDepth: maxDepth - 1,
                includeFiles,
                includeDirs,
                followSymlinks,
                exts,
                match,
                skip,
            });
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid2Fsay5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIndhbGsudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBR0EsT0FBTyxFQUFFLE1BQU0sRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQzVDLE9BQU8sRUFBRSxRQUFRLEVBQUUsSUFBSSxFQUFFLFNBQVMsRUFBRSxNQUFNLGdCQUFnQixDQUFDO0FBRzNELE1BQU0sVUFBVSxvQkFBb0IsQ0FBQyxJQUFZO0lBQy9DLElBQUksR0FBRyxTQUFTLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDdkIsTUFBTSxJQUFJLEdBQUcsUUFBUSxDQUFDLElBQUksQ0FBQyxDQUFDO0lBQzVCLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxRQUFRLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDakMsT0FBTztRQUNMLElBQUk7UUFDSixJQUFJO1FBQ0osTUFBTSxFQUFFLElBQUksQ0FBQyxNQUFNO1FBQ25CLFdBQVcsRUFBRSxJQUFJLENBQUMsV0FBVztRQUM3QixTQUFTLEVBQUUsSUFBSSxDQUFDLFNBQVM7S0FDMUIsQ0FBQztBQUNKLENBQUM7QUFHRCxNQUFNLENBQUMsS0FBSyxVQUFVLGdCQUFnQixDQUFDLElBQVk7SUFDakQsSUFBSSxHQUFHLFNBQVMsQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUN2QixNQUFNLElBQUksR0FBRyxRQUFRLENBQUMsSUFBSSxDQUFDLENBQUM7SUFDNUIsTUFBTSxJQUFJLEdBQUcsTUFBTSxJQUFJLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO0lBQ25DLE9BQU87UUFDTCxJQUFJO1FBQ0osSUFBSTtRQUNKLE1BQU0sRUFBRSxJQUFJLENBQUMsTUFBTTtRQUNuQixXQUFXLEVBQUUsSUFBSSxDQUFDLFdBQVc7UUFDN0IsU0FBUyxFQUFFLElBQUksQ0FBQyxTQUFTO0tBQzFCLENBQUM7QUFDSixDQUFDO0FBWUQsU0FBUyxPQUFPLENBQ2QsSUFBWSxFQUNaLElBQWUsRUFDZixLQUFnQixFQUNo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