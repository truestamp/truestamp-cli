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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiUHJvdmlkZXJFcnJvci5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIlByb3ZpZGVyRXJyb3IudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBU0EsTUFBTSxPQUFPLGFBQWMsU0FBUSxLQUFLO0lBQ087SUFBN0MsWUFBWSxPQUFlLEVBQWtCLGNBQXVCLElBQUk7UUFDdEUsS0FBSyxDQUFDLE9BQU8sQ0FBQyxDQUFDO1FBRDRCLGdCQUFXLEdBQVgsV0FBVyxDQUFnQjtJQUV4RSxDQUFDO0lBQ0QsTUFBTSxDQUFDLElBQUksQ0FBQyxLQUFZLEVBQUUsV0FBVyxHQUFHLElBQUk7UUFDMUMsTUFBTSxDQUFDLGNBQWMsQ0FBQyxLQUFLLEVBQUUsYUFBYSxFQUFFO1lBQzFDLEtBQUssRUFBRSxXQUFXO1lBQ2xCLFlBQVksRUFBRSxLQUFLO1lBQ25CLFVBQVUsRUFBRSxLQUFLO1lBQ2pCLFFBQVEsRUFBRSxLQUFLO1NBQ2hCLENBQUMsQ0FBQztRQUNILE9BQU8sS0FBc0IsQ0FBQztJQUNoQyxDQUFDO0NBQ0YifQ==