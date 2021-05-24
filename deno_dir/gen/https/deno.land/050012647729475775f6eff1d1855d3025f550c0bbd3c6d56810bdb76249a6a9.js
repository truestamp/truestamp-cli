import { fromFileUrl } from "../path.ts";
export function copyFile(source, destination, callback) {
    source = source instanceof URL ? fromFileUrl(source) : source;
    Deno.copyFile(source, destination).then(() => callback(null), callback);
}
export function copyFileSync(source, destination) {
    source = source instanceof URL ? fromFileUrl(source) : source;
    Deno.copyFileSync(source, destination);
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiX2ZzX2NvcHkuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJfZnNfY29weS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFFQSxPQUFPLEVBQUUsV0FBVyxFQUFFLE1BQU0sWUFBWSxDQUFDO0FBRXpDLE1BQU0sVUFBVSxRQUFRLENBQ3RCLE1BQW9CLEVBQ3BCLFdBQW1CLEVBQ25CLFFBQTJCO0lBRTNCLE1BQU0sR0FBRyxNQUFNLFlBQVksR0FBRyxDQUFDLENBQUMsQ0FBQyxXQUFXLENBQUMsTUFBTSxDQUFDLENBQUMsQ0FBQyxDQUFDLE1BQU0sQ0FBQztJQUU5RCxJQUFJLENBQUMsUUFBUSxDQUFDLE1BQU0sRUFBRSxXQUFXLENBQUMsQ0FBQyxJQUFJLENBQUMsR0FBRyxFQUFFLENBQUMsUUFBUSxDQUFDLElBQUksQ0FBQyxFQUFFLFFBQVEsQ0FBQyxDQUFDO0FBQzFFLENBQUM7QUFFRCxNQUFNLFVBQVUsWUFBWSxDQUFDLE1BQW9CLEVBQUUsV0FBbUI7SUFDcEUsTUFBTSxHQUFHLE1BQU0sWUFBWSxHQUFHLENBQUMsQ0FBQyxDQUFDLFdBQVcsQ0FBQyxNQUFNLENBQUMsQ0FBQyxDQUFDLENBQUMsTUFBTSxDQUFDO0lBQzlELElBQUksQ0FBQyxZQUFZLENBQUMsTUFBTSxFQUFFLFdBQVcsQ0FBQyxDQUFDO0FBQ3pDLENBQUMifQ==