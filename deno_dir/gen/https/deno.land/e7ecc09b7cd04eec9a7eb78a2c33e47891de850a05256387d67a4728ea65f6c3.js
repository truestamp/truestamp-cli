import { HttpRequest as __HttpRequest } from "../../protocol-http/mod.ts";
import { expectString as __expectString, extendedEncodeURIComponent as __extendedEncodeURIComponent, getValueFromTextNode as __getValueFromTextNode, } from "../../smithy-client/mod.ts";
import { decodeHTML } from "https://jspm.dev/entities";
import { parse as xmlParse } from "https://jspm.dev/fast-xml-parser";
export const serializeAws_queryAssumeRoleCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryAssumeRoleRequest(input, context),
        Action: "AssumeRole",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryAssumeRoleWithSAMLCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryAssumeRoleWithSAMLRequest(input, context),
        Action: "AssumeRoleWithSAML",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryAssumeRoleWithWebIdentityCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryAssumeRoleWithWebIdentityRequest(input, context),
        Action: "AssumeRoleWithWebIdentity",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryDecodeAuthorizationMessageCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryDecodeAuthorizationMessageRequest(input, context),
        Action: "DecodeAuthorizationMessage",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetAccessKeyInfoCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetAccessKeyInfoRequest(input, context),
        Action: "GetAccessKeyInfo",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetCallerIdentityCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetCallerIdentityRequest(input, context),
        Action: "GetCallerIdentity",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetFederationTokenCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetFederationTokenRequest(input, context),
        Action: "GetFederationToken",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const serializeAws_queryGetSessionTokenCommand = async (input, context) => {
    const headers = {
        "content-type": "application/x-www-form-urlencoded",
    };
    let body;
    body = buildFormUrlencodedString({
        ...serializeAws_queryGetSessionTokenRequest(input, context),
        Action: "GetSessionToken",
        Version: "2011-06-15",
    });
    return buildHttpRpcRequest(context, headers, "/", undefined, body);
};
export const deserializeAws_queryAssumeRoleCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryAssumeRoleCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryAssumeRoleResponse(data.AssumeRoleResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryAssumeRoleCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ExpiredTokenException":
        case "com.amazonaws.sts#ExpiredTokenException":
            response = {
                ...(await deserializeAws_queryExpiredTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryAssumeRoleWithSAMLCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryAssumeRoleWithSAMLCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryAssumeRoleWithSAMLResponse(data.AssumeRoleWithSAMLResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryAssumeRoleWithSAMLCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ExpiredTokenException":
        case "com.amazonaws.sts#ExpiredTokenException":
            response = {
                ...(await deserializeAws_queryExpiredTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "IDPRejectedClaimException":
        case "com.amazonaws.sts#IDPRejectedClaimException":
            response = {
                ...(await deserializeAws_queryIDPRejectedClaimExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "InvalidIdentityTokenException":
        case "com.amazonaws.sts#InvalidIdentityTokenException":
            response = {
                ...(await deserializeAws_queryInvalidIdentityTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryAssumeRoleWithWebIdentityCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryAssumeRoleWithWebIdentityCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryAssumeRoleWithWebIdentityResponse(data.AssumeRoleWithWebIdentityResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryAssumeRoleWithWebIdentityCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "ExpiredTokenException":
        case "com.amazonaws.sts#ExpiredTokenException":
            response = {
                ...(await deserializeAws_queryExpiredTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "IDPCommunicationErrorException":
        case "com.amazonaws.sts#IDPCommunicationErrorException":
            response = {
                ...(await deserializeAws_queryIDPCommunicationErrorExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "IDPRejectedClaimException":
        case "com.amazonaws.sts#IDPRejectedClaimException":
            response = {
                ...(await deserializeAws_queryIDPRejectedClaimExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "InvalidIdentityTokenException":
        case "com.amazonaws.sts#InvalidIdentityTokenException":
            response = {
                ...(await deserializeAws_queryInvalidIdentityTokenExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryDecodeAuthorizationMessageCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryDecodeAuthorizationMessageCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryDecodeAuthorizationMessageResponse(data.DecodeAuthorizationMessageResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryDecodeAuthorizationMessageCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidAuthorizationMessageException":
        case "com.amazonaws.sts#InvalidAuthorizationMessageException":
            response = {
                ...(await deserializeAws_queryInvalidAuthorizationMessageExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetAccessKeyInfoCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetAccessKeyInfoCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetAccessKeyInfoResponse(data.GetAccessKeyInfoResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetAccessKeyInfoCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetCallerIdentityCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetCallerIdentityCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetCallerIdentityResponse(data.GetCallerIdentityResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetCallerIdentityCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetFederationTokenCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetFederationTokenCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetFederationTokenResponse(data.GetFederationTokenResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetFederationTokenCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "MalformedPolicyDocumentException":
        case "com.amazonaws.sts#MalformedPolicyDocumentException":
            response = {
                ...(await deserializeAws_queryMalformedPolicyDocumentExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "PackedPolicyTooLargeException":
        case "com.amazonaws.sts#PackedPolicyTooLargeException":
            response = {
                ...(await deserializeAws_queryPackedPolicyTooLargeExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_queryGetSessionTokenCommand = async (output, context) => {
    if (output.statusCode >= 300) {
        return deserializeAws_queryGetSessionTokenCommandError(output, context);
    }
    const data = await parseBody(output.body, context);
    let contents = {};
    contents = deserializeAws_queryGetSessionTokenResponse(data.GetSessionTokenResult, context);
    const response = {
        $metadata: deserializeMetadata(output),
        ...contents,
    };
    return Promise.resolve(response);
};
const deserializeAws_queryGetSessionTokenCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadQueryErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "RegionDisabledException":
        case "com.amazonaws.sts#RegionDisabledException":
            response = {
                ...(await deserializeAws_queryRegionDisabledExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.Error.code || parsedBody.Error.Code || errorCode;
            response = {
                ...parsedBody.Error,
                name: `${errorCode}`,
                message: parsedBody.Error.message || parsedBody.Error.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
const deserializeAws_queryExpiredTokenExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryExpiredTokenException(body.Error, context);
    const contents = {
        name: "ExpiredTokenException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryIDPCommunicationErrorExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryIDPCommunicationErrorException(body.Error, context);
    const contents = {
        name: "IDPCommunicationErrorException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryIDPRejectedClaimExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryIDPRejectedClaimException(body.Error, context);
    const contents = {
        name: "IDPRejectedClaimException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryInvalidAuthorizationMessageExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryInvalidAuthorizationMessageException(body.Error, context);
    const contents = {
        name: "InvalidAuthorizationMessageException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryInvalidIdentityTokenExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryInvalidIdentityTokenException(body.Error, context);
    const contents = {
        name: "InvalidIdentityTokenException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryMalformedPolicyDocumentExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryMalformedPolicyDocumentException(body.Error, context);
    const contents = {
        name: "MalformedPolicyDocumentException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryPackedPolicyTooLargeExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryPackedPolicyTooLargeException(body.Error, context);
    const contents = {
        name: "PackedPolicyTooLargeException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const deserializeAws_queryRegionDisabledExceptionResponse = async (parsedOutput, context) => {
    const body = parsedOutput.body;
    const deserialized = deserializeAws_queryRegionDisabledException(body.Error, context);
    const contents = {
        name: "RegionDisabledException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        ...deserialized,
    };
    return contents;
};
const serializeAws_queryAssumeRoleRequest = (input, context) => {
    const entries = {};
    if (input.RoleArn !== undefined && input.RoleArn !== null) {
        entries["RoleArn"] = input.RoleArn;
    }
    if (input.RoleSessionName !== undefined && input.RoleSessionName !== null) {
        entries["RoleSessionName"] = input.RoleSessionName;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const memberEntries = serializeAws_querytagListType(input.Tags, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `Tags.${key}`;
            entries[loc] = value;
        });
    }
    if (input.TransitiveTagKeys !== undefined && input.TransitiveTagKeys !== null) {
        const memberEntries = serializeAws_querytagKeyListType(input.TransitiveTagKeys, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `TransitiveTagKeys.${key}`;
            entries[loc] = value;
        });
    }
    if (input.ExternalId !== undefined && input.ExternalId !== null) {
        entries["ExternalId"] = input.ExternalId;
    }
    if (input.SerialNumber !== undefined && input.SerialNumber !== null) {
        entries["SerialNumber"] = input.SerialNumber;
    }
    if (input.TokenCode !== undefined && input.TokenCode !== null) {
        entries["TokenCode"] = input.TokenCode;
    }
    if (input.SourceIdentity !== undefined && input.SourceIdentity !== null) {
        entries["SourceIdentity"] = input.SourceIdentity;
    }
    return entries;
};
const serializeAws_queryAssumeRoleWithSAMLRequest = (input, context) => {
    const entries = {};
    if (input.RoleArn !== undefined && input.RoleArn !== null) {
        entries["RoleArn"] = input.RoleArn;
    }
    if (input.PrincipalArn !== undefined && input.PrincipalArn !== null) {
        entries["PrincipalArn"] = input.PrincipalArn;
    }
    if (input.SAMLAssertion !== undefined && input.SAMLAssertion !== null) {
        entries["SAMLAssertion"] = input.SAMLAssertion;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    return entries;
};
const serializeAws_queryAssumeRoleWithWebIdentityRequest = (input, context) => {
    const entries = {};
    if (input.RoleArn !== undefined && input.RoleArn !== null) {
        entries["RoleArn"] = input.RoleArn;
    }
    if (input.RoleSessionName !== undefined && input.RoleSessionName !== null) {
        entries["RoleSessionName"] = input.RoleSessionName;
    }
    if (input.WebIdentityToken !== undefined && input.WebIdentityToken !== null) {
        entries["WebIdentityToken"] = input.WebIdentityToken;
    }
    if (input.ProviderId !== undefined && input.ProviderId !== null) {
        entries["ProviderId"] = input.ProviderId;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    return entries;
};
const serializeAws_queryDecodeAuthorizationMessageRequest = (input, context) => {
    const entries = {};
    if (input.EncodedMessage !== undefined && input.EncodedMessage !== null) {
        entries["EncodedMessage"] = input.EncodedMessage;
    }
    return entries;
};
const serializeAws_queryGetAccessKeyInfoRequest = (input, context) => {
    const entries = {};
    if (input.AccessKeyId !== undefined && input.AccessKeyId !== null) {
        entries["AccessKeyId"] = input.AccessKeyId;
    }
    return entries;
};
const serializeAws_queryGetCallerIdentityRequest = (input, context) => {
    const entries = {};
    return entries;
};
const serializeAws_queryGetFederationTokenRequest = (input, context) => {
    const entries = {};
    if (input.Name !== undefined && input.Name !== null) {
        entries["Name"] = input.Name;
    }
    if (input.Policy !== undefined && input.Policy !== null) {
        entries["Policy"] = input.Policy;
    }
    if (input.PolicyArns !== undefined && input.PolicyArns !== null) {
        const memberEntries = serializeAws_querypolicyDescriptorListType(input.PolicyArns, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `PolicyArns.${key}`;
            entries[loc] = value;
        });
    }
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    if (input.Tags !== undefined && input.Tags !== null) {
        const memberEntries = serializeAws_querytagListType(input.Tags, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            const loc = `Tags.${key}`;
            entries[loc] = value;
        });
    }
    return entries;
};
const serializeAws_queryGetSessionTokenRequest = (input, context) => {
    const entries = {};
    if (input.DurationSeconds !== undefined && input.DurationSeconds !== null) {
        entries["DurationSeconds"] = input.DurationSeconds;
    }
    if (input.SerialNumber !== undefined && input.SerialNumber !== null) {
        entries["SerialNumber"] = input.SerialNumber;
    }
    if (input.TokenCode !== undefined && input.TokenCode !== null) {
        entries["TokenCode"] = input.TokenCode;
    }
    return entries;
};
const serializeAws_querypolicyDescriptorListType = (input, context) => {
    const entries = {};
    let counter = 1;
    for (let entry of input) {
        if (entry === null) {
            continue;
        }
        const memberEntries = serializeAws_queryPolicyDescriptorType(entry, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            entries[`member.${counter}.${key}`] = value;
        });
        counter++;
    }
    return entries;
};
const serializeAws_queryPolicyDescriptorType = (input, context) => {
    const entries = {};
    if (input.arn !== undefined && input.arn !== null) {
        entries["arn"] = input.arn;
    }
    return entries;
};
const serializeAws_queryTag = (input, context) => {
    const entries = {};
    if (input.Key !== undefined && input.Key !== null) {
        entries["Key"] = input.Key;
    }
    if (input.Value !== undefined && input.Value !== null) {
        entries["Value"] = input.Value;
    }
    return entries;
};
const serializeAws_querytagKeyListType = (input, context) => {
    const entries = {};
    let counter = 1;
    for (let entry of input) {
        if (entry === null) {
            continue;
        }
        entries[`member.${counter}`] = entry;
        counter++;
    }
    return entries;
};
const serializeAws_querytagListType = (input, context) => {
    const entries = {};
    let counter = 1;
    for (let entry of input) {
        if (entry === null) {
            continue;
        }
        const memberEntries = serializeAws_queryTag(entry, context);
        Object.entries(memberEntries).forEach(([key, value]) => {
            entries[`member.${counter}.${key}`] = value;
        });
        counter++;
    }
    return entries;
};
const deserializeAws_queryAssumedRoleUser = (output, context) => {
    let contents = {
        AssumedRoleId: undefined,
        Arn: undefined,
    };
    if (output["AssumedRoleId"] !== undefined) {
        contents.AssumedRoleId = __expectString(output["AssumedRoleId"]);
    }
    if (output["Arn"] !== undefined) {
        contents.Arn = __expectString(output["Arn"]);
    }
    return contents;
};
const deserializeAws_queryAssumeRoleResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        AssumedRoleUser: undefined,
        PackedPolicySize: undefined,
        SourceIdentity: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["AssumedRoleUser"] !== undefined) {
        contents.AssumedRoleUser = deserializeAws_queryAssumedRoleUser(output["AssumedRoleUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    if (output["SourceIdentity"] !== undefined) {
        contents.SourceIdentity = __expectString(output["SourceIdentity"]);
    }
    return contents;
};
const deserializeAws_queryAssumeRoleWithSAMLResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        AssumedRoleUser: undefined,
        PackedPolicySize: undefined,
        Subject: undefined,
        SubjectType: undefined,
        Issuer: undefined,
        Audience: undefined,
        NameQualifier: undefined,
        SourceIdentity: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["AssumedRoleUser"] !== undefined) {
        contents.AssumedRoleUser = deserializeAws_queryAssumedRoleUser(output["AssumedRoleUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    if (output["Subject"] !== undefined) {
        contents.Subject = __expectString(output["Subject"]);
    }
    if (output["SubjectType"] !== undefined) {
        contents.SubjectType = __expectString(output["SubjectType"]);
    }
    if (output["Issuer"] !== undefined) {
        contents.Issuer = __expectString(output["Issuer"]);
    }
    if (output["Audience"] !== undefined) {
        contents.Audience = __expectString(output["Audience"]);
    }
    if (output["NameQualifier"] !== undefined) {
        contents.NameQualifier = __expectString(output["NameQualifier"]);
    }
    if (output["SourceIdentity"] !== undefined) {
        contents.SourceIdentity = __expectString(output["SourceIdentity"]);
    }
    return contents;
};
const deserializeAws_queryAssumeRoleWithWebIdentityResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        SubjectFromWebIdentityToken: undefined,
        AssumedRoleUser: undefined,
        PackedPolicySize: undefined,
        Provider: undefined,
        Audience: undefined,
        SourceIdentity: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["SubjectFromWebIdentityToken"] !== undefined) {
        contents.SubjectFromWebIdentityToken = __expectString(output["SubjectFromWebIdentityToken"]);
    }
    if (output["AssumedRoleUser"] !== undefined) {
        contents.AssumedRoleUser = deserializeAws_queryAssumedRoleUser(output["AssumedRoleUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    if (output["Provider"] !== undefined) {
        contents.Provider = __expectString(output["Provider"]);
    }
    if (output["Audience"] !== undefined) {
        contents.Audience = __expectString(output["Audience"]);
    }
    if (output["SourceIdentity"] !== undefined) {
        contents.SourceIdentity = __expectString(output["SourceIdentity"]);
    }
    return contents;
};
const deserializeAws_queryCredentials = (output, context) => {
    let contents = {
        AccessKeyId: undefined,
        SecretAccessKey: undefined,
        SessionToken: undefined,
        Expiration: undefined,
    };
    if (output["AccessKeyId"] !== undefined) {
        contents.AccessKeyId = __expectString(output["AccessKeyId"]);
    }
    if (output["SecretAccessKey"] !== undefined) {
        contents.SecretAccessKey = __expectString(output["SecretAccessKey"]);
    }
    if (output["SessionToken"] !== undefined) {
        contents.SessionToken = __expectString(output["SessionToken"]);
    }
    if (output["Expiration"] !== undefined) {
        contents.Expiration = new Date(output["Expiration"]);
    }
    return contents;
};
const deserializeAws_queryDecodeAuthorizationMessageResponse = (output, context) => {
    let contents = {
        DecodedMessage: undefined,
    };
    if (output["DecodedMessage"] !== undefined) {
        contents.DecodedMessage = __expectString(output["DecodedMessage"]);
    }
    return contents;
};
const deserializeAws_queryExpiredTokenException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryFederatedUser = (output, context) => {
    let contents = {
        FederatedUserId: undefined,
        Arn: undefined,
    };
    if (output["FederatedUserId"] !== undefined) {
        contents.FederatedUserId = __expectString(output["FederatedUserId"]);
    }
    if (output["Arn"] !== undefined) {
        contents.Arn = __expectString(output["Arn"]);
    }
    return contents;
};
const deserializeAws_queryGetAccessKeyInfoResponse = (output, context) => {
    let contents = {
        Account: undefined,
    };
    if (output["Account"] !== undefined) {
        contents.Account = __expectString(output["Account"]);
    }
    return contents;
};
const deserializeAws_queryGetCallerIdentityResponse = (output, context) => {
    let contents = {
        UserId: undefined,
        Account: undefined,
        Arn: undefined,
    };
    if (output["UserId"] !== undefined) {
        contents.UserId = __expectString(output["UserId"]);
    }
    if (output["Account"] !== undefined) {
        contents.Account = __expectString(output["Account"]);
    }
    if (output["Arn"] !== undefined) {
        contents.Arn = __expectString(output["Arn"]);
    }
    return contents;
};
const deserializeAws_queryGetFederationTokenResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
        FederatedUser: undefined,
        PackedPolicySize: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    if (output["FederatedUser"] !== undefined) {
        contents.FederatedUser = deserializeAws_queryFederatedUser(output["FederatedUser"], context);
    }
    if (output["PackedPolicySize"] !== undefined) {
        contents.PackedPolicySize = parseInt(output["PackedPolicySize"]);
    }
    return contents;
};
const deserializeAws_queryGetSessionTokenResponse = (output, context) => {
    let contents = {
        Credentials: undefined,
    };
    if (output["Credentials"] !== undefined) {
        contents.Credentials = deserializeAws_queryCredentials(output["Credentials"], context);
    }
    return contents;
};
const deserializeAws_queryIDPCommunicationErrorException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryIDPRejectedClaimException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryInvalidAuthorizationMessageException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryInvalidIdentityTokenException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryMalformedPolicyDocumentException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryPackedPolicyTooLargeException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeAws_queryRegionDisabledException = (output, context) => {
    let contents = {
        message: undefined,
    };
    if (output["message"] !== undefined) {
        contents.message = __expectString(output["message"]);
    }
    return contents;
};
const deserializeMetadata = (output) => ({
    httpStatusCode: output.statusCode,
    requestId: output.headers["x-amzn-requestid"] ?? output.headers["x-amzn-request-id"],
    extendedRequestId: output.headers["x-amz-id-2"],
    cfId: output.headers["x-amz-cf-id"],
});
const collectBody = (streamBody = new Uint8Array(), context) => {
    if (streamBody instanceof Uint8Array) {
        return Promise.resolve(streamBody);
    }
    return context.streamCollector(streamBody) || Promise.resolve(new Uint8Array());
};
const collectBodyString = (streamBody, context) => collectBody(streamBody, context).then((body) => context.utf8Encoder(body));
const buildHttpRpcRequest = async (context, headers, path, resolvedHostname, body) => {
    const { hostname, protocol = "https", port } = await context.endpoint();
    const contents = {
        protocol,
        hostname,
        port,
        method: "POST",
        path,
        headers,
    };
    if (resolvedHostname !== undefined) {
        contents.hostname = resolvedHostname;
    }
    if (body !== undefined) {
        contents.body = body;
    }
    return new __HttpRequest(contents);
};
const parseBody = (streamBody, context) => collectBodyString(streamBody, context).then((encoded) => {
    if (encoded.length) {
        const parsedObj = xmlParse(encoded, {
            attributeNamePrefix: "",
            ignoreAttributes: false,
            parseNodeValue: false,
            trimValues: false,
            tagValueProcessor: (val) => (val.trim() === "" ? "" : decodeHTML(val)),
        });
        const textNodeName = "#text";
        const key = Object.keys(parsedObj)[0];
        const parsedObjToReturn = parsedObj[key];
        if (parsedObjToReturn[textNodeName]) {
            parsedObjToReturn[key] = parsedObjToReturn[textNodeName];
            delete parsedObjToReturn[textNodeName];
        }
        return __getValueFromTextNode(parsedObjToReturn);
    }
    return {};
});
const buildFormUrlencodedString = (formEntries) => Object.entries(formEntries)
    .map(([key, value]) => __extendedEncodeURIComponent(key) + "=" + __extendedEncodeURIComponent(value))
    .join("&");
const loadQueryErrorCode = (output, data) => {
    if (data.Error.Code !== undefined) {
        return data.Error.Code;
    }
    if (output.statusCode == 404) {
        return "NotFound";
    }
    return "";
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQXdzX3F1ZXJ5LmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiQXdzX3F1ZXJ5LnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQTZDQSxPQUFPLEVBQUUsV0FBVyxJQUFJLGFBQWEsRUFBa0MsTUFBTSw0QkFBNEIsQ0FBQztBQUMxRyxPQUFPLEVBQ0wsWUFBWSxJQUFJLGNBQWMsRUFDOUIsMEJBQTBCLElBQUksNEJBQTRCLEVBQzFELG9CQUFvQixJQUFJLHNCQUFzQixHQUMvQyxNQUFNLDRCQUE0QixDQUFDO0FBU3BDLE9BQU8sRUFBRSxVQUFVLEVBQUUsTUFBTSwyQkFBMkIsQ0FBQztBQUN2RCxPQUFPLEVBQUUsS0FBSyxJQUFJLFFBQVEsRUFBRSxNQUFNLGtDQUFrQyxDQUFDO0FBRXJFLE1BQU0sQ0FBQyxNQUFNLG1DQUFtQyxHQUFHLEtBQUssRUFDdEQsS0FBNkIsRUFDN0IsT0FBdUIsRUFDQyxFQUFFO0lBQzFCLE1BQU0sT0FBTyxHQUFnQjtRQUMzQixjQUFjLEVBQUUsbUNBQW1DO0tBQ3BELENBQUM7SUFDRixJQUFJLElBQVMsQ0FBQztJQUNkLElBQUksR0FBRyx5QkFBeUIsQ0FBQztRQUMvQixHQUFHLG1DQUFtQyxDQUFDLEtBQUssRUFBRSxPQUFPLENBQUM7UUFDdEQsTUFBTSxFQUFFLFlBQVk7UUFDcEIsT0FBTyxFQUFFLFlBQVk7S0FDdEIsQ0FBQyxDQUFDO0lBQ0gsT0FBTyxtQkFBbUIsQ0FBQyxPQUFPLEVBQUUsT0FBTyxFQUFFLEdBQUcsRUFBRSxTQUFTLEVBQUUsSUFBSSxDQUFDLENBQUM7QUFDckUsQ0FBQyxDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sMkNBQTJDLEdBQUcsS0FBSyxFQUM5RCxLQUFxQyxFQUNyQyxPQUF1QixFQUNDLEVBQUU7SUFDMUIsTUFBTSxPQUFPLEdBQWdCO1FBQzNCLGNBQWMsRUFBRSxtQ0FBbUM7S0FDcEQsQ0FBQztJQUNGLElBQUksSUFBUyxDQUFDO0lBQ2QsSUFBSSxHQUFHLHlCQUF5QixDQUFDO1FB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