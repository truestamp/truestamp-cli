import { access, accessSync } from "./_fs/_fs_access.ts";
import { appendFile, appendFileSync } from "./_fs/_fs_appendFile.ts";
import { chmod, chmodSync } from "./_fs/_fs_chmod.ts";
import { chown, chownSync } from "./_fs/_fs_chown.ts";
import { close, closeSync } from "./_fs/_fs_close.ts";
import * as constants from "./_fs/_fs_constants.ts";
import { readFile, readFileSync } from "./_fs/_fs_readFile.ts";
import { readlink, readlinkSync } from "./_fs/_fs_readlink.ts";
import { exists, existsSync } from "./_fs/_fs_exists.ts";
import { mkdir, mkdirSync } from "./_fs/_fs_mkdir.ts";
import { mkdtemp, mkdtempSync } from "./_fs/_fs_mkdtemp.ts";
import { copyFile, copyFileSync } from "./_fs/_fs_copy.ts";
import { writeFile, writeFileSync } from "./_fs/_fs_writeFile.ts";
import { readdir, readdirSync } from "./_fs/_fs_readdir.ts";
import { realpath, realpathSync } from "./_fs/_fs_realpath.ts";
import { rename, renameSync } from "./_fs/_fs_rename.ts";
import { rmdir, rmdirSync } from "./_fs/_fs_rmdir.ts";
import { unlink, unlinkSync } from "./_fs/_fs_unlink.ts";
import { watch } from "./_fs/_fs_watch.ts";
import { open, openSync } from "./_fs/_fs_open.ts";
import { stat, statSync } from "./_fs/_fs_stat.ts";
import { lstat, lstatSync } from "./_fs/_fs_lstat.ts";
import { truncate, truncateSync } from "./_fs/_fs_truncate.ts";
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
    exists,
    existsSync,
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
    unlink,
    unlinkSync,
    watch,
    writeFile,
    writeFileSync,
    truncate,
    truncateSync,
};
export { access, accessSync, appendFile, appendFileSync, chmod, chmodSync, chown, chownSync, close, closeSync, constants, copyFile, copyFileSync, exists, existsSync, lstat, lstatSync, mkdir, mkdirSync, mkdtemp, mkdtempSync, open, openSync, promises, readdir, readdirSync, readFile, readFileSync, readlink, readlinkSync, realpath, realpathSync, rename, renameSync, rmdir, rmdirSync, stat, statSync, truncate, truncateSync, unlink, unlinkSync, watch, writeFile, writeFileSync, };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZnMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJmcy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsTUFBTSxFQUFFLFVBQVUsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3pELE9BQU8sRUFBRSxVQUFVLEVBQUUsY0FBYyxFQUFFLE1BQU0seUJBQXlCLENBQUM7QUFDckUsT0FBTyxFQUFFLEtBQUssRUFBRSxTQUFTLEVBQUUsTUFBTSxvQkFBb0IsQ0FBQztBQUN0RCxPQUFPLEVBQUUsS0FBSyxFQUFFLFNBQVMsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ3RELE9BQU8sRUFBRSxLQUFLLEVBQUUsU0FBUyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDdEQsT0FBTyxLQUFLLFNBQVMsTUFBTSx3QkFBd0IsQ0FBQztBQUNwRCxPQUFPLEVBQUUsUUFBUSxFQUFFLFlBQVksRUFBRSxNQUFNLHVCQUF1QixDQUFDO0FBQy9ELE9BQU8sRUFBRSxRQUFRLEVBQUUsWUFBWSxFQUFFLE1BQU0sdUJBQXVCLENBQUM7QUFDL0QsT0FBTyxFQUFFLE1BQU0sRUFBRSxVQUFVLEVBQUUsTUFBTSxxQkFBcUIsQ0FBQztBQUN6RCxPQUFPLEVBQUUsS0FBSyxFQUFFLFNBQVMsRUFBRSxNQUFNLG9CQUFvQixDQUFDO0FBQ3RELE9BQU8sRUFBRSxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDNUQsT0FBTyxFQUFFLFFBQVEsRUFBRSxZQUFZLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUMzRCxPQUFPLEVBQUUsU0FBUyxFQUFFLGFBQWEsRUFBRSxNQUFNLHdCQUF3QixDQUFDO0FBQ2xFLE9BQU8sRUFBRSxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0sc0JBQXNCLENBQUM7QUFDNUQsT0FBTyxFQUFFLFFBQVEsRUFBRSxZQUFZLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUMvRCxPQUFPLEVBQUUsTUFBTSxFQUFFLFVBQVUsRUFBRSxNQUFNLHFCQUFxQixDQUFDO0FBQ3pELE9BQU8sRUFBRSxLQUFLLEVBQUUsU0FBUyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDdEQsT0FBTyxFQUFFLE1BQU0sRUFBRSxVQUFVLEVBQUUsTUFBTSxxQkFBcUIsQ0FBQztBQUN6RCxPQUFPLEVBQUUsS0FBSyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDM0MsT0FBTyxFQUFFLElBQUksRUFBRSxRQUFRLEVBQUUsTUFBTSxtQkFBbUIsQ0FBQztBQUNuRCxPQUFPLEVBQUUsSUFBSSxFQUFFLFFBQVEsRUFBRSxNQUFNLG1CQUFtQixDQUFDO0FBQ25ELE9BQU8sRUFBRSxLQUFLLEVBQUUsU0FBUyxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFDdEQsT0FBTyxFQUFFLFFBQVEsRUFBRSxZQUFZLEVBQUUsTUFBTSx1QkFBdUIsQ0FBQztBQUUvRCxPQUFPLEtBQUssUUFBUSxNQUFNLHVCQUF1QixDQUFDO0FBRWxELGVBQWU7SUFDYixNQUFNO0lBQ04sVUFBVTtJQUNWLFVBQVU7SUFDVixjQUFjO0lBQ2QsS0FBSztJQUNMLFNBQVM7SUFDVCxLQUFLO0lBQ0wsU0FBUztJQUNULEtBQUs7SUFDTCxTQUFTO0lBQ1QsU0FBUztJQUNULFFBQVE7SUFDUixZQUFZO0lBQ1osTUFBTTtJQUNOLFVBQVU7SUFDVixLQUFLO0lBQ0wsU0FBUztJQUNULEtBQUs7SUFDTCxTQUFTO0lBQ1QsT0FBTztJQUNQLFdBQVc7SUFDWCxJQUFJO0lBQ0osUUFBUTtJQUNSLFFBQVE7SUFDUixPQUFPO0lBQ1AsV0FBVztJQUNYLFFBQVE7SUFDUixZQUFZO0lBQ1osUUFBUTtJQUNSLFlBQVk7SUFDWixRQUFRO0lBQ1IsWUFBWTtJQUNaLE1BQU07SUFDTixVQUFVO0lBQ1YsS0FBSztJQUNMLFNBQVM7SUFDVCxJQUFJO0lBQ0osUUFBUTtJQUNSLE1BQU07SUFDTixVQUFVO0lBQ1YsS0FBSztJQUNMLFNBQVM7SUFDVCxhQUFhO0lBQ2IsUUFBUTtJQUNSLFlBQVk7Q0FDYixDQUFDO0FBRUYsT0FBTyxFQUNMLE1BQU0sRUFDTixVQUFVLEVBQ1YsVUFBVSxFQUNWLGNBQWMsRUFDZCxLQUFLLEVBQ0wsU0FBUyxFQUNULEtBQUssRUFDTCxTQUFTLEVBQ1QsS0FBSyxFQUNMLFNBQVMsRUFDVCxTQUFTLEVBQ1QsUUFBUSxFQUNSLFlBQVksRUFDWixNQUFNLEVBQ04sVUFBVSxFQUNWLEtBQUssRUFDTCxTQUFTLEVBQ1QsS0FBSyxFQUNMLFNBQVMsRUFDVCxPQUFPLEVBQ1AsV0FBVyxFQUNYLElBQUksRUFDSixRQUFRLEVBQ1IsUUFBUSxFQUNSLE9BQU8sRUFDUCxXQUFXLEVBQ1gsUUFBUSxFQUNSLFlBQVksRUFDWixRQUFRLEVBQ1IsWUFBWSxFQUNaLFFBQVEsRUFDUixZQUFZLEVBQ1osTUFBTSxFQUNOLFVBQVUsRUFDVixLQUFLLEVBQ0wsU0FBUyxFQUNULElBQUksRUFDSixRQUFRLEVBQ1IsUUFBUSxFQUNSLFlBQVksRUFDWixNQUFNLEVBQ04sVUFBVSxFQUNWLEtBQUssRUFDTCxTQUFTLEVBQ1QsYUFBYSxHQUNkLENBQUMifQ==