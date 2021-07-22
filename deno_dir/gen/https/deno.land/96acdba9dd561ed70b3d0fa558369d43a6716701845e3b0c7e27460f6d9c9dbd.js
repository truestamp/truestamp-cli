const AWS_TEMPLATE = "portal.sso.{region}.amazonaws.com";
const AWS_CN_TEMPLATE = "portal.sso.{region}.amazonaws.com.cn";
const AWS_ISO_TEMPLATE = "portal.sso.{region}.c2s.ic.gov";
const AWS_ISO_B_TEMPLATE = "portal.sso.{region}.sc2s.sgov.gov";
const AWS_US_GOV_TEMPLATE = "portal.sso.{region}.amazonaws.com";
const AWS_REGIONS = new Set([
    "af-south-1",
    "ap-east-1",
    "ap-northeast-1",
    "ap-northeast-2",
    "ap-northeast-3",
    "ap-south-1",
    "ap-southeast-1",
    "ap-southeast-2",
    "ca-central-1",
    "eu-central-1",
    "eu-north-1",
    "eu-south-1",
    "eu-west-1",
    "eu-west-2",
    "eu-west-3",
    "me-south-1",
    "sa-east-1",
    "us-east-1",
    "us-east-2",
    "us-west-1",
    "us-west-2",
]);
const AWS_CN_REGIONS = new Set(["cn-north-1", "cn-northwest-1"]);
const AWS_ISO_REGIONS = new Set(["us-iso-east-1"]);
const AWS_ISO_B_REGIONS = new Set(["us-isob-east-1"]);
const AWS_US_GOV_REGIONS = new Set(["us-gov-east-1", "us-gov-west-1"]);
export const defaultRegionInfoProvider = (region, options) => {
    let regionInfo = undefined;
    switch (region) {
        case "ap-southeast-1":
            regionInfo = {
                hostname: "portal.sso.ap-southeast-1.amazonaws.com",
                partition: "aws",
                signingRegion: "ap-southeast-1",
            };
            break;
        case "ap-southeast-2":
            regionInfo = {
                hostname: "portal.sso.ap-southeast-2.amazonaws.com",
                partition: "aws",
                signingRegion: "ap-southeast-2",
            };
            break;
        case "ca-central-1":
            regionInfo = {
                hostname: "portal.sso.ca-central-1.amazonaws.com",
                partition: "aws",
                signingRegion: "ca-central-1",
            };
            break;
        case "eu-central-1":
            regionInfo = {
                hostname: "portal.sso.eu-central-1.amazonaws.com",
                partition: "aws",
                signingRegion: "eu-central-1",
            };
            break;
        case "eu-west-1":
            regionInfo = {
                hostname: "portal.sso.eu-west-1.amazonaws.com",
                partition: "aws",
                signingRegion: "eu-west-1",
            };
            break;
        case "eu-west-2":
            regionInfo = {
                hostname: "portal.sso.eu-west-2.amazonaws.com",
                partition: "aws",
                signingRegion: "eu-west-2",
            };
            break;
        case "us-east-1":
            regionInfo = {
                hostname: "portal.sso.us-east-1.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "us-east-2":
            regionInfo = {
                hostname: "portal.sso.us-east-2.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-2",
            };
            break;
        case "us-west-2":
            regionInfo = {
                hostname: "portal.sso.us-west-2.amazonaws.com",
                partition: "aws",
                signingRegion: "us-west-2",
            };
            break;
        default:
            if (AWS_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_TEMPLATE.replace("{region}", region),
                    partition: "aws",
                };
            }
            if (AWS_CN_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_CN_TEMPLATE.replace("{region}", region),
                    partition: "aws-cn",
                };
            }
            if (AWS_ISO_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_ISO_TEMPLATE.replace("{region}", region),
                    partition: "aws-iso",
                };
            }
            if (AWS_ISO_B_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_ISO_B_TEMPLATE.replace("{region}", region),
                    partition: "aws-iso-b",
                };
            }
            if (AWS_US_GOV_REGIONS.has(region)) {
                regionInfo = {
                    hostname: AWS_US_GOV_TEMPLATE.replace("{region}", region),
                    partition: "aws-us-gov",
                };
            }
            if (regionInfo === undefined) {
                regionInfo = {
                    hostname: AWS_TEMPLATE.replace("{region}", region),
                    partition: "aws",
                };
            }
    }
    return Promise.resolve({ signingService: "awsssoportal", ...regionInfo });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZW5kcG9pbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUdBLE1BQU0sWUFBWSxHQUFHLG1DQUFtQyxDQUFDO0FBQ3pELE1BQU0sZUFBZSxHQUFHLHNDQUFzQyxDQUFDO0FBQy9ELE1BQU0sZ0JBQWdCLEdBQUcsZ0NBQWdDLENBQUM7QUFDMUQsTUFBTSxrQkFBa0IsR0FBRyxtQ0FBbUMsQ0FBQztBQUMvRCxNQUFNLG1CQUFtQixHQUFHLG1DQUFtQyxDQUFDO0FBR2hFLE1BQU0sV0FBVyxHQUFHLElBQUksR0FBRyxDQUFDO0lBQzFCLFlBQVk7SUFDWixXQUFXO0lBQ1gsZ0JBQWdCO0lBQ2hCLGdCQUFnQjtJQUNoQixnQkFBZ0I7SUFDaEIsWUFBWTtJQUNaLGdCQUFnQjtJQUNoQixnQkFBZ0I7SUFDaEIsY0FBYztJQUNkLGNBQWM7SUFDZCxZQUFZO0lBQ1osWUFBWTtJQUNaLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztJQUNYLFlBQVk7SUFDWixXQUFXO0lBQ1gsV0FBVztJQUNYLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztDQUNaLENBQUMsQ0FBQztBQUNILE1BQU0sY0FBYyxHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsWUFBWSxFQUFFLGdCQUFnQixDQUFDLENBQUMsQ0FBQztBQUNqRSxNQUFNLGVBQWUsR0FBRyxJQUFJLEdBQUcsQ0FBQyxDQUFDLGVBQWUsQ0FBQyxDQUFDLENBQUM7QUFDbkQsTUFBTSxpQkFBaUIsR0FBRyxJQUFJLEdBQUcsQ0FBQyxDQUFDLGdCQUFnQixDQUFDLENBQUMsQ0FBQztBQUN0RCxNQUFNLGtCQUFrQixHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsZUFBZSxFQUFFLGVBQWUsQ0FBQyxDQUFDLENBQUM7QUFFdkUsTUFBTSxDQUFDLE1BQU0seUJBQXlCLEdBQXVCLENBQUMsTUFBYyxFQUFFLE9BQWEsRUFBRSxFQUFFO0lBQzdGLElBQUksVUFBVSxH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