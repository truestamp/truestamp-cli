import { HttpRequest as __HttpRequest } from "../../protocol-http/mod.ts";
export const serializeAws_restJson1GetRoleCredentialsCommand = async (input, context) => {
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = "/federation/credentials";
    const query = {
        ...(input.roleName !== undefined && { role_name: input.roleName }),
        ...(input.accountId !== undefined && { account_id: input.accountId }),
    };
    let body;
    const { hostname, protocol = "https", port } = await context.endpoint();
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restJson1ListAccountRolesCommand = async (input, context) => {
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = "/assignment/roles";
    const query = {
        ...(input.nextToken !== undefined && { next_token: input.nextToken }),
        ...(input.maxResults !== undefined && { max_result: input.maxResults.toString() }),
        ...(input.accountId !== undefined && { account_id: input.accountId }),
    };
    let body;
    const { hostname, protocol = "https", port } = await context.endpoint();
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restJson1ListAccountsCommand = async (input, context) => {
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = "/assignment/accounts";
    const query = {
        ...(input.nextToken !== undefined && { next_token: input.nextToken }),
        ...(input.maxResults !== undefined && { max_result: input.maxResults.toString() }),
    };
    let body;
    const { hostname, protocol = "https", port } = await context.endpoint();
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "GET",
        headers,
        path: resolvedPath,
        query,
        body,
    });
};
export const serializeAws_restJson1LogoutCommand = async (input, context) => {
    const headers = {
        ...(isSerializableHeaderValue(input.accessToken) && { "x-amz-sso_bearer_token": input.accessToken }),
    };
    let resolvedPath = "/logout";
    let body;
    const { hostname, protocol = "https", port } = await context.endpoint();
    return new __HttpRequest({
        protocol,
        hostname,
        port,
        method: "POST",
        headers,
        path: resolvedPath,
        body,
    });
};
export const deserializeAws_restJson1GetRoleCredentialsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1GetRoleCredentialsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        roleCredentials: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.roleCredentials !== undefined && data.roleCredentials !== null) {
        contents.roleCredentials = deserializeAws_restJson1RoleCredentials(data.roleCredentials, context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restJson1GetRoleCredentialsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "ResourceNotFoundException":
        case "com.amazonaws.sso#ResourceNotFoundException":
            response = {
                ...(await deserializeAws_restJson1ResourceNotFoundExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_restJson1ListAccountRolesCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1ListAccountRolesCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        nextToken: undefined,
        roleList: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.nextToken !== undefined && data.nextToken !== null) {
        contents.nextToken = data.nextToken;
    }
    if (data.roleList !== undefined && data.roleList !== null) {
        contents.roleList = deserializeAws_restJson1RoleListType(data.roleList, context);
    }
    return Promise.resolve(contents);
};
const deserializeAws_restJson1ListAccountRolesCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "ResourceNotFoundException":
        case "com.amazonaws.sso#ResourceNotFoundException":
            response = {
                ...(await deserializeAws_restJson1ResourceNotFoundExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_restJson1ListAccountsCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1ListAccountsCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
        accountList: undefined,
        nextToken: undefined,
    };
    const data = await parseBody(output.body, context);
    if (data.accountList !== undefined && data.accountList !== null) {
        contents.accountList = deserializeAws_restJson1AccountListType(data.accountList, context);
    }
    if (data.nextToken !== undefined && data.nextToken !== null) {
        contents.nextToken = data.nextToken;
    }
    return Promise.resolve(contents);
};
const deserializeAws_restJson1ListAccountsCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "ResourceNotFoundException":
        case "com.amazonaws.sso#ResourceNotFoundException":
            response = {
                ...(await deserializeAws_restJson1ResourceNotFoundExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
export const deserializeAws_restJson1LogoutCommand = async (output, context) => {
    if (output.statusCode !== 200 && output.statusCode >= 300) {
        return deserializeAws_restJson1LogoutCommandError(output, context);
    }
    const contents = {
        $metadata: deserializeMetadata(output),
    };
    await collectBody(output.body, context);
    return Promise.resolve(contents);
};
const deserializeAws_restJson1LogoutCommandError = async (output, context) => {
    const parsedOutput = {
        ...output,
        body: await parseBody(output.body, context),
    };
    let response;
    let errorCode = "UnknownError";
    errorCode = loadRestJsonErrorCode(output, parsedOutput.body);
    switch (errorCode) {
        case "InvalidRequestException":
        case "com.amazonaws.sso#InvalidRequestException":
            response = {
                ...(await deserializeAws_restJson1InvalidRequestExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "TooManyRequestsException":
        case "com.amazonaws.sso#TooManyRequestsException":
            response = {
                ...(await deserializeAws_restJson1TooManyRequestsExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        case "UnauthorizedException":
        case "com.amazonaws.sso#UnauthorizedException":
            response = {
                ...(await deserializeAws_restJson1UnauthorizedExceptionResponse(parsedOutput, context)),
                name: errorCode,
                $metadata: deserializeMetadata(output),
            };
            break;
        default:
            const parsedBody = parsedOutput.body;
            errorCode = parsedBody.code || parsedBody.Code || errorCode;
            response = {
                ...parsedBody,
                name: `${errorCode}`,
                message: parsedBody.message || parsedBody.Message || errorCode,
                $fault: "client",
                $metadata: deserializeMetadata(output),
            };
    }
    const message = response.message || response.Message || errorCode;
    response.message = message;
    delete response.Message;
    return Promise.reject(Object.assign(new Error(message), response));
};
const deserializeAws_restJson1InvalidRequestExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "InvalidRequestException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = data.message;
    }
    return contents;
};
const deserializeAws_restJson1ResourceNotFoundExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "ResourceNotFoundException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = data.message;
    }
    return contents;
};
const deserializeAws_restJson1TooManyRequestsExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "TooManyRequestsException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = data.message;
    }
    return contents;
};
const deserializeAws_restJson1UnauthorizedExceptionResponse = async (parsedOutput, context) => {
    const contents = {
        name: "UnauthorizedException",
        $fault: "client",
        $metadata: deserializeMetadata(parsedOutput),
        message: undefined,
    };
    const data = parsedOutput.body;
    if (data.message !== undefined && data.message !== null) {
        contents.message = data.message;
    }
    return contents;
};
const deserializeAws_restJson1AccountInfo = (output, context) => {
    return {
        accountId: output.accountId !== undefined && output.accountId !== null ? output.accountId : undefined,
        accountName: output.accountName !== undefined && output.accountName !== null ? output.accountName : undefined,
        emailAddress: output.emailAddress !== undefined && output.emailAddress !== null ? output.emailAddress : undefined,
    };
};
const deserializeAws_restJson1AccountListType = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restJson1AccountInfo(entry, context);
    });
};
const deserializeAws_restJson1RoleCredentials = (output, context) => {
    return {
        accessKeyId: output.accessKeyId !== undefined && output.accessKeyId !== null ? output.accessKeyId : undefined,
        expiration: output.expiration !== undefined && output.expiration !== null ? output.expiration : undefined,
        secretAccessKey: output.secretAccessKey !== undefined && output.secretAccessKey !== null ? output.secretAccessKey : undefined,
        sessionToken: output.sessionToken !== undefined && output.sessionToken !== null ? output.sessionToken : undefined,
    };
};
const deserializeAws_restJson1RoleInfo = (output, context) => {
    return {
        accountId: output.accountId !== undefined && output.accountId !== null ? output.accountId : undefined,
        roleName: output.roleName !== undefined && output.roleName !== null ? output.roleName : undefined,
    };
};
const deserializeAws_restJson1RoleListType = (output, context) => {
    return (output || [])
        .filter((e) => e != null)
        .map((entry) => {
        if (entry === null) {
            return null;
        }
        return deserializeAws_restJson1RoleInfo(entry, context);
    });
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
const isSerializableHeaderValue = (value) => value !== undefined &&
    value !== null &&
    value !== "" &&
    (!Object.getOwnPropertyNames(value).includes("length") || value.length != 0) &&
    (!Object.getOwnPropertyNames(value).includes("size") || value.size != 0);
const parseBody = (streamBody, context) => collectBodyString(streamBody, context).then((encoded) => {
    if (encoded.length) {
        return JSON.parse(encoded);
    }
    return {};
});
const loadRestJsonErrorCode = (output, data) => {
    const findKey = (object, key) => Object.keys(object).find((k) => k.toLowerCase() === key.toLowerCase());
    const sanitizeErrorCode = (rawValue) => {
        let cleanValue = rawValue;
        if (cleanValue.indexOf(":") >= 0) {
            cleanValue = cleanValue.split(":")[0];
        }
        if (cleanValue.indexOf("#") >= 0) {
            cleanValue = cleanValue.split("#")[1];
        }
        return cleanValue;
    };
    const headerKey = findKey(output.headers, "x-amzn-errortype");
    if (headerKey !== undefined) {
        return sanitizeErrorCode(output.headers[headerKey]);
    }
    if (data.code !== undefined) {
        return sanitizeErrorCode(data.code);
    }
    if (data["__type"] !== undefined) {
        return sanitizeErrorCode(data["__type"]);
    }
    return "";
};
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiQXdzX3Jlc3RKc29uMS5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIkF3c19yZXN0SnNvbjEudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBYUEsT0FBTyxFQUFFLFdBQVcsSUFBSSxhQUFhLEVBQWtDLE1BQU0sNEJBQTRCLENBQUM7QUFZMUcsTUFBTSxDQUFDLE1BQU0sK0NBQStDLEdBQUcsS0FBSyxFQUNsRSxLQUFxQyxFQUNyQyxPQUF1QixFQUNDLEVBQUU7SUFDMUIsTUFBTSxPQUFPLEdBQVE7UUFDbkIsR0FBRyxDQUFDLHlCQUF5QixDQUFDLEtBQUssQ0FBQyxXQUFXLENBQUMsSUFBSSxFQUFFLHdCQUF3QixFQUFFLEtBQUssQ0FBQyxXQUFZLEVBQUUsQ0FBQztLQUN0RyxDQUFDO0lBQ0YsSUFBSSxZQUFZLEdBQUcseUJBQXlCLENBQUM7SUFDN0MsTUFBTSxLQUFLLEdBQVE7UUFDakIsR0FBRyxDQUFDLEtBQUssQ0FBQyxRQUFRLEtBQUssU0FBUyxJQUFJLEVBQUUsU0FBUyxFQUFFLEtBQUssQ0FBQyxRQUFRLEVBQUUsQ0FBQztRQUNsRSxHQUFHLENBQUMsS0FBSyxDQUFDLFNBQVMsS0FBSyxTQUFTLElBQUksRUFBRSxVQUFVLEVBQUUsS0FBSyxDQUFDLFNBQVMsRUFBRSxDQUFDO0tBQ3RFLENBQUM7SUFDRixJQUFJLElBQVMsQ0FBQztJQUNkLE1BQU0sRUFBRSxRQUFRLEVBQUUsUUFBUSxHQUFHLE9BQU8sRUFBRSxJQUFJLEVBQUUsR0FBRyxNQUFNLE9BQU8sQ0FBQyxRQUFRLEVBQUUsQ0FBQztJQUN4RSxPQUFPLElBQUksYUFBYSxDQUFDO1FBQ3ZCLFFBQVE7UUFDUixRQUFRO1FBQ1IsSUFBSTtRQUNKLE1BQU0sRUFBRSxLQUFLO1FBQ2IsT0FBTztRQUNQLElBQUksRUFBRSxZQUFZO1FBQ2xCLEtBQUs7UUFDTCxJQUFJO0tBQ0wsQ0FBQyxDQUFDO0FBQ0wsQ0FBQyxDQUFDO0FBRUYsTUFBTSxDQUFDLE1BQU0sNkNBQTZDLEdBQUcsS0FB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