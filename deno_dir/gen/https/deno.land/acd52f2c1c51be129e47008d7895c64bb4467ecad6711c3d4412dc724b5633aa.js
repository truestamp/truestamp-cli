import { getStr } from "./wasm.ts";
import { Status } from "./constants.ts";
/**
 * Errors which can be thrown while interacting with
 * a database.
 */ export class SqliteError extends Error {
    /**
   * Extension over the standard JS Error object
   * to also contain class members for error code
   * and error code name.
   *
   * Instances of this class should not be constructed
   * directly and should only be obtained
   * from exceptions raised in this module.
   */ constructor(context, code){
        let message;
        let status;
        if (typeof context === "string") {
            message = context;
            status = Status.Unknown;
        } else {
            message = getStr(context, context.get_sqlite_error_str());
            status = context.get_status();
        }
        super(message);
        this.code = code ?? status;
        this.name = "SqliteError";
    }
    /**
   * The SQLite status code which caused this error.
   *
   * Errors that originate in the JavaScript part of
   * the library will not have an associated status
   * code. For these errors, the code will be
   * `Status.Unknown`.
   *
   * These codes are accessible via
   * the exported `Status` object.
   */ code;
    /**
   * Key of code in exported `status`
   * object.
   *
   * E.g. if `code` is `19`,
   * `codeName` would be `SqliteConstraint`.
   */ get codeName() {
        return Status[this.code];
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvc3FsaXRlQHYzLjQuMC9zcmMvZXJyb3IudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgV2FzbSB9IGZyb20gXCIuLi9idWlsZC9zcWxpdGUuanNcIjtcbmltcG9ydCB7IGdldFN0ciB9IGZyb20gXCIuL3dhc20udHNcIjtcbmltcG9ydCB7IFN0YXR1cyB9IGZyb20gXCIuL2NvbnN0YW50cy50c1wiO1xuXG4vKipcbiAqIEVycm9ycyB3aGljaCBjYW4gYmUgdGhyb3duIHdoaWxlIGludGVyYWN0aW5nIHdpdGhcbiAqIGEgZGF0YWJhc2UuXG4gKi9cbmV4cG9ydCBjbGFzcyBTcWxpdGVFcnJvciBleHRlbmRzIEVycm9yIHtcbiAgLyoqXG4gICAqIEV4dGVuc2lvbiBvdmVyIHRoZSBzdGFuZGFyZCBKUyBFcnJvciBvYmplY3RcbiAgICogdG8gYWxzbyBjb250YWluIGNsYXNzIG1lbWJlcnMgZm9yIGVycm9yIGNvZGVcbiAgICogYW5kIGVycm9yIGNvZGUgbmFtZS5cbiAgICpcbiAgICogSW5zdGFuY2VzIG9mIHRoaXMgY2xhc3Mgc2hvdWxkIG5vdCBiZSBjb25zdHJ1Y3RlZFxuICAgKiBkaXJlY3RseSBhbmQgc2hvdWxkIG9ubHkgYmUgb2J0YWluZWRcbiAgICogZnJvbSBleGNlcHRpb25zIHJhaXNlZCBpbiB0aGlzIG1vZHVsZS5cbiAgICovXG4gIGNvbnN0cnVjdG9yKGNvbnRleHQ6IFdhc20gfCBzdHJpbmcsIGNvZGU/OiBTdGF0dXMpIHtcbiAgICBsZXQgbWVzc2FnZTtcbiAgICBsZXQgc3RhdHVzO1xuICAgIGlmICh0eXBlb2YgY29udGV4dCA9PT0gXCJzdHJpbmdcIikge1xuICAgICAgbWVzc2FnZSA9IGNvbnRleHQ7XG4gICAgICBzdGF0dXMgPSBTdGF0dXMuVW5rbm93bjtcbiAgICB9IGVsc2Uge1xuICAgICAgbWVzc2FnZSA9IGdldFN0cihjb250ZXh0LCBjb250ZXh0LmdldF9zcWxpdGVfZXJyb3Jfc3RyKCkpO1xuICAgICAgc3RhdHVzID0gY29udGV4dC5nZXRfc3RhdHVzKCk7XG4gICAgfVxuICAgIHN1cGVyKG1lc3NhZ2UpO1xuICAgIHRoaXMuY29kZSA9IGNvZGUgPz8gc3RhdHVzO1xuICAgIHRoaXMubmFtZSA9IFwiU3FsaXRlRXJyb3JcIjtcbiAgfVxuXG4gIC8qKlxuICAgKiBUaGUgU1FMaXRlIHN0YXR1cyBjb2RlIHdoaWNoIGNhdXNlZCB0aGlzIGVycm9yLlxuICAgKlxuICAgKiBFcnJvcnMgdGhhdCBvcmlnaW5hdGUgaW4gdGhlIEphdmFTY3JpcHQgcGFydCBvZlxuICAgKiB0aGUgbGlicmFyeSB3aWxsIG5vdCBoYXZlIGFuIGFzc29jaWF0ZWQgc3RhdHVzXG4gICAqIGNvZGUuIEZvciB0aGVzZSBlcnJvcnMsIHRoZSBjb2RlIHdpbGwgYmVcbiAgICogYFN0YXR1cy5Vbmtub3duYC5cbiAgICpcbiAgICogVGhlc2UgY29kZXMgYXJlIGFjY2Vzc2libGUgdmlhXG4gICAqIHRoZSBleHBvcnRlZCBgU3RhdHVzYCBvYmplY3QuXG4gICAqL1xuICBjb2RlOiBTdGF0dXM7XG5cbiAgLyoqXG4gICAqIEtleSBvZiBjb2RlIGluIGV4cG9ydGVkIGBzdGF0dXNgXG4gICAqIG9iamVjdC5cbiAgICpcbiAgICogRS5nLiBpZiBgY29kZWAgaXMgYDE5YCxcbiAgICogYGNvZGVOYW1lYCB3b3VsZCBiZSBgU3FsaXRlQ29uc3RyYWludGAuXG4gICAqL1xuICBnZXQgY29kZU5hbWUoKToga2V5b2YgdHlwZW9mIFN0YXR1cyB7XG4gICAgcmV0dXJuIFN0YXR1c1t0aGlzLmNvZGVdIGFzIGtleW9mIHR5cGVvZiBTdGF0dXM7XG4gIH1cbn1cbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxTQUFTLE1BQU0sUUFBUSxXQUFXLENBQUM7QUFDbkMsU0FBUyxNQUFNLFFBQVEsZ0JBQWdCLENBQUM7QUFFeEM7OztHQUdHLENBQ0gsT0FBTyxNQUFNLFdBQVcsU0FBUyxLQUFLO0lBQ3BDOzs7Ozs7OztLQVFHLENBQ0gsWUFBWSxPQUFzQixFQUFFLElBQWEsQ0FBRTtRQUNqRCxJQUFJLE9BQU8sQUFBQztRQUNaLElBQUksTUFBTSxBQUFDO1FBQ1gsSUFBSSxPQUFPLE9BQU8sS0FBSyxRQUFRLEVBQUU7WUFDL0IsT0FBTyxHQUFHLE9BQU8sQ0FBQztZQUNsQixNQUFNLEdBQUcsTUFBTSxDQUFDLE9BQU8sQ0FBQztTQUN6QixNQUFNO1lBQ0wsT0FBTyxHQUFHLE1BQU0sQ0FBQyxPQUFPLEVBQUUsT0FBTyxDQUFDLG9CQUFvQixFQUFFLENBQUMsQ0FBQztZQUMxRCxNQUFNLEdBQUcsT0FBTyxDQUFDLFVBQVUsRUFBRSxDQUFDO1NBQy9CO1FBQ0QsS0FBSyxDQUFDLE9BQU8sQ0FBQyxDQUFDO1FBQ2YsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLElBQUksTUFBTSxDQUFDO1FBQzNCLElBQUksQ0FBQyxJQUFJLEdBQUcsYUFBYSxDQUFDO0tBQzNCO0lBRUQ7Ozs7Ozs7Ozs7S0FVRyxDQUNILElBQUksQ0FBUztJQUViOzs7Ozs7S0FNRyxDQUNILElBQUksUUFBUSxHQUF3QjtRQUNsQyxPQUFPLE1BQU0sQ0FBQyxJQUFJLENBQUMsSUFBSSxDQUFDLENBQXdCO0tBQ2pEO0NBQ0YifQ==