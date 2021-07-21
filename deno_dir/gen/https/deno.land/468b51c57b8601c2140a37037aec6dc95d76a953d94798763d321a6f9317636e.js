import { isClockSkewError, isRetryableByTrait, isThrottlingError, isTransientError, } from "../service-error-classification/mod.ts";
export const defaultRetryDecider = (error) => {
    if (!error) {
        return false;
    }
    return isRetryableByTrait(error) || isClockSkewError(error) || isThrottlingError(error) || isTransientError(error);
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicmV0cnlEZWNpZGVyLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsicmV0cnlEZWNpZGVyLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFDTCxnQkFBZ0IsRUFDaEIsa0JBQWtCLEVBQ2xCLGlCQUFpQixFQUNqQixnQkFBZ0IsR0FDakIsTUFBTSx3Q0FBd0MsQ0FBQztBQUdoRCxNQUFNLENBQUMsTUFBTSxtQkFBbUIsR0FBRyxDQUFDLEtBQWUsRUFBRSxFQUFFO0lBQ3JELElBQUksQ0FBQyxLQUFLLEVBQUU7UUFDVixPQUFPLEtBQUssQ0FBQztLQUNkO0lBRUQsT0FBTyxrQkFBa0IsQ0FBQyxLQUFLLENBQUMsSUFBSSxnQkFBZ0IsQ0FBQyxLQUFLLENBQUMsSUFBSSxpQkFBaUIsQ0FBQyxLQUFLLENBQUMsSUFBSSxnQkFBZ0IsQ0FBQyxLQUFLLENBQUMsQ0FBQztBQUNySCxDQUFDLENBQUMifQ==