import { rsa_oaep_decrypt, rsa_oaep_encrypt, rsa_pkcs1_decrypt, rsa_pkcs1_encrypt, rsa_pkcs1_sign, rsa_pkcs1_verify, } from "./rsa_internal.ts";
import { RawBinary } from "./../binary.ts";
import { digest } from "../hash.ts";
import { rsassa_pss_sign, rsassa_pss_verify } from "./rsassa_pss.ts";
export class PureRSA {
    static async encrypt(key, message, options) {
        if (!key.e)
            throw "Invalid RSA key";
        if (options.padding === "oaep") {
            return new RawBinary(rsa_oaep_encrypt(key.length, key.n, key.e, message, options.hash));
        }
        else if (options.padding === "pkcs1") {
            return new RawBinary(rsa_pkcs1_encrypt(key.length, key.n, key.e, message));
        }
        throw "Invalid parameters";
    }
    static async decrypt(key, ciper, options) {
        if (!key.d)
            throw "Invalid RSA key";
        if (options.padding === "oaep") {
            return new RawBinary(rsa_oaep_decrypt(key, ciper, options.hash));
        }
        else if (options.padding === "pkcs1") {
            return new RawBinary(rsa_pkcs1_decrypt(key, ciper));
        }
        throw "Invalid parameters";
    }
    static async verify(key, signature, message, options) {
        if (!key.e)
            throw "Invalid RSA key";
        if (options.algorithm === "rsassa-pkcs1-v1_5") {
            return rsa_pkcs1_verify(key, signature, digest(options.hash, message));
        }
        else {
            return rsassa_pss_verify(key, message, signature, options.hash);
        }
    }
    static async sign(key, message, options) {
        if (!key.d)
            throw "You need private key to sign the message";
        if (options.algorithm === "rsassa-pkcs1-v1_5") {
            return rsa_pkcs1_sign(key.length, key.n, key.d, digest(options.hash, message), options.hash);
        }
        else {
            return rsassa_pss_sign(key, message, options.hash);
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicnNhX2pzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsicnNhX2pzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFDTCxnQkFBZ0IsRUFDaEIsZ0JBQWdCLEVBQ2hCLGlCQUFpQixFQUNqQixpQkFBaUIsRUFDakIsY0FBYyxFQUNkLGdCQUFnQixHQUNqQixNQUFNLG1CQUFtQixDQUFDO0FBQzNCLE9BQU8sRUFBRSxTQUFTLEVBQUUsTUFBTSxnQkFBZ0IsQ0FBQztBQUUzQyxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBRXBDLE9BQU8sRUFBRSxlQUFlLEVBQUUsaUJBQWlCLEVBQUUsTUFBTSxpQkFBaUIsQ0FBQztBQUVyRSxNQUFNLE9BQU8sT0FBTztJQUNsQixNQUFNLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQyxHQUFXLEVBQUUsT0FBbUIsRUFBRSxPQUFrQjtRQUN2RSxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7WUFBRSxNQUFNLGlCQUFpQixDQUFDO1FBRXBDLElBQUksT0FBTyxDQUFDLE9BQU8sS0FBSyxNQUFNLEVBQUU7WUFDOUIsT0FBTyxJQUFJLFNBQVMsQ0FDbEIsZ0JBQWdCLENBQUMsR0FBRyxDQUFDLE1BQU0sRUFBRSxHQUFHLENBQUMsQ0FBQyxFQUFFLEdBQUcsQ0FBQyxDQUFDLEVBQUUsT0FBTyxFQUFFLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FDbEUsQ0FBQztTQUNIO2FBQU0sSUFBSSxPQUFPLENBQUMsT0FBTyxLQUFLLE9BQU8sRUFBRTtZQUN0QyxPQUFPLElBQUksU0FBUyxDQUNsQixpQkFBaUIsQ0FBQyxHQUFHLENBQUMsTUFBTSxFQUFFLEdBQUcsQ0FBQyxDQUFDLEVBQUUsR0FBRyxDQUFDLENBQUMsRUFBRSxPQUFPLENBQUMsQ0FDckQsQ0FBQztTQUNIO1FBRUQsTUFBTSxvQkFBb0IsQ0FBQztJQUM3QixDQUFDO0lBRUQsTUFBTSxDQUFDLEtBQUssQ0FBQyxPQUFPLENBQUMsR0FBVyxFQUFFLEtBQWlCLEVBQUUsT0FBa0I7UUFDckUsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO1lBQUUsTUFBTSxpQkFBaUIsQ0FBQztRQUVwQyxJQUFJLE9BQU8sQ0FBQyxPQUFPLEtBQUssTUFBTSxFQUFFO1lBQzlCLE9BQU8sSUFBSSxTQUFTLENBQUMsZ0JBQWdCLENBQUMsR0FBRyxFQUFFLEtBQUssRUFBRSxPQUFPLENBQUMsSUFBSSxDQUFDLENBQUMsQ0FBQztTQUNsRTthQUFNLElBQUksT0FBTyxDQUFDLE9BQU8sS0FBSyxPQUFPLEVBQUU7WUFDdEMsT0FBTyxJQUFJLFNBQVMsQ0FBQyxpQkFBaUIsQ0FBQyxHQUFHLEVBQUUsS0FBSyxDQUFDLENBQUMsQ0FBQztTQUNyRDtRQUVELE1BQU0sb0JBQW9CLENBQUM7SUFDN0IsQ0FBQztJQUVELE1BQU0sQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUNqQixHQUFXLEVBQ1gsU0FBcUIsRUFDckIsT0FBbUIsRUFDbkIsT0FBc0I7UUFFdEIsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO1lBQUUsTUFBTSxpQkFBaUIsQ0FBQztRQUVwQyxJQUFJLE9BQU8sQ0FBQyxTQUFTLEtBQUssbUJBQW1CLEVBQUU7WUFDN0MsT0FBTyxnQkFBZ0IsQ0FDckIsR0FBRyxFQUNILFNBQVMsRUFDVCxNQUFNLENBQUMsT0FBTyxDQUFDLElBQUksRUFBRSxPQUFPLENBQUMsQ0FDOUIsQ0FBQztTQUNIO2FBQU07WUFDTCxPQUFPLGlCQUFpQixDQUFDLEdBQUcsRUFBRSxPQUFPLEVBQUUsU0FBUyxFQUFFLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQztTQUNqRTtJQUNILENBQUM7SUFFRCxNQUFNLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxHQUFXLEVBQUUsT0FBbUIsRUFBRSxPQUFzQjtRQUN4RSxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7WUFBRSxNQUFNLDBDQUEwQyxDQUFDO1FBRTdELElBQUksT0FBTyxDQUFDLFNBQVMsS0FBSyxtQkFBbUIsRUFBRTtZQUM3QyxPQUFPLGNBQWMsQ0FDbkIsR0FBRyxDQUFDLE1BQU0sRUFDVixHQUFHLENBQUMsQ0FBQyxFQUNMLEdBQUcsQ0FBQyxDQUFDLEVBQ0wsTUFBTSxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUUsT0FBTyxDQUFDLEVBQzdCLE9BQU8sQ0FBQyxJQUFJLENBQ2IsQ0FBQztTQUNIO2FBQU07WUFDTCxPQUFPLGVBQWUsQ0FBQyxHQUFHLEVBQUUsT0FBTyxFQUFFLE9BQU8sQ0FBQyxJQUFJLENBQUMsQ0FBQztTQUNwRDtJQUNILENBQUM7Q0FDRiJ9