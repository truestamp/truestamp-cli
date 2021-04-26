// From https://github.com/sindresorhus/type-fest/blob/198f2c2e01091f9c4de0c3ad52d286dfaf47089c/source/basic.d.ts
/**
Matches a JSON object.
*/
type JsonObject = {[Key in string]?: JsonValue}

/**
Matches a JSON array.
*/
interface JsonArray extends Array<JsonValue> {}

/**
Matches any valid JSON value.
*/
type JsonValue = string | number | boolean | null | JsonObject | JsonArray

/**
Read and parse a JSON file.
@param filePath The path to the JSON file to read.
*/
const loadJsonFile = async <ValueType = JsonValue>(filePath: string): Promise<ValueType> => {
	return JSON.parse(await Deno.readTextFile(filePath))
}

/**
Synchronously read and parse a JSON file.
@param filePath The path to the JSON file to read.
*/
loadJsonFile.sync = <ValueType = JsonValue>(filePath: string): ValueType => {
	return JSON.parse(Deno.readTextFileSync(filePath))
}

export default loadJsonFile
