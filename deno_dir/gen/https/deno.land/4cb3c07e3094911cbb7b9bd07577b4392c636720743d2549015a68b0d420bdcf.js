export function normalize(args) {
    const normalized = [];
    let inLiteral = false;
    for (const arg of args) {
        if (inLiteral) {
            normalized.push(arg);
        }
        else if (arg === "--") {
            inLiteral = true;
            normalized.push(arg);
        }
        else if (arg.length > 1 && arg[0] === "-") {
            const isLong = arg[1] === "-";
            const isDotted = !isLong && arg[2] === ".";
            if (arg.includes("=")) {
                const parts = arg.split("=");
                const flag = parts.shift();
                if (isLong) {
                    normalized.push(flag);
                }
                else {
                    normalizeShortFlags(flag);
                }
                normalized.push(parts.join("="));
            }
            else if (isLong || isDotted) {
                normalized.push(arg);
            }
            else {
                normalizeShortFlags(arg);
            }
        }
        else {
            normalized.push(arg);
        }
    }
    return normalized;
    function normalizeShortFlags(flag) {
        const flags = flag.slice(1).split("");
        if (isNaN(Number(flag[flag.length - 1]))) {
            flags.forEach((val) => normalized.push(`-${val}`));
        }
        else {
            normalized.push(`-${flags.shift()}`);
            normalized.push(flags.join(""));
        }
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoibm9ybWFsaXplLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsibm9ybWFsaXplLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUlBLE1BQU0sVUFBVSxTQUFTLENBQUMsSUFBYztJQUN0QyxNQUFNLFVBQVUsR0FBRyxFQUFFLENBQUM7SUFDdEIsSUFBSSxTQUFTLEdBQUcsS0FBSyxDQUFDO0lBRXRCLEtBQUssTUFBTSxHQUFHLElBQUksSUFBSSxFQUFFO1FBQ3RCLElBQUksU0FBUyxFQUFFO1lBQ2IsVUFBVSxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsQ0FBQztTQUN0QjthQUFNLElBQUksR0FBRyxLQUFLLElBQUksRUFBRTtZQUN2QixTQUFTLEdBQUcsSUFBSSxDQUFDO1lBQ2pCLFVBQVUsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7U0FDdEI7YUFBTSxJQUFJLEdBQUcsQ0FBQyxNQUFNLEdBQUcsQ0FBQyxJQUFJLEdBQUcsQ0FBQyxDQUFDLENBQUMsS0FBSyxHQUFHLEVBQUU7WUFDM0MsTUFBTSxNQUFNLEdBQUcsR0FBRyxDQUFDLENBQUMsQ0FBQyxLQUFLLEdBQUcsQ0FBQztZQUM5QixNQUFNLFFBQVEsR0FBRyxDQUFDLE1BQU0sSUFBSSxHQUFHLENBQUMsQ0FBQyxDQUFDLEtBQUssR0FBRyxDQUFDO1lBRTNDLElBQUksR0FBRyxDQUFDLFFBQVEsQ0FBQyxHQUFHLENBQUMsRUFBRTtnQkFDckIsTUFBTSxLQUFLLEdBQUcsR0FBRyxDQUFDLEtBQUssQ0FBQyxHQUFHLENBQUMsQ0FBQztnQkFDN0IsTUFBTSxJQUFJLEdBQUcsS0FBSyxDQUFDLEtBQUssRUFBWSxDQUFDO2dCQUVyQyxJQUFJLE1BQU0sRUFBRTtvQkFDVixVQUFVLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO2lCQUN2QjtxQkFBTTtvQkFDTCxtQkFBbUIsQ0FBQyxJQUFJLENBQUMsQ0FBQztpQkFDM0I7Z0JBQ0QsVUFBVSxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDLENBQUM7YUFDbEM7aUJBQU0sSUFBSSxNQUFNLElBQUksUUFBUSxFQUFFO2dCQUM3QixVQUFVLENBQUMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQ3RCO2lCQUFNO2dCQUNMLG1CQUFtQixDQUFDLEdBQUcsQ0FBQyxDQUFDO2FBQzFCO1NBQ0Y7YUFBTTtZQUNMLFVBQVUsQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLENBQUM7U0FDdEI7S0FDRjtJQUVELE9BQU8sVUFBVSxDQUFDO0lBTWxCLFNBQVMsbUJBQW1CLENBQUMsSUFBWTtRQUN2QyxNQUFNLEtBQUssR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLENBQUMsQ0FBQyxDQUFDLEtBQUssQ0FBQyxFQUFFLENBQUMsQ0FBQztRQUV0QyxJQUFJLEtBQUssQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxNQUFNLEdBQUcsQ0FBQyxDQUFDLENBQUMsQ0FBQyxFQUFFO1lBQ3hDLEtBQUssQ0FBQyxPQUFPLENBQUMsQ0FBQyxHQUFHLEVBQUUsRUFBRSxDQUFDLFVBQVUsQ0FBQyxJQUFJLENBQUMsSUFBSSxHQUFHLEVBQUUsQ0FBQyxDQUFDLENBQUM7U0FDcEQ7YUFBTTtZQUNMLFVBQVUsQ0FBQyxJQUFJLENBQUMsSUFBSSxLQUFLLENBQUMsS0FBSyxFQUFFLEVBQUUsQ0FBQyxDQUFDO1lBQ3JDLFVBQVUsQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxFQUFFLENBQUMsQ0FBQyxDQUFDO1NBQ2pDO0lBQ0gsQ0FBQztBQUNILENBQUMifQ==