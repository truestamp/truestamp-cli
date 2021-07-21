import { Buffer } from "../buffer.ts";
export const MAX_RANDOM_VALUES = 65536;
export const MAX_SIZE = 4294967295;
function generateRandomBytes(size) {
    if (size > MAX_SIZE) {
        throw new RangeError(`The value of "size" is out of range. It must be >= 0 && <= ${MAX_SIZE}. Received ${size}`);
    }
    const bytes = Buffer.allocUnsafe(size);
    if (size > MAX_RANDOM_VALUES) {
        for (let generated = 0; generated < size; generated += MAX_RANDOM_VALUES) {
            crypto.getRandomValues(bytes.slice(generated, generated + MAX_RANDOM_VALUES));
        }
    }
    else {
        crypto.getRandomValues(bytes);
    }
    return bytes;
}
export default function randomBytes(size, cb) {
    if (typeof cb === "function") {
        let err = null, bytes;
        try {
            bytes = generateRandomBytes(size);
        }
        catch (e) {
            if (e instanceof RangeError &&
                e.message.includes('The value of "size" is out of range')) {
                throw e;
            }
            else {
                err = e;
            }
        }
        setTimeout(() => {
            if (err) {
                cb(err);
            }
            else {
                cb(null, bytes);
            }
        }, 0);
    }
    else {
        return generateRandomBytes(size);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicmFuZG9tQnl0ZXMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJyYW5kb21CeXRlcy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsTUFBTSxFQUFFLE1BQU0sY0FBYyxDQUFDO0FBRXRDLE1BQU0sQ0FBQyxNQUFNLGlCQUFpQixHQUFHLEtBQUssQ0FBQztBQUN2QyxNQUFNLENBQUMsTUFBTSxRQUFRLEdBQUcsVUFBVSxDQUFDO0FBRW5DLFNBQVMsbUJBQW1CLENBQUMsSUFBWTtJQUN2QyxJQUFJLElBQUksR0FBRyxRQUFRLEVBQUU7UUFDbkIsTUFBTSxJQUFJLFVBQVUsQ0FDbEIsOERBQThELFFBQVEsY0FBYyxJQUFJLEVBQUUsQ0FDM0YsQ0FBQztLQUNIO0lBRUQsTUFBTSxLQUFLLEdBQUcsTUFBTSxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUd2QyxJQUFJLElBQUksR0FBRyxpQkFBaUIsRUFBRTtRQUM1QixLQUFLLElBQUksU0FBUyxHQUFHLENBQUMsRUFBRSxTQUFTLEdBQUcsSUFBSSxFQUFFLFNBQVMsSUFBSSxpQkFBaUIsRUFBRTtZQUN4RSxNQUFNLENBQUMsZUFBZSxDQUNwQixLQUFLLENBQUMsS0FBSyxDQUFDLFNBQVMsRUFBRSxTQUFTLEdBQUcsaUJBQWlCLENBQUMsQ0FDdEQsQ0FBQztTQUNIO0tBQ0Y7U0FBTTtRQUNMLE1BQU0sQ0FBQyxlQUFlLENBQUMsS0FBSyxDQUFDLENBQUM7S0FDL0I7SUFFRCxPQUFPLEtBQUssQ0FBQztBQUNmLENBQUM7QUFVRCxNQUFNLENBQUMsT0FBTyxVQUFVLFdBQVcsQ0FDakMsSUFBWSxFQUNaLEVBQThDO0lBRTlDLElBQUksT0FBTyxFQUFFLEtBQUssVUFBVSxFQUFFO1FBQzVCLElBQUksR0FBRyxHQUFpQixJQUFJLEVBQUUsS0FBYSxDQUFDO1FBQzVDLElBQUk7WUFDRixLQUFLLEdBQUcsbUJBQW1CLENBQUMsSUFBSSxDQUFDLENBQUM7U0FDbkM7UUFBQyxPQUFPLENBQUMsRUFBRTtZQUdWLElBQ0UsQ0FBQyxZQUFZLFVBQVU7Z0JBQ3ZCLENBQUMsQ0FBQyxPQUFPLENBQUMsUUFBUSxDQUFDLHFDQUFxQyxDQUFDLEVBQ3pEO2dCQUNBLE1BQU0sQ0FBQyxDQUFDO2FBQ1Q7aUJBQU07Z0JBQ0wsR0FBRyxHQUFHLENBQUMsQ0FBQzthQUNUO1NBQ0Y7UUFDRCxVQUFVLENBQUMsR0FBRyxFQUFFO1lBQ2QsSUFBSSxHQUFHLEVBQUU7Z0JBQ1AsRUFBRSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQ1Q7aUJBQU07Z0JBQ0wsRUFBRSxDQUFDLElBQUksRUFBRSxLQUFLLENBQUMsQ0FBQzthQUNqQjtRQUNILENBQUMsRUFBRSxDQUFDLENBQUMsQ0FBQztLQUNQO1NBQU07UUFDTCxPQUFPLG1CQUFtQixDQUFDLElBQUksQ0FBQyxDQUFDO0tBQ2xDO0FBQ0gsQ0FBQyJ9