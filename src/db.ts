// Copyright © 2020-2022 Truestamp Inc. All rights reserved.

import { appPaths, DB, decodeUnsafely, Row } from "./deps.ts";

// e.g.  sqlite3 "/Users/glenn/Library/Application Support/com.truestamp.cli.development/db.sqlite3"
export function createDataDir(env: string): string {
  const appDir = appPaths(`com.truestamp.cli.${env}`);
  Deno.mkdirSync(appDir.data, { recursive: true });
  return appDir.data;
}

export function writeItemToDb(
  env: string,
  id: string,
  envelope: Record<string, any>,
): void {
  const dbDir = createDataDir(env);
  // console.log(`${dbDir}/db.sqlite3`)
  const db = new DB(`${dbDir}/db.sqlite3`);

  db.query(`
CREATE TABLE IF NOT EXISTS items (
  id TEXT NOT NULL UNIQUE PRIMARY KEY,
  id_json TEXT NOT NULL,
  envelope_json TEXT NOT NULL
)
`);

  const decodedId = decodeUnsafely(id);

  db.query(
    "INSERT INTO items (id, id_json, envelope_json) VALUES (?, json(?), json(?))",
    [id, JSON.stringify(decodedId), JSON.stringify(envelope)],
  );

  // // Print out data in table
  // for (const [id] of db.query("SELECT id FROM items")) {
  //   console.log(id);
  // }

  db.close();
}

// See : https://stackoverflow.com/questions/33432421/sqlite-json1-example-for-json-extract-set
// sqlite> select json_extract(items.id_json, '$.ulid') from items;
// 01GABY7BPZTMW0J7SA8DGR2JAZ

// sqlite> select i.* from items i where json_extract(id_json, '$.ulid') LIKE '01GABY7BPZTMW0J7SA8DGR2JAZ';
// truestamp-2SF5JQLhBHmtRC35G6z4M7bjhcnJrGs99nEg6reqW61ThzXLx1pzk3VXjNQsw|{"test":true,"ulid":"01FZV53XYHS0AWTHT8G8STSB93","timestamp":1649105041590000}|{"hash":"47ccf5f773af45059af050cfdfafcbc36ca9b4a81ad5fafc516334b199804299","hashType":"sha-256","data":{"hash":"72ef467524c22a2d17607ad7a818c30ce17d21d51b8a8ef0b36c0f2c4e2b679b","hashType":"sha-256","type":"item","request":{"type":"item_req_props","asn":701,"colo":"IAD","country":"US","city":"Ashburn","continent":"NA","latitude":"39.01800","longitude":"-77.53900","postalCode":"20147","metroCode":"511","region":"Virginia","regionCode":"VA","timezone":"America/New_York"},"observableEntropy":"bc137c5569cdd1910dd227a574a38d30689c9806d9ca9af740f7a387c1e1a96d"},"signatures":[{"type":"signature","publicKey":"Qrx1usC1HsvSNNuod9HM7eVc93p9n5Zt9Rd_v1YnBr0=","signature":"qTN_IWnaw_6G0AsB_W68BUntk_hfvdJWTPILndOhUAUo-m5_yYdwpgXJnYHFR57LQbflELQH0iNjjt5TPy8WDQ==","signatureType":"ed25519","signer":{"type":"person","organizationName":"Truestamp Inc.","email":"support@truestamp.com","uri":"https://www.truestamp.com"}}],"id":"T11_01FZV53XYHS0AWTHT8G8STSB93_1649105041590000_3B3F3607558F8590ABA6AE6B9D579E8F","timestamp":"2022-04-04T20:44:01.590+00:00","type":"envelope"}

// sqlite> SELECT json_extract(items.envelope_json, '$.data.hash') FROM items WHERE json_extract(id_json, '$.ulid') LIKE '01FZV53XYHS0AWTHT8G8STSB93';
// 72ef467524c22a2d17607ad7a818c30ce17d21d51b8a8ef0b36c0f2c4e2b679b
// 72ef467524c22a2d17607ad7a818c30ce17d21d51b8a8ef0b36c0f2c4e2b679b

export function getItemEnvelopesByUlid(env: string, ulid: string): Row[] {
  const dbDir = createDataDir(env);
  const db = new DB(`${dbDir}/db.sqlite3`);
  const result = db.query(
    "SELECT envelope_json FROM items WHERE json_extract(id_json, '$.ulid') IS ?",
    [ulid],
  );
  db.close();
  return result;
}

export function getItemHashById(env: string, id: string): Row[] {
  const dbDir = createDataDir(env);
  const db = new DB(`${dbDir}/db.sqlite3`);
  const result = db.query(
    "SELECT json_extract(items.envelope_json, '$.data.hash') FROM items WHERE id IS ?;",
    [id],
  );
  db.close();
  return result;
}
