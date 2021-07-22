export const validate = (str) => typeof str === "string" && str.indexOf("arn:") === 0 && str.split(":").length >= 6;
export const parse = (arn) => {
    const segments = arn.split(":");
    if (segments.length < 6 || segments[0] !== "arn")
        throw new Error("Malformed ARN");
    const [, partition, service, region, accountId, ...resource] = segments;
    return {
        partition,
        service,
        region,
        accountId,
        resource: resource.join(":"),
    };
};
export const build = (arnObject) => {
    const { partition = "aws", service, region, accountId, resource } = arnObject;
    if ([service, region, accountId, resource].some((segment) => typeof segment !== "string")) {
        throw new Error("Input ARN object is invalid");
    }
    return `arn:${partition}:${service}:${region}:${accountId}:${resource}`;
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibW9kLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQVVBLE1BQU0sQ0FBQyxNQUFNLFFBQVEsR0FBRyxDQUFDLEdBQVEsRUFBVyxFQUFFLENBQzVDLE9BQU8sR0FBRyxLQUFLLFFBQVEsSUFBSSxHQUFHLENBQUMsT0FBTyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUMsSUFBSSxHQUFHLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDLE1BQU0sSUFBSSxDQUFDLENBQUM7QUFLckYsTUFBTSxDQUFDLE1BQU0sS0FBSyxHQUFHLENBQUMsR0FBVyxFQUFPLEVBQUU7SUFDeEMsTUFBTSxRQUFRLEdBQUcsR0FBRyxDQUFDLEtBQUssQ0FBQyxHQUFHLENBQUMsQ0FBQztJQUNoQyxJQUFJLFFBQVEsQ0FBQyxNQUFNLEdBQUcsQ0FBQyxJQUFJLFFBQVEsQ0FBQyxDQUFDLENBQUMsS0FBSyxLQUFLO1FBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQyxlQUFlLENBQUMsQ0FBQztJQUNuRixNQUFNLENBQ0osQUFESyxFQUdMLFNBQVMsRUFDVCxPQUFPLEVBQ1AsTUFBTSxFQUNOLFNBQVMsRUFDVCxHQUFHLFFBQVEsQ0FDWixHQUFHLFFBQVEsQ0FBQztJQUViLE9BQU87UUFDTCxTQUFTO1FBQ1QsT0FBTztRQUNQLE1BQU07UUFDTixTQUFTO1FBQ1QsUUFBUSxFQUFFLFFBQVEsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDO0tBQzdCLENBQUM7QUFDSixDQUFDLENBQUM7QUFPRixNQUFNLENBQUMsTUFBTSxLQUFLLEdBQUcsQ0FBQyxTQUF1QixFQUFVLEVBQUU7SUFDdkQsTUFBTSxFQUFFLFNBQVMsR0FBRyxLQUFLLEVBQUUsT0FBTyxFQUFFLE1BQU0sRUFBRSxTQUFTLEVBQUUsUUFBUSxFQUFFLEdBQUcsU0FBUyxDQUFDO0lBQzlFLElBQUksQ0FBQyxPQUFPLEVBQUUsTUFBTSxFQUFFLFNBQVMsRUFBRSxRQUFRLENBQUMsQ0FBQyxJQUFJLENBQUMsQ0FBQyxPQUFPLEVBQUUsRUFBRSxDQUFDLE9BQU8sT0FBTyxLQUFLLFFBQVEsQ0FBQyxFQUFFO1FBQ3pGLE1BQU0sSUFBSSxLQUFLLENBQUMsNkJBQTZCLENBQUMsQ0FBQztLQUNoRDtJQUNELE9BQU8sT0FBTyxTQUFTLElBQUksT0FBTyxJQUFJLE1BQU0sSUFBSSxTQUFTLElBQUksUUFBUSxFQUFFLENBQUM7QUFDMUUsQ0FBQyxDQUFDIn0=