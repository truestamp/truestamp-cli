export function debounce(fn, wait) {
    let timeout = null;
    let flush = null;
    const debounced = ((...args) => {
        debounced.clear();
        flush = () => {
            debounced.clear();
            fn.call(debounced, ...args);
        };
        timeout = setTimeout(flush, wait);
    });
    debounced.clear = () => {
        if (typeof timeout === "number") {
            clearTimeout(timeout);
            timeout = null;
            flush = null;
        }
    };
    debounced.flush = () => {
        flush?.();
    };
    Object.defineProperty(debounced, "pending", {
        get: () => typeof timeout === "number",
    });
    return debounced;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZGVib3VuY2UuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJkZWJvdW5jZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUF5Q0EsTUFBTSxVQUFVLFFBQVEsQ0FDdEIsRUFBb0QsRUFDcEQsSUFBWTtJQUVaLElBQUksT0FBTyxHQUFrQixJQUFJLENBQUM7SUFDbEMsSUFBSSxLQUFLLEdBQXdCLElBQUksQ0FBQztJQUV0QyxNQUFNLFNBQVMsR0FBeUIsQ0FBQyxDQUFDLEdBQUcsSUFBTyxFQUFRLEVBQUU7UUFDNUQsU0FBUyxDQUFDLEtBQUssRUFBRSxDQUFDO1FBQ2xCLEtBQUssR0FBRyxHQUFTLEVBQUU7WUFDakIsU0FBUyxDQUFDLEtBQUssRUFBRSxDQUFDO1lBQ2xCLEVBQUUsQ0FBQyxJQUFJLENBQUMsU0FBUyxFQUFFLEdBQUcsSUFBSSxDQUFDLENBQUM7UUFDOUIsQ0FBQyxDQUFDO1FBQ0YsT0FBTyxHQUFHLFVBQVUsQ0FBQyxLQUFLLEVBQUUsSUFBSSxDQUFDLENBQUM7SUFDcEMsQ0FBQyxDQUF5QixDQUFDO0lBRTNCLFNBQVMsQ0FBQyxLQUFLLEdBQUcsR0FBUyxFQUFFO1FBQzNCLElBQUksT0FBTyxPQUFPLEtBQUssUUFBUSxFQUFFO1lBQy9CLFlBQVksQ0FBQyxPQUFPLENBQUMsQ0FBQztZQUN0QixPQUFPLEdBQUcsSUFBSSxDQUFDO1lBQ2YsS0FBSyxHQUFHLElBQUksQ0FBQztTQUNkO0lBQ0gsQ0FBQyxDQUFDO0lBRUYsU0FBUyxDQUFDLEtBQUssR0FBRyxHQUFTLEVBQUU7UUFDM0IsS0FBSyxFQUFFLEVBQUUsQ0FBQztJQUNaLENBQUMsQ0FBQztJQUVGLE1BQU0sQ0FBQyxjQUFjLENBQUMsU0FBUyxFQUFFLFNBQVMsRUFBRTtRQUMxQyxHQUFHLEVBQUUsR0FBRyxFQUFFLENBQUMsT0FBTyxPQUFPLEtBQUssUUFBUTtLQUN2QyxDQUFDLENBQUM7SUFFSCxPQUFPLFNBQVMsQ0FBQztBQUNuQixDQUFDIn0=