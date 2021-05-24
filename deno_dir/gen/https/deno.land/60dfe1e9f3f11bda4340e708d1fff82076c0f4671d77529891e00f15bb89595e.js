export function requestTimeout(timeoutInMs = 0) {
    return new Promise((resolve, reject) => {
        if (timeoutInMs) {
            setTimeout(() => {
                const timeoutError = new Error(`Request did not complete within ${timeoutInMs} ms`);
                timeoutError.name = "TimeoutError";
                reject(timeoutError);
            }, timeoutInMs);
        }
    });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicmVxdWVzdC10aW1lb3V0LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsicmVxdWVzdC10aW1lb3V0LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE1BQU0sVUFBVSxjQUFjLENBQUMsV0FBVyxHQUFHLENBQUM7SUFDNUMsT0FBTyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxNQUFNLEVBQUUsRUFBRTtRQUNyQyxJQUFJLFdBQVcsRUFBRTtZQUNmLFVBQVUsQ0FBQyxHQUFHLEVBQUU7Z0JBQ2QsTUFBTSxZQUFZLEdBQUcsSUFBSSxLQUFLLENBQUMsbUNBQW1DLFdBQVcsS0FBSyxDQUFDLENBQUM7Z0JBQ3BGLFlBQVksQ0FBQyxJQUFJLEdBQUcsY0FBYyxDQUFDO2dCQUNuQyxNQUFNLENBQUMsWUFBWSxDQUFDLENBQUM7WUFDdkIsQ0FBQyxFQUFFLFdBQVcsQ0FBQyxDQUFDO1NBQ2pCO0lBQ0gsQ0FBQyxDQUFDLENBQUM7QUFDTCxDQUFDIn0=