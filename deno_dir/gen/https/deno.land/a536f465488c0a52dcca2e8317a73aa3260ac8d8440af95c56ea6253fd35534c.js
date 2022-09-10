/*
 * JSW §1: Cryptographic algorithms and identifiers for use with this specification
 * are described in the separate JSON Web Algorithms (JWA) specification:
 * https://www.rfc-editor.org/rfc/rfc7518
 */ // Still needs an 'any' type! Does anyone have an idea?
// https://github.com/denoland/deno/blob/main/ext/crypto/lib.deno_crypto.d.ts
function isHashedKeyAlgorithm(// deno-lint-ignore no-explicit-any
algorithm) {
    return typeof algorithm.hash?.name === "string";
}
function isEcKeyAlgorithm(// deno-lint-ignore no-explicit-any
algorithm) {
    return typeof algorithm.namedCurve === "string";
}
export function verify(alg, key) {
    if (alg === "none") {
        if (key !== null) throw new Error(`The alg '${alg}' does not allow a key.`);
        else return true;
    } else {
        if (!key) throw new Error(`The alg '${alg}' demands a key.`);
        const keyAlgorithm = key.algorithm;
        const algAlgorithm = getAlgorithm(alg);
        if (keyAlgorithm.name === algAlgorithm.name) {
            if (isHashedKeyAlgorithm(keyAlgorithm)) {
                return keyAlgorithm.hash.name === algAlgorithm.hash.name;
            } else if (isEcKeyAlgorithm(keyAlgorithm)) {
                return keyAlgorithm.namedCurve === algAlgorithm.namedCurve;
            }
        }
        return false;
    }
}
export function getAlgorithm(alg) {
    switch(alg){
        case "HS256":
            return {
                hash: {
                    name: "SHA-256"
                },
                name: "HMAC"
            };
        case "HS384":
            return {
                hash: {
                    name: "SHA-384"
                },
                name: "HMAC"
            };
        case "HS512":
            return {
                hash: {
                    name: "SHA-512"
                },
                name: "HMAC"
            };
        case "PS256":
            return {
                hash: {
                    name: "SHA-256"
                },
                name: "RSA-PSS",
                saltLength: 256 >> 3
            };
        case "PS384":
            return {
                hash: {
                    name: "SHA-384"
                },
                name: "RSA-PSS",
                saltLength: 384 >> 3
            };
        case "PS512":
            return {
                hash: {
                    name: "SHA-512"
                },
                name: "RSA-PSS",
                saltLength: 512 >> 3
            };
        case "RS256":
            return {
                hash: {
                    name: "SHA-256"
                },
                name: "RSASSA-PKCS1-v1_5"
            };
        case "RS384":
            return {
                hash: {
                    name: "SHA-384"
                },
                name: "RSASSA-PKCS1-v1_5"
            };
        case "RS512":
            return {
                hash: {
                    name: "SHA-512"
                },
                name: "RSASSA-PKCS1-v1_5"
            };
        case "ES256":
            return {
                hash: {
                    name: "SHA-256"
                },
                name: "ECDSA",
                namedCurve: "P-256"
            };
        case "ES384":
            return {
                hash: {
                    name: "SHA-384"
                },
                name: "ECDSA",
                namedCurve: "P-384"
            };
        // case "ES512":
        // return { hash: { name: "SHA-512" }, name: "ECDSA", namedCurve: "P-521" };
        default:
            throw new Error(`The jwt's alg '${alg}' is not supported.`);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvZGp3dEB2Mi43L2FsZ29yaXRobS50cyJdLCJzb3VyY2VzQ29udGVudCI6WyIvKlxuICogSlNXIMKnMTogQ3J5cHRvZ3JhcGhpYyBhbGdvcml0aG1zIGFuZCBpZGVudGlmaWVycyBmb3IgdXNlIHdpdGggdGhpcyBzcGVjaWZpY2F0aW9uXG4gKiBhcmUgZGVzY3JpYmVkIGluIHRoZSBzZXBhcmF0ZSBKU09OIFdlYiBBbGdvcml0aG1zIChKV0EpIHNwZWNpZmljYXRpb246XG4gKiBodHRwczovL3d3dy5yZmMtZWRpdG9yLm9yZy9yZmMvcmZjNzUxOFxuICovXG5leHBvcnQgdHlwZSBBbGdvcml0aG0gPVxuICB8IFwiSFMyNTZcIlxuICB8IFwiSFMzODRcIlxuICB8IFwiSFM1MTJcIlxuICB8IFwiUFMyNTZcIlxuICB8IFwiUFMzODRcIlxuICB8IFwiUFM1MTJcIlxuICB8IFwiUlMyNTZcIlxuICB8IFwiUlMzODRcIlxuICB8IFwiUlM1MTJcIlxuICB8IFwiRVMyNTZcIlxuICB8IFwiRVMzODRcIlxuICAvLyBQLTUyMSBpcyBub3QgeWV0IHN1cHBvcnRlZC5cbiAgLy8gaHR0cHM6Ly9naXRodWIuY29tL2Rlbm9sYW5kL2Rlbm8vYmxvYi9tYWluL2V4dC9jcnlwdG8vMDBfY3J5cHRvLmpzXG4gIHwgXCJub25lXCI7XG5cbi8vIFN0aWxsIG5lZWRzIGFuICdhbnknIHR5cGUhIERvZXMgYW55b25lIGhhdmUgYW4gaWRlYT9cbi8vIGh0dHBzOi8vZ2l0aHViLmNvbS9kZW5vbGFuZC9kZW5vL2Jsb2IvbWFpbi9leHQvY3J5cHRvL2xpYi5kZW5vX2NyeXB0by5kLnRzXG5mdW5jdGlvbiBpc0hhc2hlZEtleUFsZ29yaXRobShcbiAgLy8gZGVuby1saW50LWlnbm9yZSBuby1leHBsaWNpdC1hbnlcbiAgYWxnb3JpdGhtOiBSZWNvcmQ8c3RyaW5nLCBhbnk+LFxuKTogYWxnb3JpdGhtIGlzIEhtYWNLZXlBbGdvcml0aG0gfCBSc2FIYXNoZWRLZXlBbGdvcml0aG0ge1xuICByZXR1cm4gdHlwZW9mIGFsZ29yaXRobS5oYXNoPy5uYW1lID09PSBcInN0cmluZ1wiO1xufVxuXG5mdW5jdGlvbiBpc0VjS2V5QWxnb3JpdGhtKFxuICAvLyBkZW5vLWxpbnQtaWdub3JlIG5vLWV4cGxpY2l0LWFueVxuICBhbGdvcml0aG06IFJlY29yZDxzdHJpbmcsIGFueT4sXG4pOiBhbGdvcml0aG0gaXMgRWNLZXlBbGdvcml0aG0ge1xuICByZXR1cm4gdHlwZW9mIGFsZ29yaXRobS5uYW1lZEN1cnZlID09PSBcInN0cmluZ1wiO1xufVxuXG5leHBvcnQgZnVuY3Rpb24gdmVyaWZ5KGFsZzogQWxnb3JpdGhtLCBrZXk6IENyeXB0b0tleSB8IG51bGwpOiBib29sZWFuIHtcbiAgaWYgKGFsZyA9PT0gXCJub25lXCIpIHtcbiAgICBpZiAoa2V5ICE9PSBudWxsKSB0aHJvdyBuZXcgRXJyb3IoYFRoZSBhbGcgJyR7YWxnfScgZG9lcyBub3QgYWxsb3cgYSBrZXkuYCk7XG4gICAgZWxzZSByZXR1cm4gdHJ1ZTtcbiAgfSBlbHNlIHtcbiAgICBpZiAoIWtleSkgdGhyb3cgbmV3IEVycm9yKGBUaGUgYWxnICcke2FsZ30nIGRlbWFuZHMgYSBrZXkuYCk7XG4gICAgY29uc3Qga2V5QWxnb3JpdGhtID0ga2V5LmFsZ29yaXRobTtcbiAgICBjb25zdCBhbGdBbGdvcml0aG0gPSBnZXRBbGdvcml0aG0oYWxnKTtcbiAgICBpZiAoa2V5QWxnb3JpdGhtLm5hbWUgPT09IGFsZ0FsZ29yaXRobS5uYW1lKSB7XG4gICAgICBpZiAoaXNIYXNoZWRLZXlBbGdvcml0aG0oa2V5QWxnb3JpdGhtKSkge1xuICAgICAgICByZXR1cm4ga2V5QWxnb3JpdGhtLmhhc2gubmFtZSA9PT0gYWxnQWxnb3JpdGhtLmhhc2gubmFtZTtcbiAgICAgIH0gZWxzZSBpZiAoaXNFY0tleUFsZ29yaXRobShrZXlBbGdvcml0aG0pKSB7XG4gICAgICAgIHJldHVybiBrZXlBbGdvcml0aG0ubmFtZWRDdXJ2ZSA9PT0gYWxnQWxnb3JpdGhtLm5hbWVkQ3VydmU7XG4gICAgICB9XG4gICAgfVxuICAgIHJldHVybiBmYWxzZTtcbiAgfVxufVxuXG5leHBvcnQgZnVuY3Rpb24gZ2V0QWxnb3JpdGhtKGFsZzogQWxnb3JpdGhtKSB7XG4gIHN3aXRjaCAoYWxnKSB7XG4gICAgY2FzZSBcIkhTMjU2XCI6XG4gICAgICByZXR1cm4geyBoYXNoOiB7IG5hbWU6IFwiU0hBLTI1NlwiIH0sIG5hbWU6IFwiSE1BQ1wiIH07XG4gICAgY2FzZSBcIkhTMzg0XCI6XG4gICAgICByZXR1cm4geyBoYXNoOiB7IG5hbWU6IFwiU0hBLTM4NFwiIH0sIG5hbWU6IFwiSE1BQ1wiIH07XG4gICAgY2FzZSBcIkhTNTEyXCI6XG4gICAgICByZXR1cm4geyBoYXNoOiB7IG5hbWU6IFwiU0hBLTUxMlwiIH0sIG5hbWU6IFwiSE1BQ1wiIH07XG4gICAgY2FzZSBcIlBTMjU2XCI6XG4gICAgICByZXR1cm4ge1xuICAgICAgICBoYXNoOiB7IG5hbWU6IFwiU0hBLTI1NlwiIH0sXG4gICAgICAgIG5hbWU6IFwiUlNBLVBTU1wiLFxuICAgICAgICBzYWx0TGVuZ3RoOiAyNTYgPj4gMyxcbiAgICAgIH07XG4gICAgY2FzZSBcIlBTMzg0XCI6XG4gICAgICByZXR1cm4ge1xuICAgICAgICBoYXNoOiB7IG5hbWU6IFwiU0hBLTM4NFwiIH0sXG4gICAgICAgIG5hbWU6IFwiUlNBLVBTU1wiLFxuICAgICAgICBzYWx0TGVuZ3RoOiAzODQgPj4gMyxcbiAgICAgIH07XG4gICAgY2FzZSBcIlBTNTEyXCI6XG4gICAgICByZXR1cm4ge1xuICAgICAgICBoYXNoOiB7IG5hbWU6IFwiU0hBLTUxMlwiIH0sXG4gICAgICAgIG5hbWU6IFwiUlNBLVBTU1wiLFxuICAgICAgICBzYWx0TGVuZ3RoOiA1MTIgPj4gMyxcbiAgICAgIH07XG4gICAgY2FzZSBcIlJTMjU2XCI6XG4gICAgICByZXR1cm4geyBoYXNoOiB7IG5hbWU6IFwiU0hBLTI1NlwiIH0sIG5hbWU6IFwiUlNBU1NBLVBLQ1MxLXYxXzVcIiB9O1xuICAgIGNhc2UgXCJSUzM4NFwiOlxuICAgICAgcmV0dXJuIHsgaGFzaDogeyBuYW1lOiBcIlNIQS0zODRcIiB9LCBuYW1lOiBcIlJTQVNTQS1QS0NTMS12MV81XCIgfTtcbiAgICBjYXNlIFwiUlM1MTJcIjpcbiAgICAgIHJldHVybiB7IGhhc2g6IHsgbmFtZTogXCJTSEEtNTEyXCIgfSwgbmFtZTogXCJSU0FTU0EtUEtDUzEtdjFfNVwiIH07XG4gICAgY2FzZSBcIkVTMjU2XCI6XG4gICAgICByZXR1cm4geyBoYXNoOiB7IG5hbWU6IFwiU0hBLTI1NlwiIH0sIG5hbWU6IFwiRUNEU0FcIiwgbmFtZWRDdXJ2ZTogXCJQLTI1NlwiIH07XG4gICAgY2FzZSBcIkVTMzg0XCI6XG4gICAgICByZXR1cm4geyBoYXNoOiB7IG5hbWU6IFwiU0hBLTM4NFwiIH0sIG5hbWU6IFwiRUNEU0FcIiwgbmFtZWRDdXJ2ZTogXCJQLTM4NFwiIH07XG4gICAgLy8gY2FzZSBcIkVTNTEyXCI6XG4gICAgLy8gcmV0dXJuIHsgaGFzaDogeyBuYW1lOiBcIlNIQS01MTJcIiB9LCBuYW1lOiBcIkVDRFNBXCIsIG5hbWVkQ3VydmU6IFwiUC01MjFcIiB9O1xuICAgIGRlZmF1bHQ6XG4gICAgICB0aHJvdyBuZXcgRXJyb3IoYFRoZSBqd3QncyBhbGcgJyR7YWxnfScgaXMgbm90IHN1cHBvcnRlZC5gKTtcbiAgfVxufVxuIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBOzs7O0NBSUMsR0FDRCxBQWdCQSx1REFBdUQ7QUFDdkQsNkVBQTZFO0FBQzdFLFNBQVMsb0JBQW9CLENBQzNCLG1DQUFtQztBQUNuQyxTQUE4QixFQUN5QjtJQUN2RCxPQUFPLE9BQU8sU0FBUyxDQUFDLElBQUksRUFBRSxJQUFJLEtBQUssUUFBUSxDQUFDO0FBQ2xELENBQUM7QUFFRCxTQUFTLGdCQUFnQixDQUN2QixtQ0FBbUM7QUFDbkMsU0FBOEIsRUFDRDtJQUM3QixPQUFPLE9BQU8sU0FBUyxDQUFDLFVBQVUs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