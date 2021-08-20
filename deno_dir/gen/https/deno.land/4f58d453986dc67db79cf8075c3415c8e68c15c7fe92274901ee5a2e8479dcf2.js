export function verify(alg, key) {
    if (alg === "none") {
        if (key)
            throw new Error(`The alg '${alg}' does not allow a key.`);
        else
            return true;
    }
    else {
        if (!key)
            throw new Error(`The alg '${alg}' demands a key.`);
        const algorithm = getAlgorithm(alg);
        if (key.algorithm.name === algorithm.name) {
            if (key.algorithm.hash?.name &&
                key.algorithm.hash?.name ===
                    algorithm.hash.name) {
                return true;
            }
        }
    }
    return false;
}
export function getAlgorithm(alg) {
    switch (alg) {
        case "HS256":
            return { hash: { name: "SHA-256" }, name: "HMAC" };
        case "HS384":
            return { hash: { name: "SHA-384" }, name: "HMAC" };
        case "HS512":
            return { hash: { name: "SHA-512" }, name: "HMAC" };
        case "PS256":
            return {
                hash: { name: "SHA-256" },
                name: "RSA-PSS",
                saltLength: 256 >> 3,
            };
        case "PS384":
            return {
                hash: { name: "SHA-384" },
                name: "RSA-PSS",
                saltLength: 384 >> 3,
            };
        case "PS512":
            return {
                hash: { name: "SHA-512" },
                name: "RSA-PSS",
                saltLength: 512 >> 3,
            };
        case "RS256":
            return { hash: { name: "SHA-256" }, name: "RSASSA-PKCS1-v1_5" };
        case "RS384":
            return { hash: { name: "SHA-384" }, name: "RSASSA-PKCS1-v1_5" };
        case "RS512":
            return { hash: { name: "SHA-512" }, name: "RSASSA-PKCS1-v1_5" };
        default:
            throw new Error(`The jwt's alg '${alg}' is not supported.`);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYWxnb3JpdGhtLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiYWxnb3JpdGhtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQXFCQSxNQUFNLFVBQVUsTUFBTSxDQUNwQixHQUFjLEVBQ2QsR0FBcUI7SUFFckIsSUFBSSxHQUFHLEtBQUssTUFBTSxFQUFFO1FBQ2xCLElBQUksR0FBRztZQUFFLE1BQU0sSUFBSSxLQUFLLENBQUMsWUFBWSxHQUFHLHlCQUF5QixDQUFDLENBQUM7O1lBQzlELE9BQU8sSUFBSSxDQUFDO0tBQ2xCO1NBQU07UUFDTCxJQUFJLENBQUMsR0FBRztZQUFFLE1BQU0sSUFBSSxLQUFLLENBQUMsWUFBWSxHQUFHLGtCQUFrQixDQUFDLENBQUM7UUFDN0QsTUFBTSxTQUFTLEdBQUcsWUFBWSxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ3BDLElBQUksR0FBRyxDQUFDLFNBQVMsQ0FBQyxJQUFJLEtBQUssU0FBUyxDQUFDLElBQUksRUFBRTtZQUN6QyxJQUdHLEdBQUcsQ0FBQyxTQUF5QyxDQUFDLElBQUksRUFBRSxJQUFJO2dCQUN4RCxHQUFHLENBQUMsU0FBeUMsQ0FBQyxJQUFJLEVBQUUsSUFBSTtvQkFDdkQsU0FBUyxDQUFDLElBQUksQ0FBQyxJQUFJLEVBQ3JCO2dCQUNBLE9BQU8sSUFBSSxDQUFDO2FBT2I7U0FDRjtLQUNGO0lBQ0QsT0FBTyxLQUFLLENBQUM7QUFDZixDQUFDO0FBRUQsTUFBTSxVQUFVLFlBQVksQ0FBQyxHQUFjO0lBQ3pDLFFBQVEsR0FBRyxFQUFFO1FBQ1gsS0FBSyxPQUFPO1lBQ1YsT0FBTyxFQUFFLElBQUksRUFBRSxFQUFFLElBQUksRUFBRSxTQUFTLEVBQUUsRUFBRSxJQUFJLEVBQUUsTUFBTSxFQUFFLENBQUM7UUFDckQsS0FBSyxPQUFPO1lBQ1YsT0FBTyxFQUFFLElBQUksRUFBRSxFQUFFLElBQUksRUFBRSxTQUFTLEVBQUUsRUFBRSxJQUFJLEVBQUUsTUFBTSxFQUFFLENBQUM7UUFDckQsS0FBSyxPQUFPO1lBQ1YsT0FBTyxFQUFFLElBQUksRUFBRSxFQUFFLElBQUksRUFBRSxTQUFTLEVBQUUsRUFBRSxJQUFJLEVBQUUsTUFBTSxFQUFFLENBQUM7UUFDckQsS0FBSyxPQUFPO1lBQ1YsT0FBTztnQkFDTCxJQUFJLEVBQUUsRUFBRSxJQUFJLEVBQUUsU0FBUyxFQUFFO2dCQUN6QixJQUFJLEVBQUUsU0FBUztnQkFDZixVQUFVLEVBQUUsR0FBRyxJQUFJLENBQUM7YUFDckIsQ0FBQztRQUNKLEtBQUssT0FBTztZQUNWLE9BQU87Z0JBQ0wsSUFBSSxFQUFFLEVBQUUsSUFBSSxFQUFFLFNBQVMsRUFBRTtnQkFDekIsSUFBSSxFQUFFLFNBQVM7Z0JBQ2YsVUFBVSxFQUFFLEdBQUcsSUFBSSxDQUFDO2FBQ3JCLENBQUM7UUFDSixLQUFLLE9BQU87WUFDVixPQUFPO2dCQUNMLElBQUksRUFBRSxFQUFFLElBQUksRUFBRSxTQUFTLEVBQUU7Z0JBQ3pCLElBQUksRUFBRSxTQUFTO2dCQUNmLFVBQVUsRUFBRSxHQUFHLElBQUksQ0FBQzthQUNyQixDQUFDO1FBQ0osS0FBSyxPQUFPO1lBQ1YsT0FBTyxFQUFFLElBQUksRUFBRSxFQUFFLElBQUksRUFBRSxTQUFTLEVBQUUsRUFBRSxJQUFJLEVBQUUsbUJBQW1CLEVBQUUsQ0FBQztRQUNsRSxLQUFLLE9BQU87WUFDVixPQUFPLEVBQUUsSUFBSSxFQUFFLEVBQUUsSUFBSSxFQUFFLFNBQVMsRUFBRSxFQUFFLElBQUksRUFBRSxtQkFBbUIsRUFBRSxDQUFDO1FBQ2xFLEtBQUssT0FBTztZQUNWLE9BQU8sRUFBRSxJQUFJLEVBQUUsRUFBRSxJQUFJLEVBQUUsU0FBUyxFQUFFLEVBQUUsSUFBSSxFQUFFLG1CQUFtQixFQUFFLENBQUM7UUFPbEU7WUFDRSxNQUFNLElBQUksS0FBSyxDQUFDLGtCQUFrQixHQUFHLHFCQUFxQixDQUFDLENBQUM7S0FDL0Q7QUFDSCxDQUFDIn0=