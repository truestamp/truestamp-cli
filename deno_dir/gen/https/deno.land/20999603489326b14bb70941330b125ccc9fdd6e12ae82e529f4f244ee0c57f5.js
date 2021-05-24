import pl from "./pipeline.ts";
import eos from "./end_of_stream.ts";
export function pipeline(...streams) {
    return new Promise((resolve, reject) => {
        pl(...streams, (err, value) => {
            if (err) {
                reject(err);
            }
            else {
                resolve(value);
            }
        });
    });
}
export function finished(stream, opts) {
    return new Promise((resolve, reject) => {
        eos(stream, opts || null, (err) => {
            if (err) {
                reject(err);
            }
            else {
                resolve();
            }
        });
    });
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicHJvbWlzZXMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJwcm9taXNlcy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLEVBQUUsTUFBTSxlQUFlLENBQUM7QUFFL0IsT0FBTyxHQUFHLE1BQU0sb0JBQW9CLENBQUM7QUFNckMsTUFBTSxVQUFVLFFBQVEsQ0FBQyxHQUFHLE9BQTBCO0lBQ3BELE9BQU8sSUFBSSxPQUFPLENBQUMsQ0FBQyxPQUFPLEVBQUUsTUFBTSxFQUFFLEVBQUU7UUFDckMsRUFBRSxDQUNBLEdBQUcsT0FBTyxFQUNWLENBQUMsR0FBRyxFQUFFLEtBQUssRUFBRSxFQUFFO1lBQ2IsSUFBSSxHQUFHLEVBQUU7Z0JBQ1AsTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQ2I7aUJBQU07Z0JBQ0wsT0FBTyxDQUFDLEtBQUssQ0FBQyxDQUFDO2FBQ2hCO1FBQ0gsQ0FBQyxDQUNGLENBQUM7SUFDSixDQUFDLENBQUMsQ0FBQztBQUNMLENBQUM7QUFFRCxNQUFNLFVBQVUsUUFBUSxDQUN0QixNQUF1QixFQUN2QixJQUFzQjtJQUV0QixPQUFPLElBQUksT0FBTyxDQUFPLENBQUMsT0FBTyxFQUFFLE1BQU0sRUFBRSxFQUFFO1FBQzNDLEdBQUcsQ0FDRCxNQUFNLEVBQ04sSUFBSSxJQUFJLElBQUksRUFDWixDQUFDLEdBQUcsRUFBRSxFQUFFO1lBQ04sSUFBSSxHQUFHLEVBQUU7Z0JBQ1AsTUFBTSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQ2I7aUJBQU07Z0JBQ0wsT0FBTyxFQUFFLENBQUM7YUFDWDtRQUNILENBQUMsQ0FDRixDQUFDO0lBQ0osQ0FBQyxDQUFDLENBQUM7QUFDTCxDQUFDIn0=