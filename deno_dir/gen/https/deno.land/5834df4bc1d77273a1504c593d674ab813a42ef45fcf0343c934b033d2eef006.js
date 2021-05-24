export function blobReader(blob, onChunk, chunkSize = 1024 * 1024) {
    return new Promise((resolve, reject) => {
        const fileReader = new FileReader();
        fileReader.addEventListener("error", reject);
        fileReader.addEventListener("abort", reject);
        const size = blob.size;
        let totalBytesRead = 0;
        function read() {
            if (totalBytesRead >= size) {
                resolve();
                return;
            }
            fileReader.readAsArrayBuffer(blob.slice(totalBytesRead, Math.min(size, totalBytesRead + chunkSize)));
        }
        fileReader.addEventListener("load", (event) => {
            const result = event.target.result;
            onChunk(new Uint8Array(result));
            totalBytesRead += result.byteLength;
            read();
        });
        read();
    });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE1BQU0sVUFBVSxVQUFVLENBQ3hCLElBQVUsRUFDVixPQUFvQyxFQUNwQyxZQUFvQixJQUFJLEdBQUcsSUFBSTtJQUUvQixPQUFPLElBQUksT0FBTyxDQUFDLENBQUMsT0FBTyxFQUFFLE1BQU0sRUFBRSxFQUFFO1FBQ3JDLE1BQU0sVUFBVSxHQUFHLElBQUksVUFBVSxFQUFFLENBQUM7UUFFcEMsVUFBVSxDQUFDLGdCQUFnQixDQUFDLE9BQU8sRUFBRSxNQUFNLENBQUMsQ0FBQztRQUM3QyxVQUFVLENBQUMsZ0JBQWdCLENBQUMsT0FBTyxFQUFFLE1BQU0sQ0FBQyxDQUFDO1FBRTdDLE1BQU0sSUFBSSxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUM7UUFDdkIsSUFBSSxjQUFjLEdBQUcsQ0FBQyxDQUFDO1FBRXZCLFNBQVMsSUFBSTtZQUNYLElBQUksY0FBYyxJQUFJLElBQUksRUFBRTtnQkFDMUIsT0FBTyxFQUFFLENBQUM7Z0JBQ1YsT0FBTzthQUNSO1lBQ0QsVUFBVSxDQUFDLGlCQUFpQixDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsY0FBYyxFQUFFLElBQUksQ0FBQyxHQUFHLENBQUMsSUFBSSxFQUFFLGNBQWMsR0FBRyxTQUFTLENBQUMsQ0FBQyxDQUFDLENBQUM7UUFDdkcsQ0FBQztRQUVELFVBQVUsQ0FBQyxnQkFBZ0IsQ0FBQyxNQUFNLEVBQUUsQ0FBQyxLQUFLLEVBQUUsRUFBRTtZQUM1QyxNQUFNLE1BQU0sR0FBaUIsS0FBSyxDQUFDLE1BQWMsQ0FBQyxNQUFNLENBQUM7WUFDekQsT0FBTyxDQUFDLElBQUksVUFBVSxDQUFDLE1BQU0sQ0FBQyxDQUFDLENBQUM7WUFDaEMsY0FBYyxJQUFJLE1BQU0sQ0FBQyxVQUFVLENBQUM7WUFFcEMsSUFBSSxFQUFFLENBQUM7UUFDVCxDQUFDLENBQUMsQ0FBQztRQUdILElBQUksRUFBRSxDQUFDO0lBQ1QsQ0FBQyxDQUFDLENBQUM7QUFDTCxDQUFDIn0=