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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibW9kZWxzXzAuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJtb2RlbHNfMC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZ0JBQWdCLEVBQXdDLE1BQU0sNEJBQTRCLENBQUM7QUF1QnBHLE1BQU0sS0FBVyxXQUFXLENBTzNCO0FBUEQsV0FBaUIsV0FBVztJQUliLDhCQUFrQixHQUFHLENBQUMsR0FBZ0IsRUFBTyxFQUFFLENBQUMsQ0FBQztRQUM1RCxHQUFHLEdBQUc7S0FDUCxDQUFDLENBQUM7QUFDTCxDQUFDLEVBUGdCLFdBQVcsS0FBWCxXQUFXLFFBTzNCO0FBb0JELE1BQU0sS0FBVyx5QkFBeUIsQ0FRekM7QUFSRCxXQUFpQix5QkFBeUI7SUFJM0IsNENBQWtCLEdBQUcsQ0FBQyxHQUE4QixFQUFPLEVBQUUsQ0FBQyxDQUFDO1FBQzFFLEdBQUcsR0FBRztRQUNOLEdBQUcsQ0FBQyxHQUFHLENBQUMsV0FBVyxJQUFJLEVBQUUsV0FBVyxFQUFFLGdCQUFnQixFQUFFLENBQUM7S0FDMUQsQ0FBQyxDQUFDO0FBQ0wsQ0FBQyxFQVJnQix5QkFBeUIsS0FBekIseUJBQXlCLFFBUXpDO0FBK0JELE1BQU0sS0FBVyxlQUFlLENBUy9CO0FBVEQsV0FBaUIsZUFBZTtJQUlqQixrQ0FBa0IsR0FBRyxDQUFDLEdBQW9CLEVBQU8sRUFBRSxDQUFDLENBQUM7UUFDaEUsR0FBRyxHQUFHO1FBQ04sR0FBRyxDQUFDLEdBQUcsQ0FBQyxlQUFlLElBQUksRUFBRSxlQUFlLEVBQUUsZ0JBQWdCLEVBQUUsQ0FBQztRQUNqRSxHQUFHLENBQUMsR0FBRyxDQUFDLFlBQVksSUFBSSxFQUFFLFlBQVksRUFBRSxnQkFBZ0IsRUFBRSxDQUFDO0tBQzVELENBQUMsQ0FBQztBQUNMLENBQUMsRUFUZ0IsZUFBZSxLQUFmLGVBQWUsUUFTL0I7QUFTRCxNQUFNLEtBQVcsMEJBQTBCLENBUTFDO0FBUkQsV0FBaUIsMEJBQTBCO0lBSTVCLDZDQUFrQixHQUFH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