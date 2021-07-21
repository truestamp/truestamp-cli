import { access, accessSync } from "./_fs/_fs_access.ts";
import { appendFile, appendFileSync } from "./_fs/_fs_appendFile.ts";
import { chmod, chmodSync } from "./_fs/_fs_chmod.ts";
import { chown, chownSync } from "./_fs/_fs_chown.ts";
import { close, closeSync } from "./_fs/_fs_close.ts";
import * as constants from "./_fs/_fs_constants.ts";
import { copyFile, copyFileSync } from "./_fs/_fs_copy.ts";
import Dir from "./_fs/_fs_dir.ts";
import Dirent from "./_fs/_fs_dirent.ts";
import { exists, existsSync } from "./_fs/_fs_exists.ts";
import { fdatasync, fdatasyncSync } from "./_fs/_fs_fdatasync.ts";
import { fstat, fstatSync } from "./_fs/_fs_fstat.ts";
import { fsync, fsyncSync } from "./_fs/_fs_fsync.ts";
import { ftruncate, ftruncateSync } from "./_fs/_fs_ftruncate.ts";
import { futimes, futimesSync } from "./_fs/_fs_futimes.ts";
import { link, linkSync } from "./_fs/_fs_link.ts";
import { lstat, lstatSync } from "./_fs/_fs_lstat.ts";
import { mkdir, mkdirSync } from "./_fs/_fs_mkdir.ts";
import { mkdtemp, mkdtempSync } from "./_fs/_fs_mkdtemp.ts";
import { open, openSync } from "./_fs/_fs_open.ts";
import { readdir, readdirSync } from "./_fs/_fs_readdir.ts";
import { readFile, readFileSync } from "./_fs/_fs_readFile.ts";
import { readlink, readlinkSync } from "./_fs/_fs_readlink.ts";
import { realpath, realpathSync } from "./_fs/_fs_realpath.ts";
import { rename, renameSync } from "./_fs/_fs_rename.ts";
import { rmdir, rmdirSync } from "./_fs/_fs_rmdir.ts";
import { stat, statSync } from "./_fs/_fs_stat.ts";
import { symlink, symlinkSync } from "./_fs/_fs_symlink.ts";
import { truncate, truncateSync } from "./_fs/_fs_truncate.ts";
import { unlink, unlinkSync } from "./_fs/_fs_unlink.ts";
import { utimes, utimesSync } from "./_fs/_fs_utimes.ts";
import { watch } from "./_fs/_fs_watch.ts";
import { writeFile, writeFileSync } from "./_fs/_fs_writeFile.ts";
import * as promises from "./_fs/promises/mod.ts";
export default {
    access,
    accessSync,
    appendFile,
    appendFileSync,
    chmod,
    chmodSync,
    chown,
    chownSync,
    close,
    closeSync,
    constants,
    copyFile,
    copyFileSync,
    Dir,
    Dirent,
    exists,
    existsSync,
    fdatasync,
    fdatasyncSync,
    fstat,
    fstatSync,
    fsync,
    fsyncSync,
    ftruncate,
    ftruncateSync,
    futimes,
    futimesSync,
    link,
    linkSync,
    lstat,
    lstatSync,
    mkdir,
    mkdirSync,
    mkdtemp,
    mkdtempSync,
    open,
    openSync,
    promises,
    readdir,
    readdirSync,
    readFile,
    readFileSync,
    readlink,
    readlinkSync,
    realpath,
    realpathSync,
    rename,
    renameSync,
    rmdir,
    rmdirSync,
    stat,
    statSync,
    symlink,
    symlinkSync,
    truncate,
    truncateSync,
    unlink,
    unlinkSync,
    utimes,
    utimesSync,
    watch,
    writeFile,
    writeFileSync,
};
export { access, accessSync, appendFile, appendFileSync, chmod, chmodSync, chown, chownSync, close, closeSync, constants, copyFile, copyFileSync, Dir, Dirent, exists, existsSync, fdatasync, fdatasyncSync, fstat, fstatSync, fsync, fsyncSync, ftruncate, ftruncateSync, futimes, futimesSync, link, linkSync, lstat, lstatSync, mkdir, mkdirSync, mkdtemp, mkdtempSync, open, openSync, promises, readdir, readdirSync, readFile, readFileSync, readlink, readlinkSync, realpath, realpathSync, rename, renameSync, rmdir, rmdirSync, stat, statSync, symlink, symlinkSync, truncate, truncateSync, unlink, unlinkSync, utimes, utimesSync, watch, writeFile, writeFileSync, };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJmcy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsTUFBTSxFQUFFLFVBQVUsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3pELE9BQU8sRUFBRSxVQUFVLEVBQUUsY0FBYyxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFDckUsT0FBTyxFQUFFLEtBQUssRUFBRSxTQUFTLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUN0RCxPQUFPLEVBQUUsS0FBSyxFQUFFLFNBQVMsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ3RELE9BQU8sRUFBRSxLQUFLLEVBQUUsU0FBUyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDdEQsT0FBTyxLQUFLLFNBQVMsTUFBTSx3QkFBd0IsQ0FBQztBQUNwRCxPQUFPLEVBQUUsUUFBUSxFQUFFLFlBQVksRUFBRSxNQUFNLG1CQUFtQixDQUFDO0FBQzNELE9BQU8sR0FBRyxNQUFNLGtCQUFrQixDQUFDO0FBQ25DLE9BQU8sTUFBTSxNQUFNLHFCQUFxQixDQUFDO0FBQ3pDLE9BQU8sRUFBRSxNQUFNLEVBQUUsVUFBVSxFQUFFLE1BQU0scUJBQXFCLENBQUM7QUFDekQsT0FBTyxFQUFFLFNBQVMsRUFBRSxhQUFhLEVBQUUsTUFBTSx3QkFBd0IsQ0FBQztBQUNsRSxPQUFPLEVBQUUsS0FBSyxFQUFFLFNBQVMsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ3RELE9BQU8sRUFBRSxLQUFLLEVBQUUsU0FBUyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDdEQsT0FBTyxFQUFFLFNBQVMsRUFBRSxhQUFhLEVBQUUsTUFBTSx3QkFBd0IsQ0FBQztBQUNsRSxPQUFPLEVBQUUsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLHNCQUFzQixDQUFDO0FBQzVELE9BQU8sRUFBRSxJQUFJLEVBQUUsUUFBUSxFQUFFLE1BQU0sbUJBQW1CLENBQUM7QUFDbkQsT0FBTyxFQUFFLEtBQUss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