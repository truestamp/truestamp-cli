export const memoize = (provider, isExpired, requiresRefresh) => {
    let result;
    let hasResult;
    if (isExpired === undefined) {
        return () => {
            if (!hasResult) {
                result = provider();
                hasResult = true;
            }
            return result;
        };
    }
    let isConstant = false;
    return async () => {
        if (!hasResult) {
            result = provider();
            hasResult = true;
        }
        if (isConstant) {
            return result;
        }
        const resolved = await result;
        if (requiresRefresh && !requiresRefresh(resolved)) {
            isConstant = true;
            return resolved;
        }
        if (isExpired(resolved)) {
            return (result = provider());
        }
        return resolved;
    };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibWVtb2l6ZS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIm1lbW9pemUudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBMENBLE1BQU0sQ0FBQyxNQUFNLE9BQU8sR0FBb0IsQ0FDdEMsUUFBcUIsRUFDckIsU0FBb0MsRUFDcEMsZUFBMEMsRUFDN0IsRUFBRTtJQUNmLElBQUksTUFBVyxDQUFDO0lBQ2hCLElBQUksU0FBa0IsQ0FBQztJQUN2QixJQUFJLFNBQVMsS0FBSyxTQUFTLEVBQUU7UUFFM0IsT0FBTyxHQUFHLEVBQUU7WUFDVixJQUFJLENBQUMsU0FBUyxFQUFFO2dCQUNkLE1BQU0sR0FBRyxRQUFRLEVBQUUsQ0FBQztnQkFDcEIsU0FBUyxHQUFHLElBQUksQ0FBQzthQUNsQjtZQUNELE9BQU8sTUFBTSxDQUFDO1FBQ2hCLENBQUMsQ0FBQztLQUNIO0lBRUQsSUFBSSxVQUFVLEdBQUcsS0FBSyxDQUFDO0lBRXZCLE9BQU8sS0FBSyxJQUFJLEVBQUU7UUFDaEIsSUFBSSxDQUFDLFNBQVMsRUFBRTtZQUNkLE1BQU0sR0FBRyxRQUFRLEVBQUUsQ0FBQztZQUNwQixTQUFTLEdBQUcsSUFBSSxDQUFDO1NBQ2xCO1FBQ0QsSUFBSSxVQUFVLEVBQUU7WUFDZCxPQUFPLE1BQU0sQ0FBQztTQUNmO1FBRUQsTUFBTSxRQUFRLEdBQUcsTUFBTSxNQUFNLENBQUM7UUFDOUIsSUFBSSxlQUFlLElBQUksQ0FBQyxlQUFlLENBQUMsUUFBUSxDQUFDLEVBQUU7WUFDakQsVUFBVSxHQUFHLElBQUksQ0FBQztZQUNsQixPQUFPLFFBQVEsQ0FBQztTQUNqQjtRQUNELElBQUksU0FBUyxDQUFDLFFBQVEsQ0FBQyxFQUFFO1lBQ3ZCLE9BQU8sQ0FBQyxNQUFNLEdBQUcsUUFBUSxFQUFFLENBQUMsQ0FBQztTQUM5QjtRQUNELE9BQU8sUUFBUSxDQUFDO0lBQ2xCLENBQUMsQ0FBQztBQUNKLENBQUMsQ0FBQyJ9