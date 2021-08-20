import { base64url } from "./deps.ts";
import { create as createSignature, verify as verifySignature, } from "./signature.ts";
import { verify as verifyAlgorithm } from "./algorithm.ts";
export const encoder = new TextEncoder();
export const decoder = new TextDecoder();
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
                    return uint8Array;
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
export async function verify(jwt, key) {
    const { header, payload, signature } = validate(decode(jwt));
    if (verifyAlgorithm(header.alg, key)) {
        if (!(await verifySignature(signature, key, header.alg, jwt.slice(0, jwt.lastIndexOf("."))))) {
            throw new Error("The jwt's signature does not match the verification signature.");
        }
        return payload;
    }
    else {
        throw new Error(`The jwt's alg '${header.alg}' does not match the key's algorithm.`);
    }
}
function createSigningInput(header, payload) {
    return `${base64url.encode(encoder.encode(JSON.stringify(header)))}.${base64url.encode(encoder.encode(JSON.stringify(payload)))}`;
}
export async function create(header, payload, key) {
    if (verifyAlgorithm(header.alg, key)) {
        const signingInput = createSigningInput(header, payload);
        const signature = await createSignature(header.alg, key, signingInput);
        return `${signingInput}.${signature}`;
    }
    else {
        throw new Error(`The jwt's alg '${header.alg}' does not match the key's algorithm.`);
    }
}
export function getNumericDate(exp) {
    return Math.round((exp instanceof Date ? exp.getTime() : Date.now() + exp * 1000) / 1000);
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxTQUFTLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDdEMsT0FBTyxFQUNMLE1BQU0sSUFBSSxlQUFlLEVBQ3pCLE1BQU0sSUFBSSxlQUFlLEdBQzFCLE1BQU0sZ0JBQWdCLENBQUM7QUFDeEIsT0FBTyxFQUFFLE1BQU0sSUFBSSxlQUFlLEVBQUUsTUFBTSxnQkFBZ0IsQ0FBQztBQWtDM0QsTUFBTSxDQUFDLE1BQU0sT0FBTyxHQUFHLElBQUksV0FBVyxFQUFFLENBQUM7QUFDekMsTUFBTSxDQUFDLE1BQU0sT0FBTyxHQUFHLElBQUksV0FBVyxFQUFFLENBQUM7QUFNekMsU0FBUyxTQUFTLENBQUMsR0FBVyxFQUFFLE1BQU0sR0FBRyxDQUFDO0lBQ3hDLE9BQU8sR0FBRyxHQUFHLE1BQU0sR0FBRyxJQUFJLENBQUMsR0FBRyxFQUFFLEdBQUcsSUFBSSxDQUFDO0FBQzFDLENBQUM7QUFFRCxTQUFTLFVBQVUsQ0FBQyxHQUFXLEVBQUUsTUFBTSxHQUFHLENBQUM7SUFDekMsT0FBTyxHQUFHLEdBQUcsTUFBTSxHQUFHLElBQUksQ0FBQyxHQUFHLEVBQUUsR0FBRyxJQUFJLENBQUM7QUFDMUMsQ0FBQztBQUVELFNBQVMsUUFBUSxDQUFDLEdBQVk7SUFDNUIsT0FBTyxDQUNMLEdBQUcsS0FBSyxJQUFJLElBQUksT0FBTyxHQUFHLEtBQUssUUFBUSxJQUFJLEtBQUssQ0FBQyxPQUFPLENBQUMsR0FBRyxDQUFDLEtBQUssS0FBSyxDQUN4RSxDQUFDO0FBQ0osQ0FBQztBQUVELFNBQVMsUUFBUSxDQUFDLEdBQVU7SUFDMUIsT0FBTyxHQUFHLENBQUMsTUFBTSxLQUFLLENBQUMsQ0FBQztBQUMxQixDQUFDO0FBRUQsU0FBUyxzQkFBc0IsQ0FBQyxHQUFHLFdBQXNCO0lBQ3ZELE9BQU8sV0FBVyxDQUFDLElBQUksQ0FBQyxDQUFDLFVB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