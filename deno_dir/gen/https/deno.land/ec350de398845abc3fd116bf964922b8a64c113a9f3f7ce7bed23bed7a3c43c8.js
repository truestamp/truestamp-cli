import { STSClient } from "./STSClient.ts";
import { AssumeRoleCommand } from "./commands/AssumeRoleCommand.ts";
import { AssumeRoleWithSAMLCommand, } from "./commands/AssumeRoleWithSAMLCommand.ts";
import { AssumeRoleWithWebIdentityCommand, } from "./commands/AssumeRoleWithWebIdentityCommand.ts";
import { DecodeAuthorizationMessageCommand, } from "./commands/DecodeAuthorizationMessageCommand.ts";
import { GetAccessKeyInfoCommand, } from "./commands/GetAccessKeyInfoCommand.ts";
import { GetCallerIdentityCommand, } from "./commands/GetCallerIdentityCommand.ts";
import { GetFederationTokenCommand, } from "./commands/GetFederationTokenCommand.ts";
import { GetSessionTokenCommand, } from "./commands/GetSessionTokenCommand.ts";
export class STS extends STSClient {
    assumeRole(args, optionsOrCb, cb) {
        const command = new AssumeRoleCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    assumeRoleWithSAML(args, optionsOrCb, cb) {
        const command = new AssumeRoleWithSAMLCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    assumeRoleWithWebIdentity(args, optionsOrCb, cb) {
        const command = new AssumeRoleWithWebIdentityCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    decodeAuthorizationMessage(args, optionsOrCb, cb) {
        const command = new DecodeAuthorizationMessageCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getAccessKeyInfo(args, optionsOrCb, cb) {
        const command = new GetAccessKeyInfoCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getCallerIdentity(args, optionsOrCb, cb) {
        const command = new GetCallerIdentityCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getFederationToken(args, optionsOrCb, cb) {
        const command = new GetFederationTokenCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
    getSessionToken(args, optionsOrCb, cb) {
        const command = new GetSessionTokenCommand(args);
        if (typeof optionsOrCb === "function") {
            this.send(command, optionsOrCb);
        }
        else if (typeof cb === "function") {
            if (typeof optionsOrCb !== "object")
                throw new Error(`Expect http options but get ${typeof optionsOrCb}`);
            this.send(command, optionsOrCb || {}, cb);
        }
        else {
            return this.send(command, optionsOrCb);
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiU1RTLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiU1RTLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUFBLE9BQU8sRUFBRSxTQUFTLEVBQUUsTUFBTSxnQkFBZ0IsQ0FBQztBQUMzQyxPQUFPLEVBQUUsaUJBQWlCLEVBQW1ELE1BQU0saUNBQWlDLENBQUM7QUFDckgsT0FBTyxFQUNMLHlCQUF5QixHQUcxQixNQUFNLHlDQUF5QyxDQUFDO0FBQ2pELE9BQU8sRUFDTCxnQ0FBZ0MsR0FHakMsTUFBTSxnREFBZ0QsQ0FBQztBQUN4RCxPQUFPLEVBQ0wsaUNBQWlDLEdBR2xDLE1BQU0saURBQWlELENBQUM7QUFDekQsT0FBTyxFQUNMLHVCQUF1QixHQUd4QixNQUFNLHVDQUF1QyxDQUFDO0FBQy9DLE9BQU8sRUFDTCx3QkFBd0IsR0FHekIsTUFBTSx3Q0FBd0MsQ0FBQztBQUNoRCxPQUFPLEVBQ0wseUJBQXlCLEdBRzFCLE1BQU0seUNBQXlDLENBQUM7QUFDakQsT0FBTyxFQUNMLHNCQUFzQixHQUd2QixNQUFNLHNDQUFzQyxDQUFDO0FBVTlDLE1BQU0sT0FBTyxHQUFJLFNBQVEsU0FBUztJQThGekIsVUFBVSxDQUNmLElBQTRCLEVBQzVCLFdBQXlGLEVBQ3pGLEVBQXVEO1FBRXZELE1BQU0sT0FBTyxHQUFHLElBQUksaUJBQWlCLENBQUMsSUFBSSxDQUFDLENBQUM7UUFDNUMsSUFBSSxPQUFPLFdBQVcsS0FBSyxVQUFVLEVBQUU7WUFDckMsSUFBSSxDQUFDLElBQUksQ0FBQyxPQUFPLEVBQUUsV0FBVyxDQUFDLENBQUM7U0FDakM7YUFBTSxJQUFJLE9BQU8sRUFBRSxLQUFLLFVBQVUsRUFBRTtZQUNuQyxJQUFJLE9BQU8sV0FBVyxLQUFLLFFBQVE7Z0JBQUUsTUFBTSxJQUFJLEtBQUssQ0FBQywrQkFBK0IsT0FBTyxXQUFXLEVBQUUsQ0FBQyxDQUFDO1lBQzFHLElBQUksQ0FBQyxJQUFJLENBQUMsT0FBTyxFQUFFLFdBQVcsSUFBSSxFQUFFLEVB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