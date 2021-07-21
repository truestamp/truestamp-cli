import { SENSITIVE_STRING } from "../../smithy-client/mod.ts";
export var AccountInfo;
(function (AccountInfo) {
    AccountInfo.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(AccountInfo || (AccountInfo = {}));
export var GetRoleCredentialsRequest;
(function (GetRoleCredentialsRequest) {
    GetRoleCredentialsRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.accessToken && { accessToken: SENSITIVE_STRING }),
    });
})(GetRoleCredentialsRequest || (GetRoleCredentialsRequest = {}));
export var RoleCredentials;
(function (RoleCredentials) {
    RoleCredentials.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.secretAccessKey && { secretAccessKey: SENSITIVE_STRING }),
        ...(obj.sessionToken && { sessionToken: SENSITIVE_STRING }),
    });
})(RoleCredentials || (RoleCredentials = {}));
export var GetRoleCredentialsResponse;
(function (GetRoleCredentialsResponse) {
    GetRoleCredentialsResponse.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.roleCredentials && { roleCredentials: RoleCredentials.filterSensitiveLog(obj.roleCredentials) }),
    });
})(GetRoleCredentialsResponse || (GetRoleCredentialsResponse = {}));
export var InvalidRequestException;
(function (InvalidRequestException) {
    InvalidRequestException.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(InvalidRequestException || (InvalidRequestException = {}));
export var ResourceNotFoundException;
(function (ResourceNotFoundException) {
    ResourceNotFoundException.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ResourceNotFoundException || (ResourceNotFoundException = {}));
export var TooManyRequestsException;
(function (TooManyRequestsException) {
    TooManyRequestsException.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(TooManyRequestsException || (TooManyRequestsException = {}));
export var UnauthorizedException;
(function (UnauthorizedException) {
    UnauthorizedException.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(UnauthorizedException || (UnauthorizedException = {}));
export var ListAccountRolesRequest;
(function (ListAccountRolesRequest) {
    ListAccountRolesRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.accessToken && { accessToken: SENSITIVE_STRING }),
    });
})(ListAccountRolesRequest || (ListAccountRolesRequest = {}));
export var RoleInfo;
(function (RoleInfo) {
    RoleInfo.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(RoleInfo || (RoleInfo = {}));
export var ListAccountRolesResponse;
(function (ListAccountRolesResponse) {
    ListAccountRolesResponse.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ListAccountRolesResponse || (ListAccountRolesResponse = {}));
export var ListAccountsRequest;
(function (ListAccountsRequest) {
    ListAccountsRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.accessToken && { accessToken: SENSITIVE_STRING }),
    });
})(ListAccountsRequest || (ListAccountsRequest = {}));
export var ListAccountsResponse;
(function (ListAccountsResponse) {
    ListAccountsResponse.filterSensitiveLog = (obj) => ({
        ...obj,
    });
})(ListAccountsResponse || (ListAccountsResponse = {}));
export var LogoutRequest;
(function (LogoutRequest) {
    LogoutRequest.filterSensitiveLog = (obj) => ({
        ...obj,
        ...(obj.accessToken && { accessToken: SENSITIVE_STRING }),
    });
})(LogoutRequest || (LogoutRequest = {}));
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kZWxzXzAuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJtb2RlbHNfMC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZ0JBQWdCLEVBQUUsTUFBTSw0QkFBNEIsQ0FBQztBQXVCOUQsTUFBTSxLQUFXLFdBQVcsQ0FPM0I7QUFQRCxXQUFpQixXQUFXO0lBSWIsOEJBQWtCLEdBQUcsQ0FBQyxHQUFnQixFQUFPLEVBQUUsQ0FBQyxDQUFDO1FBQzVELEdBQUcsR0FBRztLQUNQLENBQUMsQ0FBQztBQUNMLENBQUMsRUFQZ0IsV0FBVyxLQUFYLFdBQVcsUUFPM0I7QUFvQkQsTUFBTSxLQUFXLHlCQUF5QixDQVF6QztBQVJELFdBQWlCLHlCQUF5QjtJQUkzQiw0Q0FBa0IsR0FBRyxDQUFDLEdBQThCLEVBQU8sRUFBRSxDQUFDLENBQUM7UUFDMUUsR0FBRyxHQUFHO1FBQ04sR0FBRyxDQUFDLEdBQUcsQ0FBQyxXQUFXLElBQUksRUFBRSxXQUFXLEVBQUUsZ0JBQWdCLEVBQUUsQ0FBQztLQUMxRCxDQUFDLENBQUM7QUFDTCxDQUFDLEVBUmdCLHlCQUF5QixLQUF6Qix5QkFBeUIsUUFRekM7QUErQkQsTUFBTSxLQUFXLGVBQWUsQ0FTL0I7QUFURCxXQUFpQixlQUFlO0lBSWpCLGtDQUFrQixHQUFHLENBQUMsR0FBb0IsRUFBTyxFQUFFLENBQUMsQ0FBQztRQUNoRSxHQUFHLEdBQUc7UUFDTixHQUFHLENBQUMsR0FBRyxDQUFDLGVBQWUsSUFBSSxFQUFFLGVBQWUsRUFBRSxnQkFBZ0IsRUFBRSxDQUFDO1FBQ2pFLEdBQUcsQ0FBQyxHQUFHLENBQUMsWUFBWSxJQUFJLEVBQUUsWUFBWSxFQUFFLGdCQUFnQixFQUFFLENBQUM7S0FDNUQsQ0FBQyxDQUFDO0FBQ0wsQ0FBQyxFQVRnQixlQUFlLEtBQWYsZUFBZSxRQVMvQjtBQVNELE1BQU0sS0FBVywwQkFBMEIsQ0FRMUM7QUFSRCxXQUFpQiwwQkFBMEI7SUFJNUIsNkNBQWtCLEdBQUcs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