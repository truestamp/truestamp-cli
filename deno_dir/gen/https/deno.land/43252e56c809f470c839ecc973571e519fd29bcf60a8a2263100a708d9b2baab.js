const AWS_TEMPLATE = "s3.{region}.amazonaws.com";
const AWS_CN_TEMPLATE = "s3.{region}.amazonaws.com.cn";
const AWS_ISO_TEMPLATE = "s3.{region}.c2s.ic.gov";
const AWS_ISO_B_TEMPLATE = "s3.{region}.sc2s.sgov.gov";
const AWS_US_GOV_TEMPLATE = "s3.{region}.amazonaws.com";
const AWS_REGIONS = new Set([
    "af-south-1",
    "ap-east-1",
    "ap-northeast-1",
    "ap-northeast-2",
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
        case "af-south-1":
            regionInfo = {
                hostname: "s3.af-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-east-1":
            regionInfo = {
                hostname: "s3.ap-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-1":
            regionInfo = {
                hostname: "s3.ap-northeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-northeast-2":
            regionInfo = {
                hostname: "s3.ap-northeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-south-1":
            regionInfo = {
                hostname: "s3.ap-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-southeast-1":
            regionInfo = {
                hostname: "s3.ap-southeast-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "ap-southeast-2":
            regionInfo = {
                hostname: "s3.ap-southeast-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "aws-global":
            regionInfo = {
                hostname: "s3.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "ca-central-1":
            regionInfo = {
                hostname: "s3.ca-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "cn-north-1":
            regionInfo = {
                hostname: "s3.cn-north-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "cn-northwest-1":
            regionInfo = {
                hostname: "s3.cn-northwest-1.amazonaws.com.cn",
                partition: "aws-cn",
            };
            break;
        case "eu-central-1":
            regionInfo = {
                hostname: "s3.eu-central-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-north-1":
            regionInfo = {
                hostname: "s3.eu-north-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-south-1":
            regionInfo = {
                hostname: "s3.eu-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-1":
            regionInfo = {
                hostname: "s3.eu-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-2":
            regionInfo = {
                hostname: "s3.eu-west-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "eu-west-3":
            regionInfo = {
                hostname: "s3.eu-west-3.amazonaws.com",
                partition: "aws",
            };
            break;
        case "fips-us-gov-west-1":
            regionInfo = {
                hostname: "s3-fips.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
                signingRegion: "us-gov-west-1",
            };
            break;
        case "me-south-1":
            regionInfo = {
                hostname: "s3.me-south-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "s3-external-1":
            regionInfo = {
                hostname: "s3-external-1.amazonaws.com",
                partition: "aws",
                signingRegion: "us-east-1",
            };
            break;
        case "sa-east-1":
            regionInfo = {
                hostname: "s3.sa-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-1":
            regionInfo = {
                hostname: "s3.us-east-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-east-2":
            regionInfo = {
                hostname: "s3.us-east-2.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-gov-east-1":
            regionInfo = {
                hostname: "s3.us-gov-east-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "us-gov-west-1":
            regionInfo = {
                hostname: "s3.us-gov-west-1.amazonaws.com",
                partition: "aws-us-gov",
            };
            break;
        case "us-iso-east-1":
            regionInfo = {
                hostname: "s3.us-iso-east-1.c2s.ic.gov",
                partition: "aws-iso",
            };
            break;
        case "us-isob-east-1":
            regionInfo = {
                hostname: "s3.us-isob-east-1.sc2s.sgov.gov",
                partition: "aws-iso-b",
            };
            break;
        case "us-west-1":
            regionInfo = {
                hostname: "s3.us-west-1.amazonaws.com",
                partition: "aws",
            };
            break;
        case "us-west-2":
            regionInfo = {
                hostname: "s3.us-west-2.amazonaws.com",
                partition: "aws",
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
    return Promise.resolve({ signingService: "s3", ...regionInfo });
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZW5kcG9pbnRzLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiZW5kcG9pbnRzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUdBLE1BQU0sWUFBWSxHQUFHLDJCQUEyQixDQUFDO0FBQ2pELE1BQU0sZUFBZSxHQUFHLDhCQUE4QixDQUFDO0FBQ3ZELE1BQU0sZ0JBQWdCLEdBQUcsd0JBQXdCLENBQUM7QUFDbEQsTUFBTSxrQkFBa0IsR0FBRywyQkFBMkIsQ0FBQztBQUN2RCxNQUFNLG1CQUFtQixHQUFHLDJCQUEyQixDQUFDO0FBR3hELE1BQU0sV0FBVyxHQUFHLElBQUksR0FBRyxDQUFDO0lBQzFCLFlBQVk7SUFDWixXQUFXO0lBQ1gsZ0JBQWdCO0lBQ2hCLGdCQUFnQjtJQUNoQixZQUFZO0lBQ1osZ0JBQWdCO0lBQ2hCLGdCQUFnQjtJQUNoQixjQUFjO0lBQ2QsY0FBYztJQUNkLFlBQVk7SUFDWixZQUFZO0lBQ1osV0FBVztJQUNYLFdBQVc7SUFDWCxXQUFXO0lBQ1gsWUFBWTtJQUNaLFdBQVc7SUFDWCxXQUFXO0lBQ1gsV0FBVztJQUNYLFdBQVc7SUFDWCxXQUFXO0NBQ1osQ0FBQyxDQUFDO0FBQ0gsTUFBTSxjQUFjLEdBQUcsSUFBSSxHQUFHLENBQUMsQ0FBQyxZQUFZLEVBQUUsZ0JBQWdCLENBQUMsQ0FBQyxDQUFDO0FBQ2pFLE1BQU0sZUFBZSxHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsZUFBZSxDQUFDLENBQUMsQ0FBQztBQUNuRCxNQUFNLGlCQUFpQixHQUFHLElBQUksR0FBRyxDQUFDLENBQUMsZ0JBQWdCLENBQUMsQ0FBQyxDQUFDO0FBQ3RELE1BQU0sa0JBQWtCLEdBQUcsSUFBSSxHQUFHLENBQUMsQ0FBQyxlQUFlLEVBQUUsZUFBZSxDQUFDLENBQUMsQ0FBQztBQUV2RSxNQUFNLENBQUMsTUFBTSx5QkFBeUIsR0FBdUIsQ0FBQyxNQUFjLEVBQUUsT0FBYSxFQUFFLEVBQUU7SUFDN0YsSUFBSSxVQUFVLEdBQTJCLFNBQVMsQ0FB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