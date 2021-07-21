export function getChunkedStream(source) {
    let currentMessageTotalLength = 0;
    let currentMessagePendingLength = 0;
    let currentMessage = null;
    let messageLengthBuffer = null;
    const allocateMessage = (size) => {
        if (typeof size !== "number") {
            throw new Error("Attempted to allocate an event message where size was not a number: " + size);
        }
        currentMessageTotalLength = size;
        currentMessagePendingLength = 4;
        currentMessage = new Uint8Array(size);
        const currentMessageView = new DataView(currentMessage.buffer);
        currentMessageView.setUint32(0, size, false);
    };
    const iterator = async function* () {
        const sourceIterator = source[Symbol.asyncIterator]();
        while (true) {
            const { value, done } = await sourceIterator.next();
            if (done) {
                if (!currentMessageTotalLength) {
                    return;
                }
                else if (currentMessageTotalLength === currentMessagePendingLength) {
                    yield currentMessage;
                }
                else {
                    throw new Error("Truncated event message received.");
                }
                return;
            }
            const chunkLength = value.length;
            let currentOffset = 0;
            while (currentOffset < chunkLength) {
                if (!currentMessage) {
                    const bytesRemaining = chunkLength - currentOffset;
                    if (!messageLengthBuffer) {
                        messageLengthBuffer = new Uint8Array(4);
                    }
                    const numBytesForTotal = Math.min(4 - currentMessagePendingLength, bytesRemaining);
                    messageLengthBuffer.set(value.slice(currentOffset, currentOffset + numBytesForTotal), currentMessagePendingLength);
                    currentMessagePendingLength += numBytesForTotal;
                    currentOffset += numBytesForTotal;
                    if (currentMessagePendingLength < 4) {
                        break;
                    }
                    allocateMessage(new DataView(messageLengthBuffer.buffer).getUint32(0, false));
                    messageLengthBuffer = null;
                }
                const numBytesToWrite = Math.min(currentMessageTotalLength - currentMessagePendingLength, chunkLength - currentOffset);
                currentMessage.set(value.slice(currentOffset, currentOffset + numBytesToWrite), currentMessagePendingLength);
                currentMessagePendingLength += numBytesToWrite;
                currentOffset += numBytesToWrite;
                if (currentMessageTotalLength && currentMessageTotalLength === currentMessagePendingLength) {
                    yield currentMessage;
                    currentMessage = null;
                    currentMessageTotalLength = 0;
                    currentMessagePendingLength = 0;
                }
            }
        }
    };
    return {
        [Symbol.asyncIterator]: iterator,
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZ2V0Q2h1bmtlZFN0cmVhbS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbImdldENodW5rZWRTdHJlYW0udHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsTUFBTSxVQUFVLGdCQUFnQixDQUFDLE1BQWlDO0lBQ2hFLElBQUkseUJBQXlCLEdBQUcsQ0FBQyxDQUFDO0lBQ2xDLElBQUksMkJBQTJCLEdBQUcsQ0FBQyxDQUFDO0lBQ3BDLElBQUksY0FBYyxHQUFzQixJQUFJLENBQUM7SUFDN0MsSUFBSSxtQkFBbUIsR0FBc0IsSUFBSSxDQUFDO0lBQ2xELE1BQU0sZUFBZSxHQUFHLENBQUMsSUFBWSxFQUFFLEVBQUU7UUFDdkMsSUFBSSxPQUFPLElBQUksS0FBSyxRQUFRLEVBQUU7WUFDNUIsTUFBTSxJQUFJLEtBQUssQ0FBQyxzRUFBc0UsR0FBRyxJQUFJLENBQUMsQ0FBQztTQUNoRztRQUNELHlCQUF5QixHQUFHLElBQUksQ0FBQztRQUNqQywyQkFBMkIsR0FBRyxDQUFDLENBQUM7UUFDaEMsY0FBYyxHQUFHLElBQUksVUFBVSxDQUFDLElBQUksQ0FBQyxDQUFDO1FBQ3RDLE1BQU0sa0JBQWtCLEdBQUcsSUFBSSxRQUFRLENBQUMsY0FBYyxDQUFDLE1BQU0sQ0FBQyxDQUFDO1FBQy9ELGtCQUFrQixDQUFDLFNBQVMsQ0FBQyxDQUFDLEVBQUUsSUFBSSxFQUFFLEtBQUssQ0FBQyxDQUFDO0lBQy9DLENBQUMsQ0FBQztJQUVGLE1BQU0sUUFBUSxHQUFHLEtBQUssU0FBUyxDQUFDO1FBQzlCLE1BQU0sY0FBYyxHQUFHLE1BQU0sQ0FBQyxNQUFNLENBQUMsYUFBYSxDQUFDLEVBQUUsQ0FBQztRQUN0RCxPQUFPLElBQUksRUFBRTtZQUNYLE1BQU0sRUFBRSxLQUFLLEVBQUUsSUFBSSxFQUFFLEdBQUcsTUFBTSxjQUFjLENBQUMsSUFBSSxFQUFFLENBQUM7WUFDcEQsSUFBSSxJQUFJLEVBQUU7Z0JBQ1IsSUFBSSxDQUFDLHlCQUF5QixFQUFFO29CQUM5QixPQUFPO2lCQUNSO3FCQUFNLElBQUkseUJBQXlCLEtBQUssMkJBQTJCLEVBQUU7b0JBQ3BFLE1BQU0sY0FBNEIsQ0FBQztpQkFDcEM7cUJBQU07b0JBQ0wsTUFBTSxJQUFJLEtBQUssQ0FBQyxtQ0FBbUMsQ0FBQyxDQUFDO2lCQUN0RDtnQkFDRCxPQUFPO2FBQ1I7WUFFRCxNQUFNLFdBQVcsR0FBRyxLQUFLLENBQUMsTUFBTSxDQUFDO1lBQ2pDLElBQUksYUFBYSxHQUFHLENBQUMsQ0FBQztZQUV0QixPQUFPLGFBQWEsR0FBRyxXQUFXLEVBQUU7Z0JBRWxDLElBQUksQ0FBQyxjQUFjLEVBQUU7b0JBRW5CLE1BQU0sY0FBYyxHQUFHLFdBQVcsR0FBRyxhQUFhLENBQUM7b0JBRW5ELElBQUksQ0FBQyxtQkFBbUIsRUFBRTt3QkFDeEIsbUJBQW1CLEdBQUcsSUFBSSxVQUFVLENBQUMsQ0FBQyxDQUFDLENBQUM7cUJBQ3pDO29CQUNELE1BQU0sZ0JBQWdCLEdBQUcsSUFBSSxDQUFDLEdBQUcsQ0FDL0IsQ0FBQyxHQUFHLDJCQUEyQixFQUMvQixjQUFjLENBQ2YsQ0FBQztvQkFFRixtQkFBbUIsQ0FBQyxHQUFHLENBRXJCLEtBQUssQ0FBQyxLQUFLLENBQUMsYUFBYSxFQUFFLGFBQWEsR0FBRyxnQkFBZ0IsQ0FBQyxFQUM1RCwyQkFBMkIsQ0FDNUIsQ0FBQztvQkFFRiwyQkFBMkIsSUFBSSxnQkFBZ0IsQ0FBQztvQkFDaEQsYUFBYSxJQUFJLGdCQUFnQixDQUFDO29CQUVsQyxJQUFJLDJCQUEyQixHQUFHLENBQUMsRUFBRTt3QkFFbkMsTUFBTTtxQkFDUDtvQkFDRCxlQUFlLENBQUMsSUFBSSxRQUFRLENBQUMsbUJBQW1CLENBQUMsTUFBTSxDQUFDLENBQUMsU0FBUyxDQUFDLENBQUMsRUFBRSxLQUFLLENBQUMsQ0FBQyxDQUFDO29CQUM5RSxtQkFBbUIsR0FBRyxJQUFJLENBQUM7aUJBQzVCO2dCQUdELE1BQU0sZUFBZSxHQUFHLElBQUksQ0FBQyxHQUFHLENBQzlCLHlCQUF5QixHQUFHLDJCQUEyQixFQUN2RCxXQUFXLEdBQUcsYUFBYSxDQUM1QixDQUFDO2dCQUNGLGNBQWUsQ0FBQyxHQUFHLENBRWpCLEtBQUssQ0FBQyxLQUFLLENBQUMsYUFBYSxFQUFFLGFBQWEsR0FBRyxlQUFlLENBQUMsRUFDM0QsMkJBQTJCLENBQzVCLENBQUM7Z0JBQ0YsMkJBQTJCLElBQUksZUFBZSxDQUFDO2dCQUMvQyxhQUFhLElBQUksZUFBZSxDQUFDO2dCQUdqQyxJQUFJLHlCQUF5QixJQUFJLHlCQUF5QixLQUFLLDJCQUEyQixFQUFFO29CQUUxRixNQUFNLGNBQTRCLENBQUM7b0JBRW5DLGNBQWMsR0FBRyxJQUFJLENBQUM7b0JBQ3RCLHlCQUF5QixHQUFHLENBQUMsQ0FBQztvQkFDOUIsMkJBQTJCLEdBQUcsQ0FBQyxDQUFDO2lCQUNqQzthQUNGO1NBQ0Y7SUFDSCxDQUFDLENBQUM7SUFFRixPQUFPO1FBQ0wsQ0FBQyxNQUFNLENBQUMsYUFBYSxDQUFDLEVBQUUsUUFBUTtLQUNqQyxDQUFDO0FBQ0osQ0FBQyJ9