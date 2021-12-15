export function sleep(seconds) {
    return new Promise((resolve) => setTimeout(resolve, seconds * 1000));
}
export function sleepRandomAmountOfSeconds(minimumSeconds, maximumSeconds) {
    const secondsOfSleep = getRandomArbitrary(minimumSeconds, maximumSeconds);
    return new Promise((resolve) => setTimeout(resolve, secondsOfSleep * 1000));
}
function getRandomArbitrary(min, max) {
    return Math.random() * (max - min) + min;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic2xlZXAuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJzbGVlcC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxNQUFNLFVBQVUsS0FBSyxDQUFDLE9BQWU7SUFDbkMsT0FBTyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxFQUFFLENBQUMsVUFBVSxDQUFDLE9BQU8sRUFBRSxPQUFPLEdBQUcsSUFBSSxDQUFDLENBQUMsQ0FBQztBQUN2RSxDQUFDO0FBRUQsTUFBTSxVQUFVLDBCQUEwQixDQUN4QyxjQUFzQixFQUN0QixjQUFzQjtJQUV0QixNQUFNLGNBQWMsR0FBRyxrQkFBa0IsQ0FBQyxjQUFjLEVBQUUsY0FBYyxDQUFDLENBQUM7SUFDMUUsT0FBTyxJQUFJLE9BQU8sQ0FBQyxDQUFDLE9BQU8sRUFBRSxFQUFFLENBQUMsVUFBVSxDQUFDLE9BQU8sRUFBRSxjQUFjLEdBQUcsSUFBSSxDQUFDLENBQUMsQ0FBQztBQUM5RSxDQUFDO0FBRUQsU0FBUyxrQkFBa0IsQ0FBQyxHQUFXLEVBQUUsR0FBVztJQUNsRCxPQUFPLElBQUksQ0FBQyxNQUFNLEVBQUUsR0FBRyxDQUFDLEdBQUcsR0FBRyxHQUFHLENBQUMsR0FBRyxHQUFHLENBQUM7QUFDM0MsQ0FBQyJ9