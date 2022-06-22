import { instantiate } from "../build/sqlite.js";
import { setStr } from "./wasm.ts";
import { OpenFlags, Status, Values } from "./constants.ts";
import { SqliteError } from "./error.ts";
import { PreparedQuery } from "./query.ts";
/**
 * A database handle that can be used to run
 * queries.
 */ export class DB {
    _wasm;
    _open;
    _statements;
    _transactionDepth;
    /**
   * Create a new database. The file at the
   * given path will be opened with the
   * mode specified in options. The default
   * mode is `create`.
   *
   * If no path is given, or if the `memory`
   * option is set, the database is opened in
   * memory.
   *
   * # Examples
   *
   * Create an in-memory database.
   * ```typescript
   * const db = new DB();
   * ```
   *
   * Open a database backed by a file on disk.
   * ```typescript
   * const db = new DB("path/to/database.sqlite");
   * ```
   *
   * Pass options to open a read-only database.
   * ```typescript
   * const db = new DB("path/to/database.sqlite", { mode: "read" });
   * ```
   */ constructor(path = ":memory:", options = {}){
        this._wasm = instantiate().exports;
        this._open = false;
        this._statements = new Set();
        this._transactionDepth = 0;
        // Configure flags
        let flags = 0;
        switch(options.mode){
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
        // Try to open the database
        const status = setStr(this._wasm, path, (ptr)=>this._wasm.open(ptr, flags));
        if (status !== Status.SqliteOk) {
            throw new SqliteError(this._wasm, status);
        }
        this._open = true;
    }
    /**
   * Query the database and return all matching
   * rows.
   *
   * This is equivalent to calling `all` on
   * a prepared query which is then immediately
   * finalized.
   *
   * The type parameter `R` may be supplied by
   * the user to indicated the type for the rows returned
   * by the query. Notice that the user is responsible
   * for ensuring the correctness of the supplied type.
   *
   * To avoid SQL injection, user-provided values
   * should always be passed to the database through
   * a query parameter.
   *
   * See `QueryParameterSet` for documentation on
   * how values can be bound to SQL statements.
   *
   * See `QueryParameter` for documentation on how
   * values are returned from the database.
   *
   * # Examples
   *
   * ```typescript
   * const rows = db.query<[string, number]>("SELECT name, age FROM people WHERE city = ?", [city]);
   * // rows = [["Peter Parker", 21], ...]
   * ```
   */ query(sql, params) {
        const query = this.prepareQuery(sql);
        try {
            const rows = query.all(params);
            query.finalize();
            return rows;
        } catch (err) {
            query.finalize();
            throw err;
        }
    }
    /**
   * Like `query` except each row is returned
   * as an object containing key-value pairs.
   *
   * # Examples
   *
   * ```typescript
   * const rows = db.queryEntries<{ name: string, age: number }>("SELECT name, age FROM people");
   * // rows = [{ name: "Peter Parker", age: 21 }, ...]
   * ```
   */ queryEntries(sql, params) {
        const query = this.prepareQuery(sql);
        try {
            const rows = query.allEntries(params);
            query.finalize();
            return rows;
        } catch (err) {
            query.finalize();
            throw err;
        }
    }
    /**
   * Prepares the given SQL query, so that it
   * can be run multiple times and potentially
   * with different parameters.
   *
   * If a query will be issued a lot, this is more
   * efficient than using `query`. A prepared
   * query also provides more control over how
   * the query is run, as well as access to meta-data
   * about the issued query.
   *
   * The returned `PreparedQuery` object must be
   * finalized by calling its `finalize` method
   * once it is no longer needed.
   *
   * # Typing Queries
   *
   * Prepared query objects accept three type parameters
   * to specify precise types for returned data and
   * query parameters.
   *
   * The first type parameter `R` indicates the tuple type
   * for rows returned by the query.
   *
   * The second type parameter `O` indicates the record type
   * for rows returned as entries (mappings from column names
   * to values).
   *
   * The third type parameter `P` indicates the type this query
   * accepts as parameters.
   *
   * Note, that the correctness of those types must
   * be guaranteed by the caller of this function.
   *
   * # Examples
   *
   * ```typescript
   * const query = db.prepareQuery<
   *   [string, number],
   *   { name: string, age: number },
   *   { city: string },
   *  >("SELECT name, age FROM people WHERE city = :city");
   * // use query ...
   * query.finalize();
   * ```
   */ prepareQuery(sql) {
        if (!this._open) {
            throw new SqliteError("Database was closed.");
        }
        const stmt = setStr(this._wasm, sql, (ptr)=>this._wasm.prepare(ptr));
        if (stmt === Values.Null) {
            throw new SqliteError(this._wasm);
        }
        this._statements.add(stmt);
        return new PreparedQuery(this._wasm, stmt, this._statements);
    }
    /**
   * Run multiple semicolon-separated statements from a single
   * string.
   *
   * This method cannot bind any query parameters, and any
   * result rows are discarded. It is only for running a chunk
   * of raw SQL; for example, to initialize a database.
   *
   * # Examples
   *
   * ```typescript
   * db.execute(`
   *   CREATE TABLE people (
   *     id INTEGER PRIMARY KEY AUTOINCREMENT,
   *     name TEXT,
   *     age REAL,
   *     city TEXT
   *   );
   *   INSERT INTO people (name, age, city) VALUES ("Peter Parker", 21, "nyc");
   * `);
   * ```
   */ execute(sql) {
        const status = setStr(this._wasm, sql, (ptr)=>this._wasm.exec(ptr));
        if (status !== Status.SqliteOk) {
            throw new SqliteError(this._wasm, status);
        }
    }
    /**
   * Run a function within the context of a database
   * transaction. If the function throws an error,
   * the transaction is rolled back. Otherwise, the
   * transaction is committed when the function returns.
   *
   * Calls to `transaction` may be nested. Nested transactions
   * behave like SQLite save points.
   */ transaction(closure) {
        this._transactionDepth += 1;
        this.query(`SAVEPOINT _deno_sqlite_sp_${this._transactionDepth}`);
        let value;
        try {
            value = closure();
        } catch (err) {
            this.query(`ROLLBACK TO _deno_sqlite_sp_${this._transactionDepth}`);
            this._transactionDepth -= 1;
            throw err;
        }
        this.query(`RELEASE _deno_sqlite_sp_${this._transactionDepth}`);
        this._transactionDepth -= 1;
        return value;
    }
    /**
   * Close the database. This must be called if
   * the database is no longer used to avoid leaking
   * open file descriptors.
   *
   * If `force` is specified, any active `PreparedQuery`
   * will be finalized. Otherwise, this throws if there
   * are active queries.
   *
   * `close` may safely be called multiple
   * times.
   */ close(force = false) {
        if (!this._open) {
            return;
        }
        if (force) {
            for (const stmt of this._statements){
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
    /**
   * Get last inserted row id. This corresponds to
   * the SQLite function `sqlite3_last_insert_rowid`.
   *
   * Before a row is inserted for the first time (since
   * the database was opened), this returns `0`.
   */ get lastInsertRowId() {
        return this._wasm.last_insert_rowid();
    }
    /**
   * Return the number of rows modified, inserted or
   * deleted by the most recently completed query.
   * This corresponds to the SQLite function
   * `sqlite3_changes`.
   */ get changes() {
        return this._wasm.changes();
    }
    /**
   * Return the number of rows modified, inserted or
   * deleted since the database was opened.
   * This corresponds to the SQLite function
   * `sqlite3_total_changes`.
   */ get totalChanges() {
        return this._wasm.total_changes();
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvc3FsaXRlQHYzLjQuMC9zcmMvZGIudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgaW5zdGFudGlhdGUsIFN0YXRlbWVudFB0ciwgV2FzbSB9IGZyb20gXCIuLi9idWlsZC9zcWxpdGUuanNcIjtcbmltcG9ydCB7IHNldFN0ciB9IGZyb20gXCIuL3dhc20udHNcIjtcbmltcG9ydCB7IE9wZW5GbGFncywgU3RhdHVzLCBWYWx1ZXMgfSBmcm9tIFwiLi9jb25zdGFudHMudHNcIjtcbmltcG9ydCB7IFNxbGl0ZUVycm9yIH0gZnJvbSBcIi4vZXJyb3IudHNcIjtcbmltcG9ydCB7IFByZXBhcmVkUXVlcnksIFF1ZXJ5UGFyYW1ldGVyU2V0LCBSb3csIFJvd09iamVjdCB9IGZyb20gXCIuL3F1ZXJ5LnRzXCI7XG5cbi8qKlxuICogT3B0aW9ucyBmb3Igb3BlbmluZyBhIGRhdGFiYXNlLlxuICovXG5leHBvcnQgaW50ZXJmYWNlIFNxbGl0ZU9wdGlvbnMge1xuICAvKipcbiAgICogTW9kZSBpbiB3aGljaCB0byBvcGVuIHRoZSBkYXRhYmFzZS5cbiAgICpcbiAgICogLSBgcmVhZGA6IHJlYWQtb25seSwgdGhyb3dzIGFuIGVycm9yIGlmXG4gICAqICAgdGhlIGRhdGFiYXNlIGZpbGUgZG9lcyBub3QgZXhpc3RzXG4gICAqIC0gYHdyaXRlYDogcmVhZC13cml0ZSwgdGhyb3dzIGFuIGVycm9yXG4gICAqICAgaWYgdGhlIGRhdGFiYXNlIGZpbGUgZG9lcyBub3QgZXhpc3RzXG4gICAqIC0gYGNyZWF0ZWA6IHJlYWQtd3JpdGUsIGNyZWF0ZSB0aGUgZGF0YWJhc2VcbiAgICogICBpZiB0aGUgZmlsZSBkb2VzIG5vdCBleGlzdFxuICAgKlxuICAgKiBgY3JlYXRlYCBpcyB0aGUgZGVmYXVsdCBpZiBubyBtb2RlIGlzXG4gICAqIHNwZWNpZmllZC5cbiAgICovXG4gIG1vZGU/OiBcInJlYWRcIiB8IFwid3JpdGVcIiB8IFwiY3JlYXRlXCI7XG4gIC8qKlxuICAgKiBGb3JjZSB0aGUgZGF0YWJhc2UgdG8gYmUgaW4tbWVtb3J5LiBXaGVuXG4gICAqIHRoaXMgb3B0aW9uIGlzIHNldCwgdGhlIGRhdGFiYXNlIGlzIG9wZW5lZFxuICAgKiBpbiBtZW1vcnksIHJlZ2FyZGxlc3Mgb2YgdGhlIHNwZWNpZmllZFxuICAgKiBmaWxlbmFtZS5cbiAgICovXG4gIG1lbW9yeT86IGJvb2xlYW47XG4gIC8qKlxuICAgKiBJbnRlcnByZXQgdGhlIGZpbGUgbmFtZSBhcyBhIFVSSS5cbiAgICogU2VlIGh0dHBzOi8vc3FsaXRlLm9yZy91cmkuaHRtbFxuICAgKiBmb3IgbW9yZSBpbmZvcm1hdGlvbi5cbiAgICovXG4gIHVyaT86IGJvb2xlYW47XG59XG5cbi8qKlxuICogQSBkYXRhYmFzZSBoYW5kbGUgdGhhdCBjYW4gYmUgdXNlZCB0byBydW5cbiAqIHF1ZXJpZXMuXG4gKi9cbmV4cG9ydCBjbGFzcyBEQiB7XG4gIHByaXZhdGUgX3dhc206IFdhc207XG4gIHByaXZhdGUgX29wZW46IGJvb2xlYW47XG4gIHByaXZhdGUgX3N0YXRlbWVudHM6IFNldDxTdGF0ZW1lbnRQdHI+O1xuICBwcml2YXRlIF90cmFuc2FjdGlvbkRlcHRoOiBudW1iZXI7XG5cbiAgLyoqXG4gICAqIENyZWF0ZSBhIG5ldyBkYXRhYmFzZS4gVGhlIGZpbGUgYXQgdGhlXG4gICAqIGdpdmVuIHBhdGggd2lsbCBiZSBvcGVuZWQgd2l0aCB0aGVcbiAgICogbW9kZSBzcGVjaWZpZWQgaW4gb3B0aW9ucy4gVGhlIGRlZmF1bHRcbiAgICogbW9kZSBpcyBgY3JlYXRlYC5cbiAgICpcbiAgICogSWYgbm8gcGF0aCBpcyBnaXZlbiwgb3IgaWYgdGhlIGBtZW1vcnlgXG4gICAqIG9wdGlvbiBpcyBzZXQsIHRoZSBkYXRhYmFzZSBpcyBvcGVuZWQgaW5cbiAgICogbWVtb3J5LlxuICAgKlxuICAgKiAjIEV4YW1wbGVzXG4gICAqXG4gICAqIENyZWF0ZSBhbiBpbi1tZW1vcnkgZGF0YWJhc2UuXG4gICAqIGBgYHR5cGVzY3JpcHRcbiAgICogY29uc3QgZGIgPSBuZXcgREIoKTtcbiAgICogYGBgXG4gICAqXG4gICAqIE9wZW4gYSBkYXRhYmFzZSBiYWNrZWQgYnkgYSBmaWxlIG9uIGRpc2suXG4gICAqIGBgYHR5cGVzY3JpcHRcbiAgICogY29uc3QgZGIgPSBuZXcgREIoXCJwYXRoL3RvL2RhdGFiYXNlLnNxbGl0ZVwiKTtcbiAgICogYGBgXG4gICAqXG4gICAqIFBhc3Mgb3B0aW9ucyB0byBvcGVuIGEgcmVhZC1vbmx5IGRhdGFiYXNlLlxuICAgKiBgYGB0eXBlc2NyaXB0XG4gICAqIGNvbnN0IGRiID0gbmV3IERCKFwicGF0aC90by9kYXRhYmFzZS5zcWxpdGVcIiwgeyBtb2RlOiBcInJlYWRcIiB9KTtcbiAgICogYGBgXG4gICAqL1xuICBjb25zdHJ1Y3RvcihwYXRoOiBzdHJpbmcgPSBcIjptZW1vcnk6XCIsIG9wdGlvbnM6IFNxbGl0ZU9wdGlvbnMgPSB7fSkge1xuICAgIHRoaXMuX3dhc20gPSBpbnN0YW50aWF0ZSgpLmV4cG9ydHM7XG4gICAgdGhpcy5fb3BlbiA9IGZhbHNlO1xuICAgIHRoaXMuX3N0YXRlbWVudHMgPSBuZXcgU2V0KCk7XG4gICAgdGhpcy5fdHJhbnNhY3Rpb25EZXB0aCA9IDA7XG5cbiAgICAvLyBDb25maWd1cmUgZmxhZ3NcbiAgICBsZXQgZmxhZ3MgPSAwO1xuICAgIHN3aXRjaCAob3B0aW9ucy5tb2RlKSB7XG4gICAgICBjYXNlIFwicmVhZFwiOlxuICAgICAgICBmbGFncyA9IE9wZW5GbGFncy5SZWFkT25seTtcbiAgICAgICAgYnJlYWs7XG4gICAgICBjYXNlIFwid3JpdGVcIjpcbiAgICAgICAgZmxhZ3MgPSBPcGVuRmxhZ3MuUmVhZFdyaXRlO1xuICAgICAgICBicmVhaztcbiAgICAgIGNhc2UgXCJjcmVhdGVcIjogLy8gZmFsbCB0aHJvdWdoXG4gICAgICBkZWZhdWx0OlxuICAgICAgICBmbGFncyA9IE9wZW5GbGFncy5SZWFkV3JpdGUgfCBPcGVuRmxhZ3MuQ3JlYXRlO1xuICAgICAgICBicmVhaztcbiAgICB9XG4gICAgaWYgKG9wdGlvbnMubWVtb3J5ID09PSB0cnVlKSB7XG4gICAgICBmbGFncyB8PSBPcGVuRmxhZ3MuTWVtb3J5O1xuICAgIH1cbiAgICBpZiAob3B0aW9ucy51cmkgPT09IHRydWUpIHtcbiAgICAgIGZsYWdzIHw9IE9wZW5GbGFncy5Vcmk7XG4gICAgfVxuXG4gICAgLy8gVHJ5IHRvIG9wZW4gdGhlIGRhdGFiYXNlXG4gICAgY29uc3Qgc3RhdHVzID0gc2V0U3RyKFxuICAgICAgdGhpcy5fd2FzbSxcbiAgICAgIHBhdGgsXG4gICAgICAocHRyKSA9PiB0aGlzLl93YXNtLm9wZW4ocHRyLCBmbGFncyksXG4gICAgKTtcbiAgICBpZiAoc3RhdHVzICE9PSBTdGF0dXMuU3FsaXRlT2spIHtcbiAgICAgIHRocm93IG5ldyBTcWxpdGVFcnJvcih0aGlzLl93YXNtLCBzdGF0dXMpO1xuICAgIH1cbiAgICB0aGlzLl9vcGVuID0gdHJ1ZTtcbiAgfVxuXG4gIC8qKlxuICAgKiBRdWVyeSB0aGUgZGF0YWJhc2UgYW5kIHJldHVybiBhbGwgbWF0Y2hpbmdcbiAgICogcm93cy5cbiAgICpcbiAgICogVGhpcyBpcyBlcXVpdmFsZW50IHRvIGNhbGxpbmcgYGFsbGAgb25cbiAgICogYSBwcmVwYXJlZCBxdWVyeSB3aGljaCBpcyB0aGVuIGltbWVkaWF0ZWx5XG4gICAqIGZpbmFsaXplZC5cbiAgICpcbiAgICogVGhlIHR5cGUgcGFyYW1ldGVyIGBSYCBtYXkgYmUgc3VwcGxpZWQgYnlcbiAgICogdGhlIHVzZXIgdG8gaW5kaWNhdGVkIHRoZSB0eXBlIGZvciB0aGUgcm93cyByZXR1cm5lZFxuICAgKiBieSB0aGUgcXVlcnkuIE5vdGljZSB0aGF0IHRoZSB1c2VyIGlzIHJlc3BvbnNpYmxlXG4gICAqIGZvciBlbnN1cmluZyB0aGUgY29ycmVjdG5lc3Mgb2YgdGhlIHN1cHBsaWVkIHR5cGUuXG4gICAqXG4gICAqIFRvIGF2b2lkIFNRTCBpbmplY3Rpb24sIHVzZXItcHJvdmlkZWQgdmFsdWVzXG4gICAqIHNob3VsZCBhbHdheXMgYmUgcGFzc2VkIHRvIHRoZSBkYXRhYmFzZSB0aHJvdWdoXG4gICAqIGEgcXVlcnkgcGFyYW1ldGVyLlxuICAgKlxuICAgKiBTZWUgYFF1ZXJ5UGFyYW1ldGVyU2V0YCBmb3IgZG9jdW1lbnRhdGlvbiBvblxuICAgKiBob3cgdmFsdWVzIGNhbiBiZSBib3VuZCB0byBTUUwgc3RhdGVtZW50cy5cbiAgICpcbiAgICogU2VlIGBRdWVyeVBhcmFtZXRlcmAgZm9yIGRvY3VtZW50YXRpb24gb24gaG93XG4gICAqIHZhbHVlcyBhcmUgcmV0dXJuZWQgZnJvbSB0aGUgZGF0YWJhc2UuXG4gICAqXG4gICAqICMgRXhhbXBsZXNcbiAgICpcbiAgICogYGBgdHlwZXNjcmlwdFxuICAgKiBjb25zdCByb3dzID0gZGIucXVlcnk8W3N0cmluZywgbnVtYmVyXT4oXCJTRUxFQ1QgbmFtZSwgYWdlIEZST00gcGVvcGxlIFdIRVJFIGNpdHkgPSA/XCIsIFtjaXR5XSk7XG4gICAqIC8vIHJvd3MgPSBbW1wiUGV0ZXIgUGFya2VyXCIsIDIxXSwgLi4uXVxuICAgKiBgYGBcbiAgICovXG4gIHF1ZXJ5PFIgZXh0ZW5kcyBSb3cgPSBSb3c+KFxuICAgIHNxbDogc3RyaW5nLFxuICAgIHBhcmFtcz86IFF1ZXJ5UGFyYW1ldGVyU2V0LFxuICApOiBBcnJheTxSPiB7XG4gICAgY29uc3QgcXVlcnkgPSB0aGlzLnByZXBhcmVRdWVyeTxSPihzcWwpO1xuICAgIHRyeSB7XG4gICAgICBjb25zdCByb3dzID0gcXVlcnkuYWxsKHBhcmFtcyk7XG4gICAgICBxdWVyeS5maW5hbGl6ZSgpO1xuICAgICAgcmV0dXJuIHJvd3M7XG4gICAgfSBjYXRjaCAoZXJyKSB7XG4gICAgICBxdWVyeS5maW5hbGl6ZSgpO1xuICAgICAgdGhyb3cgZXJyO1xuICAgIH1cbiAgfVxuXG4gIC8qKlxuICAgKiBMaWtlIGBxdWVyeWAgZXhjZXB0IGVhY2ggcm93IGlzIHJldHVybmVkXG4gICAqIGFzIGFuIG9iamVjdCBjb250YWluaW5nIGtleS12YWx1ZSBwYWlycy5cbiAgICpcbiAgICogIyBFeGFtcGxlc1xuICAgKlxuICAgKiBgYGB0eXBlc2NyaXB0XG4gICAqIGNvbnN0IHJvd3MgPSBkYi5xdWVyeUVudHJpZXM8eyBuYW1lOiBzdHJpbmcsIGFnZTogbnVtYmVyIH0+KFwiU0VMRUNUIG5hbWUsIGFnZSBGUk9NIHBlb3BsZVwiKTtcbiAgICogLy8gcm93cyA9IFt7IG5hbWU6IFwiUGV0ZXIgUGFya2VyXCIsIGFnZTogMjEgfSwgLi4uXVxuICAgKiBgYGBcbiAgICovXG4gIHF1ZXJ5RW50cmllczxPIGV4dGVuZHMgUm93T2JqZWN0ID0gUm93T2JqZWN0PihcbiAgICBzcWw6IHN0cmluZyxcbiAgICBwYXJhbXM/OiBRdWVyeVBhcmFtZXRlclNldCxcbiAgKTogQXJyYXk8Tz4ge1xuICAgIGNvbnN0IHF1ZXJ5ID0gdGhpcy5wcmVwYXJlUXVlcnk8Um93LCBPPihzcWwpO1xuICAgIHRyeSB7XG4gICAgICBjb25zdCByb3dzID0gcXVlcnkuYWxsRW50cmllcyhwYXJhbXMpO1xuICAgICAgcXVlcnkuZmluYWxpemUoKTtcbiAgICAgIHJldHVybiByb3dzO1xuICAgIH0gY2F0Y2ggKGVycikge1xuICAgICAgcXVlcnkuZmluYWxpemUoKTtcbiAgICAgIHRocm93IGVycjtcbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogUHJlcGFyZXMgdGhlIGdpdmVuIFNRTCBxdWVyeSwgc28gdGhhdCBpdFxuICAgKiBjYW4gYmUgcnVuIG11bHRpcGxlIHRpbWVzIGFuZCBwb3RlbnRpYWxseVxuICAgKiB3aXRoIGRpZmZlcmVudCBwYXJhbWV0ZXJzLlxuICAgKlxuICAgKiBJZiBhIHF1ZXJ5IHdpbGwgYmUgaXNzdWVkIGEgbG90LCB0aGlzIGlzIG1vcmVcbiAgICogZWZmaWNpZW50IHRoYW4gdXNpbmcgYHF1ZXJ5YC4gQSBwcmVwYXJlZFxuICAgKiBxdWVyeSBhbHNvIHByb3ZpZGVzIG1vcmUgY29udHJvbCBvdmVyIGhvd1xuICAgKiB0aGUgcXVlcnkgaXMgcnVuLCBhcyB3ZWxsIGFzIGFjY2VzcyB0byBtZXRhLWRhdGFcbiAgICogYWJvdXQgdGhlIGlzc3VlZCBxdWVyeS5cbiAgICpcbiAgICogVGhlIHJldHVybmVkIGBQcmVwYXJlZFF1ZXJ5YCBvYmplY3QgbXVzdCBiZVxuICAgKiBmaW5hbGl6ZWQgYnkgY2FsbGluZyBpdHMgYGZpbmFsaXplYCBtZXRob2RcbiAgICogb25jZSBpdCBpcyBubyBsb25nZXIgbmVlZGVkLlxuICAgKlxuICAgKiAjIFR5cGluZyBRdWVyaWVzXG4gICAqXG4gICAqIFByZXBhcmVkIHF1ZXJ5IG9iamVjdHMgYWNjZXB0IHRocmVlIHR5cGUgcGFyYW1ldGVyc1xuICAgKiB0byBzcGVjaWZ5IHByZWNpc2UgdHlwZXMgZm9yIHJldHVybmVkIGRhdGEgYW5kXG4gICAqIHF1ZXJ5IHBhcmFtZXRlcnMuXG4gICAqXG4gICAqIFRoZSBmaXJzdCB0eXBlIHBhcmFtZXRlciBgUmAgaW5kaWNhdGVzIHRoZSB0dXBsZSB0eXBlXG4gICAqIGZvciByb3dzIHJldHVybmVkIGJ5IHRoZSBxdWVyeS5cbiAgICpcbiAgICogVGhlIHNlY29uZCB0eXBlIHBhcmFtZXRlciBgT2AgaW5kaWNhdGVzIHRoZSByZWNvcmQgdHlwZVxuICAgKiBmb3Igcm93cyByZXR1cm5lZCBhcyBlbnRyaWVzIChtYXBwaW5ncyBmcm9tIGNvbHVtbiBuYW1lc1xuICAgKiB0byB2YWx1ZXMpLlxuICAgKlxuICAgKiBUaGUgdGhpcmQgdHlwZSBwYXJhbWV0ZXIgYFBgIGluZGljYXRlcyB0aGUgdHlwZSB0aGlzIHF1ZXJ5XG4gICAqIGFjY2VwdHMgYXMgcGFyYW1ldGVycy5cbiAgICpcbiAgICogTm90ZSwgdGhhdCB0aGUgY29ycmVjdG5lc3Mgb2YgdGhvc2UgdHlwZXMgbXVzdFxuICAgKiBiZSBndWFyYW50ZWVkIGJ5IHRoZSBjYWxsZXIgb2YgdGhpcyBmdW5jdGlvbi5cbiAgICpcbiAgICogIyBFeGFtcGxlc1xuICAgKlxuICAgKiBgYGB0eXBlc2NyaXB0XG4gICAqIGNvbnN0IHF1ZXJ5ID0gZGIucHJlcGFyZVF1ZXJ5PFxuICAgKiAgIFtzdHJpbmcsIG51bWJlcl0sXG4gICAqICAgeyBuYW1lOiBzdHJpbmcsIGFnZTogbnVtYmVyIH0sXG4gICAqICAgeyBjaXR5OiBzdHJpbmcgfSxcbiAgICogID4oXCJTRUxFQ1QgbmFtZSwgYWdlIEZST00gcGVvcGxlIFdIRVJFIGNpdHkgPSA6Y2l0eVwiKTtcbiAgICogLy8gdXNlIHF1ZXJ5IC4uLlxuICAgKiBxdWVyeS5maW5hbGl6ZSgpO1xuICAgKiBgYGBcbiAgICovXG4gIHByZXBhcmVRdWVyeTxcbiAgICBSIGV4dGVuZHMgUm93ID0gUm93LFxuICAgIE8gZXh0ZW5kcyBSb3dPYmplY3QgPSBSb3dPYmplY3QsXG4gICAgUCBleHRlbmRzIFF1ZXJ5UGFyYW1ldGVyU2V0ID0gUXVlcnlQYXJhbWV0ZXJTZXQsXG4gID4oXG4gICAgc3FsOiBzdHJpbmcsXG4gICk6IFByZXBhcmVkUXVlcnk8UiwgTywgUD4ge1xuICAgIGlmICghdGhpcy5fb3Blbikge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKFwiRGF0YWJhc2Ugd2FzIGNsb3NlZC5cIik7XG4gICAgfVxuXG4gICAgY29uc3Qgc3RtdCA9IHNldFN0cihcbiAgICAgIHRoaXMuX3dhc20sXG4gICAgICBzcWwsXG4gICAgICAocHRyKSA9PiB0aGlzLl93YXNtLnByZXBhcmUocHRyKSxcbiAgICApO1xuICAgIGlmIChzdG10ID09PSBWYWx1ZXMuTnVsbCkge1xuICAgICAgdGhyb3cgbmV3IFNxbGl0ZUVycm9yKHRoaXMuX3dhc20pO1xuICAgIH1cblxuICAgIHRoaXMuX3N0YXRlbWVudHMuYWRkKHN0bXQpO1xuICAgIHJldHVybiBuZXcgUHJlcGFyZWRRdWVyeTxSLCBPLCBQPih0aGlzLl93YXNtLCBzdG10LCB0aGlzLl9zdGF0ZW1lbnRzKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSdW4gbXVsdGlwbGUgc2VtaWNvbG9uLXNlcGFyYXRlZCBzdGF0ZW1lbnRzIGZyb20gYSBzaW5nbGVcbiAgICogc3RyaW5nLlxuICAgKlxuICAgKiBUaGlzIG1ldGhvZCBjYW5ub3QgYmluZCBhbnkgcXVlcnkgcGFyYW1ldGVycywgYW5kIGFueVxuICAgKiByZXN1bHQgcm93cyBhcmUgZGlzY2FyZGVkLiBJdCBpcyBvbmx5IGZvciBydW5uaW5nIGEgY2h1bmtcbiAgICogb2YgcmF3IFNRTDsgZm9yIGV4YW1wbGUsIHRvIGluaXRpYWxpemUgYSBkYXRhYmFzZS5cbiAgICpcbiAgICogIyBFeGFtcGxlc1xuICAgKlxuICAgKiBgYGB0eXBlc2NyaXB0XG4gICAqIGRiLmV4ZWN1dGUoYFxuICAgKiAgIENSRUFURSBUQUJMRSBwZW9wbGUgKFxuICAgKiAgICAgaWQgSU5URUdFUiBQUklNQVJZIEtFWSBBVVRPSU5DUkVNRU5ULFxuICAgKiAgICAgbmFtZSBURVhULFxuICAgKiAgICAgYWdlIFJFQUwsXG4gICAqICAgICBjaXR5IFRFWFRcbiAgICogICApO1xuICAgKiAgIElOU0VSVCBJTlRPIHBlb3BsZSAobmFtZSwgYWdlLCBjaXR5KSBWQUxVRVMgKFwiUGV0ZXIgUGFya2VyXCIsIDIxLCBcIm55Y1wiKTtcbiAgICogYCk7XG4gICAqIGBgYFxuICAgKi9cbiAgZXhlY3V0ZShzcWw6IHN0cmluZykge1xuICAgIGNvbnN0IHN0YXR1cyA9IHNldFN0cihcbiAgICAgIHRoaXMuX3dhc20sXG4gICAgICBzcWwsXG4gICAgICAocHRyKSA9PiB0aGlzLl93YXNtLmV4ZWMocHRyKSxcbiAgICApO1xuXG4gICAgaWYgKHN0YXR1cyAhPT0gU3RhdHVzLlNxbGl0ZU9rKSB7XG4gICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSwgc3RhdHVzKTtcbiAgICB9XG4gIH1cblxuICAvKipcbiAgICogUnVuIGEgZnVuY3Rpb24gd2l0aGluIHRoZSBjb250ZXh0IG9mIGEgZGF0YWJhc2VcbiAgICogdHJhbnNhY3Rpb24uIElmIHRoZSBmdW5jdGlvbiB0aHJvd3MgYW4gZXJyb3IsXG4gICAqIHRoZSB0cmFuc2FjdGlvbiBpcyByb2xsZWQgYmFjay4gT3RoZXJ3aXNlLCB0aGVcbiAgICogdHJhbnNhY3Rpb24gaXMgY29tbWl0dGVkIHdoZW4gdGhlIGZ1bmN0aW9uIHJldHVybnMuXG4gICAqXG4gICAqIENhbGxzIHRvIGB0cmFuc2FjdGlvbmAgbWF5IGJlIG5lc3RlZC4gTmVzdGVkIHRyYW5zYWN0aW9uc1xuICAgKiBiZWhhdmUgbGlrZSBTUUxpdGUgc2F2ZSBwb2ludHMuXG4gICAqL1xuICB0cmFuc2FjdGlvbjxWPihjbG9zdXJlOiAoKSA9PiBWKTogViB7XG4gICAgdGhpcy5fdHJhbnNhY3Rpb25EZXB0aCArPSAxO1xuICAgIHRoaXMucXVlcnkoYFNBVkVQT0lOVCBfZGVub19zcWxpdGVfc3BfJHt0aGlzLl90cmFuc2FjdGlvbkRlcHRofWApO1xuICAgIGxldCB2YWx1ZTtcbiAgICB0cnkge1xuICAgICAgdmFsdWUgPSBjbG9zdXJlKCk7XG4gICAgfSBjYXRjaCAoZXJyKSB7XG4gICAgICB0aGlzLnF1ZXJ5KGBST0xMQkFDSyBUTyBfZGVub19zcWxpdGVfc3BfJHt0aGlzLl90cmFuc2FjdGlvbkRlcHRofWApO1xuICAgICAgdGhpcy5fdHJhbnNhY3Rpb25EZXB0aCAtPSAxO1xuICAgICAgdGhyb3cgZXJyO1xuICAgIH1cbiAgICB0aGlzLnF1ZXJ5KGBSRUxFQVNFIF9kZW5vX3NxbGl0ZV9zcF8ke3RoaXMuX3RyYW5zYWN0aW9uRGVwdGh9YCk7XG4gICAgdGhpcy5fdHJhbnNhY3Rpb25EZXB0aCAtPSAxO1xuICAgIHJldHVybiB2YWx1ZTtcbiAgfVxuXG4gIC8qKlxuICAgKiBDbG9zZSB0aGUgZGF0YWJhc2UuIFRoaXMgbXVzdCBiZSBjYWxsZWQgaWZcbiAgICogdGhlIGRhdGFiYXNlIGlzIG5vIGxvbmdlciB1c2VkIHRvIGF2b2lkIGxlYWtpbmdcbiAgICogb3BlbiBmaWxlIGRlc2NyaXB0b3JzLlxuICAgKlxuICAgKiBJZiBgZm9yY2VgIGlzIHNwZWNpZmllZCwgYW55IGFjdGl2ZSBgUHJlcGFyZWRRdWVyeWBcbiAgICogd2lsbCBiZSBmaW5hbGl6ZWQuIE90aGVyd2lzZSwgdGhpcyB0aHJvd3MgaWYgdGhlcmVcbiAgICogYXJlIGFjdGl2ZSBxdWVyaWVzLlxuICAgKlxuICAgKiBgY2xvc2VgIG1heSBzYWZlbHkgYmUgY2FsbGVkIG11bHRpcGxlXG4gICAqIHRpbWVzLlxuICAgKi9cbiAgY2xvc2UoZm9yY2UgPSBmYWxzZSkge1xuICAgIGlmICghdGhpcy5fb3Blbikge1xuICAgICAgcmV0dXJuO1xuICAgIH1cbiAgICBpZiAoZm9yY2UpIHtcbiAgICAgIGZvciAoY29uc3Qgc3RtdCBvZiB0aGlzLl9zdGF0ZW1lbnRzKSB7XG4gICAgICAgIGlmICh0aGlzLl93YXNtLmZpbmFsaXplKHN0bXQpICE9PSBTdGF0dXMuU3FsaXRlT2spIHtcbiAgICAgICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSk7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9XG4gICAgaWYgKHRoaXMuX3dhc20uY2xvc2UoKSAhPT0gU3RhdHVzLlNxbGl0ZU9rKSB7XG4gICAgICB0aHJvdyBuZXcgU3FsaXRlRXJyb3IodGhpcy5fd2FzbSk7XG4gICAgfVxuICAgIHRoaXMuX29wZW4gPSBmYWxzZTtcbiAgfVxuXG4gIC8qKlxuICAgKiBHZXQgbGFzdCBpbnNlcnRlZCByb3cgaWQuIFRoaXMgY29ycmVzcG9uZHMgdG9cbiAgICogdGhlIFNRTGl0ZSBmdW5jdGlvbiBgc3FsaXRlM19sYXN0X2luc2VydF9yb3dpZGAuXG4gICAqXG4gICAqIEJlZm9yZSBhIHJvdyBpcyBpbnNlcnRlZCBmb3IgdGhlIGZpcnN0IHRpbWUgKHNpbmNlXG4gICAqIHRoZSBkYXRhYmFzZSB3YXMgb3BlbmVkKSwgdGhpcyByZXR1cm5zIGAwYC5cbiAgICovXG4gIGdldCBsYXN0SW5zZXJ0Um93SWQoKTogbnVtYmVyIHtcbiAgICByZXR1cm4gdGhpcy5fd2FzbS5sYXN0X2luc2VydF9yb3dpZCgpO1xuICB9XG5cbiAgLyoqXG4gICAqIFJldHVybiB0aGUgbnVtYmVyIG9mIHJvd3MgbW9kaWZpZWQsIGluc2VydGVkIG9yXG4gICAqIGRlbGV0ZWQgYnkgdGhlIG1vc3QgcmVjZW50bHkgY29tcGxldGVkIHF1ZXJ5LlxuICAgKiBUaGlzIGNvcnJlc3BvbmRzIHRvIHRoZSBTUUxpdGUgZnVuY3Rpb25cbiAgICogYHNxbGl0ZTNfY2hhbmdlc2AuXG4gICAqL1xuICBnZXQgY2hhbmdlcygpOiBudW1iZXIge1xuICAgIHJldHVybiB0aGlzLl93YXNtLmNoYW5nZXMoKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBSZXR1cm4gdGhlIG51bWJlciBvZiByb3dzIG1vZGlmaWVkLCBpbnNlcnRlZCBvclxuICAgKiBkZWxldGVkIHNpbmNlIHRoZSBkYXRhYmFzZSB3YXMgb3BlbmVkLlxuICAgKiBUaGlzIGNvcnJlc3BvbmRzIHRvIHRoZSBTUUxpdGUgZnVuY3Rpb25cbiAgICogYHNxbGl0ZTNfdG90YWxfY2hhbmdlc2AuXG4gICAqL1xuICBnZXQgdG90YWxDaGFuZ2VzKCk6IG51bWJlciB7XG4gICAgcmV0dXJuIHRoaXMuX3dhc20udG90YWxfY2hhbmdlcygpO1xuICB9XG59XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsU0FBUyxXQUFXLFFBQTRCLG9CQUFvQixDQUFDO0FBQ3JFLFNBQVMsTUFBTSxRQUFRLFdBQVcsQ0FBQztBQUNuQyxTQUFTLFNBQVMsRUFBRSxNQUFNLEVBQUUsTUFBTSxRQUFRLGdCQUFnQixDQUFDO0FBQzNELFNBQVMsV0FBVyxRQUFRLFlBQVksQ0FBQztBQUN6QyxTQUFTLGFBQWEsUUFBMkMsWUFBWSxDQUFDO0FBbUM5RTs7O0dBR0csQ0FDSCxPQUFPLE1BQU0sRUFBRTtJQUNiLEFBQVEsS0FBSyxDQUFPO0lBQ3BCLEFBQVEsS0FBSyxDQUFVO0lBQ3ZCLEFBQVEsV0FBVyxDQUFvQjtJQUN2QyxBQUFRLGlCQUFpQixDQUFTO0lBRWxDOzs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7OztLQTBC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