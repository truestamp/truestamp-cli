export class ProviderError extends Error {
    tryNextLink;
    constructor(message, tryNextLink = true) {
        super(message);
        this.tryNextLink = tryNextLink;
    }
    static from(error, tryNextLink = true) {
        Object.defineProperty(error, "tryNextLink", {
            value: tryNextLink,
            configurable: false,
            enumerable: false,
            writable: false,
        });
        return error;
    }
}
export class CredentialsProviderError extends Error {
    tryNextLink;
    name = "CredentialsProviderError";
    constructor(message, tryNextLink = true) {
        super(message);
        this.tryNextLink = tryNextLink;
    }
    static from(error, tryNextLink = true) {
        Object.defineProperty(error, "tryNextLink", {
            value: tryNextLink,
            configurable: false,
            enumerable: false,
            writable: false,
        });
        return error;
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUHJvdmlkZXJFcnJvci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIlByb3ZpZGVyRXJyb3IudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBV0EsTUFBTSxPQUFPLGFBQWMsU0FBUSxLQUFLO0lBQ087SUFBN0MsWUFBWSxPQUFlLEVBQWtCLGNBQXVCLElBQUk7UUFDdEUsS0FBSyxDQUFDLE9BQU8sQ0FBQyxDQUFDO1FBRDRCLGdCQUFXLEdBQVgsV0FBVyxDQUFnQjtJQUV4RSxDQUFDO0lBQ0QsTUFBTSxDQUFDLElBQUksQ0FBQyxLQUFZLEVBQUUsV0FBVyxHQUFHLElBQUk7UUFDMUMsTUFBTSxDQUFDLGNBQWMsQ0FBQyxLQUFLLEVBQUUsYUFBYSxFQUFFO1lBQzFDLEtBQUssRUFBRSxXQUFXO1lBQ2xCLFlBQVksRUFBRSxLQUFLO1lBQ25CLFVBQVUsRUFBRSxLQUFLO1lBQ2pCLFFBQVEsRUFBRSxLQUFLO1NBQ2hCLENBQUMsQ0FBQztRQUNILE9BQU8sS0FBc0IsQ0FBQztJQUNoQyxDQUFDO0NBQ0Y7QUFXRCxNQUFNLE9BQU8sd0JBQXlCLFNBQVEsS0FBSztJQUVKO0lBRHBDLElBQUksR0FBRywwQkFBMEIsQ0FBQztJQUMzQyxZQUFZLE9BQWUsRUFBa0IsY0FBdUIsSUFBSTtRQUN0RSxLQUFLLENBQUMsT0FBTyxDQUFDLENBQUM7UUFENEIsZ0JBQVcsR0FBWCxXQUFXLENBQWdCO0lBRXhFLENBQUM7SUFDRCxNQUFNLENBQUMsSUFBSSxDQUFDLEtBQVksRUFBRSxXQUFXLEdBQUcsSUFBSTtRQUMxQyxNQUFNLENBQUMsY0FBYyxDQUFDLEtBQUssRUFBRSxhQUFhLEVBQUU7WUFDMUMsS0FBSyxFQUFFLFdBQVc7WUFDbEIsWUFBWSxFQUFFLEtBQUs7WUFDbkIsVUFBVSxFQUFFLEtBQUs7WUFDakIsUUFBUSxFQUFFLEtBQUs7U0FDaEIsQ0FBQyxDQUFDO1FBQ0gsT0FBTyxLQUFpQyxDQUFDO0lBQzNDLENBQUM7Q0FDRiJ9