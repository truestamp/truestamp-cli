import * as log from "https://deno.land/std/log/mod.ts";
export function sleep(seconds) {
    return new Promise((resolve) => setTimeout(resolve, seconds * 1000));
}
export function sleepRandomAmountOfSeconds(minimumSeconds, maximumSeconds, sleepLog = false) {
    const secondsOfSleep = getRandomArbitrary(minimumSeconds, maximumSeconds);
    if (sleepLog) {
        log.info(`I will sleep for ${secondsOfSleep} seconds.`);
    }
    return new Promise((resolve) => setTimeout(resolve, secondsOfSleep * 1000));
}
function getRandomArbitrary(min, max) {
    return Math.random() * (max - min) + min;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2xlZXAuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJzbGVlcC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEtBQUssR0FBRyxNQUFNLGtDQUFrQyxDQUFDO0FBRXhELE1BQU0sVUFBVSxLQUFLLENBQUMsT0FBZTtJQUNqQyxPQUFPLElBQUksT0FBTyxDQUFDLENBQUMsT0FBTyxFQUFFLEVBQUUsQ0FBQyxVQUFVLENBQUMsT0FBTyxFQUFFLE9BQU8sR0FBRyxJQUFJLENBQUMsQ0FBQyxDQUFBO0FBQ3hFLENBQUM7QUFFRCxNQUFNLFVBQVUsMEJBQTBCLENBQUMsY0FBc0IsRUFBRSxjQUFzQixFQUFFLFdBQW9CLEtBQUs7SUFDaEgsTUFBTSxjQUFjLEdBQUcsa0JBQWtCLENBQUMsY0FBYyxFQUFFLGNBQWMsQ0FBQyxDQUFBO0lBQ3pFLElBQUksUUFBUSxFQUFDO1FBQ1QsR0FBRyxDQUFDLElBQUksQ0FBQyxvQkFBb0IsY0FBYyxXQUFXLENBQUMsQ0FBQTtLQUMxRDtJQUNELE9BQU8sSUFBSSxPQUFPLENBQUMsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUFDLFVBQVUsQ0FBQyxPQUFPLEVBQUUsY0FBYyxHQUFHLElBQUksQ0FBQyxDQUFDLENBQUE7QUFDL0UsQ0FBQztBQUVELFNBQVMsa0JBQWtCLENBQUMsR0FBVyxFQUFFLEdBQVc7SUFDaEQsT0FBTyxJQUFJLENBQUMsTUFBTSxFQUFFLEdBQUcsQ0FBQyxHQUFHLEdBQUcsR0FBRyxDQUFDLEdBQUcsR0FBRyxDQUFBO0FBQzVDLENBQUMifQ==