// Copyright © 2020-2023 Truestamp Inc. All rights reserved.

import { appPaths, DB } from "./deps.ts";

// e.g.  sqlite3 "/Users/glenn/Library/Application Support/com.truestamp.cli.development/db.sqlite3"
export function createDataDir(env: string): string {
  const appDir = appPaths(`com.truestamp.cli.${env}`);
  Deno.mkdirSync(appDir.data, { recursive: true });
  return appDir.data;
}

export function writeItemToDb(env: string, id: string): void {
  const dbDir = createDataDir(env);
  // console.log(`${dbDir}/db.sqlite3`)
  const db = new DB(`${dbDir}/db.sqlite3`);

  db.query(`
CREATE TABLE IF NOT EXISTS items (
  id TEXT NOT NULL UNIQUE PRIMARY KEY
)
`);

  db.query("INSERT INTO items (id) VALUES (?)", [
    id,
  ]);

  db.close();
}

// export function writeItemToDb(env: string, id: string): void {
//   const dbDir = createDataDir(env);
//   // console.log(`${dbDir}/db.sqlite3`)
//   const db = new DB(`${dbDir}/db.sqlite3`);

//   db.query(`
// CREATE TABLE IF NOT EXISTS items (
//   id TEXT NOT NULL UNIQUE PRIMARY KEY,
//   id_json TEXT NOT NULL
// )
// `);

//   const decodedId = decodeUnsafely(id);

//   db.query("INSERT INTO items (id, id_json) VALUES (?, json(?))", [
//     id,
//     JSON.stringify(decodedId),
//   ]);

//   // // Print out data in table
//   // for (const [id] of db.query("SELECT id FROM items")) {
//   //   console.log(id);
//   // }

//   db.close();
// }

// See : https://stackoverflow.com/questions/33432421/sqlite-json1-example-for-json-extract-set
// e.g.  sqlite3 "/Users/glenn/Library/Application Support/com.truestamp.cli.development/db.sqlite3"
// sqlite> select json_extract(items.id_json, '$.ulid') from items;
// 01GFW799RWQ03ZHCBFQK7EDZ5N
// sqlite> select i.* from items i where json_extract(id_json, '$.ulid') LIKE '01GFW799RWQ03ZHCBFQK7EDZ5N';
// ts_11SHzvRaFSVkLu4qpcDx4Aqxrg6dsxHTDcMhrjH1dYbGkfdAWxHenuNa2uVo|{"test":false,"timestamp":"1666320738370000","ulid":"01GFW799RWQ03ZHCBFQK7EDZ5N"}
// sqlite> SELECT json_extract(items.id_json, '$.timestamp') FROM items WHERE json_extract(id_json, '$.ulid') LIKE '01GFW799RWQ03ZHCBFQK7EDZ5N';
// 1666320738370000
// sqlite>

// export function getItemEnvelopesByUlid(env: string, ulid: string): Row[] {
//   const dbDir = createDataDir(env)
//   const db = new DB(`${dbDir}/db.sqlite3`)
//   const result = db.query(
//     "SELECT envelope_json FROM items WHERE json_extract(id_json, '$.ulid') IS ?",
//     [ulid]
//   )
//   db.close()
//   return result
// }
