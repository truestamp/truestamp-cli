import { ProviderError } from "./ProviderError.ts";
export function chain(...providers) {
    return () => {
        let promise = Promise.reject(new ProviderError("No providers in chain"));
        for (const provider of providers) {
            promise = promise.catch((err) => {
                if (err?.tryNextLink) {
                    return provider();
                }
                throw err;
            });
        }
        return promise;
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY2hhaW4uanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJjaGFpbi50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFFQSxPQUFPLEVBQUUsYUFBYSxFQUFFLE1BQU0sb0JBQW9CLENBQUM7QUFXbkQsTUFBTSxVQUFVLEtBQUssQ0FBSSxHQUFHLFNBQTZCO0lBQ3ZELE9BQU8sR0FBRyxFQUFFO1FBQ1YsSUFBSSxPQUFPLEdBQWUsT0FBTyxDQUFDLE1BQU0sQ0FBQyxJQUFJLGFBQWEsQ0FBQyx1QkFBdUIsQ0FBQyxDQUFDLENBQUM7UUFDckYsS0FBSyxNQUFNLFFBQVEsSUFBSSxTQUFTLEVBQUU7WUFDaEMsT0FBTyxHQUFHLE9BQU8sQ0FBQyxLQUFLLENBQUMsQ0FBQyxHQUFRLEVBQUUsRUFBRTtnQkFDbkMsSUFBSSxHQUFHLEVBQUUsV0FBVyxFQUFFO29CQUNwQixPQUFPLFFBQVEsRUFBRSxDQUFDO2lCQUNuQjtnQkFFRCxNQUFNLEdBQUcsQ0FBQztZQUNaLENBQUMsQ0FBQyxDQUFDO1NBQ0o7UUFFRCxPQUFPLE9BQU8sQ0FBQztJQUNqQixDQUFDLENBQUM7QUFDSixDQUFDIn0=