import { instantiate } from "../build/sqlite.js";
import { setStr } from "./wasm.ts";
import { OpenFlags, Status, Values } from "./constants.ts";
import { SqliteError } from "./error.ts";
import { PreparedQuery } from "./query.ts";
export class DB {
    _wasm;
    _open;
    _statements;
    _transactionDepth;
    constructor(path = ":memory:", options = {}) {
        this._wasm = instantiate().exports;
        this._open = false;
        this._statements = new Set();
        this._transactionDepth = 0;
        let flags = 0;
        switch (options.mode) {
            case "read":
                flags = OpenFlags.ReadOnly;
                break;
            case "write":
                flags = OpenFlags.ReadWrite;
                break;
            case "create":
            default:
                flags = OpenFlags.ReadWrite | OpenFlags.Create;
                break;
        }
        if (options.memory === true) {
            flags |= OpenFlags.Memory;
        }
        if (options.uri === true) {
            flags |= OpenFlags.Uri;
        }
        const status = setStr(this._wasm, path, (ptr) => this._wasm.open(ptr, flags));
        if (status !== Status.SqliteOk) {
            throw new SqliteError(this._wasm, status);
        }
        this._open = true;
    }
    query(sql, params) {
        const query = this.prepareQuery(sql);
        try {
            const rows = query.all(params);
            query.finalize();
            return rows;
        }
        catch (err) {
            query.finalize();
            throw err;
        }
    }
    queryEntries(sql, params) {
        const query = this.prepareQuery(sql);
        try {
            const rows = query.allEntries(params);
            query.finalize();
            return rows;
        }
        catch (err) {
            query.finalize();
            throw err;
        }
    }
    prepareQuery(sql) {
        if (!this._open) {
            throw new SqliteError("Database was closed.");
        }
        const stmt = setStr(this._wasm, sql, (ptr) => this._wasm.prepare(ptr));
        if (stmt === Values.Null) {
            throw new SqliteError(this._wasm);
        }
        this._statements.add(stmt);
        return new PreparedQuery(this._wasm, stmt, this._statements);
    }
    execute(sql) {
        const status = setStr(this._wasm, sql, (ptr) => this._wasm.exec(ptr));
        if (status !== Status.SqliteOk) {
            throw new SqliteError(this._wasm, status);
        }
    }
    transaction(closure) {
        this._transactionDepth += 1;
        this.query(`SAVEPOINT _deno_sqlite_sp_${this._transactionDepth}`);
        let value;
        try {
            value = closure();
        }
        catch (err) {
            this.query(`ROLLBACK TO _deno_sqlite_sp_${this._transactionDepth}`);
            this._transactionDepth -= 1;
            throw err;
        }
        this.query(`RELEASE _deno_sqlite_sp_${this._transactionDepth}`);
        this._transactionDepth -= 1;
        return value;
    }
    close(force = false) {
        if (!this._open) {
            return;
        }
        if (force) {
            for (const stmt of this._statements) {
                if (this._wasm.finalize(stmt) !== Status.SqliteOk) {
                    throw new SqliteError(this._wasm);
                }
            }
        }
        if (this._wasm.close() !== Status.SqliteOk) {
            throw new SqliteError(this._wasm);
        }
        this._open = false;
    }
    get lastInsertRowId() {
        return this._wasm.last_insert_rowid();
    }
    get changes() {
        return this._wasm.changes();
    }
    get totalChanges() {
        return this._wasm.total_changes();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGIuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJkYi50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsV0FBVyxFQUFzQixNQUFNLG9CQUFvQixDQUFDO0FBQ3JFLE9BQU8sRUFBRSxNQUFNLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDbkMsT0FBTyxFQUFFLFNBQVMsRUFBRSxNQUFNLEVBQUUsTUFBTSxFQUFFLE1BQU0sZ0JBQWdCLENBQUM7QUFDM0QsT0FBTyxFQUFFLFdBQVcsRUFBRSxNQUFNLFlBQVksQ0FBQztBQUN6QyxPQUFPLEVBQUUsYUFBYSxFQUFxQyxNQUFNLFlBQVksQ0FBQztBQXVDOUUsTUFBTSxPQUFPLEVBQUU7SUFDTCxLQUFLLENBQU87SUFDWixLQUFLLENBQVU7SUFDZixXQUFXLENBQW9CO0lBQy9CLGlCQUFpQixDQUFTO0lBNkJsQyxZQUFZLE9BQWUsVUFBVSxFQUFFLFVBQXlCLEVBQUU7UUFDaEUsSUFBSSxDQUFDLEtBQUssR0FBRyxXQUFXLEVBQUUsQ0FBQyxPQUFPLENBQUM7UUFDbkMsSUFBSSxDQUFDLEtBQUssR0FBRyxLQUFLLENBQUM7UUFDbkIsSUFBSSxDQUFDLFdBQVcsR0FBRyxJQUFJLEdBQUcsRUFBRSxDQUFDO1FBQzdCLElBQUksQ0FBQyxpQkFBaUIsR0FBRyxDQUFDLENBQUM7UUFHM0IsSUFBSSxLQUFLLEdBQUcsQ0FBQyxDQUFDO1FBQ2QsUUFBUSxPQUFPLENBQUMsSUFBSSxFQUFFO1lBQ3BCLEtBQUssTUFBTTtnQkFDVCxLQUFLLEdBQUcsU0FBUyxDQUFDLFFBQVEsQ0FBQztnQkFDM0IsTUFBTTtZQUNSLEtBQUssT0FBTztnQkFDVixLQUFLLEdBQUcsU0FBUyxDQUFDLFNBQVMsQ0FBQztnQkFDNUIsTUFBTTtZQUNSLEtBQUssUUFBUSxDQUFDO1lBQ2Q7Z0JBQ0UsS0FBSyxHQUFHLFNBQVMsQ0FBQyxTQUFTLEdBQUcsU0FBUyxDQUFDLE1BQU0sQ0FBQztnQkFDL0MsTUFBTTtTQUNUO1FB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RCxJQUFJLENBQUMsS0FBSyxHQUFHLEtBQUssQ0FBQztJQUNyQixDQUFDO0lBU0QsSUFBSSxlQUFlO1FBQ2pCLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxpQkFBaUIsRUFBRSxDQUFDO0lBQ3hDLENBQUM7SUFRRCxJQUFJLE9BQU87UUFDVCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsT0FBTyxFQUFFLENBQUM7SUFDOUIsQ0FBQztJQVFELElBQUksWUFBWTtRQUNkLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxhQUFhLEVBQUUsQ0FBQztJQUNwQyxDQUFDO0NBQ0YiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgeyBpbnN0YW50aWF0ZSwgU3RhdGVtZW50UHRyLCBXYXNtIH0gZnJvbSBcIi4uL2J1aWxkL3NxbGl0ZS5qc1wiO1xuaW1wb3J0IHsgc2V0U3RyIH0gZnJvbSBcIi4vd2FzbS50c1wiO1xuaW1wb3J0IHsgT3BlbkZsYWdzLCBTdGF0dXMsIFZhbHVlcyB9IGZyb20gXCIuL2NvbnN0YW50cy50c1wiO1xuaW1wb3J0IHsgU3FsaXRlRXJyb3IgfSBmcm9tIFwiLi9lcnJvci50c1wiO1xuaW1wb3J0IHsgUHJlcGFyZWRRdWVyeSwgUXVlcnlQYXJhbWV0ZXJTZXQsIFJvdywgUm93T2JqZWN0IH0gZnJvbSBcIi4vcXVlcnkudHNcIjtcblxuLyoqXG4gKiBPcHRpb25zIGZvciBvcGVuaW5nIGEgZGF0YWJhc2UuXG4gKi9cbmV4cG9ydCBpbnRlcmZhY2UgU3FsaXRlT3B0aW9ucyB7XG4gIC8qKlxuICAgKiBNb2RlIGluIHdoaWNoIHRvIG9wZW4gdGhlIGRhdGFiYXNlLlxuICAgKlxuICAgKiAtIGByZWFkYDogcmVhZC1vbmx5LCB0aHJvd3MgYW4gZXJyb3IgaWZcbiAgICogICB0aGUgZGF0YWJhc2UgZmlsZSBkb2VzIG5vdCBleGlzdHNcbiAgICogLSBgd3JpdGVgOiByZWFkLXdyaXRlLCB0aHJvd3MgYW4gZXJyb3JcbiAgICogICBpZiB0aGUgZGF0YWJhc2UgZmlsZSBkb2VzIG5vdCBleGlzdHNcbiAgICogLSBgY3JlYXRlYDogcmVhZC13cml0ZSwgY3JlYXRlIHRoZSBkYXRhYmFzZVxuICAgKiAgIGlmIHRoZSBmaWxlIGRvZXMgbm90IGV4aXN0XG4gICAqXG4gICAqIGBjcmVhdGVgIGlzIHRoZSBkZWZhdWx0IGlmIG5vIG1vZGUgaXNcbiAgICogc3BlY2lmaWVkLlxuICAgKi9cbiAgbW9kZT86IFwicmVhZFwiIHwgXCJ3cml0ZVwiIHwgXCJjcmVhdGVcIjtcbiAgLyoqXG4gICAqIEZvcmNlIHRoZSBkYXRhYmFzZSB0byBiZSBpbi1tZW1vcnkuIFdoZW5cbiAgICogdGhpcyBvcHRpb24gaXMgc2V0LCB0aGUgZGF0YWJhc2UgaXMgb3BlbmVkXG4gICAqIGluIG1lbW9yeSwgcmVnYXJkbGVzcyBvZiB0aGUgc3BlY2lmaWVkXG4gICAqIGZpbGVuYW1lLlxuICAgKi9cbiAgbWVtb3J5PzogYm9vbGVhbjtcbiAgLyoqXG4gICAqIEludGVycHJldCB0aGUgZmlsZSBuYW1lIGFzIGEgVVJJLlxuICAgKiBTZWUgaHR0cHM6Ly9zcWxpdGUub3JnL3VyaS5odG1sXG4gICAqIGZvciBtb3JlIGluZm9ybWF0aW9uLlxuICAgKi9cbiAgdXJpPzogYm9vbGVhbjtcbn1cblxuLyoqXG4gKiBBIGRhdGFiYXNlIGhhbmRsZSB0aGF0IGNhbiBiZSB1c2VkIHRvIHJ1blxuICogcXVlcmllcy5cbiAqL1xuZXhwb3J0IGNsYXNzIERCIHtcbiAgcHJpdmF0ZSBfd2FzbTogV2FzbTtcbiAgcHJpdmF0ZSBfb3BlbjogYm9vbGVhbjtcbiAgcHJpdmF0ZSBfc3RhdGVtZW50czogU2V0PFN0YXRlbWVudFB0cj47XG4gIHByaXZhdGUgX3RyYW5zYWN0aW9uRGVwdGg6IG51bWJlcjtcblxuICAvKipcbiAgICogQ3JlYXRlIGEgbmV3IGRhdGFiYXNlLiBUaGUgZmlsZSBhdCB0aGVcbiAgICogZ2l2ZW4gcGF0aCB3aWxsIGJlIG9wZW5lZCB3aXRoIHRoZVxuICAgKiBtb2RlIHNwZWNpZmllZCBpbiBvcHRpb25zLiBUaGUgZGVmYXVsdFxuICAgKiBtb2RlIGlzIGBjcmVhdGVgLlxuICAgKlxuICAgKiBJZiBubyBwYXRoIGlzIGdpdmVuLCBvciBpZiB0aGUgYG1lbW9yeWBcbiAgICogb3B0aW9uIGlzIHNldCwgdGhlIGRhdGFiYXNlIGlzIG9wZW5lZCBpblxuICAgKiBtZW1vcnkuXG4gICAqXG4gICAqICMgRXhhbXBsZXNcbiAgICpcbiAgICogQ3JlYXRlIGFuIGluLW1lbW9yeSBkYXRhYmFzZS5cbiAgICogYGBgdHlwZXNjcmlwdFxuICAgKiBjb25zdCBkYiA9IG5ldyBEQigpO1xuICAgKiBgYGBcbiAgICpcbiAgICogT3BlbiBhIGRhdGFiYXNlIGJhY2tlZCBieSBhIGZpbGUgb24gZGlzay5cbiAgICogYGBgdHlwZXNjcmlwdFxuICAgKiBjb25zdCBkYiA9IG5ldyBEQihcInBhdGgvdG8vZGF0YWJhc2Uuc3FsaXRlXCIpO1xuICAgKiBgYGBcbiAgICpcbiAgICogUGFzcyBvcHRpb25zIHRvIG9wZW4gYSByZWFkLW9ubHkgZGF0YWJhc2UuXG4gICAqIGBgYHR5cGVzY3JpcHRcbiAgICogY29uc3QgZGIgPSBuZXcgREIoXCJwYXRoL3RvL2RhdGFiYXNlLnNxbGl0ZVwiLCB7IG1vZGU6IFwicmVhZFwiIH0pO1xuICAgKiBgYGBcbiAgICovXG4gIGNvbnN0cnVjdG9yKHBhdGg6IHN0cmluZyA9IFwiOm1lbW9yeTpcIiwgb3B0aW9uczogU3FsaXRlT3B0aW9ucyA9IHt9KSB7XG4gICAgdGhpcy5fd2FzbSA9IGluc3RhbnRpYXRlKCkuZXhwb3J0cztcbiAgICB0aGlzLl9vcGVuID0gZmFsc2U7XG4gICAgdGhpcy5fc3RhdGVtZW50cyA9IG5ldyBTZXQoKTtcbiAgICB0aGlzLl90cmFuc2FjdGlvbkRlcHRoID0gMDtcblxuICAgIC8vIENvbmZpZ3VyZSBmbGFnc1xuICAgIGxldCBmbGFncyA9IDA7XG4gICAgc3dpdGNoIChvcHRpb25zLm1vZGUpIHtcbiAgICAgIGNhc2UgXCJyZWFkXCI6XG4gICAgICAgIGZsYWdzID0gT3BlbkZsYWdzLlJlYWRPbmx5O1xuICAgICAgICBicmVhaztcbiAgICAgIGNhc2UgXCJ3cml0ZVwiOlxuICAgICAgICBmbGFncyA9IE9wZW5GbGFncy5SZWFkV3JpdGU7XG4gICAgICAgIGJyZWFrO1xuICAgICAgY2FzZSBcImNyZWF0ZVwiOiAvLyBmYWxsIHRocm91Z2hcbiAgICAgIGRlZmF1bHQ6XG4gICAgICAgIGZsYWdzID0gT3BlbkZsYWdzLlJlYWRXcml0ZSB8IE9wZW5GbGFncy5DcmVhdGU7XG4gICAgICAgIGJyZWFrO1xuICAgIH1cbiAgICBpZiAob3B0aW9ucy5tZW1vcnkgPT09IHRydWUpIHtcbiAgICAgIGZsYWdzIHw9IE9wZW5GbGFncy5NZW1vcnk7XG4gICAgfVxuICAgIGlmIChvcHRpb25zLnVyaSA9PT0gdHJ1ZSkge1xuICAgICAgZmxhZ3MgfD0gT3BlbkZsYWdzLlVyaTtcbiAgICB9XG5cbiAgICAvLyBUcnkgdG8gb3BlbiB0aGUgZGF0YWJhc2VcbiAgICBjb25zdCBzdGF0dXMgPSBzZXRTdHIoXG4gICAgICB0aGlzLl93YXNtLFxuICAgICAgcGF0aCxcbiAgICAgIChwdHIpID0+IHRoaXMuX3dhc20ub3BlbihwdHIsIGZsYWdzKSxcbiAgICApO1xuICAgIGlmIChzdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVPaykge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKHRoaXMuX3dhc20sIHN0YXR1cyk7XG4gICAgfVxuICAgIHRoaXMuX29wZW4gPSB0cnVlO1xuICB9XG5cbiAgLyoqXG4gICAqIFF1ZXJ5IHRoZSBkYXRhYmFzZSBhbmQgcmV0dXJuIGFsbCBtYXRjaGluZ1xuICAgKiByb3dzLlxuICAgKlxuICAgKiBUaGlzIGlzIGVxdWl2YWxlbnQgdG8gY2FsbGluZyBgYWxsYCBvblxuICAgKiBhIHByZXBhcmVkIHF1ZXJ5IHdoaWNoIGlzIHRoZW4gaW1tZWRpYXRlbHlcbiAgICogZmluYWxpemVkLlxuICAgKlxuICAgKiBUaGUgdHlwZSBwYXJhbWV0ZXIgYFJgIG1heSBiZSBzdXBwbGllZCBieVxuICAgKiB0aGUgdXNlciB0byBpbmRpY2F0ZWQgdGhlIHR5cGUgZm9yIHRoZSByb3dzIHJldHVybmVkXG4gICAqIGJ5IHRoZSBxdWVyeS4gTm90aWNlIHRoYXQgdGhlIHVzZXIgaXMgcmVzcG9uc2libGVcbiAgICogZm9yIGVuc3VyaW5nIHRoZSBjb3JyZWN0bmVzcyBvZiB0aGUgc3VwcGxpZWQgdHlwZS5cbiAgICpcbiAgICogVG8gYXZvaWQgU1FMIGluamVjdGlvbiwgdXNlci1wcm92aWRlZCB2YWx1ZXNcbiAgICogc2hvdWxkIGFsd2F5cyBiZSBwYXNzZWQgdG8gdGhlIGRhdGFiYXNlIHRocm91Z2hcbiAgICogYSBxdWVyeSBwYXJhbWV0ZXIuXG4gICAqXG4gICAqIFNlZSBgUXVlcnlQYXJhbWV0ZXJTZXRgIGZvciBkb2N1bWVudGF0aW9uIG9uXG4gICAqIGhvdyB2YWx1ZXMgY2FuIGJlIGJvdW5kIHRvIFNRTCBzdGF0ZW1lbnRzLlxuICAgKlxuICAgKiBTZWUgYFF1ZXJ5UGFyYW1ldGVyYCBmb3IgZG9jdW1lbnRhdGlvbiBvbiBob3dcbiAgICogdmFsdWVzIGFyZSByZXR1cm5lZCBmcm9tIHRoZSBkYXRhYmFzZS5cbiAgICpcbiAgICogIyBFeGFtcGxlc1xuICAgKlxuICAgKiBgYGB0eXBlc2NyaXB0XG4gICAqIGNvbnN0IHJvd3MgPSBkYi5xdWVyeTxbc3RyaW5nLCBudW1iZXJdPihcIlNFTEVDVCBuYW1lLCBhZ2UgRlJPTSBwZW9wbGUgV0hFUkUgY2l0eSA9ID9cIiwgW2NpdHldKTtcbiAgICogLy8gcm93cyA9IFtbXCJQZXRlciBQYXJrZXJcIiwgMjFdLCAuLi5dXG4gICAqIGBgYFxuICAgKi9cbiAgcXVlcnk8UiBleHRlbmRzIFJvdyA9IFJvdz4oXG4gICAgc3FsOiBzdHJpbmcsXG4gICAgcGFyYW1zPzogUXVlcnlQYXJhbWV0ZXJTZXQsXG4gICk6IEFycmF5PFI+IHtcbiAgICBjb25zdCBxdWVyeSA9IHRoaXMucHJlcGFyZVF1ZXJ5PFI+KHNxbCk7XG4gICAgdHJ5IHtcbiAgICAgIGNvbnN0IHJvd3MgPSBxdWVyeS5hbGwocGFyYW1zKTtcbiAgICAgIHF1ZXJ5LmZpbmFsaXplKCk7XG4gICAgICByZXR1cm4gcm93cztcbiAgICB9IGNhdGNoIChlcnIpIHtcbiAgICAgIHF1ZXJ5LmZpbmFsaXplKCk7XG4gICAgICB0aHJvdyBlcnI7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIExpa2UgYHF1ZXJ5YCBleGNlcHQgZWFjaCByb3cgaXMgcmV0dXJuZWRcbiAgICogYXMgYW4gb2JqZWN0IGNvbnRhaW5pbmcga2V5LXZhbHVlIHBhaXJzLlxuICAgKlxuICAgKiAjIEV4YW1wbGVzXG4gICAqXG4gICAqIGBgYHR5cGVzY3JpcHRcbiAgICogY29uc3Qgcm93cyA9IGRiLnF1ZXJ5PHsgbmFtZTogc3RyaW5nLCBhZ2U6IG51bWJlciB9PihcIlNFTEVDVCBuYW1lLCBhZ2UgRlJPTSBwZW9wbGVcIik7XG4gICAqIC8vIHJvd3MgPSBbeyBuYW1lOiBcIlBldGVyIFBhcmtlclwiLCBhZ2U6IDIxIH0sIC4uLl1cbiAgICogYGBgXG4gICAqL1xuICBxdWVyeUVudHJpZXM8TyBleHRlbmRzIFJvd09iamVjdCA9IFJvd09iamVjdD4oXG4gICAgc3FsOiBzdHJpbmcsXG4gICAgcGFyYW1zPzogUXVlcnlQYXJhbWV0ZXJTZXQsXG4gICk6IEFycmF5PE8+IHtcbiAgICBjb25zdCBxdWVyeSA9IHRoaXMucHJlcGFyZVF1ZXJ5PFJvdywgTz4oc3FsKTtcbiAgICB0cnkge1xuICAgICAgY29uc3Qgcm93cyA9IHF1ZXJ5LmFsbEVudHJpZXMocGFyYW1zKTtcbiAgICAgIHF1ZXJ5LmZpbmFsaXplKCk7XG4gICAgICByZXR1cm4gcm93cztcbiAgICB9IGNhdGNoIChlcnIpIHtcbiAgICAgIHF1ZXJ5LmZpbmFsaXplKCk7XG4gICAgICB0aHJvdyBlcnI7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIFByZXBhcmVzIHRoZSBnaXZlbiBTUUwgcXVlcnksIHNvIHRoYXQgaXRcbiAgICogY2FuIGJlIHJ1biBtdWx0aXBsZSB0aW1lcyBhbmQgcG90ZW50aWFsbHlcbiAgICogd2l0aCBkaWZmZXJlbnQgcGFyYW1ldGVycy5cbiAgICpcbiAgICogSWYgYSBxdWVyeSB3aWxsIGJlIGlzc3VlZCBhIGxvdCwgdGhpcyBpcyBtb3JlXG4gICAqIGVmZmljaWVudCB0aGFuIHVzaW5nIGBxdWVyeWAuIEEgcHJlcGFyZWRcbiAgICogcXVlcnkgYWxzbyBwcm92aWRlcyBtb3JlIGNvbnRyb2wgb3ZlciBob3dcbiAgICogdGhlIHF1ZXJ5IGlzIHJ1biwgYXMgd2VsbCBhcyBhY2Nlc3MgdG8gbWV0YS1kYXRhXG4gICAqIGFib3V0IHRoZSBpc3N1ZWQgcXVlcnkuXG4gICAqXG4gICAqIFRoZSByZXR1cm5lZCBgUHJlcGFyZWRRdWVyeWAgb2JqZWN0IG11c3QgYmVcbiAgICogZmluYWxpemVkIGJ5IGNhbGxpbmcgaXRzIGBmaW5hbGl6ZWAgbWV0aG9kXG4gICAqIG9uY2UgaXQgaXMgbm8gbG9uZ2VyIG5lZWRlZC5cbiAgICpcbiAgICogIyBUeXBpbmcgUXVlcmllc1xuICAgKlxuICAgKiBQcmVwYXJlZCBxdWVyeSBvYmplY3RzIGFjY2VwdCB0aHJlZSB0eXBlIHBhcmFtZXRlcnNcbiAgICogdG8gc3BlY2lmeSBwcmVjaXNlIHR5cGVzIGZvciByZXR1cm5lZCBkYXRhIGFuZFxuICAgKiBxdWVyeSBwYXJhbWV0ZXJzLlxuICAgKlxuICAgKiBUaGUgZmlyc3QgdHlwZSBwYXJhbWV0ZXIgYFJgIGluZGljYXRlcyB0aGUgdHVwbGUgdHlwZVxuICAgKiBmb3Igcm93cyByZXR1cm5lZCBieSB0aGUgcXVlcnkuXG4gICAqXG4gICAqIFRoZSBzZWNvbmQgdHlwZSBwYXJhbWV0ZXIgYE9gIGluZGljYXRlcyB0aGUgcmVjb3JkIHR5cGVcbiAgICogZm9yIHJvd3MgcmV0dXJuZWQgYXMgZW50cmllcyAobWFwcGluZ3MgZnJvbSBjb2x1bW4gbmFtZXNcbiAgICogdG8gdmFsdWVzKS5cbiAgICpcbiAgICogVGhlIHRoaXJkIHR5cGUgcGFyYW1ldGVyIGBQYCBpbmRpY2F0ZXMgdGhlIHR5cGUgdGhpcyBxdWVyeVxuICAgKiBhY2NlcHRzIGFzIHBhcmFtZXRlcnMuXG4gICAqXG4gICAqIE5vdGUsIHRoYXQgdGhlIGNvcnJlY3RuZXNzIG9mIHRob3NlIHR5cGVzIG11c3RcbiAgICogYmUgZ3VhcmFudGVlZCBieSB0aGUgY2FsbGVyIG9mIHRoaXMgZnVuY3Rpb24uXG4gICAqXG4gICAqICMgRXhhbXBsZXNcbiAgICpcbiAgICogYGBgdHlwZXNjcmlwdFxuICAgKiBjb25zdCBxdWVyeSA9IGRiLnByZXBhcmVRdWVyeTxcbiAgICogICBbc3RyaW5nLCBudW1iZXJdLFxuICAgKiAgIHsgbmFtZTogc3RyaW5nLCBhZ2U6IG51bWJlciB9LFxuICAgKiAgIHsgY2l0eTogc3RyaW5nIH0sXG4gICAqICA+KFwiU0VMRUNUIG5hbWUsIGFnZSBGUk9NIHBlb3BsZSBXSEVSRSBjaXR5ID0gOmNpdHlcIik7XG4gICAqIC8vIHVzZSBxdWVyeSAuLi5cbiAgICogcXVlcnkuZmluYWxpemUoKTtcbiAgICogYGBgXG4gICAqL1xuICBwcmVwYXJlUXVlcnk8XG4gICAgUiBleHRlbmRzIFJvdyA9IFJvdyxcbiAgICBPIGV4dGVuZHMgUm93T2JqZWN0ID0gUm93T2JqZWN0LFxuICAgIFAgZXh0ZW5kcyBRdWVyeVBhcmFtZXRlclNldCA9IFF1ZXJ5UGFyYW1ldGVyU2V0LFxuICA+KFxuICAgIHNxbDogc3RyaW5nLFxuICApOiBQcmVwYXJlZFF1ZXJ5PFIsIE8sIFA+IHtcbiAgICBpZiAoIXRoaXMuX29wZW4pIHtcbiAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcihcIkRhdGFiYXNlIHdhcyBjbG9zZWQuXCIpO1xuICAgIH1cblxuICAgIGNvbnN0IHN0bXQgPSBzZXRTdHIoXG4gICAgICB0aGlzLl93YXNtLFxuICAgICAgc3FsLFxuICAgICAgKHB0cikgPT4gdGhpcy5fd2FzbS5wcmVwYXJlKHB0ciksXG4gICAgKTtcbiAgICBpZiAoc3RtdCA9PT0gVmFsdWVzLk51bGwpIHtcbiAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcih0aGlzLl93YXNtKTtcbiAgICB9XG5cbiAgICB0aGlzLl9zdGF0ZW1lbnRzLmFkZChzdG10KTtcbiAgICByZXR1cm4gbmV3IFByZXBhcmVkUXVlcnk8UiwgTywgUD4odGhpcy5fd2FzbSwgc3RtdCwgdGhpcy5fc3RhdGVtZW50cyk7XG4gIH1cblxuICAvKipcbiAgICogUnVuIG11bHRpcGxlIHNlbWljb2xvbi1zZXBhcmF0ZWQgc3RhdGVtZW50cyBmcm9tIGEgc2luZ2xlXG4gICAqIHN0cmluZy5cbiAgICpcbiAgICogVGhpcyBtZXRob2QgY2Fubm90IGJpbmQgYW55IHF1ZXJ5IHBhcmFtZXRlcnMsIGFuZCBhbnlcbiAgICogcmVzdWx0IHJvd3MgYXJlIGRpc2NhcmRlZC4gSXQgaXMgb25seSBmb3IgcnVubmluZyBhIGNodW5rXG4gICAqIG9mIHJhdyBTUUw7IGZvciBleGFtcGxlLCB0byBpbml0aWFsaXplIGEgZGF0YWJhc2UuXG4gICAqXG4gICAqICMgRXhhbXBsZXNcbiAgICpcbiAgICogYGBgdHlwZXNjcmlwdFxuICAgKiBkYi5leGVjdXRlKGBcbiAgICogICBDUkVBVEUgVEFCTEUgcGVvcGxlIChcbiAgICogICAgIGlkIElOVEVHRVIgUFJJTUFSWSBLRVkgQVVUT0lOQ1JFTUVOVCxcbiAgICogICAgIG5hbWUgVEVYVCxcbiAgICogICAgIGFnZSBSRUFMLFxuICAgKiAgICAgY2l0eSBURVhUXG4gICAqICAgKTtcbiAgICogICBJTlNFUlQgSU5UTyBwZW9wbGUgKG5hbWUsIGFnZSwgY2l0eSkgVkFMVUVTIChcIlBldGVyIFBhcmtlclwiLCAyMSwgXCJueWNcIik7XG4gICAqIGApO1xuICAgKiBgYGBcbiAgICovXG4gIGV4ZWN1dGUoc3FsOiBzdHJpbmcpIHtcbiAgICBjb25zdCBzdGF0dXMgPSBzZXRTdHIoXG4gICAgICB0aGlzLl93YXNtLFxuICAgICAgc3FsLFxuICAgICAgKHB0cikgPT4gdGhpcy5fd2FzbS5leGVjKHB0ciksXG4gICAgKTtcblxuICAgIGlmIChzdGF0dXMgIT09IFN0YXR1cy5TcWxpdGVPaykge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKHRoaXMuX3dhc20sIHN0YXR1cyk7XG4gICAgfVxuICB9XG5cbiAgLyoqXG4gICAqIFJ1biBhIGZ1bmN0aW9uIHdpdGhpbiB0aGUgY29udGV4dCBvZiBhIGRhdGFiYXNlXG4gICAqIHRyYW5zYWN0aW9uLiBJZiB0aGUgZnVuY3Rpb24gdGhyb3dzIGFuIGVycm9yLFxuICAgKiB0aGUgdHJhbnNhY3Rpb24gaXMgcm9sbGVkIGJhY2suIE90aGVyd2lzZSwgdGhlXG4gICAqIHRyYW5zYWN0aW9uIGlzIGNvbW1pdHRlZCB3aGVuIHRoZSBmdW5jdGlvbiByZXR1cm5zLlxuICAgKlxuICAgKiBDYWxscyB0byBgdHJhbnNhY3Rpb25gIG1heSBiZSBuZXN0ZWQuIE5lc3RlZCB0cmFuc2FjdGlvbnNcbiAgICogYmVoYXZlIGxpa2UgU1FMaXRlIHNhdmUgcG9pbnRzLlxuICAgKi9cbiAgdHJhbnNhY3Rpb248Vj4oY2xvc3VyZTogKCkgPT4gVik6IFYge1xuICAgIHRoaXMuX3RyYW5zYWN0aW9uRGVwdGggKz0gMTtcbiAgICB0aGlzLnF1ZXJ5KGBTQVZFUE9JTlQgX2Rlbm9fc3FsaXRlX3NwXyR7dGhpcy5fdHJhbnNhY3Rpb25EZXB0aH1gKTtcbiAgICBsZXQgdmFsdWU7XG4gICAgdHJ5IHtcbiAgICAgIHZhbHVlID0gY2xvc3VyZSgpO1xuICAgIH0gY2F0Y2ggKGVycikge1xuICAgICAgdGhpcy5xdWVyeShgUk9MTEJBQ0sgVE8gX2Rlbm9fc3FsaXRlX3NwXyR7dGhpcy5fdHJhbnNhY3Rpb25EZXB0aH1gKTtcbiAgICAgIHRoaXMuX3RyYW5zYWN0aW9uRGVwdGggLT0gMTtcbiAgICAgIHRocm93IGVycjtcbiAgICB9XG4gICAgdGhpcy5xdWVyeShgUkVMRUFTRSBfZGVub19zcWxpdGVfc3BfJHt0aGlzLl90cmFuc2FjdGlvbkRlcHRofWApO1xuICAgIHRoaXMuX3RyYW5zYWN0aW9uRGVwdGggLT0gMTtcbiAgICByZXR1cm4gdmFsdWU7XG4gIH1cblxuICAvKipcbiAgICogQ2xvc2UgdGhlIGRhdGFiYXNlLiBUaGlzIG11c3QgYmUgY2FsbGVkIGlmXG4gICAqIHRoZSBkYXRhYmFzZSBpcyBubyBsb25nZXIgdXNlZCB0byBhdm9pZCBsZWFraW5nXG4gICAqIG9wZW4gZmlsZSBkZXNjcmlwdG9ycy5cbiAgICpcbiAgICogSWYgYGZvcmNlYCBpcyBzcGVjaWZpZWQsIGFueSBhY3RpdmUgYFByZXBhcmVkUXVlcnlgXG4gICAqIHdpbGwgYmUgZmluYWxpemVkLiBPdGhlcndpc2UsIHRoaXMgdGhyb3dzIGlmIHRoZXJlXG4gICAqIGFyZSBhY3RpdmUgcXVlcmllcy5cbiAgICpcbiAgICogYGNsb3NlYCBtYXkgc2FmZWx5IGJlIGNhbGxlZCBtdWx0aXBsZVxuICAgKiB0aW1lcy5cbiAgICovXG4gIGNsb3NlKGZvcmNlID0gZmFsc2UpIHtcbiAgICBpZiAoIXRoaXMuX29wZW4pIHtcbiAgICAgIHJldHVybjtcbiAgICB9XG4gICAgaWYgKGZvcmNlKSB7XG4gICAgICBmb3IgKGNvbnN0IHN0bXQgb2YgdGhpcy5fc3RhdGVtZW50cykge1xuICAgICAgICBpZiAodGhpcy5fd2FzbS5maW5hbGl6ZShzdG10KSAhPT0gU3RhdHVzLlNxbGl0ZU9rKSB7XG4gICAgICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKHRoaXMuX3dhc20pO1xuICAgICAgICB9XG4gICAgICB9XG4gICAgfVxuICAgIGlmICh0aGlzLl93YXNtLmNsb3NlKCkgIT09IFN0YXR1cy5TcWxpdGVPaykge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKHRoaXMuX3dhc20pO1xuICAgIH1cbiAgICB0aGlzLl9vcGVuID0gZmFsc2U7XG4gIH1cblxuICAvKipcbiAgICogR2V0IGxhc3QgaW5zZXJ0ZWQgcm93IGlkLiBUaGlzIGNvcnJlc3BvbmRzIHRvXG4gICAqIHRoZSBTUUxpdGUgZnVuY3Rpb24gYHNxbGl0ZTNfbGFzdF9pbnNlcnRfcm93aWRgLlxuICAgKlxuICAgKiBCZWZvcmUgYSByb3cgaXMgaW5zZXJ0ZWQgZm9yIHRoZSBmaXJzdCB0aW1lIChzaW5jZVxuICAgKiB0aGUgZGF0YWJhc2Ugd2FzIG9wZW5lZCksIHRoaXMgcmV0dXJucyBgMGAuXG4gICAqL1xuICBnZXQgbGFzdEluc2VydFJvd0lkKCk6IG51bWJlciB7XG4gICAgcmV0dXJuIHRoaXMuX3dhc20ubGFzdF9pbnNlcnRfcm93aWQoKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZXR1cm4gdGhlIG51bWJlciBvZiByb3dzIG1vZGlmaWVkLCBpbnNlcnRlZCBvclxuICAgKiBkZWxldGVkIGJ5IHRoZSBtb3N0IHJlY2VudGx5IGNvbXBsZXRlZCBxdWVyeS5cbiAgICogVGhpcyBjb3JyZXNwb25kcyB0byB0aGUgU1FMaXRlIGZ1bmN0aW9uXG4gICAqIGBzcWxpdGUzX2NoYW5nZXNgLlxuICAgKi9cbiAgZ2V0IGNoYW5nZXMoKTogbnVtYmVyIHtcbiAgICByZXR1cm4gdGhpcy5fd2FzbS5jaGFuZ2VzKCk7XG4gIH1cblxuICAvKipcbiAgICogUmV0dXJuIHRoZSBudW1iZXIgb2Ygcm93cyBtb2RpZmllZCwgaW5zZXJ0ZWQgb3JcbiAgICogZGVsZXRlZCBzaW5jZSB0aGUgZGF0YWJhc2Ugd2FzIG9wZW5lZC5cbiAgICogVGhpcyBjb3JyZXNwb25kcyB0byB0aGUgU1FMaXRlIGZ1bmN0aW9uXG4gICAqIGBzcWxpdGUzX3RvdGFsX2NoYW5nZXNgLlxuICAgKi9cbiAgZ2V0IHRvdGFsQ2hhbmdlcygpOiBudW1iZXIge1xuICAgIHJldHVybiB0aGlzLl93YXNtLnRvdGFsX2NoYW5nZXMoKTtcbiAgfVxufVxuIl19