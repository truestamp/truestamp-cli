import { base64url, convertUint8ArrayToHex } from "./deps.ts";
import { create as createSignature, verify as verifySignature, } from "./signature.ts";
import { verify as verifyAlgorithm } from "./algorithm.ts";
const encoder = new TextEncoder();
const decoder = new TextDecoder();
function isExpired(exp, leeway = 0) {
    return exp + leeway < Date.now() / 1000;
}
function isTooEarly(nbf, leeway = 0) {
    return nbf - leeway > Date.now() / 1000;
}
function isObject(obj) {
    return (obj !== null && typeof obj === "object" && Array.isArray(obj) === false);
}
function is3Tuple(arr) {
    return arr.length === 3;
}
function hasInvalidTimingClaims(...claimValues) {
    return claimValues.some((claimValue) => claimValue !== undefined ? typeof claimValue !== "number" : false);
}
export function decode(jwt) {
    try {
        const arr = jwt
            .split(".")
            .map(base64url.decode)
            .map((uint8Array, index) => {
            switch (index) {
                case 0:
                case 1:
                    return JSON.parse(decoder.decode(uint8Array));
                case 2:
                    return convertUint8ArrayToHex(uint8Array);
            }
        });
        if (is3Tuple(arr))
            return arr;
        else
            throw new Error();
    }
    catch {
        throw TypeError("The serialization of the jwt is invalid.");
    }
}
export function validate([header, payload, signature]) {
    if (typeof signature !== "string") {
        throw new Error(`The signature of the jwt must be a string.`);
    }
    if (typeof header?.alg !== "string") {
        throw new Error(`The header 'alg' parameter of the jwt must be a string.`);
    }
    if (isObject(payload)) {
        if (hasInvalidTimingClaims(payload.exp, payload.nbf)) {
            throw new Error(`The jwt has an invalid 'exp' or 'nbf' claim.`);
        }
        if (typeof payload.exp === "number" && isExpired(payload.exp, 1)) {
            throw RangeError("The jwt is expired.");
        }
        if (typeof payload.nbf === "number" && isTooEarly(payload.nbf, 1)) {
            throw RangeError("The jwt is used too early.");
        }
        return {
            header,
            payload,
            signature,
        };
    }
    else {
        throw new Error(`The jwt claims set is not a JSON object.`);
    }
}
export async function verify(jwt, key, algorithm) {
    const { header, payload, signature } = validate(decode(jwt));
    if (!verifyAlgorithm(algorithm, header.alg)) {
        throw new Error(`The jwt's algorithm does not match the specified algorithm '${algorithm}'.`);
    }
    if (!(await verifySignature({
        signature,
        key,
        algorithm: header.alg,
        signingInput: jwt.slice(0, jwt.lastIndexOf(".")),
    }))) {
        throw new Error("The jwt's signature does not match the verification signature.");
    }
    return payload;
}
function createSigningInput(header, payload) {
    return `${base64url.encode(encoder.encode(JSON.stringify(header)))}.${base64url.encode(encoder.encode(JSON.stringify(payload)))}`;
}
export async function create(header, payload, key) {
    const signingInput = createSigningInput(header, payload);
    const signature = await createSignature(header.alg, key, signingInput);
    return `${signingInput}.${signature}`;
}
export function getNumericDate(exp) {
    return Math.round((exp instanceof Date ? exp.getTime() : Date.now() + exp * 1000) / 1000);
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxTQUFTLEVBQUUsc0JBQXNCLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDOUQsT0FBTyxFQUNMLE1BQU0sSUFBSSxlQUFlLEVBQ3pCLE1BQU0sSUFBSSxlQUFlLEdBQzFCLE1BQU0sZ0JBQWdCLENBQUM7QUFDeEIsT0FBTyxFQUFFLE1BQU0sSUFBSSxlQUFlLEVBQUUsTUFBTSxnQkFBZ0IsQ0FBQztBQWtDM0QsTUFBTSxPQUFPLEdBQUcsSUFBSSxXQUFXLEVBQUUsQ0FBQztBQUNsQyxNQUFNLE9BQU8sR0FBRyxJQUFJLFdBQVcsRUFBRSxDQUFDO0FBTWxDLFNBQVMsU0FBUyxDQUFDLEdBQVcsRUFBRSxNQUFNLEdBQUcsQ0FBQztJQUN4QyxPQUFPLEdBQUcsR0FBRyxNQUFNLEdBQUcsSUFBSSxDQUFDLEdBQUcsRUFBRSxHQUFHLElBQUksQ0FBQztBQUMxQyxDQUFDO0FBRUQsU0FBUyxVQUFVLENBQUMsR0FBVyxFQUFFLE1BQU0sR0FBRyxDQUFDO0lBQ3pDLE9BQU8sR0FBRyxHQUFHLE1BQU0sR0FBRyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsSUFBSSxDQUFDO0FBQzFDLENBQUM7QUFFRCxTQUFTLFFBQVEsQ0FBQyxHQUFZO0lBQzVCLE9BQU8sQ0FDTCxHQUFHLEtBQUssSUFBSSxJQUFJLE9BQU8sR0FBRyxLQUFLLFFBQVEsSUFBSSxLQUFLLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxLQUFLLEtBQUssQ0FDeEUsQ0FBQztBQUNKLENBQUM7QUFFRCxTQUFTLFFBQVEsQ0FBQyxHQUFVO0lBQzFCLE9BQU8sR0FBRyxDQUFDLE1BQU0sS0FBSyxDQUFDLENBQUM7QUFDMUIsQ0FBQztBQUVELFNBQVMsc0JBQXNCLENBQUMsR0FBRyxXQUFzQjtJQUN2RCxPQUFPLFdBQVcsQ0FBQyxJQUFJLENBQUMsQ0FBQyxVQUFVLEVBQUUs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