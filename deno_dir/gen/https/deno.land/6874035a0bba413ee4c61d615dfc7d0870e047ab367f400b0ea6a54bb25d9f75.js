import { DOT_PATTERN, getArnResources, getPseudoRegion, getSuffix, getSuffixForArnEndpoint, isBucketNameOptions, isDnsCompatibleBucketName, isFipsRegion, validateAccountId, validateArnEndpointOptions, validateDNSHostLabel, validateNoDualstack, validateNoFIPS, validateOutpostService, validatePartition, validateRegion, validateRegionalClient, validateS3Service, validateService, } from "./bucketHostnameUtils.ts";
export const bucketHostname = (options) => {
    const { isCustomEndpoint, baseHostname, dualstackEndpoint, accelerateEndpoint } = options;
    if (isCustomEndpoint) {
        if (dualstackEndpoint)
            throw new Error("Dualstack endpoint is not supported with custom endpoint");
        if (accelerateEndpoint)
            throw new Error("Accelerate endpoint is not supported with custom endpoint");
    }
    return isBucketNameOptions(options)
        ?
            getEndpointFromBucketName({ ...options, isCustomEndpoint })
        :
            getEndpointFromArn({ ...options, isCustomEndpoint });
};
const getEndpointFromArn = (options) => {
    const { isCustomEndpoint, baseHostname, clientRegion } = options;
    const hostnameSuffix = isCustomEndpoint ? baseHostname : getSuffixForArnEndpoint(baseHostname)[1];
    const { pathStyleEndpoint, dualstackEndpoint = false, accelerateEndpoint = false, tlsCompatible = true, useArnRegion, bucketName, clientPartition = "aws", clientSigningRegion = clientRegion, } = options;
    validateArnEndpointOptions({ pathStyleEndpoint, accelerateEndpoint, tlsCompatible });
    const { service, partition, accountId, region, resource } = bucketName;
    validateService(service);
    validatePartition(partition, { clientPartition });
    validateAccountId(accountId);
    validateRegionalClient(clientRegion);
    const { accesspointName, outpostId } = getArnResources(resource);
    const DNSHostLabel = `${accesspointName}-${accountId}`;
    validateDNSHostLabel(DNSHostLabel, { tlsCompatible });
    const endpointRegion = useArnRegion ? region : clientRegion;
    const signingRegion = useArnRegion ? region : clientSigningRegion;
    if (service === "s3-object-lambda") {
        validateRegion(region, { useArnRegion, clientRegion, clientSigningRegion, allowFipsRegion: true });
        validateNoDualstack(dualstackEndpoint);
        return {
            bucketEndpoint: true,
            hostname: `${DNSHostLabel}.${service}${isFipsRegion(clientRegion) ? "-fips" : ""}.${getPseudoRegion(endpointRegion)}.${hostnameSuffix}`,
            signingRegion,
            signingService: service,
        };
    }
    else if (outpostId) {
        validateRegion(region, { useArnRegion, clientRegion, clientSigningRegion });
        validateOutpostService(service);
        validateDNSHostLabel(outpostId, { tlsCompatible });
        validateNoDualstack(dualstackEndpoint);
        validateNoFIPS(endpointRegion);
        const hostnamePrefix = `${DNSHostLabel}.${outpostId}`;
        return {
            bucketEndpoint: true,
            hostname: `${hostnamePrefix}${isCustomEndpoint ? "" : `.s3-outposts.${endpointRegion}`}.${hostnameSuffix}`,
            signingRegion,
            signingService: "s3-outposts",
        };
    }
    validateRegion(region, { useArnRegion, clientRegion, clientSigningRegion, allowFipsRegion: true });
    validateS3Service(service);
    const hostnamePrefix = `${DNSHostLabel}`;
    return {
        bucketEndpoint: true,
        hostname: `${hostnamePrefix}${isCustomEndpoint
            ? ""
            : `.s3-accesspoint${isFipsRegion(clientRegion) ? "-fips" : ""}${dualstackEndpoint ? ".dualstack" : ""}.${getPseudoRegion(endpointRegion)}`}.${hostnameSuffix}`,
        signingRegion,
    };
};
const getEndpointFromBucketName = ({ accelerateEndpoint = false, clientRegion: region, baseHostname, bucketName, dualstackEndpoint = false, pathStyleEndpoint = false, tlsCompatible = true, isCustomEndpoint = false, }) => {
    const [clientRegion, hostnameSuffix] = isCustomEndpoint ? [region, baseHostname] : getSuffix(baseHostname);
    if (pathStyleEndpoint || !isDnsCompatibleBucketName(bucketName) || (tlsCompatible && DOT_PATTERN.test(bucketName))) {
        return {
            bucketEndpoint: false,
            hostname: dualstackEndpoint ? `s3.dualstack.${clientRegion}.${hostnameSuffix}` : baseHostname,
        };
    }
    if (accelerateEndpoint) {
        baseHostname = `s3-accelerate${dualstackEndpoint ? ".dualstack" : ""}.${hostnameSuffix}`;
    }
    else if (dualstackEndpoint) {
        baseHostname = `s3.dualstack.${clientRegion}.${hostnameSuffix}`;
    }
    return {
        bucketEndpoint: true,
        hostname: `${bucketName}.${baseHostname}`,
    };
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYnVja2V0SG9zdG5hbWUuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJidWNrZXRIb3N0bmFtZS50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBR0wsV0FBVyxFQUNYLGVBQWUsRUFDZixlQUFlLEVBQ2YsU0FBUyxFQUNULHVCQUF1QixFQUN2QixtQkFBbUIsRUFDbkIseUJBQXlCLEVBQ3pCLFlBQVksRUFDWixpQkFBaUIsRUFDakIsMEJBQTBCLEVBQzFCLG9CQUFvQixFQUNwQixtQkFBbUIsRUFDbkIsY0FBYyxFQUNkLHNCQUFzQixFQUN0QixpQkFBaUIsRUFDakIsY0FBYyxFQUNkLHNCQUFzQixFQUN0QixpQkFBaUIsRUFDakIsZUFBZSxHQUNoQixNQUFNLDBCQUEwQixDQUFDO0FBU2xDLE1BQU0sQ0FBQyxNQUFNLGNBQWMsR0FBRyxDQUFDLE9BQWlELEVBQWtCLEVBQUU7SUFDbEcsTUFBTSxFQUFFLGdCQUFnQixFQUFFLFlBQVksRUFBRSxpQkFBaUIsRUFBRSxrQkFBa0IsRUFBRSxHQUFHLE9BQU8sQ0FBQztJQUUxRixJQUFJLGdCQUFnQixFQUFFO1FBQ3BCLElBQUksaUJBQWlCO1lBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQywwREFBMEQsQ0FBQyxDQUFDO1FBQ25HLElBQUksa0JBQWtCO1lBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQywyREFBMkQsQ0FBQyxDQUFDO0tBQ3RHO0lBRUQsT0FBTyxtQkFBbUIsQ0FBQyxPQUFPLENBQUM7UUFDakMsQ0FBQztZQUNDLHlCQUF5QixDQUFDLEVBQUUsR0FBRyxPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsQ0FBQztRQUM3RCxDQUFDO1lBQ0Msa0JBQWtCLENBQUMsRUFBRSxHQUFHLE9BQU8sRUFBRSxnQkFBZ0IsRUFBRSxDQUFDLENBQUM7QUFDM0QsQ0FBQyxDQUFDO0FBRUYsTUFBTSxrQkFBa0IsR0FBRyxDQUFDLE9BQTBELEVBQWtCLEVBQUU7SUFDeEcsTUFBTSxFQUFFLGdCQUFnQixFQUFF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