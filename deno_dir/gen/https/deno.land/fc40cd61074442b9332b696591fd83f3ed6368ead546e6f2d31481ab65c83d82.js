const _toString = Object.prototype.toString;
const _isObjectLike = (value) => value !== null && typeof value === "object";
const _isFunctionLike = (value) => value !== null && typeof value === "function";
export function isAnyArrayBuffer(value) {
    return (_isObjectLike(value) &&
        (_toString.call(value) === "[object ArrayBuffer]" ||
            _toString.call(value) === "[object SharedArrayBuffer]"));
}
export function isArrayBufferView(value) {
    return ArrayBuffer.isView(value);
}
export function isArgumentsObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Arguments]";
}
export function isArrayBuffer(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object ArrayBuffer]");
}
export function isAsyncFunction(value) {
    return (_isFunctionLike(value) && _toString.call(value) === "[object AsyncFunction]");
}
export function isBigInt64Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object BigInt64Array]");
}
export function isBigUint64Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object BigUint64Array]");
}
export function isBooleanObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Boolean]";
}
export function isBoxedPrimitive(value) {
    return (isBooleanObject(value) ||
        isStringObject(value) ||
        isNumberObject(value) ||
        isSymbolObject(value) ||
        isBigIntObject(value));
}
export function isDataView(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object DataView]";
}
export function isDate(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Date]";
}
export function isFloat32Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Float32Array]");
}
export function isFloat64Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Float64Array]");
}
export function isGeneratorFunction(value) {
    return (_isFunctionLike(value) &&
        _toString.call(value) === "[object GeneratorFunction]");
}
export function isGeneratorObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Generator]";
}
export function isInt8Array(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Int8Array]";
}
export function isInt16Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Int16Array]");
}
export function isInt32Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Int32Array]");
}
export function isMap(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Map]";
}
export function isMapIterator(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Map Iterator]");
}
export function isModuleNamespaceObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Module]";
}
export function isNativeError(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Error]";
}
export function isNumberObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Number]";
}
export function isBigIntObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object BigInt]";
}
export function isPromise(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Promise]";
}
export function isRegExp(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object RegExp]";
}
export function isSet(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Set]";
}
export function isSetIterator(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Set Iterator]");
}
export function isSharedArrayBuffer(value) {
    return (_isObjectLike(value) &&
        _toString.call(value) === "[object SharedArrayBuffer]");
}
export function isStringObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object String]";
}
export function isSymbolObject(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object Symbol]";
}
export function isTypedArray(value) {
    const reTypedTag = /^\[object (?:Float(?:32|64)|(?:Int|Uint)(?:8|16|32)|Uint8Clamped)Array\]$/;
    return _isObjectLike(value) && reTypedTag.test(_toString.call(value));
}
export function isUint8Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Uint8Array]");
}
export function isUint8ClampedArray(value) {
    return (_isObjectLike(value) &&
        _toString.call(value) === "[object Uint8ClampedArray]");
}
export function isUint16Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Uint16Array]");
}
export function isUint32Array(value) {
    return (_isObjectLike(value) && _toString.call(value) === "[object Uint32Array]");
}
export function isWeakMap(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object WeakMap]";
}
export function isWeakSet(value) {
    return _isObjectLike(value) && _toString.call(value) === "[object WeakSet]";
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX3V0aWxfdHlwZXMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfdXRpbF90eXBlcy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUF1QkEsTUFBTSxTQUFTLEdBQUcsTUFBTSxDQUFDLFNBQVMsQ0FBQyxRQUFRLENBQUM7QUFFNUMsTUFBTSxhQUFhLEdBQUcsQ0FBQyxLQUFjLEVBQVcsRUFBRSxDQUNoRCxLQUFLLEtBQUssSUFBSSxJQUFJLE9BQU8sS0FBSyxLQUFLLFFBQVEsQ0FBQztBQUU5QyxNQUFNLGVBQWUsR0FBRyxDQUFDLEtBQWMsRUFBVyxFQUFFLENBQ2xELEtBQUssS0FBSyxJQUFJLElBQUksT0FBTyxLQUFLLEtBQUssVUFBVSxDQUFDO0FBRWhELE1BQU0sVUFBVSxnQkFBZ0IsQ0FBQyxLQUFjO0lBQzdDLE9BQU8sQ0FDTCxhQUFhLENBQUMsS0FBSyxDQUFDO1FBQ3BCLENBQUMsU0FBUyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyxzQkFBc0I7WUFDL0MsU0FBUyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyw0QkFBNEIsQ0FBQyxDQUMxRCxDQUFDO0FBQ0osQ0FBQztBQUVELE1BQU0sVUFBVSxpQkFBaUIsQ0FBQyxLQUFjO0lBQzlDLE9BQU8sV0FBVyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUMsQ0FBQztBQUNuQyxDQUFDO0FBRUQsTUFBTSxVQUFVLGlCQUFpQixDQUFDLEtBQWM7SUFDOUMsT0FBTyxhQUFhLENBQUMsS0FBSyxDQUFDLElBQUksU0FBUyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyxvQkFBb0IsQ0FBQztBQUNoRixDQUFDO0FBRUQsTUFBTSxVQUFVLGFBQWEsQ0FBQyxLQUFjO0lBQzFDLE9BQU8sQ0FDTCxhQUFhLENBQUMsS0FBSyxDQUFDLElBQUksU0FBUyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsS0FBSyxzQkFBc0IsQ0FDekUsQ0FBQztBQUNKLENBQUM7QUFFRCxNQUFNLFVBQVUs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