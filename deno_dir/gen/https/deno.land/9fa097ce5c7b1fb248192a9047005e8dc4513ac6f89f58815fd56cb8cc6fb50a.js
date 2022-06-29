import { errorUtil } from "./helpers/errorUtil.ts";
import { addIssueToContext, INVALID, isAborted, isAsync, isDirty, isValid, makeIssue, OK, ParseStatus } from "./helpers/parseUtil.ts";
import { getParsedType, util, ZodParsedType } from "./helpers/util.ts";
import { defaultErrorMap, overrideErrorMap, ZodError, ZodIssueCode } from "./ZodError.ts";
class ParseInputLazyPath {
    parent;
    data;
    _path;
    _key;
    constructor(parent, value, path, key){
        this.parent = parent;
        this.data = value;
        this._path = path;
        this._key = key;
    }
    get path() {
        return this._path.concat(this._key);
    }
}
const handleResult = (ctx, result)=>{
    if (isValid(result)) {
        return {
            success: true,
            data: result.value
        };
    } else {
        if (!ctx.common.issues.length) {
            throw new Error("Validation failed but no issues detected.");
        }
        const error = new ZodError(ctx.common.issues);
        return {
            success: false,
            error
        };
    }
};
function processCreateParams(params) {
    if (!params) return {};
    const { errorMap , invalid_type_error , required_error , description  } = params;
    if (errorMap && (invalid_type_error || required_error)) {
        throw new Error(`Can't use "invalid" or "required" in conjunction with custom error map.`);
    }
    if (errorMap) return {
        errorMap: errorMap,
        description
    };
    const customMap = (iss, ctx)=>{
        if (iss.code !== "invalid_type") return {
            message: ctx.defaultError
        };
        if (typeof ctx.data === "undefined" && required_error) return {
            message: required_error
        };
        if (params.invalid_type_error) return {
            message: params.invalid_type_error
        };
        return {
            message: ctx.defaultError
        };
    };
    return {
        errorMap: customMap,
        description
    };
}
export class ZodType {
    _type;
    _output;
    _input;
    _def;
    get description() {
        return this._def.description;
    }
    _getType(input) {
        return getParsedType(input.data);
    }
    _getOrReturnCtx(input, ctx) {
        return ctx || {
            common: input.parent.common,
            data: input.data,
            parsedType: getParsedType(input.data),
            schemaErrorMap: this._def.errorMap,
            path: input.path,
            parent: input.parent
        };
    }
    _processInputParams(input) {
        return {
            status: new ParseStatus(),
            ctx: {
                common: input.parent.common,
                data: input.data,
                parsedType: getParsedType(input.data),
                schemaErrorMap: this._def.errorMap,
                path: input.path,
                parent: input.parent
            }
        };
    }
    _parseSync(input) {
        const result = this._parse(input);
        if (isAsync(result)) {
            throw new Error("Synchronous parse encountered promise.");
        }
        return result;
    }
    _parseAsync(input) {
        const result = this._parse(input);
        return Promise.resolve(result);
    }
    parse(data, params) {
        const result = this.safeParse(data, params);
        if (result.success) return result.data;
        throw result.error;
    }
    safeParse(data, params) {
        const ctx = {
            common: {
                issues: [],
                async: params?.async ?? false,
                contextualErrorMap: params?.errorMap
            },
            path: params?.path || [],
            schemaErrorMap: this._def.errorMap,
            parent: null,
            data,
            parsedType: getParsedType(data)
        };
        const result = this._parseSync({
            data,
            path: ctx.path,
            parent: ctx
        });
        return handleResult(ctx, result);
    }
    async parseAsync(data, params) {
        const result = await this.safeParseAsync(data, params);
        if (result.success) return result.data;
        throw result.error;
    }
    async safeParseAsync(data, params) {
        const ctx = {
            common: {
                issues: [],
                contextualErrorMap: params?.errorMap,
                async: true
            },
            path: params?.path || [],
            schemaErrorMap: this._def.errorMap,
            parent: null,
            data,
            parsedType: getParsedType(data)
        };
        const maybeAsyncResult = this._parse({
            data,
            path: [],
            parent: ctx
        });
        const result = await (isAsync(maybeAsyncResult) ? maybeAsyncResult : Promise.resolve(maybeAsyncResult));
        return handleResult(ctx, result);
    }
    /** Alias of safeParseAsync */ spa = this.safeParseAsync;
    refine(check, message) {
        const getIssueProperties = (val)=>{
            if (typeof message === "string" || typeof message === "undefined") {
                return {
                    message
                };
            } else if (typeof message === "function") {
                return message(val);
            } else {
                return message;
            }
        };
        return this._refinement((val, ctx)=>{
            const result = check(val);
            const setError = ()=>ctx.addIssue({
                    code: ZodIssueCode.custom,
                    ...getIssueProperties(val)
                });
            if (typeof Promise !== "undefined" && result instanceof Promise) {
                return result.then((data)=>{
                    if (!data) {
                        setError();
                        return false;
                    } else {
                        return true;
                    }
                });
            }
            if (!result) {
                setError();
                return false;
            } else {
                return true;
            }
        });
    }
    refinement(check, refinementData) {
        return this._refinement((val, ctx)=>{
            if (!check(val)) {
                ctx.addIssue(typeof refinementData === "function" ? refinementData(val, ctx) : refinementData);
                return false;
            } else {
                return true;
            }
        });
    }
    _refinement(refinement) {
        return new ZodEffects({
            schema: this,
            typeName: ZodFirstPartyTypeKind.ZodEffects,
            effect: {
                type: "refinement",
                refinement
            }
        });
    }
    superRefine = this._refinement;
    constructor(def){
        this._def = def;
        this.parse = this.parse.bind(this);
        this.safeParse = this.safeParse.bind(this);
        this.parseAsync = this.parseAsync.bind(this);
        this.safeParseAsync = this.safeParseAsync.bind(this);
        this.spa = this.spa.bind(this);
        this.refine = this.refine.bind(this);
        this.refinement = this.refinement.bind(this);
        this.superRefine = this.superRefine.bind(this);
        this.optional = this.optional.bind(this);
        this.nullable = this.nullable.bind(this);
        this.nullish = this.nullish.bind(this);
        this.array = this.array.bind(this);
        this.promise = this.promise.bind(this);
        this.or = this.or.bind(this);
        this.and = this.and.bind(this);
        this.transform = this.transform.bind(this);
        this.default = this.default.bind(this);
        this.describe = this.describe.bind(this);
        this.isNullable = this.isNullable.bind(this);
        this.isOptional = this.isOptional.bind(this);
    }
    optional() {
        return ZodOptional.create(this);
    }
    nullable() {
        return ZodNullable.create(this);
    }
    nullish() {
        return this.optional().nullable();
    }
    array() {
        return ZodArray.create(this);
    }
    promise() {
        return ZodPromise.create(this);
    }
    or(option) {
        return ZodUnion.create([
            this,
            option
        ]);
    }
    and(incoming) {
        return ZodIntersection.create(this, incoming);
    }
    transform(transform) {
        return new ZodEffects({
            schema: this,
            typeName: ZodFirstPartyTypeKind.ZodEffects,
            effect: {
                type: "transform",
                transform
            }
        });
    }
    default(def) {
        const defaultValueFunc = typeof def === "function" ? def : ()=>def;
        return new ZodDefault({
            innerType: this,
            defaultValue: defaultValueFunc,
            typeName: ZodFirstPartyTypeKind.ZodDefault
        });
    }
    describe(description) {
        const This = this.constructor;
        return new This({
            ...this._def,
            description
        });
    }
    isOptional() {
        return this.safeParse(undefined).success;
    }
    isNullable() {
        return this.safeParse(null).success;
    }
}
const cuidRegex = /^c[^\s-]{8,}$/i;
const uuidRegex = /^([a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[a-f0-9]{4}-[a-f0-9]{12}|00000000-0000-0000-0000-000000000000)$/i;
// from https://stackoverflow.com/a/46181/1550155
// old version: too slow, didn't support unicode
// const emailRegex = /^((([a-z]|\d|[!#\$%&'\*\+\-\/=\?\^_`{\|}~]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])+(\.([a-z]|\d|[!#\$%&'\*\+\-\/=\?\^_`{\|}~]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])+)*)|((\x22)((((\x20|\x09)*(\x0d\x0a))?(\x20|\x09)+)?(([\x01-\x08\x0b\x0c\x0e-\x1f\x7f]|\x21|[\x23-\x5b]|[\x5d-\x7e]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])|(\\([\x01-\x09\x0b\x0c\x0d-\x7f]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF]))))*(((\x20|\x09)*(\x0d\x0a))?(\x20|\x09)+)?(\x22)))@((([a-z]|\d|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])|(([a-z]|\d|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])([a-z]|\d|-|\.|_|~|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])*([a-z]|\d|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])))\.)+(([a-z]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])|(([a-z]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])([a-z]|\d|-|\.|_|~|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])*([a-z]|[\u00A0-\uD7FF\uF900-\uFDCF\uFDF0-\uFFEF])))$/i;
// eslint-disable-next-line
const emailRegex = /^(([^<>()[\]\.,;:\s@\"]+(\.[^<>()[\]\.,;:\s@\"]+)*)|(\".+\"))@(([^<>()[\]\.,;:\s@\"]+\.)+[^<>()[\]\.,;:\s@\"]{2,})$/i;
export class ZodString extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.string) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.string,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const status = new ParseStatus();
        let ctx = undefined;
        for (const check of this._def.checks){
            if (check.kind === "min") {
                if (input.data.length < check.value) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_small,
                        minimum: check.value,
                        type: "string",
                        inclusive: true,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "max") {
                if (input.data.length > check.value) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_big,
                        maximum: check.value,
                        type: "string",
                        inclusive: true,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "email") {
                if (!emailRegex.test(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "email",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "uuid") {
                if (!uuidRegex.test(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "uuid",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "cuid") {
                if (!cuidRegex.test(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "cuid",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "url") {
                try {
                    new URL(input.data);
                } catch  {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "url",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "regex") {
                check.regex.lastIndex = 0;
                const testResult = check.regex.test(input.data);
                if (!testResult) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        validation: "regex",
                        code: ZodIssueCode.invalid_string,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "trim") {
                input.data = input.data.trim();
            } else {
                util.assertNever(check);
            }
        }
        return {
            status: status.value,
            value: input.data
        };
    }
    _regex = (regex, validation, message)=>this.refinement((data)=>regex.test(data), {
            validation,
            code: ZodIssueCode.invalid_string,
            ...errorUtil.errToObj(message)
        });
    _addCheck(check) {
        return new ZodString({
            ...this._def,
            checks: [
                ...this._def.checks,
                check
            ]
        });
    }
    email(message) {
        return this._addCheck({
            kind: "email",
            ...errorUtil.errToObj(message)
        });
    }
    url(message) {
        return this._addCheck({
            kind: "url",
            ...errorUtil.errToObj(message)
        });
    }
    uuid(message) {
        return this._addCheck({
            kind: "uuid",
            ...errorUtil.errToObj(message)
        });
    }
    cuid(message) {
        return this._addCheck({
            kind: "cuid",
            ...errorUtil.errToObj(message)
        });
    }
    regex(regex, message) {
        return this._addCheck({
            kind: "regex",
            regex: regex,
            ...errorUtil.errToObj(message)
        });
    }
    min(minLength, message) {
        return this._addCheck({
            kind: "min",
            value: minLength,
            ...errorUtil.errToObj(message)
        });
    }
    max(maxLength, message) {
        return this._addCheck({
            kind: "max",
            value: maxLength,
            ...errorUtil.errToObj(message)
        });
    }
    length(len, message) {
        return this.min(len, message).max(len, message);
    }
    /**
   * @deprecated Use z.string().min(1) instead.
   * @see {@link ZodString.min}
   */ nonempty = (message)=>this.min(1, errorUtil.errToObj(message));
    trim = ()=>new ZodString({
            ...this._def,
            checks: [
                ...this._def.checks,
                {
                    kind: "trim"
                }
            ]
        });
    get isEmail() {
        return !!this._def.checks.find((ch)=>ch.kind === "email");
    }
    get isURL() {
        return !!this._def.checks.find((ch)=>ch.kind === "url");
    }
    get isUUID() {
        return !!this._def.checks.find((ch)=>ch.kind === "uuid");
    }
    get isCUID() {
        return !!this._def.checks.find((ch)=>ch.kind === "cuid");
    }
    get minLength() {
        let min = -Infinity;
        this._def.checks.map((ch)=>{
            if (ch.kind === "min") {
                if (min === null || ch.value > min) {
                    min = ch.value;
                }
            }
        });
        return min;
    }
    get maxLength() {
        let max = null;
        this._def.checks.map((ch)=>{
            if (ch.kind === "max") {
                if (max === null || ch.value < max) {
                    max = ch.value;
                }
            }
        });
        return max;
    }
    static create = (params)=>{
        return new ZodString({
            checks: [],
            typeName: ZodFirstPartyTypeKind.ZodString,
            ...processCreateParams(params)
        });
    };
}
// https://stackoverflow.com/questions/3966484/why-does-modulus-operator-return-fractional-number-in-javascript/31711034#31711034
function floatSafeRemainder(val, step) {
    const valDecCount = (val.toString().split(".")[1] || "").length;
    const stepDecCount = (step.toString().split(".")[1] || "").length;
    const decCount = valDecCount > stepDecCount ? valDecCount : stepDecCount;
    const valInt = parseInt(val.toFixed(decCount).replace(".", ""));
    const stepInt = parseInt(step.toFixed(decCount).replace(".", ""));
    return valInt % stepInt / Math.pow(10, decCount);
}
export class ZodNumber extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.number) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.number,
                received: ctx.parsedType
            });
            return INVALID;
        }
        let ctx = undefined;
        const status = new ParseStatus();
        for (const check of this._def.checks){
            if (check.kind === "int") {
                if (!util.isInteger(input.data)) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.invalid_type,
                        expected: "integer",
                        received: "float",
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "min") {
                const tooSmall = check.inclusive ? input.data < check.value : input.data <= check.value;
                if (tooSmall) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_small,
                        minimum: check.value,
                        type: "number",
                        inclusive: check.inclusive,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "max") {
                const tooBig = check.inclusive ? input.data > check.value : input.data >= check.value;
                if (tooBig) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.too_big,
                        maximum: check.value,
                        type: "number",
                        inclusive: check.inclusive,
                        message: check.message
                    });
                    status.dirty();
                }
            } else if (check.kind === "multipleOf") {
                if (floatSafeRemainder(input.data, check.value) !== 0) {
                    ctx = this._getOrReturnCtx(input, ctx);
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.not_multiple_of,
                        multipleOf: check.value,
                        message: check.message
                    });
                    status.dirty();
                }
            } else {
                util.assertNever(check);
            }
        }
        return {
            status: status.value,
            value: input.data
        };
    }
    static create = (params)=>{
        return new ZodNumber({
            checks: [],
            typeName: ZodFirstPartyTypeKind.ZodNumber,
            ...processCreateParams(params)
        });
    };
    gte(value, message) {
        return this.setLimit("min", value, true, errorUtil.toString(message));
    }
    min = this.gte;
    gt(value, message) {
        return this.setLimit("min", value, false, errorUtil.toString(message));
    }
    lte(value, message) {
        return this.setLimit("max", value, true, errorUtil.toString(message));
    }
    max = this.lte;
    lt(value, message) {
        return this.setLimit("max", value, false, errorUtil.toString(message));
    }
    setLimit(kind, value, inclusive, message) {
        return new ZodNumber({
            ...this._def,
            checks: [
                ...this._def.checks,
                {
                    kind,
                    value,
                    inclusive,
                    message: errorUtil.toString(message)
                }, 
            ]
        });
    }
    _addCheck(check) {
        return new ZodNumber({
            ...this._def,
            checks: [
                ...this._def.checks,
                check
            ]
        });
    }
    int(message) {
        return this._addCheck({
            kind: "int",
            message: errorUtil.toString(message)
        });
    }
    positive(message) {
        return this._addCheck({
            kind: "min",
            value: 0,
            inclusive: false,
            message: errorUtil.toString(message)
        });
    }
    negative(message) {
        return this._addCheck({
            kind: "max",
            value: 0,
            inclusive: false,
            message: errorUtil.toString(message)
        });
    }
    nonpositive(message) {
        return this._addCheck({
            kind: "max",
            value: 0,
            inclusive: true,
            message: errorUtil.toString(message)
        });
    }
    nonnegative(message) {
        return this._addCheck({
            kind: "min",
            value: 0,
            inclusive: true,
            message: errorUtil.toString(message)
        });
    }
    multipleOf(value, message) {
        return this._addCheck({
            kind: "multipleOf",
            value: value,
            message: errorUtil.toString(message)
        });
    }
    step = this.multipleOf;
    get minValue() {
        let min = null;
        for (const ch of this._def.checks){
            if (ch.kind === "min") {
                if (min === null || ch.value > min) min = ch.value;
            }
        }
        return min;
    }
    get maxValue() {
        let max = null;
        for (const ch of this._def.checks){
            if (ch.kind === "max") {
                if (max === null || ch.value < max) max = ch.value;
            }
        }
        return max;
    }
    get isInt() {
        return !!this._def.checks.find((ch)=>ch.kind === "int");
    }
}
export class ZodBigInt extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.bigint) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.bigint,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
    static create = (params)=>{
        return new ZodBigInt({
            typeName: ZodFirstPartyTypeKind.ZodBigInt,
            ...processCreateParams(params)
        });
    };
}
export class ZodBoolean extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.boolean) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.boolean,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
    static create = (params)=>{
        return new ZodBoolean({
            typeName: ZodFirstPartyTypeKind.ZodBoolean,
            ...processCreateParams(params)
        });
    };
}
export class ZodDate extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.date) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.date,
                received: ctx.parsedType
            });
            return INVALID;
        }
        if (isNaN(input.data.getTime())) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_date
            });
            return INVALID;
        }
        return {
            status: "valid",
            value: new Date(input.data.getTime())
        };
    }
    static create = (params)=>{
        return new ZodDate({
            typeName: ZodFirstPartyTypeKind.ZodDate,
            ...processCreateParams(params)
        });
    };
}
export class ZodUndefined extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.undefined) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.undefined,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
    params;
    static create = (params)=>{
        return new ZodUndefined({
            typeName: ZodFirstPartyTypeKind.ZodUndefined,
            ...processCreateParams(params)
        });
    };
}
export class ZodNull extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.null) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.null,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
    static create = (params)=>{
        return new ZodNull({
            typeName: ZodFirstPartyTypeKind.ZodNull,
            ...processCreateParams(params)
        });
    };
}
export class ZodAny extends ZodType {
    // to prevent instances of other classes from extending ZodAny. this causes issues with catchall in ZodObject.
    _any = true;
    _parse(input) {
        return OK(input.data);
    }
    static create = (params)=>{
        return new ZodAny({
            typeName: ZodFirstPartyTypeKind.ZodAny,
            ...processCreateParams(params)
        });
    };
}
export class ZodUnknown extends ZodType {
    // required
    _unknown = true;
    _parse(input) {
        return OK(input.data);
    }
    static create = (params)=>{
        return new ZodUnknown({
            typeName: ZodFirstPartyTypeKind.ZodUnknown,
            ...processCreateParams(params)
        });
    };
}
export class ZodNever extends ZodType {
    _parse(input) {
        const ctx = this._getOrReturnCtx(input);
        addIssueToContext(ctx, {
            code: ZodIssueCode.invalid_type,
            expected: ZodParsedType.never,
            received: ctx.parsedType
        });
        return INVALID;
    }
    static create = (params)=>{
        return new ZodNever({
            typeName: ZodFirstPartyTypeKind.ZodNever,
            ...processCreateParams(params)
        });
    };
}
export class ZodVoid extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.undefined) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.void,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return OK(input.data);
    }
    static create = (params)=>{
        return new ZodVoid({
            typeName: ZodFirstPartyTypeKind.ZodVoid,
            ...processCreateParams(params)
        });
    };
}
export class ZodArray extends ZodType {
    _parse(input) {
        const { ctx , status  } = this._processInputParams(input);
        const def = this._def;
        if (ctx.parsedType !== ZodParsedType.array) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.array,
                received: ctx.parsedType
            });
            return INVALID;
        }
        if (def.minLength !== null) {
            if (ctx.data.length < def.minLength.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_small,
                    minimum: def.minLength.value,
                    type: "array",
                    inclusive: true,
                    message: def.minLength.message
                });
                status.dirty();
            }
        }
        if (def.maxLength !== null) {
            if (ctx.data.length > def.maxLength.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_big,
                    maximum: def.maxLength.value,
                    type: "array",
                    inclusive: true,
                    message: def.maxLength.message
                });
                status.dirty();
            }
        }
        if (ctx.common.async) {
            return Promise.all(ctx.data.map((item, i)=>{
                return def.type._parseAsync(new ParseInputLazyPath(ctx, item, ctx.path, i));
            })).then((result)=>{
                return ParseStatus.mergeArray(status, result);
            });
        }
        const result1 = ctx.data.map((item, i)=>{
            return def.type._parseSync(new ParseInputLazyPath(ctx, item, ctx.path, i));
        });
        return ParseStatus.mergeArray(status, result1);
    }
    get element() {
        return this._def.type;
    }
    min(minLength, message) {
        return new ZodArray({
            ...this._def,
            minLength: {
                value: minLength,
                message: errorUtil.toString(message)
            }
        });
    }
    max(maxLength, message) {
        return new ZodArray({
            ...this._def,
            maxLength: {
                value: maxLength,
                message: errorUtil.toString(message)
            }
        });
    }
    length(len, message) {
        return this.min(len, message).max(len, message);
    }
    nonempty(message) {
        return this.min(1, message);
    }
    static create = (schema, params)=>{
        return new ZodArray({
            type: schema,
            minLength: null,
            maxLength: null,
            typeName: ZodFirstPartyTypeKind.ZodArray,
            ...processCreateParams(params)
        });
    };
}
/////////////////////////////////////////
/////////////////////////////////////////
//////////                     //////////
//////////      ZodObject      //////////
//////////                     //////////
/////////////////////////////////////////
/////////////////////////////////////////
export var objectUtil;
(function(objectUtil1) {
    var mergeShapes = objectUtil1.mergeShapes = (first, second)=>{
        return {
            ...first,
            ...second
        };
    };
})(objectUtil || (objectUtil = {}));
const AugmentFactory = (def)=>(augmentation)=>{
        return new ZodObject({
            ...def,
            shape: ()=>({
                    ...def.shape(),
                    ...augmentation
                })
        });
    };
function deepPartialify(schema) {
    if (schema instanceof ZodObject) {
        const newShape = {};
        for(const key in schema.shape){
            const fieldSchema = schema.shape[key];
            newShape[key] = ZodOptional.create(deepPartialify(fieldSchema));
        }
        return new ZodObject({
            ...schema._def,
            shape: ()=>newShape
        });
    } else if (schema instanceof ZodArray) {
        return ZodArray.create(deepPartialify(schema.element));
    } else if (schema instanceof ZodOptional) {
        return ZodOptional.create(deepPartialify(schema.unwrap()));
    } else if (schema instanceof ZodNullable) {
        return ZodNullable.create(deepPartialify(schema.unwrap()));
    } else if (schema instanceof ZodTuple) {
        return ZodTuple.create(schema.items.map((item)=>deepPartialify(item)));
    } else {
        return schema;
    }
}
export class ZodObject extends ZodType {
    _shape;
    _unknownKeys;
    _catchall;
    _cached = null;
    _getCached() {
        if (this._cached !== null) return this._cached;
        const shape = this._def.shape();
        const keys = util.objectKeys(shape);
        return this._cached = {
            shape,
            keys
        };
    }
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.object) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.object,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const { status , ctx  } = this._processInputParams(input);
        const { shape , keys: shapeKeys  } = this._getCached();
        const extraKeys = [];
        for(const key2 in ctx.data){
            if (!shapeKeys.includes(key2)) {
                extraKeys.push(key2);
            }
        }
        const pairs = [];
        for (const key1 of shapeKeys){
            const keyValidator = shape[key1];
            const value = ctx.data[key1];
            pairs.push({
                key: {
                    status: "valid",
                    value: key1
                },
                value: keyValidator._parse(new ParseInputLazyPath(ctx, value, ctx.path, key1)),
                alwaysSet: key1 in ctx.data
            });
        }
        if (this._def.catchall instanceof ZodNever) {
            const unknownKeys = this._def.unknownKeys;
            if (unknownKeys === "passthrough") {
                for (const key of extraKeys){
                    pairs.push({
                        key: {
                            status: "valid",
                            value: key
                        },
                        value: {
                            status: "valid",
                            value: ctx.data[key]
                        }
                    });
                }
            } else if (unknownKeys === "strict") {
                if (extraKeys.length > 0) {
                    addIssueToContext(ctx, {
                        code: ZodIssueCode.unrecognized_keys,
                        keys: extraKeys
                    });
                    status.dirty();
                }
            } else if (unknownKeys === "strip") {} else {
                throw new Error(`Internal ZodObject error: invalid unknownKeys value.`);
            }
        } else {
            // run catchall validation
            const catchall = this._def.catchall;
            for (const key of extraKeys){
                const value = ctx.data[key];
                pairs.push({
                    key: {
                        status: "valid",
                        value: key
                    },
                    value: catchall._parse(new ParseInputLazyPath(ctx, value, ctx.path, key) //, ctx.child(key), value, getParsedType(value)
                    ),
                    alwaysSet: key in ctx.data
                });
            }
        }
        if (ctx.common.async) {
            return Promise.resolve().then(async ()=>{
                const syncPairs = [];
                for (const pair of pairs){
                    const key = await pair.key;
                    syncPairs.push({
                        key,
                        value: await pair.value,
                        alwaysSet: pair.alwaysSet
                    });
                }
                return syncPairs;
            }).then((syncPairs)=>{
                return ParseStatus.mergeObjectSync(status, syncPairs);
            });
        } else {
            return ParseStatus.mergeObjectSync(status, pairs);
        }
    }
    get shape() {
        return this._def.shape();
    }
    strict(message) {
        errorUtil.errToObj;
        return new ZodObject({
            ...this._def,
            unknownKeys: "strict",
            ...message !== undefined ? {
                errorMap: (issue, ctx)=>{
                    const defaultError = this._def.errorMap?.(issue, ctx).message ?? ctx.defaultError;
                    if (issue.code === "unrecognized_keys") return {
                        message: errorUtil.errToObj(message).message ?? defaultError
                    };
                    return {
                        message: defaultError
                    };
                }
            } : {}
        });
    }
    strip() {
        return new ZodObject({
            ...this._def,
            unknownKeys: "strip"
        });
    }
    passthrough() {
        return new ZodObject({
            ...this._def,
            unknownKeys: "passthrough"
        });
    }
    /**
   * @deprecated In most cases, this is no longer needed - unknown properties are now silently stripped.
   * If you want to pass through unknown properties, use `.passthrough()` instead.
   */ nonstrict = this.passthrough;
    augment = AugmentFactory(this._def);
    extend = AugmentFactory(this._def);
    setKey(key, schema) {
        return this.augment({
            [key]: schema
        });
    }
    /**
   * Prior to zod@1.0.12 there was a bug in the
   * inferred type of merged objects. Please
   * upgrade if you are experiencing issues.
   */ merge(merging) {
        // const mergedShape = objectUtil.mergeShapes(
        //   this._def.shape(),
        //   merging._def.shape()
        // );
        const merged = new ZodObject({
            unknownKeys: merging._def.unknownKeys,
            catchall: merging._def.catchall,
            shape: ()=>objectUtil.mergeShapes(this._def.shape(), merging._def.shape()),
            typeName: ZodFirstPartyTypeKind.ZodObject
        });
        return merged;
    }
    catchall(index) {
        return new ZodObject({
            ...this._def,
            catchall: index
        });
    }
    pick(mask) {
        const shape = {};
        util.objectKeys(mask).map((key)=>{
            // only add to shape if key corresponds to an element of the current shape
            if (this.shape[key]) shape[key] = this.shape[key];
        });
        return new ZodObject({
            ...this._def,
            shape: ()=>shape
        });
    }
    omit(mask) {
        const shape = {};
        util.objectKeys(this.shape).map((key)=>{
            if (util.objectKeys(mask).indexOf(key) === -1) {
                shape[key] = this.shape[key];
            }
        });
        return new ZodObject({
            ...this._def,
            shape: ()=>shape
        });
    }
    deepPartial() {
        return deepPartialify(this);
    }
    partial(mask) {
        const newShape = {};
        if (mask) {
            util.objectKeys(this.shape).map((key)=>{
                if (util.objectKeys(mask).indexOf(key) === -1) {
                    newShape[key] = this.shape[key];
                } else {
                    newShape[key] = this.shape[key].optional();
                }
            });
            return new ZodObject({
                ...this._def,
                shape: ()=>newShape
            });
        } else {
            for(const key in this.shape){
                const fieldSchema = this.shape[key];
                newShape[key] = fieldSchema.optional();
            }
        }
        return new ZodObject({
            ...this._def,
            shape: ()=>newShape
        });
    }
    required() {
        const newShape = {};
        for(const key in this.shape){
            const fieldSchema = this.shape[key];
            let newField = fieldSchema;
            while(newField instanceof ZodOptional){
                newField = newField._def.innerType;
            }
            newShape[key] = newField;
        }
        return new ZodObject({
            ...this._def,
            shape: ()=>newShape
        });
    }
    static create = (shape, params)=>{
        return new ZodObject({
            shape: ()=>shape,
            unknownKeys: "strip",
            catchall: ZodNever.create(),
            typeName: ZodFirstPartyTypeKind.ZodObject,
            ...processCreateParams(params)
        });
    };
    static strictCreate = (shape, params)=>{
        return new ZodObject({
            shape: ()=>shape,
            unknownKeys: "strict",
            catchall: ZodNever.create(),
            typeName: ZodFirstPartyTypeKind.ZodObject,
            ...processCreateParams(params)
        });
    };
    static lazycreate = (shape, params)=>{
        return new ZodObject({
            shape,
            unknownKeys: "strip",
            catchall: ZodNever.create(),
            typeName: ZodFirstPartyTypeKind.ZodObject,
            ...processCreateParams(params)
        });
    };
}
export class ZodUnion extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        const options = this._def.options;
        function handleResults(results) {
            // return first issue-free validation if it exists
            for (const result3 of results){
                if (result3.result.status === "valid") {
                    return result3.result;
                }
            }
            for (const result2 of results){
                if (result2.result.status === "dirty") {
                    // add issues from dirty option
                    ctx.common.issues.push(...result2.ctx.common.issues);
                    return result2.result;
                }
            }
            // return invalid
            const unionErrors = results.map((result)=>new ZodError(result.ctx.common.issues));
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_union,
                unionErrors
            });
            return INVALID;
        }
        if (ctx.common.async) {
            return Promise.all(options.map(async (option)=>{
                const childCtx = {
                    ...ctx,
                    common: {
                        ...ctx.common,
                        issues: []
                    },
                    parent: null
                };
                return {
                    result: await option._parseAsync({
                        data: ctx.data,
                        path: ctx.path,
                        parent: childCtx
                    }),
                    ctx: childCtx
                };
            })).then(handleResults);
        } else {
            let dirty = undefined;
            const issues1 = [];
            for (const option of options){
                const childCtx = {
                    ...ctx,
                    common: {
                        ...ctx.common,
                        issues: []
                    },
                    parent: null
                };
                const result = option._parseSync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: childCtx
                });
                if (result.status === "valid") {
                    return result;
                } else if (result.status === "dirty" && !dirty) {
                    dirty = {
                        result,
                        ctx: childCtx
                    };
                }
                if (childCtx.common.issues.length) {
                    issues1.push(childCtx.common.issues);
                }
            }
            if (dirty) {
                ctx.common.issues.push(...dirty.ctx.common.issues);
                return dirty.result;
            }
            const unionErrors = issues1.map((issues)=>new ZodError(issues));
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_union,
                unionErrors
            });
            return INVALID;
        }
    }
    get options() {
        return this._def.options;
    }
    static create = (types, params)=>{
        return new ZodUnion({
            options: types,
            typeName: ZodFirstPartyTypeKind.ZodUnion,
            ...processCreateParams(params)
        });
    };
}
export class ZodDiscriminatedUnion extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.object) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.object,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const discriminator = this.discriminator;
        const discriminatorValue = ctx.data[discriminator];
        const option = this.options.get(discriminatorValue);
        if (!option) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_union_discriminator,
                options: this.validDiscriminatorValues,
                path: [
                    discriminator
                ]
            });
            return INVALID;
        }
        if (ctx.common.async) {
            return option._parseAsync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            });
        } else {
            return option._parseSync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            });
        }
    }
    get discriminator() {
        return this._def.discriminator;
    }
    get validDiscriminatorValues() {
        return Array.from(this.options.keys());
    }
    get options() {
        return this._def.options;
    }
    /**
   * The constructor of the discriminated union schema. Its behaviour is very similar to that of the normal z.union() constructor.
   * However, it only allows a union of objects, all of which need to share a discriminator property. This property must
   * have a different value for each object in the union.
   * @param discriminator the name of the discriminator property
   * @param types an array of object schemas
   * @param params
   */ static create(discriminator, types, params) {
        // Get all the valid discriminator values
        const options = new Map();
        try {
            types.forEach((type)=>{
                const discriminatorValue = type.shape[discriminator].value;
                options.set(discriminatorValue, type);
            });
        } catch (e) {
            throw new Error("The discriminator value could not be extracted from all the provided schemas");
        }
        // Assert that all the discriminator values are unique
        if (options.size !== types.length) {
            throw new Error("Some of the discriminator values are not unique");
        }
        return new ZodDiscriminatedUnion({
            typeName: ZodFirstPartyTypeKind.ZodDiscriminatedUnion,
            discriminator,
            options,
            ...processCreateParams(params)
        });
    }
}
function mergeValues(a, b) {
    const aType = getParsedType(a);
    const bType = getParsedType(b);
    if (a === b) {
        return {
            valid: true,
            data: a
        };
    } else if (aType === ZodParsedType.object && bType === ZodParsedType.object) {
        const bKeys = util.objectKeys(b);
        const sharedKeys = util.objectKeys(a).filter((key)=>bKeys.indexOf(key) !== -1);
        const newObj = {
            ...a,
            ...b
        };
        for (const key3 of sharedKeys){
            const sharedValue = mergeValues(a[key3], b[key3]);
            if (!sharedValue.valid) {
                return {
                    valid: false
                };
            }
            newObj[key3] = sharedValue.data;
        }
        return {
            valid: true,
            data: newObj
        };
    } else if (aType === ZodParsedType.array && bType === ZodParsedType.array) {
        if (a.length !== b.length) {
            return {
                valid: false
            };
        }
        const newArray = [];
        for(let index = 0; index < a.length; index++){
            const itemA = a[index];
            const itemB = b[index];
            const sharedValue = mergeValues(itemA, itemB);
            if (!sharedValue.valid) {
                return {
                    valid: false
                };
            }
            newArray.push(sharedValue.data);
        }
        return {
            valid: true,
            data: newArray
        };
    } else if (aType === ZodParsedType.date && bType === ZodParsedType.date && +a === +b) {
        return {
            valid: true,
            data: a
        };
    } else {
        return {
            valid: false
        };
    }
}
export class ZodIntersection extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        const handleParsed = (parsedLeft, parsedRight)=>{
            if (isAborted(parsedLeft) || isAborted(parsedRight)) {
                return INVALID;
            }
            const merged = mergeValues(parsedLeft.value, parsedRight.value);
            if (!merged.valid) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.invalid_intersection_types
                });
                return INVALID;
            }
            if (isDirty(parsedLeft) || isDirty(parsedRight)) {
                status.dirty();
            }
            return {
                status: status.value,
                value: merged.data
            };
        };
        if (ctx.common.async) {
            return Promise.all([
                this._def.left._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }),
                this._def.right._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }), 
            ]).then(([left, right])=>handleParsed(left, right));
        } else {
            return handleParsed(this._def.left._parseSync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            }), this._def.right._parseSync({
                data: ctx.data,
                path: ctx.path,
                parent: ctx
            }));
        }
    }
    static create = (left, right, params)=>{
        return new ZodIntersection({
            left: left,
            right: right,
            typeName: ZodFirstPartyTypeKind.ZodIntersection,
            ...processCreateParams(params)
        });
    };
}
export class ZodTuple extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.array) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.array,
                received: ctx.parsedType
            });
            return INVALID;
        }
        if (ctx.data.length < this._def.items.length) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.too_small,
                minimum: this._def.items.length,
                inclusive: true,
                type: "array"
            });
            return INVALID;
        }
        const rest = this._def.rest;
        if (!rest && ctx.data.length > this._def.items.length) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.too_big,
                maximum: this._def.items.length,
                inclusive: true,
                type: "array"
            });
            status.dirty();
        }
        const items = ctx.data.map((item, itemIndex)=>{
            const schema = this._def.items[itemIndex] || this._def.rest;
            if (!schema) return null;
            return schema._parse(new ParseInputLazyPath(ctx, item, ctx.path, itemIndex));
        }).filter((x)=>!!x); // filter nulls
        if (ctx.common.async) {
            return Promise.all(items).then((results)=>{
                return ParseStatus.mergeArray(status, results);
            });
        } else {
            return ParseStatus.mergeArray(status, items);
        }
    }
    get items() {
        return this._def.items;
    }
    rest(rest) {
        return new ZodTuple({
            ...this._def,
            rest
        });
    }
    static create = (schemas, params)=>{
        return new ZodTuple({
            items: schemas,
            typeName: ZodFirstPartyTypeKind.ZodTuple,
            rest: null,
            ...processCreateParams(params)
        });
    };
}
export class ZodRecord extends ZodType {
    get keySchema() {
        return this._def.keyType;
    }
    get valueSchema() {
        return this._def.valueType;
    }
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.object) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.object,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const pairs = [];
        const keyType = this._def.keyType;
        const valueType = this._def.valueType;
        for(const key in ctx.data){
            pairs.push({
                key: keyType._parse(new ParseInputLazyPath(ctx, key, ctx.path, key)),
                value: valueType._parse(new ParseInputLazyPath(ctx, ctx.data[key], ctx.path, key))
            });
        }
        if (ctx.common.async) {
            return ParseStatus.mergeObjectAsync(status, pairs);
        } else {
            return ParseStatus.mergeObjectSync(status, pairs);
        }
    }
    get element() {
        return this._def.valueType;
    }
    static create(first, second, third) {
        if (second instanceof ZodType) {
            return new ZodRecord({
                keyType: first,
                valueType: second,
                typeName: ZodFirstPartyTypeKind.ZodRecord,
                ...processCreateParams(third)
            });
        }
        return new ZodRecord({
            keyType: ZodString.create(),
            valueType: first,
            typeName: ZodFirstPartyTypeKind.ZodRecord,
            ...processCreateParams(second)
        });
    }
}
export class ZodMap extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.map) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.map,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const keyType = this._def.keyType;
        const valueType = this._def.valueType;
        const pairs = [
            ...ctx.data.entries()
        ].map(([key, value], index)=>{
            return {
                key: keyType._parse(new ParseInputLazyPath(ctx, key, ctx.path, [
                    index,
                    "key"
                ])),
                value: valueType._parse(new ParseInputLazyPath(ctx, value, ctx.path, [
                    index,
                    "value"
                ]))
            };
        });
        if (ctx.common.async) {
            const finalMap = new Map();
            return Promise.resolve().then(async ()=>{
                for (const pair of pairs){
                    const key = await pair.key;
                    const value = await pair.value;
                    if (key.status === "aborted" || value.status === "aborted") {
                        return INVALID;
                    }
                    if (key.status === "dirty" || value.status === "dirty") {
                        status.dirty();
                    }
                    finalMap.set(key.value, value.value);
                }
                return {
                    status: status.value,
                    value: finalMap
                };
            });
        } else {
            const finalMap = new Map();
            for (const pair of pairs){
                const key = pair.key;
                const value = pair.value;
                if (key.status === "aborted" || value.status === "aborted") {
                    return INVALID;
                }
                if (key.status === "dirty" || value.status === "dirty") {
                    status.dirty();
                }
                finalMap.set(key.value, value.value);
            }
            return {
                status: status.value,
                value: finalMap
            };
        }
    }
    static create = (keyType, valueType, params)=>{
        return new ZodMap({
            valueType,
            keyType,
            typeName: ZodFirstPartyTypeKind.ZodMap,
            ...processCreateParams(params)
        });
    };
}
export class ZodSet extends ZodType {
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.set) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.set,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const def = this._def;
        if (def.minSize !== null) {
            if (ctx.data.size < def.minSize.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_small,
                    minimum: def.minSize.value,
                    type: "set",
                    inclusive: true,
                    message: def.minSize.message
                });
                status.dirty();
            }
        }
        if (def.maxSize !== null) {
            if (ctx.data.size > def.maxSize.value) {
                addIssueToContext(ctx, {
                    code: ZodIssueCode.too_big,
                    maximum: def.maxSize.value,
                    type: "set",
                    inclusive: true,
                    message: def.maxSize.message
                });
                status.dirty();
            }
        }
        const valueType = this._def.valueType;
        function finalizeSet(elements) {
            const parsedSet = new Set();
            for (const element of elements){
                if (element.status === "aborted") return INVALID;
                if (element.status === "dirty") status.dirty();
                parsedSet.add(element.value);
            }
            return {
                status: status.value,
                value: parsedSet
            };
        }
        const elements1 = [
            ...ctx.data.values()
        ].map((item, i)=>valueType._parse(new ParseInputLazyPath(ctx, item, ctx.path, i)));
        if (ctx.common.async) {
            return Promise.all(elements1).then((elements)=>finalizeSet(elements));
        } else {
            return finalizeSet(elements1);
        }
    }
    min(minSize, message) {
        return new ZodSet({
            ...this._def,
            minSize: {
                value: minSize,
                message: errorUtil.toString(message)
            }
        });
    }
    max(maxSize, message) {
        return new ZodSet({
            ...this._def,
            maxSize: {
                value: maxSize,
                message: errorUtil.toString(message)
            }
        });
    }
    size(size, message) {
        return this.min(size, message).max(size, message);
    }
    nonempty(message) {
        return this.min(1, message);
    }
    static create = (valueType, params)=>{
        return new ZodSet({
            valueType,
            minSize: null,
            maxSize: null,
            typeName: ZodFirstPartyTypeKind.ZodSet,
            ...processCreateParams(params)
        });
    };
}
export class ZodFunction extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.function) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.function,
                received: ctx.parsedType
            });
            return INVALID;
        }
        function makeArgsIssue(args, error) {
            return makeIssue({
                data: args,
                path: ctx.path,
                errorMaps: [
                    ctx.common.contextualErrorMap,
                    ctx.schemaErrorMap,
                    overrideErrorMap,
                    defaultErrorMap, 
                ].filter((x)=>!!x),
                issueData: {
                    code: ZodIssueCode.invalid_arguments,
                    argumentsError: error
                }
            });
        }
        function makeReturnsIssue(returns, error) {
            return makeIssue({
                data: returns,
                path: ctx.path,
                errorMaps: [
                    ctx.common.contextualErrorMap,
                    ctx.schemaErrorMap,
                    overrideErrorMap,
                    defaultErrorMap, 
                ].filter((x)=>!!x),
                issueData: {
                    code: ZodIssueCode.invalid_return_type,
                    returnTypeError: error
                }
            });
        }
        const params = {
            errorMap: ctx.common.contextualErrorMap
        };
        const fn = ctx.data;
        if (this._def.returns instanceof ZodPromise) {
            return OK(async (...args)=>{
                const error = new ZodError([]);
                const parsedArgs = await this._def.args.parseAsync(args, params).catch((e)=>{
                    error.addIssue(makeArgsIssue(args, e));
                    throw error;
                });
                const result = await fn(...parsedArgs);
                const parsedReturns = await this._def.returns._def.type.parseAsync(result, params).catch((e)=>{
                    error.addIssue(makeReturnsIssue(result, e));
                    throw error;
                });
                return parsedReturns;
            });
        } else {
            return OK((...args)=>{
                const parsedArgs = this._def.args.safeParse(args, params);
                if (!parsedArgs.success) {
                    throw new ZodError([
                        makeArgsIssue(args, parsedArgs.error)
                    ]);
                }
                const result = fn(...parsedArgs.data);
                const parsedReturns = this._def.returns.safeParse(result, params);
                if (!parsedReturns.success) {
                    throw new ZodError([
                        makeReturnsIssue(result, parsedReturns.error)
                    ]);
                }
                return parsedReturns.data;
            });
        }
    }
    parameters() {
        return this._def.args;
    }
    returnType() {
        return this._def.returns;
    }
    args(...items) {
        return new ZodFunction({
            ...this._def,
            args: ZodTuple.create(items).rest(ZodUnknown.create())
        });
    }
    returns(returnType) {
        return new ZodFunction({
            ...this._def,
            returns: returnType
        });
    }
    implement(func) {
        const validatedFunc = this.parse(func);
        return validatedFunc;
    }
    strictImplement(func) {
        const validatedFunc = this.parse(func);
        return validatedFunc;
    }
    validate = this.implement;
    static create = (args, returns, params)=>{
        return new ZodFunction({
            args: args ? args.rest(ZodUnknown.create()) : ZodTuple.create([]).rest(ZodUnknown.create()),
            returns: returns || ZodUnknown.create(),
            typeName: ZodFirstPartyTypeKind.ZodFunction,
            ...processCreateParams(params)
        });
    };
}
export class ZodLazy extends ZodType {
    get schema() {
        return this._def.getter();
    }
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        const lazySchema = this._def.getter();
        return lazySchema._parse({
            data: ctx.data,
            path: ctx.path,
            parent: ctx
        });
    }
    static create = (getter, params)=>{
        return new ZodLazy({
            getter: getter,
            typeName: ZodFirstPartyTypeKind.ZodLazy,
            ...processCreateParams(params)
        });
    };
}
export class ZodLiteral extends ZodType {
    _parse(input) {
        if (input.data !== this._def.value) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_literal,
                expected: this._def.value
            });
            return INVALID;
        }
        return {
            status: "valid",
            value: input.data
        };
    }
    get value() {
        return this._def.value;
    }
    static create = (value, params)=>{
        return new ZodLiteral({
            value: value,
            typeName: ZodFirstPartyTypeKind.ZodLiteral,
            ...processCreateParams(params)
        });
    };
}
function createZodEnum(values, params) {
    return new ZodEnum({
        values: values,
        typeName: ZodFirstPartyTypeKind.ZodEnum,
        ...processCreateParams(params)
    });
}
export class ZodEnum extends ZodType {
    _parse(input) {
        if (typeof input.data !== "string") {
            const ctx = this._getOrReturnCtx(input);
            const expectedValues = this._def.values;
            addIssueToContext(ctx, {
                expected: util.joinValues(expectedValues),
                received: ctx.parsedType,
                code: ZodIssueCode.invalid_type
            });
            return INVALID;
        }
        if (this._def.values.indexOf(input.data) === -1) {
            const ctx = this._getOrReturnCtx(input);
            const expectedValues = this._def.values;
            addIssueToContext(ctx, {
                received: ctx.data,
                code: ZodIssueCode.invalid_enum_value,
                options: expectedValues
            });
            return INVALID;
        }
        return OK(input.data);
    }
    get options() {
        return this._def.values;
    }
    get enum() {
        const enumValues = {};
        for (const val of this._def.values){
            enumValues[val] = val;
        }
        return enumValues;
    }
    get Values() {
        const enumValues = {};
        for (const val of this._def.values){
            enumValues[val] = val;
        }
        return enumValues;
    }
    get Enum() {
        const enumValues = {};
        for (const val of this._def.values){
            enumValues[val] = val;
        }
        return enumValues;
    }
    static create = createZodEnum;
}
export class ZodNativeEnum extends ZodType {
    _parse(input) {
        const nativeEnumValues = util.getValidEnumValues(this._def.values);
        const ctx = this._getOrReturnCtx(input);
        if (ctx.parsedType !== ZodParsedType.string && ctx.parsedType !== ZodParsedType.number) {
            const expectedValues = util.objectValues(nativeEnumValues);
            addIssueToContext(ctx, {
                expected: util.joinValues(expectedValues),
                received: ctx.parsedType,
                code: ZodIssueCode.invalid_type
            });
            return INVALID;
        }
        if (nativeEnumValues.indexOf(input.data) === -1) {
            const expectedValues = util.objectValues(nativeEnumValues);
            addIssueToContext(ctx, {
                received: ctx.data,
                code: ZodIssueCode.invalid_enum_value,
                options: expectedValues
            });
            return INVALID;
        }
        return OK(input.data);
    }
    get enum() {
        return this._def.values;
    }
    static create = (values, params)=>{
        return new ZodNativeEnum({
            values: values,
            typeName: ZodFirstPartyTypeKind.ZodNativeEnum,
            ...processCreateParams(params)
        });
    };
}
export class ZodPromise extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        if (ctx.parsedType !== ZodParsedType.promise && ctx.common.async === false) {
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.promise,
                received: ctx.parsedType
            });
            return INVALID;
        }
        const promisified = ctx.parsedType === ZodParsedType.promise ? ctx.data : Promise.resolve(ctx.data);
        return OK(promisified.then((data)=>{
            return this._def.type.parseAsync(data, {
                path: ctx.path,
                errorMap: ctx.common.contextualErrorMap
            });
        }));
    }
    static create = (schema, params)=>{
        return new ZodPromise({
            type: schema,
            typeName: ZodFirstPartyTypeKind.ZodPromise,
            ...processCreateParams(params)
        });
    };
}
export class ZodEffects extends ZodType {
    innerType() {
        return this._def.schema;
    }
    _parse(input) {
        const { status , ctx  } = this._processInputParams(input);
        const effect = this._def.effect || null;
        if (effect.type === "preprocess") {
            const processed1 = effect.transform(ctx.data);
            if (ctx.common.async) {
                return Promise.resolve(processed1).then((processed)=>{
                    return this._def.schema._parseAsync({
                        data: processed,
                        path: ctx.path,
                        parent: ctx
                    });
                });
            } else {
                return this._def.schema._parseSync({
                    data: processed1,
                    path: ctx.path,
                    parent: ctx
                });
            }
        }
        const checkCtx = {
            addIssue: (arg)=>{
                addIssueToContext(ctx, arg);
                if (arg.fatal) {
                    status.abort();
                } else {
                    status.dirty();
                }
            },
            get path () {
                return ctx.path;
            }
        };
        checkCtx.addIssue = checkCtx.addIssue.bind(checkCtx);
        if (effect.type === "refinement") {
            const executeRefinement = (acc)=>{
                const result = effect.refinement(acc, checkCtx);
                if (ctx.common.async) {
                    return Promise.resolve(result);
                }
                if (result instanceof Promise) {
                    throw new Error("Async refinement encountered during synchronous parse operation. Use .parseAsync instead.");
                }
                return acc;
            };
            if (ctx.common.async === false) {
                const inner = this._def.schema._parseSync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                });
                if (inner.status === "aborted") return INVALID;
                if (inner.status === "dirty") status.dirty();
                // return value is ignored
                executeRefinement(inner.value);
                return {
                    status: status.value,
                    value: inner.value
                };
            } else {
                return this._def.schema._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }).then((inner)=>{
                    if (inner.status === "aborted") return INVALID;
                    if (inner.status === "dirty") status.dirty();
                    return executeRefinement(inner.value).then(()=>{
                        return {
                            status: status.value,
                            value: inner.value
                        };
                    });
                });
            }
        }
        if (effect.type === "transform") {
            if (ctx.common.async === false) {
                const base = this._def.schema._parseSync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                });
                // if (base.status === "aborted") return INVALID;
                // if (base.status === "dirty") {
                //   return { status: "dirty", value: base.value };
                // }
                if (!isValid(base)) return base;
                const result = effect.transform(base.value, checkCtx);
                if (result instanceof Promise) {
                    throw new Error(`Asynchronous transform encountered during synchronous parse operation. Use .parseAsync instead.`);
                }
                return {
                    status: status.value,
                    value: result
                };
            } else {
                return this._def.schema._parseAsync({
                    data: ctx.data,
                    path: ctx.path,
                    parent: ctx
                }).then((base)=>{
                    if (!isValid(base)) return base;
                    // if (base.status === "aborted") return INVALID;
                    // if (base.status === "dirty") {
                    //   return { status: "dirty", value: base.value };
                    // }
                    return Promise.resolve(effect.transform(base.value, checkCtx)).then((result)=>({
                            status: status.value,
                            value: result
                        }));
                });
            }
        }
        util.assertNever(effect);
    }
    static create = (schema, effect, params)=>{
        return new ZodEffects({
            schema,
            typeName: ZodFirstPartyTypeKind.ZodEffects,
            effect,
            ...processCreateParams(params)
        });
    };
    static createWithPreprocess = (preprocess, schema, params)=>{
        return new ZodEffects({
            schema,
            effect: {
                type: "preprocess",
                transform: preprocess
            },
            typeName: ZodFirstPartyTypeKind.ZodEffects,
            ...processCreateParams(params)
        });
    };
}
export { ZodEffects as ZodTransformer };
export class ZodOptional extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType === ZodParsedType.undefined) {
            return OK(undefined);
        }
        return this._def.innerType._parse(input);
    }
    unwrap() {
        return this._def.innerType;
    }
    static create = (type, params)=>{
        return new ZodOptional({
            innerType: type,
            typeName: ZodFirstPartyTypeKind.ZodOptional,
            ...processCreateParams(params)
        });
    };
}
export class ZodNullable extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType === ZodParsedType.null) {
            return OK(null);
        }
        return this._def.innerType._parse(input);
    }
    unwrap() {
        return this._def.innerType;
    }
    static create = (type, params)=>{
        return new ZodNullable({
            innerType: type,
            typeName: ZodFirstPartyTypeKind.ZodNullable,
            ...processCreateParams(params)
        });
    };
}
export class ZodDefault extends ZodType {
    _parse(input) {
        const { ctx  } = this._processInputParams(input);
        let data = ctx.data;
        if (ctx.parsedType === ZodParsedType.undefined) {
            data = this._def.defaultValue();
        }
        return this._def.innerType._parse({
            data,
            path: ctx.path,
            parent: ctx
        });
    }
    removeDefault() {
        return this._def.innerType;
    }
    static create = (type, params)=>{
        return new ZodOptional({
            innerType: type,
            typeName: ZodFirstPartyTypeKind.ZodOptional,
            ...processCreateParams(params)
        });
    };
}
export class ZodNaN extends ZodType {
    _parse(input) {
        const parsedType = this._getType(input);
        if (parsedType !== ZodParsedType.nan) {
            const ctx = this._getOrReturnCtx(input);
            addIssueToContext(ctx, {
                code: ZodIssueCode.invalid_type,
                expected: ZodParsedType.nan,
                received: ctx.parsedType
            });
            return INVALID;
        }
        return {
            status: "valid",
            value: input.data
        };
    }
    static create = (params)=>{
        return new ZodNaN({
            typeName: ZodFirstPartyTypeKind.ZodNaN,
            ...processCreateParams(params)
        });
    };
}
export const custom = (check, params = {}, fatal)=>{
    if (check) return ZodAny.create().superRefine((data, ctx)=>{
        if (!check(data)) {
            const p = typeof params === "function" ? params(data) : params;
            const p2 = typeof p === "string" ? {
                message: p
            } : p;
            ctx.addIssue({
                code: "custom",
                ...p2,
                fatal
            });
        }
    });
    return ZodAny.create();
};
export { ZodType as Schema, ZodType as ZodSchema };
export const late = {
    object: ZodObject.lazycreate
};
export var ZodFirstPartyTypeKind;
(function(ZodFirstPartyTypeKind) {
    ZodFirstPartyTypeKind["ZodString"] = "ZodString";
    ZodFirstPartyTypeKind["ZodNumber"] = "ZodNumber";
    ZodFirstPartyTypeKind["ZodNaN"] = "ZodNaN";
    ZodFirstPartyTypeKind["ZodBigInt"] = "ZodBigInt";
    ZodFirstPartyTypeKind["ZodBoolean"] = "ZodBoolean";
    ZodFirstPartyTypeKind["ZodDate"] = "ZodDate";
    ZodFirstPartyTypeKind["ZodUndefined"] = "ZodUndefined";
    ZodFirstPartyTypeKind["ZodNull"] = "ZodNull";
    ZodFirstPartyTypeKind["ZodAny"] = "ZodAny";
    ZodFirstPartyTypeKind["ZodUnknown"] = "ZodUnknown";
    ZodFirstPartyTypeKind["ZodNever"] = "ZodNever";
    ZodFirstPartyTypeKind["ZodVoid"] = "ZodVoid";
    ZodFirstPartyTypeKind["ZodArray"] = "ZodArray";
    ZodFirstPartyTypeKind["ZodObject"] = "ZodObject";
    ZodFirstPartyTypeKind["ZodUnion"] = "ZodUnion";
    ZodFirstPartyTypeKind["ZodDiscriminatedUnion"] = "ZodDiscriminatedUnion";
    ZodFirstPartyTypeKind["ZodIntersection"] = "ZodIntersection";
    ZodFirstPartyTypeKind["ZodTuple"] = "ZodTuple";
    ZodFirstPartyTypeKind["ZodRecord"] = "ZodRecord";
    ZodFirstPartyTypeKind["ZodMap"] = "ZodMap";
    ZodFirstPartyTypeKind["ZodSet"] = "ZodSet";
    ZodFirstPartyTypeKind["ZodFunction"] = "ZodFunction";
    ZodFirstPartyTypeKind["ZodLazy"] = "ZodLazy";
    ZodFirstPartyTypeKind["ZodLiteral"] = "ZodLiteral";
    ZodFirstPartyTypeKind["ZodEnum"] = "ZodEnum";
    ZodFirstPartyTypeKind["ZodEffects"] = "ZodEffects";
    ZodFirstPartyTypeKind["ZodNativeEnum"] = "ZodNativeEnum";
    ZodFirstPartyTypeKind["ZodOptional"] = "ZodOptional";
    ZodFirstPartyTypeKind["ZodNullable"] = "ZodNullable";
    ZodFirstPartyTypeKind["ZodDefault"] = "ZodDefault";
    ZodFirstPartyTypeKind["ZodPromise"] = "ZodPromise";
})(ZodFirstPartyTypeKind || (ZodFirstPartyTypeKind = {}));
const instanceOfType = (cls, params = {
    message: `Input not instance of ${cls.name}`
})=>custom((data)=>data instanceof cls, params, true);
const stringType = ZodString.create;
const numberType = ZodNumber.create;
const nanType = ZodNaN.create;
const bigIntType = ZodBigInt.create;
const booleanType = ZodBoolean.create;
const dateType = ZodDate.create;
const undefinedType = ZodUndefined.create;
const nullType = ZodNull.create;
const anyType = ZodAny.create;
const unknownType = ZodUnknown.create;
const neverType = ZodNever.create;
const voidType = ZodVoid.create;
const arrayType = ZodArray.create;
const objectType = ZodObject.create;
const strictObjectType = ZodObject.strictCreate;
const unionType = ZodUnion.create;
const discriminatedUnionType = ZodDiscriminatedUnion.create;
const intersectionType = ZodIntersection.create;
const tupleType = ZodTuple.create;
const recordType = ZodRecord.create;
const mapType = ZodMap.create;
const setType = ZodSet.create;
const functionType = ZodFunction.create;
const lazyType = ZodLazy.create;
const literalType = ZodLiteral.create;
const enumType = ZodEnum.create;
const nativeEnumType = ZodNativeEnum.create;
const promiseType = ZodPromise.create;
const effectsType = ZodEffects.create;
const optionalType = ZodOptional.create;
const nullableType = ZodNullable.create;
const preprocessType = ZodEffects.createWithPreprocess;
const ostring = ()=>stringType().optional();
const onumber = ()=>numberType().optional();
const oboolean = ()=>booleanType().optional();
export { anyType as any, arrayType as array, bigIntType as bigint, booleanType as boolean, dateType as date, discriminatedUnionType as discriminatedUnion, effectsType as effect, enumType as enum, functionType as function, instanceOfType as instanceof, intersectionType as intersection, lazyType as lazy, literalType as literal, mapType as map, nanType as nan, nativeEnumType as nativeEnum, neverType as never, nullType as null, nullableType as nullable, numberType as number, objectType as object, oboolean, onumber, optionalType as optional, ostring, preprocessType as preprocess, promiseType as promise, recordType as record, setType as set, strictObjectType as strictObject, stringType as string, effectsType as transformer, tupleType as tuple, undefinedType as undefined, unionType as union, unknownType as unknown, voidType as void };
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImh0dHBzOi8vZGVuby5sYW5kL3gvem9kQHYzLjE3LjMvdHlwZXMudHMiXSwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0IHsgZXJyb3JVdGlsIH0gZnJvbSBcIi4vaGVscGVycy9lcnJvclV0aWwudHNcIjtcbmltcG9ydCB7XG4gIGFkZElzc3VlVG9Db250ZXh0LFxuICBBc3luY1BhcnNlUmV0dXJuVHlwZSxcbiAgRElSVFksXG4gIElOVkFMSUQsXG4gIGlzQWJvcnRlZCxcbiAgaXNBc3luYyxcbiAgaXNEaXJ0eSxcbiAgaXNWYWxpZCxcbiAgbWFrZUlzc3VlLFxuICBPSyxcbiAgUGFyc2VDb250ZXh0LFxuICBQYXJzZUlucHV0LFxuICBQYXJzZVBhcmFtcyxcbiAgUGFyc2VQYXRoLFxuICBQYXJzZVJldHVyblR5cGUsXG4gIFBhcnNlU3RhdHVzLFxuICBTeW5jUGFyc2VSZXR1cm5UeXBlLFxufSBmcm9tIFwiLi9oZWxwZXJzL3BhcnNlVXRpbC50c1wiO1xuaW1wb3J0IHsgcGFydGlhbFV0aWwgfSBmcm9tIFwiLi9oZWxwZXJzL3BhcnRpYWxVdGlsLnRzXCI7XG5pbXBvcnQgeyBQcmltaXRpdmUgfSBmcm9tIFwiLi9oZWxwZXJzL3R5cGVBbGlhc2VzLnRzXCI7XG5pbXBvcnQgeyBnZXRQYXJzZWRUeXBlLCB1dGlsLCBab2RQYXJzZWRUeXBlIH0gZnJvbSBcIi4vaGVscGVycy91dGlsLnRzXCI7XG5pbXBvcnQge1xuICBkZWZhdWx0RXJyb3JNYXAsXG4gIElzc3VlRGF0YSxcbiAgb3ZlcnJpZGVFcnJvck1hcCxcbiAgU3RyaW5nVmFsaWRhdGlvbixcbiAgWm9kQ3VzdG9tSXNzdWUsXG4gIFpvZEVycm9yLFxuICBab2RFcnJvck1hcCxcbiAgWm9kSXNzdWUsXG4gIFpvZElzc3VlQ29kZSxcbn0gZnJvbSBcIi4vWm9kRXJyb3IudHNcIjtcblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZFR5cGUgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5cbmV4cG9ydCB0eXBlIFJlZmluZW1lbnRDdHggPSB7XG4gIGFkZElzc3VlOiAoYXJnOiBJc3N1ZURhdGEpID0+IHZvaWQ7XG4gIHBhdGg6IChzdHJpbmcgfCBudW1iZXIpW107XG59O1xuZXhwb3J0IHR5cGUgWm9kUmF3U2hhcGUgPSB7IFtrOiBzdHJpbmddOiBab2RUeXBlQW55IH07XG5leHBvcnQgdHlwZSBab2RUeXBlQW55ID0gWm9kVHlwZTxhbnksIGFueSwgYW55PjtcbmV4cG9ydCB0eXBlIFR5cGVPZjxUIGV4dGVuZHMgWm9kVHlwZTxhbnksIGFueSwgYW55Pj4gPSBUW1wiX291dHB1dFwiXTtcbmV4cG9ydCB0eXBlIGlucHV0PFQgZXh0ZW5kcyBab2RUeXBlPGFueSwgYW55LCBhbnk+PiA9IFRbXCJfaW5wdXRcIl07XG5leHBvcnQgdHlwZSBvdXRwdXQ8VCBleHRlbmRzIFpvZFR5cGU8YW55LCBhbnksIGFueT4+ID0gVFtcIl9vdXRwdXRcIl07XG5leHBvcnQgdHlwZSB7IFR5cGVPZiBhcyBpbmZlciB9O1xuXG5leHBvcnQgdHlwZSBDdXN0b21FcnJvclBhcmFtcyA9IFBhcnRpYWw8dXRpbC5PbWl0PFpvZEN1c3RvbUlzc3VlLCBcImNvZGVcIj4+O1xuZXhwb3J0IGludGVyZmFjZSBab2RUeXBlRGVmIHtcbiAgZXJyb3JNYXA/OiBab2RFcnJvck1hcDtcbiAgZGVzY3JpcHRpb24/OiBzdHJpbmc7XG59XG5cbmNsYXNzIFBhcnNlSW5wdXRMYXp5UGF0aCBpbXBsZW1lbnRzIFBhcnNlSW5wdXQge1xuICBwYXJlbnQ6IFBhcnNlQ29udGV4dDtcbiAgZGF0YTogYW55O1xuICBfcGF0aDogUGFyc2VQYXRoO1xuICBfa2V5OiBzdHJpbmcgfCBudW1iZXIgfCAoc3RyaW5nIHwgbnVtYmVyKVtdO1xuICBjb25zdHJ1Y3RvcihcbiAgICBwYXJlbnQ6IFBhcnNlQ29udGV4dCxcbiAgICB2YWx1ZTogYW55LFxuICAgIHBhdGg6IFBhcnNlUGF0aCxcbiAgICBrZXk6IHN0cmluZyB8IG51bWJlciB8IChzdHJpbmcgfCBudW1iZXIpW11cbiAgKSB7XG4gICAgdGhpcy5wYXJlbnQgPSBwYXJlbnQ7XG4gICAgdGhpcy5kYXRhID0gdmFsdWU7XG4gICAgdGhpcy5fcGF0aCA9IHBhdGg7XG4gICAgdGhpcy5fa2V5ID0ga2V5O1xuICB9XG4gIGdldCBwYXRoKCkge1xuICAgIHJldHVybiB0aGlzLl9wYXRoLmNvbmNhdCh0aGlzLl9rZXkpO1xuICB9XG59XG5cbmNvbnN0IGhhbmRsZVJlc3VsdCA9IDxJbnB1dCwgT3V0cHV0PihcbiAgY3R4OiBQYXJzZUNvbnRleHQsXG4gIHJlc3VsdDogU3luY1BhcnNlUmV0dXJuVHlwZTxPdXRwdXQ+XG4pOlxuICB8IHsgc3VjY2VzczogdHJ1ZTsgZGF0YTogT3V0cHV0IH1cbiAgfCB7IHN1Y2Nlc3M6IGZhbHNlOyBlcnJvcjogWm9kRXJyb3I8SW5wdXQ+IH0gPT4ge1xuICBpZiAoaXNWYWxpZChyZXN1bHQpKSB7XG4gICAgcmV0dXJuIHsgc3VjY2VzczogdHJ1ZSwgZGF0YTogcmVzdWx0LnZhbHVlIH07XG4gIH0gZWxzZSB7XG4gICAgaWYgKCFjdHguY29tbW9uLmlzc3Vlcy5sZW5ndGgpIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihcIlZhbGlkYXRpb24gZmFpbGVkIGJ1dCBubyBpc3N1ZXMgZGV0ZWN0ZWQuXCIpO1xuICAgIH1cbiAgICBjb25zdCBlcnJvciA9IG5ldyBab2RFcnJvcihjdHguY29tbW9uLmlzc3Vlcyk7XG4gICAgcmV0dXJuIHsgc3VjY2VzczogZmFsc2UsIGVycm9yIH07XG4gIH1cbn07XG5cbnR5cGUgUmF3Q3JlYXRlUGFyYW1zID1cbiAgfCB7XG4gICAgICBlcnJvck1hcD86IFpvZEVycm9yTWFwO1xuICAgICAgaW52YWxpZF90eXBlX2Vycm9yPzogc3RyaW5nO1xuICAgICAgcmVxdWlyZWRfZXJyb3I/OiBzdHJpbmc7XG4gICAgICBkZXNjcmlwdGlvbj86IHN0cmluZztcbiAgICB9XG4gIHwgdW5kZWZpbmVkO1xudHlwZSBQcm9jZXNzZWRDcmVhdGVQYXJhbXMgPSB7IGVycm9yTWFwPzogWm9kRXJyb3JNYXA7IGRlc2NyaXB0aW9uPzogc3RyaW5nIH07XG5mdW5jdGlvbiBwcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtczogUmF3Q3JlYXRlUGFyYW1zKTogUHJvY2Vzc2VkQ3JlYXRlUGFyYW1zIHtcbiAgaWYgKCFwYXJhbXMpIHJldHVybiB7fTtcbiAgY29uc3QgeyBlcnJvck1hcCwgaW52YWxpZF90eXBlX2Vycm9yLCByZXF1aXJlZF9lcnJvciwgZGVzY3JpcHRpb24gfSA9IHBhcmFtcztcbiAgaWYgKGVycm9yTWFwICYmIChpbnZhbGlkX3R5cGVfZXJyb3IgfHwgcmVxdWlyZWRfZXJyb3IpKSB7XG4gICAgdGhyb3cgbmV3IEVycm9yKFxuICAgICAgYENhbid0IHVzZSBcImludmFsaWRcIiBvciBcInJlcXVpcmVkXCIgaW4gY29uanVuY3Rpb24gd2l0aCBjdXN0b20gZXJyb3IgbWFwLmBcbiAgICApO1xuICB9XG4gIGlmIChlcnJvck1hcCkgcmV0dXJuIHsgZXJyb3JNYXA6IGVycm9yTWFwLCBkZXNjcmlwdGlvbiB9O1xuICBjb25zdCBjdXN0b21NYXA6IFpvZEVycm9yTWFwID0gKGlzcywgY3R4KSA9PiB7XG4gICAgaWYgKGlzcy5jb2RlICE9PSBcImludmFsaWRfdHlwZVwiKSByZXR1cm4geyBtZXNzYWdlOiBjdHguZGVmYXVsdEVycm9yIH07XG4gICAgaWYgKHR5cGVvZiBjdHguZGF0YSA9PT0gXCJ1bmRlZmluZWRcIiAmJiByZXF1aXJlZF9lcnJvcilcbiAgICAgIHJldHVybiB7IG1lc3NhZ2U6IHJlcXVpcmVkX2Vycm9yIH07XG4gICAgaWYgKHBhcmFtcy5pbnZhbGlkX3R5cGVfZXJyb3IpXG4gICAgICByZXR1cm4geyBtZXNzYWdlOiBwYXJhbXMuaW52YWxpZF90eXBlX2Vycm9yIH07XG4gICAgcmV0dXJuIHsgbWVzc2FnZTogY3R4LmRlZmF1bHRFcnJvciB9O1xuICB9O1xuICByZXR1cm4geyBlcnJvck1hcDogY3VzdG9tTWFwLCBkZXNjcmlwdGlvbiB9O1xufVxuXG5leHBvcnQgdHlwZSBTYWZlUGFyc2VTdWNjZXNzPE91dHB1dD4gPSB7IHN1Y2Nlc3M6IHRydWU7IGRhdGE6IE91dHB1dCB9O1xuZXhwb3J0IHR5cGUgU2FmZVBhcnNlRXJyb3I8SW5wdXQ+ID0geyBzdWNjZXNzOiBmYWxzZTsgZXJyb3I6IFpvZEVycm9yPElucHV0PiB9O1xuXG5leHBvcnQgdHlwZSBTYWZlUGFyc2VSZXR1cm5UeXBlPElucHV0LCBPdXRwdXQ+ID1cbiAgfCBTYWZlUGFyc2VTdWNjZXNzPE91dHB1dD5cbiAgfCBTYWZlUGFyc2VFcnJvcjxJbnB1dD47XG5cbmV4cG9ydCBhYnN0cmFjdCBjbGFzcyBab2RUeXBlPFxuICBPdXRwdXQgPSBhbnksXG4gIERlZiBleHRlbmRzIFpvZFR5cGVEZWYgPSBab2RUeXBlRGVmLFxuICBJbnB1dCA9IE91dHB1dFxuPiB7XG4gIHJlYWRvbmx5IF90eXBlITogT3V0cHV0O1xuICByZWFkb25seSBfb3V0cHV0ITogT3V0cHV0O1xuICByZWFkb25seSBfaW5wdXQhOiBJbnB1dDtcbiAgcmVhZG9ubHkgX2RlZiE6IERlZjtcblxuICBnZXQgZGVzY3JpcHRpb24oKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5kZXNjcmlwdGlvbjtcbiAgfVxuXG4gIGFic3RyYWN0IF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTxPdXRwdXQ+O1xuXG4gIF9nZXRUeXBlKGlucHV0OiBQYXJzZUlucHV0KTogc3RyaW5nIHtcbiAgICByZXR1cm4gZ2V0UGFyc2VkVHlwZShpbnB1dC5kYXRhKTtcbiAgfVxuXG4gIF9nZXRPclJldHVybkN0eChcbiAgICBpbnB1dDogUGFyc2VJbnB1dCxcbiAgICBjdHg/OiBQYXJzZUNvbnRleHQgfCB1bmRlZmluZWRcbiAgKTogUGFyc2VDb250ZXh0IHtcbiAgICByZXR1cm4gKFxuICAgICAgY3R4IHx8IHtcbiAgICAgICAgY29tbW9uOiBpbnB1dC5wYXJlbnQuY29tbW9uLFxuICAgICAgICBkYXRhOiBpbnB1dC5kYXRhLFxuXG4gICAgICAgIHBhcnNlZFR5cGU6IGdldFBhcnNlZFR5cGUoaW5wdXQuZGF0YSksXG5cbiAgICAgICAgc2NoZW1hRXJyb3JNYXA6IHRoaXMuX2RlZi5lcnJvck1hcCxcbiAgICAgICAgcGF0aDogaW5wdXQucGF0aCxcbiAgICAgICAgcGFyZW50OiBpbnB1dC5wYXJlbnQsXG4gICAgICB9XG4gICAgKTtcbiAgfVxuXG4gIF9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQ6IFBhcnNlSW5wdXQpOiB7XG4gICAgc3RhdHVzOiBQYXJzZVN0YXR1cztcbiAgICBjdHg6IFBhcnNlQ29udGV4dDtcbiAgfSB7XG4gICAgcmV0dXJuIHtcbiAgICAgIHN0YXR1czogbmV3IFBhcnNlU3RhdHVzKCksXG4gICAgICBjdHg6IHtcbiAgICAgICAgY29tbW9uOiBpbnB1dC5wYXJlbnQuY29tbW9uLFxuICAgICAgICBkYXRhOiBpbnB1dC5kYXRhLFxuXG4gICAgICAgIHBhcnNlZFR5cGU6IGdldFBhcnNlZFR5cGUoaW5wdXQuZGF0YSksXG5cbiAgICAgICAgc2NoZW1hRXJyb3JNYXA6IHRoaXMuX2RlZi5lcnJvck1hcCxcbiAgICAgICAgcGF0aDogaW5wdXQucGF0aCxcbiAgICAgICAgcGFyZW50OiBpbnB1dC5wYXJlbnQsXG4gICAgICB9LFxuICAgIH07XG4gIH1cblxuICBfcGFyc2VTeW5jKGlucHV0OiBQYXJzZUlucHV0KTogU3luY1BhcnNlUmV0dXJuVHlwZTxPdXRwdXQ+IHtcbiAgICBjb25zdCByZXN1bHQgPSB0aGlzLl9wYXJzZShpbnB1dCk7XG4gICAgaWYgKGlzQXN5bmMocmVzdWx0KSkge1xuICAgICAgdGhyb3cgbmV3IEVycm9yKFwiU3luY2hyb25vdXMgcGFyc2UgZW5jb3VudGVyZWQgcHJvbWlzZS5cIik7XG4gICAgfVxuICAgIHJldHVybiByZXN1bHQ7XG4gIH1cblxuICBfcGFyc2VBc3luYyhpbnB1dDogUGFyc2VJbnB1dCk6IEFzeW5jUGFyc2VSZXR1cm5UeXBlPE91dHB1dD4ge1xuICAgIGNvbnN0IHJlc3VsdCA9IHRoaXMuX3BhcnNlKGlucHV0KTtcblxuICAgIHJldHVybiBQcm9taXNlLnJlc29sdmUocmVzdWx0KTtcbiAgfVxuXG4gIHBhcnNlKGRhdGE6IHVua25vd24sIHBhcmFtcz86IFBhcnRpYWw8UGFyc2VQYXJhbXM+KTogT3V0cHV0IHtcbiAgICBjb25zdCByZXN1bHQgPSB0aGlzLnNhZmVQYXJzZShkYXRhLCBwYXJhbXMpO1xuICAgIGlmIChyZXN1bHQuc3VjY2VzcykgcmV0dXJuIHJlc3VsdC5kYXRhO1xuICAgIHRocm93IHJlc3VsdC5lcnJvcjtcbiAgfVxuXG4gIHNhZmVQYXJzZShcbiAgICBkYXRhOiB1bmtub3duLFxuICAgIHBhcmFtcz86IFBhcnRpYWw8UGFyc2VQYXJhbXM+XG4gICk6IFNhZmVQYXJzZVJldHVyblR5cGU8SW5wdXQsIE91dHB1dD4ge1xuICAgIGNvbnN0IGN0eDogUGFyc2VDb250ZXh0ID0ge1xuICAgICAgY29tbW9uOiB7XG4gICAgICAgIGlzc3VlczogW10sXG4gICAgICAgIGFzeW5jOiBwYXJhbXM/LmFzeW5jID8/IGZhbHNlLFxuICAgICAgICBjb250ZXh0dWFsRXJyb3JNYXA6IHBhcmFtcz8uZXJyb3JNYXAsXG4gICAgICB9LFxuICAgICAgcGF0aDogcGFyYW1zPy5wYXRoIHx8IFtdLFxuICAgICAgc2NoZW1hRXJyb3JNYXA6IHRoaXMuX2RlZi5lcnJvck1hcCxcbiAgICAgIHBhcmVudDogbnVsbCxcbiAgICAgIGRhdGEsXG4gICAgICBwYXJzZWRUeXBlOiBnZXRQYXJzZWRUeXBlKGRhdGEpLFxuICAgIH07XG4gICAgY29uc3QgcmVzdWx0ID0gdGhpcy5fcGFyc2VTeW5jKHsgZGF0YSwgcGF0aDogY3R4LnBhdGgsIHBhcmVudDogY3R4IH0pO1xuXG4gICAgcmV0dXJuIGhhbmRsZVJlc3VsdChjdHgsIHJlc3VsdCk7XG4gIH1cblxuICBhc3luYyBwYXJzZUFzeW5jKFxuICAgIGRhdGE6IHVua25vd24sXG4gICAgcGFyYW1zPzogUGFydGlhbDxQYXJzZVBhcmFtcz5cbiAgKTogUHJvbWlzZTxPdXRwdXQ+IHtcbiAgICBjb25zdCByZXN1bHQgPSBhd2FpdCB0aGlzLnNhZmVQYXJzZUFzeW5jKGRhdGEsIHBhcmFtcyk7XG4gICAgaWYgKHJlc3VsdC5zdWNjZXNzKSByZXR1cm4gcmVzdWx0LmRhdGE7XG4gICAgdGhyb3cgcmVzdWx0LmVycm9yO1xuICB9XG5cbiAgYXN5bmMgc2FmZVBhcnNlQXN5bmMoXG4gICAgZGF0YTogdW5rbm93bixcbiAgICBwYXJhbXM/OiBQYXJ0aWFsPFBhcnNlUGFyYW1zPlxuICApOiBQcm9taXNlPFNhZmVQYXJzZVJldHVyblR5cGU8SW5wdXQsIE91dHB1dD4+IHtcbiAgICBjb25zdCBjdHg6IFBhcnNlQ29udGV4dCA9IHtcbiAgICAgIGNvbW1vbjoge1xuICAgICAgICBpc3N1ZXM6IFtdLFxuICAgICAgICBjb250ZXh0dWFsRXJyb3JNYXA6IHBhcmFtcz8uZXJyb3JNYXAsXG4gICAgICAgIGFzeW5jOiB0cnVlLFxuICAgICAgfSxcbiAgICAgIHBhdGg6IHBhcmFtcz8ucGF0aCB8fCBbXSxcbiAgICAgIHNjaGVtYUVycm9yTWFwOiB0aGlzLl9kZWYuZXJyb3JNYXAsXG4gICAgICBwYXJlbnQ6IG51bGwsXG4gICAgICBkYXRhLFxuICAgICAgcGFyc2VkVHlwZTogZ2V0UGFyc2VkVHlwZShkYXRhKSxcbiAgICB9O1xuXG4gICAgY29uc3QgbWF5YmVBc3luY1Jlc3VsdCA9IHRoaXMuX3BhcnNlKHsgZGF0YSwgcGF0aDogW10sIHBhcmVudDogY3R4IH0pO1xuICAgIGNvbnN0IHJlc3VsdCA9IGF3YWl0IChpc0FzeW5jKG1heWJlQXN5bmNSZXN1bHQpXG4gICAgICA/IG1heWJlQXN5bmNSZXN1bHRcbiAgICAgIDogUHJvbWlzZS5yZXNvbHZlKG1heWJlQXN5bmNSZXN1bHQpKTtcbiAgICByZXR1cm4gaGFuZGxlUmVzdWx0KGN0eCwgcmVzdWx0KTtcbiAgfVxuXG4gIC8qKiBBbGlhcyBvZiBzYWZlUGFyc2VBc3luYyAqL1xuICBzcGEgPSB0aGlzLnNhZmVQYXJzZUFzeW5jO1xuXG4gIHJlZmluZTxSZWZpbmVkT3V0cHV0IGV4dGVuZHMgT3V0cHV0PihcbiAgICBjaGVjazogKGFyZzogT3V0cHV0KSA9PiBhcmcgaXMgUmVmaW5lZE91dHB1dCxcbiAgICBtZXNzYWdlPzogc3RyaW5nIHwgQ3VzdG9tRXJyb3JQYXJhbXMgfCAoKGFyZzogT3V0cHV0KSA9PiBDdXN0b21FcnJvclBhcmFtcylcbiAgKTogWm9kRWZmZWN0czx0aGlzLCBSZWZpbmVkT3V0cHV0LCBSZWZpbmVkT3V0cHV0PjtcbiAgcmVmaW5lKFxuICAgIGNoZWNrOiAoYXJnOiBPdXRwdXQpID0+IHVua25vd24gfCBQcm9taXNlPHVua25vd24+LFxuICAgIG1lc3NhZ2U/OiBzdHJpbmcgfCBDdXN0b21FcnJvclBhcmFtcyB8ICgoYXJnOiBPdXRwdXQpID0+IEN1c3RvbUVycm9yUGFyYW1zKVxuICApOiBab2RFZmZlY3RzPHRoaXMsIE91dHB1dCwgSW5wdXQ+O1xuICByZWZpbmUoXG4gICAgY2hlY2s6IChhcmc6IE91dHB1dCkgPT4gdW5rbm93bixcbiAgICBtZXNzYWdlPzogc3RyaW5nIHwgQ3VzdG9tRXJyb3JQYXJhbXMgfCAoKGFyZzogT3V0cHV0KSA9PiBDdXN0b21FcnJvclBhcmFtcylcbiAgKTogWm9kRWZmZWN0czx0aGlzLCBPdXRwdXQsIElucHV0PiB7XG4gICAgY29uc3QgZ2V0SXNzdWVQcm9wZXJ0aWVzOiBhbnkgPSAodmFsOiBPdXRwdXQpID0+IHtcbiAgICAgIGlmICh0eXBlb2YgbWVzc2FnZSA9PT0gXCJzdHJpbmdcIiB8fCB0eXBlb2YgbWVzc2FnZSA9PT0gXCJ1bmRlZmluZWRcIikge1xuICAgICAgICByZXR1cm4geyBtZXNzYWdlIH07XG4gICAgICB9IGVsc2UgaWYgKHR5cGVvZiBtZXNzYWdlID09PSBcImZ1bmN0aW9uXCIpIHtcbiAgICAgICAgcmV0dXJuIG1lc3NhZ2UodmFsKTtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHJldHVybiBtZXNzYWdlO1xuICAgICAgfVxuICAgIH07XG4gICAgcmV0dXJuIHRoaXMuX3JlZmluZW1lbnQoKHZhbCwgY3R4KSA9PiB7XG4gICAgICBjb25zdCByZXN1bHQgPSBjaGVjayh2YWwpO1xuICAgICAgY29uc3Qgc2V0RXJyb3IgPSAoKSA9PlxuICAgICAgICBjdHguYWRkSXNzdWUoe1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5jdXN0b20sXG4gICAgICAgICAgLi4uZ2V0SXNzdWVQcm9wZXJ0aWVzKHZhbCksXG4gICAgICAgIH0pO1xuICAgICAgaWYgKHR5cGVvZiBQcm9taXNlICE9PSBcInVuZGVmaW5lZFwiICYmIHJlc3VsdCBpbnN0YW5jZW9mIFByb21pc2UpIHtcbiAgICAgICAgcmV0dXJuIHJlc3VsdC50aGVuKChkYXRhKSA9PiB7XG4gICAgICAgICAgaWYgKCFkYXRhKSB7XG4gICAgICAgICAgICBzZXRFcnJvcigpO1xuICAgICAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgICAgIH0gZWxzZSB7XG4gICAgICAgICAgICByZXR1cm4gdHJ1ZTtcbiAgICAgICAgICB9XG4gICAgICAgIH0pO1xuICAgICAgfVxuICAgICAgaWYgKCFyZXN1bHQpIHtcbiAgICAgICAgc2V0RXJyb3IoKTtcbiAgICAgICAgcmV0dXJuIGZhbHNlO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgcmV0dXJuIHRydWU7XG4gICAgICB9XG4gICAgfSk7XG4gIH1cblxuICByZWZpbmVtZW50PFJlZmluZWRPdXRwdXQgZXh0ZW5kcyBPdXRwdXQ+KFxuICAgIGNoZWNrOiAoYXJnOiBPdXRwdXQpID0+IGFyZyBpcyBSZWZpbmVkT3V0cHV0LFxuICAgIHJlZmluZW1lbnREYXRhOiBJc3N1ZURhdGEgfCAoKGFyZzogT3V0cHV0LCBjdHg6IFJlZmluZW1lbnRDdHgpID0+IElzc3VlRGF0YSlcbiAgKTogWm9kRWZmZWN0czx0aGlzLCBSZWZpbmVkT3V0cHV0LCBSZWZpbmVkT3V0cHV0PjtcbiAgcmVmaW5lbWVudChcbiAgICBjaGVjazogKGFyZzogT3V0cHV0KSA9PiBib29sZWFuLFxuICAgIHJlZmluZW1lbnREYXRhOiBJc3N1ZURhdGEgfCAoKGFyZzogT3V0cHV0LCBjdHg6IFJlZmluZW1lbnRDdHgpID0+IElzc3VlRGF0YSlcbiAgKTogWm9kRWZmZWN0czx0aGlzLCBPdXRwdXQsIElucHV0PjtcbiAgcmVmaW5lbWVudChcbiAgICBjaGVjazogKGFyZzogT3V0cHV0KSA9PiB1bmtub3duLFxuICAgIHJlZmluZW1lbnREYXRhOiBJc3N1ZURhdGEgfCAoKGFyZzogT3V0cHV0LCBjdHg6IFJlZmluZW1lbnRDdHgpID0+IElzc3VlRGF0YSlcbiAgKTogWm9kRWZmZWN0czx0aGlzLCBPdXRwdXQsIElucHV0PiB7XG4gICAgcmV0dXJuIHRoaXMuX3JlZmluZW1lbnQoKHZhbCwgY3R4KSA9PiB7XG4gICAgICBpZiAoIWNoZWNrKHZhbCkpIHtcbiAgICAgICAgY3R4LmFkZElzc3VlKFxuICAgICAgICAgIHR5cGVvZiByZWZpbmVtZW50RGF0YSA9PT0gXCJmdW5jdGlvblwiXG4gICAgICAgICAgICA/IHJlZmluZW1lbnREYXRhKHZhbCwgY3R4KVxuICAgICAgICAgICAgOiByZWZpbmVtZW50RGF0YVxuICAgICAgICApO1xuICAgICAgICByZXR1cm4gZmFsc2U7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gdHJ1ZTtcbiAgICAgIH1cbiAgICB9KTtcbiAgfVxuXG4gIF9yZWZpbmVtZW50KFxuICAgIHJlZmluZW1lbnQ6IFJlZmluZW1lbnRFZmZlY3Q8T3V0cHV0PltcInJlZmluZW1lbnRcIl1cbiAgKTogWm9kRWZmZWN0czx0aGlzLCBPdXRwdXQsIElucHV0PiB7XG4gICAgcmV0dXJuIG5ldyBab2RFZmZlY3RzKHtcbiAgICAgIHNjaGVtYTogdGhpcyxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kRWZmZWN0cyxcbiAgICAgIGVmZmVjdDogeyB0eXBlOiBcInJlZmluZW1lbnRcIiwgcmVmaW5lbWVudCB9LFxuICAgIH0pO1xuICB9XG4gIHN1cGVyUmVmaW5lID0gdGhpcy5fcmVmaW5lbWVudDtcblxuICBjb25zdHJ1Y3RvcihkZWY6IERlZikge1xuICAgIHRoaXMuX2RlZiA9IGRlZjtcbiAgICB0aGlzLnBhcnNlID0gdGhpcy5wYXJzZS5iaW5kKHRoaXMpO1xuICAgIHRoaXMuc2FmZVBhcnNlID0gdGhpcy5zYWZlUGFyc2UuYmluZCh0aGlzKTtcbiAgICB0aGlzLnBhcnNlQXN5bmMgPSB0aGlzLnBhcnNlQXN5bmMuYmluZCh0aGlzKTtcbiAgICB0aGlzLnNhZmVQYXJzZUFzeW5jID0gdGhpcy5zYWZlUGFyc2VBc3luYy5iaW5kKHRoaXMpO1xuICAgIHRoaXMuc3BhID0gdGhpcy5zcGEuYmluZCh0aGlzKTtcbiAgICB0aGlzLnJlZmluZSA9IHRoaXMucmVmaW5lLmJpbmQodGhpcyk7XG4gICAgdGhpcy5yZWZpbmVtZW50ID0gdGhpcy5yZWZpbmVtZW50LmJpbmQodGhpcyk7XG4gICAgdGhpcy5zdXBlclJlZmluZSA9IHRoaXMuc3VwZXJSZWZpbmUuYmluZCh0aGlzKTtcbiAgICB0aGlzLm9wdGlvbmFsID0gdGhpcy5vcHRpb25hbC5iaW5kKHRoaXMpO1xuICAgIHRoaXMubnVsbGFibGUgPSB0aGlzLm51bGxhYmxlLmJpbmQodGhpcyk7XG4gICAgdGhpcy5udWxsaXNoID0gdGhpcy5udWxsaXNoLmJpbmQodGhpcyk7XG4gICAgdGhpcy5hcnJheSA9IHRoaXMuYXJyYXkuYmluZCh0aGlzKTtcbiAgICB0aGlzLnByb21pc2UgPSB0aGlzLnByb21pc2UuYmluZCh0aGlzKTtcbiAgICB0aGlzLm9yID0gdGhpcy5vci5iaW5kKHRoaXMpO1xuICAgIHRoaXMuYW5kID0gdGhpcy5hbmQuYmluZCh0aGlzKTtcbiAgICB0aGlzLnRyYW5zZm9ybSA9IHRoaXMudHJhbnNmb3JtLmJpbmQodGhpcyk7XG4gICAgdGhpcy5kZWZhdWx0ID0gdGhpcy5kZWZhdWx0LmJpbmQodGhpcyk7XG4gICAgdGhpcy5kZXNjcmliZSA9IHRoaXMuZGVzY3JpYmUuYmluZCh0aGlzKTtcbiAgICB0aGlzLmlzTnVsbGFibGUgPSB0aGlzLmlzTnVsbGFibGUuYmluZCh0aGlzKTtcbiAgICB0aGlzLmlzT3B0aW9uYWwgPSB0aGlzLmlzT3B0aW9uYWwuYmluZCh0aGlzKTtcbiAgfVxuXG4gIG9wdGlvbmFsKCk6IFpvZE9wdGlvbmFsPHRoaXM+IHtcbiAgICByZXR1cm4gWm9kT3B0aW9uYWwuY3JlYXRlKHRoaXMpIGFzIGFueTtcbiAgfVxuICBudWxsYWJsZSgpOiBab2ROdWxsYWJsZTx0aGlzPiB7XG4gICAgcmV0dXJuIFpvZE51bGxhYmxlLmNyZWF0ZSh0aGlzKSBhcyBhbnk7XG4gIH1cbiAgbnVsbGlzaCgpOiBab2ROdWxsYWJsZTxab2RPcHRpb25hbDx0aGlzPj4ge1xuICAgIHJldHVybiB0aGlzLm9wdGlvbmFsKCkubnVsbGFibGUoKTtcbiAgfVxuICBhcnJheSgpOiBab2RBcnJheTx0aGlzPiB7XG4gICAgcmV0dXJuIFpvZEFycmF5LmNyZWF0ZSh0aGlzKTtcbiAgfVxuICBwcm9taXNlKCk6IFpvZFByb21pc2U8dGhpcz4ge1xuICAgIHJldHVybiBab2RQcm9taXNlLmNyZWF0ZSh0aGlzKTtcbiAgfVxuXG4gIG9yPFQgZXh0ZW5kcyBab2RUeXBlQW55PihvcHRpb246IFQpOiBab2RVbmlvbjxbdGhpcywgVF0+IHtcbiAgICByZXR1cm4gWm9kVW5pb24uY3JlYXRlKFt0aGlzLCBvcHRpb25dKSBhcyBhbnk7XG4gIH1cblxuICBhbmQ8VCBleHRlbmRzIFpvZFR5cGVBbnk+KGluY29taW5nOiBUKTogWm9kSW50ZXJzZWN0aW9uPHRoaXMsIFQ+IHtcbiAgICByZXR1cm4gWm9kSW50ZXJzZWN0aW9uLmNyZWF0ZSh0aGlzLCBpbmNvbWluZyk7XG4gIH1cblxuICB0cmFuc2Zvcm08TmV3T3V0PihcbiAgICB0cmFuc2Zvcm06IChhcmc6IE91dHB1dCwgY3R4OiBSZWZpbmVtZW50Q3R4KSA9PiBOZXdPdXQgfCBQcm9taXNlPE5ld091dD5cbiAgKTogWm9kRWZmZWN0czx0aGlzLCBOZXdPdXQ+IHtcbiAgICByZXR1cm4gbmV3IFpvZEVmZmVjdHMoe1xuICAgICAgc2NoZW1hOiB0aGlzLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RFZmZlY3RzLFxuICAgICAgZWZmZWN0OiB7IHR5cGU6IFwidHJhbnNmb3JtXCIsIHRyYW5zZm9ybSB9LFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIGRlZmF1bHQoZGVmOiB1dGlsLm5vVW5kZWZpbmVkPElucHV0Pik6IFpvZERlZmF1bHQ8dGhpcz47XG4gIGRlZmF1bHQoZGVmOiAoKSA9PiB1dGlsLm5vVW5kZWZpbmVkPElucHV0Pik6IFpvZERlZmF1bHQ8dGhpcz47XG4gIGRlZmF1bHQoZGVmOiBhbnkpIHtcbiAgICBjb25zdCBkZWZhdWx0VmFsdWVGdW5jID0gdHlwZW9mIGRlZiA9PT0gXCJmdW5jdGlvblwiID8gZGVmIDogKCkgPT4gZGVmO1xuXG4gICAgcmV0dXJuIG5ldyBab2REZWZhdWx0KHtcbiAgICAgIGlubmVyVHlwZTogdGhpcyxcbiAgICAgIGRlZmF1bHRWYWx1ZTogZGVmYXVsdFZhbHVlRnVuYyxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kRGVmYXVsdCxcbiAgICB9KSBhcyBhbnk7XG4gIH1cblxuICBkZXNjcmliZShkZXNjcmlwdGlvbjogc3RyaW5nKTogdGhpcyB7XG4gICAgY29uc3QgVGhpcyA9ICh0aGlzIGFzIGFueSkuY29uc3RydWN0b3I7XG4gICAgcmV0dXJuIG5ldyBUaGlzKHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIGRlc2NyaXB0aW9uLFxuICAgIH0pO1xuICB9XG5cbiAgaXNPcHRpb25hbCgpOiBib29sZWFuIHtcbiAgICByZXR1cm4gdGhpcy5zYWZlUGFyc2UodW5kZWZpbmVkKS5zdWNjZXNzO1xuICB9XG4gIGlzTnVsbGFibGUoKTogYm9vbGVhbiB7XG4gICAgcmV0dXJuIHRoaXMuc2FmZVBhcnNlKG51bGwpLnN1Y2Nlc3M7XG4gIH1cbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZFN0cmluZyAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbnR5cGUgWm9kU3RyaW5nQ2hlY2sgPVxuICB8IHsga2luZDogXCJtaW5cIjsgdmFsdWU6IG51bWJlcjsgbWVzc2FnZT86IHN0cmluZyB9XG4gIHwgeyBraW5kOiBcIm1heFwiOyB2YWx1ZTogbnVtYmVyOyBtZXNzYWdlPzogc3RyaW5nIH1cbiAgfCB7IGtpbmQ6IFwiZW1haWxcIjsgbWVzc2FnZT86IHN0cmluZyB9XG4gIHwgeyBraW5kOiBcInVybFwiOyBtZXNzYWdlPzogc3RyaW5nIH1cbiAgfCB7IGtpbmQ6IFwidXVpZFwiOyBtZXNzYWdlPzogc3RyaW5nIH1cbiAgfCB7IGtpbmQ6IFwiY3VpZFwiOyBtZXNzYWdlPzogc3RyaW5nIH1cbiAgfCB7IGtpbmQ6IFwicmVnZXhcIjsgcmVnZXg6IFJlZ0V4cDsgbWVzc2FnZT86IHN0cmluZyB9XG4gIHwgeyBraW5kOiBcInRyaW1cIjsgbWVzc2FnZT86IHN0cmluZyB9O1xuXG5leHBvcnQgaW50ZXJmYWNlIFpvZFN0cmluZ0RlZiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICBjaGVja3M6IFpvZFN0cmluZ0NoZWNrW107XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kU3RyaW5nO1xufVxuXG5jb25zdCBjdWlkUmVnZXggPSAvXmNbXlxccy1dezgsfSQvaTtcbmNvbnN0IHV1aWRSZWdleCA9XG4gIC9eKFthLWYwLTldezh9LVthLWYwLTldezR9LVsxLTVdW2EtZjAtOV17M30tW2EtZjAtOV17NH0tW2EtZjAtOV17MTJ9fDAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCkkL2k7XG4vLyBmcm9tIGh0dHBzOi8vc3RhY2tvdmVyZmxvdy5jb20vYS80NjE4MS8xNTUwMTU1XG4vLyBvbGQgdmVyc2lvbjogdG9vIHNsb3csIGRpZG4ndCBzdXBwb3J0IHVuaWNvZGVcbi8vIGNvbnN0IGVtYWlsUmVnZXggPSAvXigoKFthLXpdfFxcZHxbISNcXCQlJidcXCpcXCtcXC1cXC89XFw/XFxeX2B7XFx8fX5dfFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKSsoXFwuKFthLXpdfFxcZHxbISNcXCQlJidcXCpcXCtcXC1cXC89XFw/XFxeX2B7XFx8fX5dfFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKSspKil8KChcXHgyMikoKCgoXFx4MjB8XFx4MDkpKihcXHgwZFxceDBhKSk/KFxceDIwfFxceDA5KSspPygoW1xceDAxLVxceDA4XFx4MGJcXHgwY1xceDBlLVxceDFmXFx4N2ZdfFxceDIxfFtcXHgyMy1cXHg1Yl18W1xceDVkLVxceDdlXXxbXFx1MDBBMC1cXHVEN0ZGXFx1RjkwMC1cXHVGRENGXFx1RkRGMC1cXHVGRkVGXSl8KFxcXFwoW1xceDAxLVxceDA5XFx4MGJcXHgwY1xceDBkLVxceDdmXXxbXFx1MDBBMC1cXHVEN0ZGXFx1RjkwMC1cXHVGRENGXFx1RkRGMC1cXHVGRkVGXSkpKSkqKCgoXFx4MjB8XFx4MDkpKihcXHgwZFxceDBhKSk/KFxceDIwfFxceDA5KSspPyhcXHgyMikpKUAoKChbYS16XXxcXGR8W1xcdTAwQTAtXFx1RDdGRlxcdUY5MDAtXFx1RkRDRlxcdUZERjAtXFx1RkZFRl0pfCgoW2Etel18XFxkfFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKShbYS16XXxcXGR8LXxcXC58X3x+fFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKSooW2Etel18XFxkfFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKSkpXFwuKSsoKFthLXpdfFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKXwoKFthLXpdfFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKShbYS16XXxcXGR8LXxcXC58X3x+fFtcXHUwMEEwLVxcdUQ3RkZcXHVGOTAwLVxcdUZEQ0ZcXHVGREYwLVxcdUZGRUZdKSooW2Etel18W1xcdTAwQTAtXFx1RDdGRlxcdUY5MDAtXFx1RkRDRlxcdUZERjAtXFx1RkZFRl0pKSkkL2k7XG4vLyBlc2xpbnQtZGlzYWJsZS1uZXh0LWxpbmVcbmNvbnN0IGVtYWlsUmVnZXggPVxuICAvXigoW148PigpW1xcXVxcLiw7Olxcc0BcXFwiXSsoXFwuW148PigpW1xcXVxcLiw7Olxcc0BcXFwiXSspKil8KFxcXCIuK1xcXCIpKUAoKFtePD4oKVtcXF1cXC4sOzpcXHNAXFxcIl0rXFwuKStbXjw+KClbXFxdXFwuLDs6XFxzQFxcXCJdezIsfSkkL2k7XG5cbmV4cG9ydCBjbGFzcyBab2RTdHJpbmcgZXh0ZW5kcyBab2RUeXBlPHN0cmluZywgWm9kU3RyaW5nRGVmPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTxzdHJpbmc+IHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG5cbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5zdHJpbmcpIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KFxuICAgICAgICBjdHgsXG4gICAgICAgIHtcbiAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLnN0cmluZyxcbiAgICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICAgIH1cbiAgICAgICAgLy9cbiAgICAgICk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBjb25zdCBzdGF0dXMgPSBuZXcgUGFyc2VTdGF0dXMoKTtcbiAgICBsZXQgY3R4OiB1bmRlZmluZWQgfCBQYXJzZUNvbnRleHQgPSB1bmRlZmluZWQ7XG5cbiAgICBmb3IgKGNvbnN0IGNoZWNrIG9mIHRoaXMuX2RlZi5jaGVja3MpIHtcbiAgICAgIGlmIChjaGVjay5raW5kID09PSBcIm1pblwiKSB7XG4gICAgICAgIGlmIChpbnB1dC5kYXRhLmxlbmd0aCA8IGNoZWNrLnZhbHVlKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX3NtYWxsLFxuICAgICAgICAgICAgbWluaW11bTogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICB0eXBlOiBcInN0cmluZ1wiLFxuICAgICAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICAgICAgbWVzc2FnZTogY2hlY2subWVzc2FnZSxcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcIm1heFwiKSB7XG4gICAgICAgIGlmIChpbnB1dC5kYXRhLmxlbmd0aCA+IGNoZWNrLnZhbHVlKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX2JpZyxcbiAgICAgICAgICAgIG1heGltdW06IGNoZWNrLnZhbHVlLFxuICAgICAgICAgICAgdHlwZTogXCJzdHJpbmdcIixcbiAgICAgICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2UsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJlbWFpbFwiKSB7XG4gICAgICAgIGlmICghZW1haWxSZWdleC50ZXN0KGlucHV0LmRhdGEpKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICB2YWxpZGF0aW9uOiBcImVtYWlsXCIsXG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF9zdHJpbmcsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlLFxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGNoZWNrLmtpbmQgPT09IFwidXVpZFwiKSB7XG4gICAgICAgIGlmICghdXVpZFJlZ2V4LnRlc3QoaW5wdXQuZGF0YSkpIHtcbiAgICAgICAgICBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCwgY3R4KTtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIHZhbGlkYXRpb246IFwidXVpZFwiLFxuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfc3RyaW5nLFxuICAgICAgICAgICAgbWVzc2FnZTogY2hlY2subWVzc2FnZSxcbiAgICAgICAgICB9KTtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSBlbHNlIGlmIChjaGVjay5raW5kID09PSBcImN1aWRcIikge1xuICAgICAgICBpZiAoIWN1aWRSZWdleC50ZXN0KGlucHV0LmRhdGEpKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICB2YWxpZGF0aW9uOiBcImN1aWRcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2UsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJ1cmxcIikge1xuICAgICAgICB0cnkge1xuICAgICAgICAgIG5ldyBVUkwoaW5wdXQuZGF0YSk7XG4gICAgICAgIH0gY2F0Y2gge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgdmFsaWRhdGlvbjogXCJ1cmxcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2UsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJyZWdleFwiKSB7XG4gICAgICAgIGNoZWNrLnJlZ2V4Lmxhc3RJbmRleCA9IDA7XG4gICAgICAgIGNvbnN0IHRlc3RSZXN1bHQgPSBjaGVjay5yZWdleC50ZXN0KGlucHV0LmRhdGEpO1xuICAgICAgICBpZiAoIXRlc3RSZXN1bHQpIHtcbiAgICAgICAgICBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCwgY3R4KTtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIHZhbGlkYXRpb246IFwicmVnZXhcIixcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgICAgICAgIG1lc3NhZ2U6IGNoZWNrLm1lc3NhZ2UsXG4gICAgICAgICAgfSk7XG4gICAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICAgIH1cbiAgICAgIH0gZWxzZSBpZiAoY2hlY2sua2luZCA9PT0gXCJ0cmltXCIpIHtcbiAgICAgICAgaW5wdXQuZGF0YSA9IGlucHV0LmRhdGEudHJpbSgpO1xuICAgICAgfSBlbHNlIHtcbiAgICAgICAgdXRpbC5hc3NlcnROZXZlcihjaGVjayk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgcmV0dXJuIHsgc3RhdHVzOiBzdGF0dXMudmFsdWUsIHZhbHVlOiBpbnB1dC5kYXRhIH07XG4gIH1cblxuICBwcm90ZWN0ZWQgX3JlZ2V4ID0gKFxuICAgIHJlZ2V4OiBSZWdFeHAsXG4gICAgdmFsaWRhdGlvbjogU3RyaW5nVmFsaWRhdGlvbixcbiAgICBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2VcbiAgKSA9PlxuICAgIHRoaXMucmVmaW5lbWVudCgoZGF0YSkgPT4gcmVnZXgudGVzdChkYXRhKSwge1xuICAgICAgdmFsaWRhdGlvbixcbiAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3N0cmluZyxcbiAgICAgIC4uLmVycm9yVXRpbC5lcnJUb09iaihtZXNzYWdlKSxcbiAgICB9KTtcblxuICBfYWRkQ2hlY2soY2hlY2s6IFpvZFN0cmluZ0NoZWNrKSB7XG4gICAgcmV0dXJuIG5ldyBab2RTdHJpbmcoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgY2hlY2tzOiBbLi4udGhpcy5fZGVmLmNoZWNrcywgY2hlY2tdLFxuICAgIH0pO1xuICB9XG5cbiAgZW1haWwobWVzc2FnZT86IGVycm9yVXRpbC5FcnJNZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHsga2luZDogXCJlbWFpbFwiLCAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSkgfSk7XG4gIH1cbiAgdXJsKG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7IGtpbmQ6IFwidXJsXCIsIC4uLmVycm9yVXRpbC5lcnJUb09iaihtZXNzYWdlKSB9KTtcbiAgfVxuICB1dWlkKG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7IGtpbmQ6IFwidXVpZFwiLCAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSkgfSk7XG4gIH1cbiAgY3VpZChtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soeyBraW5kOiBcImN1aWRcIiwgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpIH0pO1xuICB9XG4gIHJlZ2V4KHJlZ2V4OiBSZWdFeHAsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcInJlZ2V4XCIsXG4gICAgICByZWdleDogcmVnZXgsXG4gICAgICAuLi5lcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSksXG4gICAgfSk7XG4gIH1cblxuICBtaW4obWluTGVuZ3RoOiBudW1iZXIsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcIm1pblwiLFxuICAgICAgdmFsdWU6IG1pbkxlbmd0aCxcbiAgICAgIC4uLmVycm9yVXRpbC5lcnJUb09iaihtZXNzYWdlKSxcbiAgICB9KTtcbiAgfVxuXG4gIG1heChtYXhMZW5ndGg6IG51bWJlciwgbWVzc2FnZT86IGVycm9yVXRpbC5FcnJNZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtcbiAgICAgIGtpbmQ6IFwibWF4XCIsXG4gICAgICB2YWx1ZTogbWF4TGVuZ3RoLFxuICAgICAgLi4uZXJyb3JVdGlsLmVyclRvT2JqKG1lc3NhZ2UpLFxuICAgIH0pO1xuICB9XG5cbiAgbGVuZ3RoKGxlbjogbnVtYmVyLCBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5taW4obGVuLCBtZXNzYWdlKS5tYXgobGVuLCBtZXNzYWdlKTtcbiAgfVxuXG4gIC8qKlxuICAgKiBAZGVwcmVjYXRlZCBVc2Ugei5zdHJpbmcoKS5taW4oMSkgaW5zdGVhZC5cbiAgICogQHNlZSB7QGxpbmsgWm9kU3RyaW5nLm1pbn1cbiAgICovXG4gIG5vbmVtcHR5ID0gKG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkgPT5cbiAgICB0aGlzLm1pbigxLCBlcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSkpO1xuXG4gIHRyaW0gPSAoKSA9PlxuICAgIG5ldyBab2RTdHJpbmcoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgY2hlY2tzOiBbLi4udGhpcy5fZGVmLmNoZWNrcywgeyBraW5kOiBcInRyaW1cIiB9XSxcbiAgICB9KTtcblxuICBnZXQgaXNFbWFpbCgpIHtcbiAgICByZXR1cm4gISF0aGlzLl9kZWYuY2hlY2tzLmZpbmQoKGNoKSA9PiBjaC5raW5kID09PSBcImVtYWlsXCIpO1xuICB9XG4gIGdldCBpc1VSTCgpIHtcbiAgICByZXR1cm4gISF0aGlzLl9kZWYuY2hlY2tzLmZpbmQoKGNoKSA9PiBjaC5raW5kID09PSBcInVybFwiKTtcbiAgfVxuICBnZXQgaXNVVUlEKCkge1xuICAgIHJldHVybiAhIXRoaXMuX2RlZi5jaGVja3MuZmluZCgoY2gpID0+IGNoLmtpbmQgPT09IFwidXVpZFwiKTtcbiAgfVxuICBnZXQgaXNDVUlEKCkge1xuICAgIHJldHVybiAhIXRoaXMuX2RlZi5jaGVja3MuZmluZCgoY2gpID0+IGNoLmtpbmQgPT09IFwiY3VpZFwiKTtcbiAgfVxuICBnZXQgbWluTGVuZ3RoKCkge1xuICAgIGxldCBtaW46IG51bWJlciB8IG51bGwgPSAtSW5maW5pdHk7XG4gICAgdGhpcy5fZGVmLmNoZWNrcy5tYXAoKGNoKSA9PiB7XG4gICAgICBpZiAoY2gua2luZCA9PT0gXCJtaW5cIikge1xuICAgICAgICBpZiAobWluID09PSBudWxsIHx8IGNoLnZhbHVlID4gbWluKSB7XG4gICAgICAgICAgbWluID0gY2gudmFsdWU7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9KTtcbiAgICByZXR1cm4gbWluO1xuICB9XG4gIGdldCBtYXhMZW5ndGgoKSB7XG4gICAgbGV0IG1heDogbnVtYmVyIHwgbnVsbCA9IG51bGw7XG4gICAgdGhpcy5fZGVmLmNoZWNrcy5tYXAoKGNoKSA9PiB7XG4gICAgICBpZiAoY2gua2luZCA9PT0gXCJtYXhcIikge1xuICAgICAgICBpZiAobWF4ID09PSBudWxsIHx8IGNoLnZhbHVlIDwgbWF4KSB7XG4gICAgICAgICAgbWF4ID0gY2gudmFsdWU7XG4gICAgICAgIH1cbiAgICAgIH1cbiAgICB9KTtcbiAgICByZXR1cm4gbWF4O1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IChwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXMpOiBab2RTdHJpbmcgPT4ge1xuICAgIHJldHVybiBuZXcgWm9kU3RyaW5nKHtcbiAgICAgIGNoZWNrczogW10sXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFN0cmluZyxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE51bWJlciAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbnR5cGUgWm9kTnVtYmVyQ2hlY2sgPVxuICB8IHsga2luZDogXCJtaW5cIjsgdmFsdWU6IG51bWJlcjsgaW5jbHVzaXZlOiBib29sZWFuOyBtZXNzYWdlPzogc3RyaW5nIH1cbiAgfCB7IGtpbmQ6IFwibWF4XCI7IHZhbHVlOiBudW1iZXI7IGluY2x1c2l2ZTogYm9vbGVhbjsgbWVzc2FnZT86IHN0cmluZyB9XG4gIHwgeyBraW5kOiBcImludFwiOyBtZXNzYWdlPzogc3RyaW5nIH1cbiAgfCB7IGtpbmQ6IFwibXVsdGlwbGVPZlwiOyB2YWx1ZTogbnVtYmVyOyBtZXNzYWdlPzogc3RyaW5nIH07XG5cbi8vIGh0dHBzOi8vc3RhY2tvdmVyZmxvdy5jb20vcXVlc3Rpb25zLzM5NjY0ODQvd2h5LWRvZXMtbW9kdWx1cy1vcGVyYXRvci1yZXR1cm4tZnJhY3Rpb25hbC1udW1iZXItaW4tamF2YXNjcmlwdC8zMTcxMTAzNCMzMTcxMTAzNFxuZnVuY3Rpb24gZmxvYXRTYWZlUmVtYWluZGVyKHZhbDogbnVtYmVyLCBzdGVwOiBudW1iZXIpIHtcbiAgY29uc3QgdmFsRGVjQ291bnQgPSAodmFsLnRvU3RyaW5nKCkuc3BsaXQoXCIuXCIpWzFdIHx8IFwiXCIpLmxlbmd0aDtcbiAgY29uc3Qgc3RlcERlY0NvdW50ID0gKHN0ZXAudG9TdHJpbmcoKS5zcGxpdChcIi5cIilbMV0gfHwgXCJcIikubGVuZ3RoO1xuICBjb25zdCBkZWNDb3VudCA9IHZhbERlY0NvdW50ID4gc3RlcERlY0NvdW50ID8gdmFsRGVjQ291bnQgOiBzdGVwRGVjQ291bnQ7XG4gIGNvbnN0IHZhbEludCA9IHBhcnNlSW50KHZhbC50b0ZpeGVkKGRlY0NvdW50KS5yZXBsYWNlKFwiLlwiLCBcIlwiKSk7XG4gIGNvbnN0IHN0ZXBJbnQgPSBwYXJzZUludChzdGVwLnRvRml4ZWQoZGVjQ291bnQpLnJlcGxhY2UoXCIuXCIsIFwiXCIpKTtcbiAgcmV0dXJuICh2YWxJbnQgJSBzdGVwSW50KSAvIE1hdGgucG93KDEwLCBkZWNDb3VudCk7XG59XG5cbmV4cG9ydCBpbnRlcmZhY2UgWm9kTnVtYmVyRGVmIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIGNoZWNrczogWm9kTnVtYmVyQ2hlY2tbXTtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROdW1iZXI7XG59XG5cbmV4cG9ydCBjbGFzcyBab2ROdW1iZXIgZXh0ZW5kcyBab2RUeXBlPG51bWJlciwgWm9kTnVtYmVyRGVmPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTxudW1iZXI+IHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUubnVtYmVyKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUubnVtYmVyLFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cblxuICAgIGxldCBjdHg6IHVuZGVmaW5lZCB8IFBhcnNlQ29udGV4dCA9IHVuZGVmaW5lZDtcbiAgICBjb25zdCBzdGF0dXMgPSBuZXcgUGFyc2VTdGF0dXMoKTtcblxuICAgIGZvciAoY29uc3QgY2hlY2sgb2YgdGhpcy5fZGVmLmNoZWNrcykge1xuICAgICAgaWYgKGNoZWNrLmtpbmQgPT09IFwiaW50XCIpIHtcbiAgICAgICAgaWYgKCF1dGlsLmlzSW50ZWdlcihpbnB1dC5kYXRhKSkge1xuICAgICAgICAgIGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0LCBjdHgpO1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgICAgIGV4cGVjdGVkOiBcImludGVnZXJcIixcbiAgICAgICAgICAgIHJlY2VpdmVkOiBcImZsb2F0XCIsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlLFxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGNoZWNrLmtpbmQgPT09IFwibWluXCIpIHtcbiAgICAgICAgY29uc3QgdG9vU21hbGwgPSBjaGVjay5pbmNsdXNpdmVcbiAgICAgICAgICA/IGlucHV0LmRhdGEgPCBjaGVjay52YWx1ZVxuICAgICAgICAgIDogaW5wdXQuZGF0YSA8PSBjaGVjay52YWx1ZTtcbiAgICAgICAgaWYgKHRvb1NtYWxsKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX3NtYWxsLFxuICAgICAgICAgICAgbWluaW11bTogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICB0eXBlOiBcIm51bWJlclwiLFxuICAgICAgICAgICAgaW5jbHVzaXZlOiBjaGVjay5pbmNsdXNpdmUsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlLFxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGNoZWNrLmtpbmQgPT09IFwibWF4XCIpIHtcbiAgICAgICAgY29uc3QgdG9vQmlnID0gY2hlY2suaW5jbHVzaXZlXG4gICAgICAgICAgPyBpbnB1dC5kYXRhID4gY2hlY2sudmFsdWVcbiAgICAgICAgICA6IGlucHV0LmRhdGEgPj0gY2hlY2sudmFsdWU7XG4gICAgICAgIGlmICh0b29CaWcpIHtcbiAgICAgICAgICBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCwgY3R4KTtcbiAgICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS50b29fYmlnLFxuICAgICAgICAgICAgbWF4aW11bTogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICB0eXBlOiBcIm51bWJlclwiLFxuICAgICAgICAgICAgaW5jbHVzaXZlOiBjaGVjay5pbmNsdXNpdmUsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlLFxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKGNoZWNrLmtpbmQgPT09IFwibXVsdGlwbGVPZlwiKSB7XG4gICAgICAgIGlmIChmbG9hdFNhZmVSZW1haW5kZXIoaW5wdXQuZGF0YSwgY2hlY2sudmFsdWUpICE9PSAwKSB7XG4gICAgICAgICAgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQsIGN0eCk7XG4gICAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUubm90X211bHRpcGxlX29mLFxuICAgICAgICAgICAgbXVsdGlwbGVPZjogY2hlY2sudmFsdWUsXG4gICAgICAgICAgICBtZXNzYWdlOiBjaGVjay5tZXNzYWdlLFxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2Uge1xuICAgICAgICB1dGlsLmFzc2VydE5ldmVyKGNoZWNrKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICByZXR1cm4geyBzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IGlucHV0LmRhdGEgfTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGUgPSAocGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zKTogWm9kTnVtYmVyID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZE51bWJlcih7XG4gICAgICBjaGVja3M6IFtdLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROdW1iZXIsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG5cbiAgZ3RlKHZhbHVlOiBudW1iZXIsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLnNldExpbWl0KFwibWluXCIsIHZhbHVlLCB0cnVlLCBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSkpO1xuICB9XG4gIG1pbiA9IHRoaXMuZ3RlO1xuXG4gIGd0KHZhbHVlOiBudW1iZXIsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLnNldExpbWl0KFwibWluXCIsIHZhbHVlLCBmYWxzZSwgZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpKTtcbiAgfVxuXG4gIGx0ZSh2YWx1ZTogbnVtYmVyLCBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5zZXRMaW1pdChcIm1heFwiLCB2YWx1ZSwgdHJ1ZSwgZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpKTtcbiAgfVxuICBtYXggPSB0aGlzLmx0ZTtcblxuICBsdCh2YWx1ZTogbnVtYmVyLCBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5zZXRMaW1pdChcIm1heFwiLCB2YWx1ZSwgZmFsc2UsIGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSk7XG4gIH1cblxuICBwcm90ZWN0ZWQgc2V0TGltaXQoXG4gICAga2luZDogXCJtaW5cIiB8IFwibWF4XCIsXG4gICAgdmFsdWU6IG51bWJlcixcbiAgICBpbmNsdXNpdmU6IGJvb2xlYW4sXG4gICAgbWVzc2FnZT86IHN0cmluZ1xuICApIHtcbiAgICByZXR1cm4gbmV3IFpvZE51bWJlcih7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBjaGVja3M6IFtcbiAgICAgICAgLi4udGhpcy5fZGVmLmNoZWNrcyxcbiAgICAgICAge1xuICAgICAgICAgIGtpbmQsXG4gICAgICAgICAgdmFsdWUsXG4gICAgICAgICAgaW5jbHVzaXZlLFxuICAgICAgICAgIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSxcbiAgICAgICAgfSxcbiAgICAgIF0sXG4gICAgfSk7XG4gIH1cblxuICBfYWRkQ2hlY2soY2hlY2s6IFpvZE51bWJlckNoZWNrKSB7XG4gICAgcmV0dXJuIG5ldyBab2ROdW1iZXIoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgY2hlY2tzOiBbLi4udGhpcy5fZGVmLmNoZWNrcywgY2hlY2tdLFxuICAgIH0pO1xuICB9XG5cbiAgaW50KG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcImludFwiLFxuICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpLFxuICAgIH0pO1xuICB9XG5cbiAgcG9zaXRpdmUobWVzc2FnZT86IGVycm9yVXRpbC5FcnJNZXNzYWdlKSB7XG4gICAgcmV0dXJuIHRoaXMuX2FkZENoZWNrKHtcbiAgICAgIGtpbmQ6IFwibWluXCIsXG4gICAgICB2YWx1ZTogMCxcbiAgICAgIGluY2x1c2l2ZTogZmFsc2UsXG4gICAgICBtZXNzYWdlOiBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSksXG4gICAgfSk7XG4gIH1cblxuICBuZWdhdGl2ZShtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soe1xuICAgICAga2luZDogXCJtYXhcIixcbiAgICAgIHZhbHVlOiAwLFxuICAgICAgaW5jbHVzaXZlOiBmYWxzZSxcbiAgICAgIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSxcbiAgICB9KTtcbiAgfVxuXG4gIG5vbnBvc2l0aXZlKG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSkge1xuICAgIHJldHVybiB0aGlzLl9hZGRDaGVjayh7XG4gICAgICBraW5kOiBcIm1heFwiLFxuICAgICAgdmFsdWU6IDAsXG4gICAgICBpbmNsdXNpdmU6IHRydWUsXG4gICAgICBtZXNzYWdlOiBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSksXG4gICAgfSk7XG4gIH1cblxuICBub25uZWdhdGl2ZShtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soe1xuICAgICAga2luZDogXCJtaW5cIixcbiAgICAgIHZhbHVlOiAwLFxuICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgbWVzc2FnZTogZXJyb3JVdGlsLnRvU3RyaW5nKG1lc3NhZ2UpLFxuICAgIH0pO1xuICB9XG5cbiAgbXVsdGlwbGVPZih2YWx1ZTogbnVtYmVyLCBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpIHtcbiAgICByZXR1cm4gdGhpcy5fYWRkQ2hlY2soe1xuICAgICAga2luZDogXCJtdWx0aXBsZU9mXCIsXG4gICAgICB2YWx1ZTogdmFsdWUsXG4gICAgICBtZXNzYWdlOiBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSksXG4gICAgfSk7XG4gIH1cblxuICBzdGVwID0gdGhpcy5tdWx0aXBsZU9mO1xuXG4gIGdldCBtaW5WYWx1ZSgpIHtcbiAgICBsZXQgbWluOiBudW1iZXIgfCBudWxsID0gbnVsbDtcbiAgICBmb3IgKGNvbnN0IGNoIG9mIHRoaXMuX2RlZi5jaGVja3MpIHtcbiAgICAgIGlmIChjaC5raW5kID09PSBcIm1pblwiKSB7XG4gICAgICAgIGlmIChtaW4gPT09IG51bGwgfHwgY2gudmFsdWUgPiBtaW4pIG1pbiA9IGNoLnZhbHVlO1xuICAgICAgfVxuICAgIH1cbiAgICByZXR1cm4gbWluO1xuICB9XG5cbiAgZ2V0IG1heFZhbHVlKCkge1xuICAgIGxldCBtYXg6IG51bWJlciB8IG51bGwgPSBudWxsO1xuICAgIGZvciAoY29uc3QgY2ggb2YgdGhpcy5fZGVmLmNoZWNrcykge1xuICAgICAgaWYgKGNoLmtpbmQgPT09IFwibWF4XCIpIHtcbiAgICAgICAgaWYgKG1heCA9PT0gbnVsbCB8fCBjaC52YWx1ZSA8IG1heCkgbWF4ID0gY2gudmFsdWU7XG4gICAgICB9XG4gICAgfVxuICAgIHJldHVybiBtYXg7XG4gIH1cblxuICBnZXQgaXNJbnQoKSB7XG4gICAgcmV0dXJuICEhdGhpcy5fZGVmLmNoZWNrcy5maW5kKChjaCkgPT4gY2gua2luZCA9PT0gXCJpbnRcIik7XG4gIH1cbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZEJpZ0ludCAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cblxuZXhwb3J0IGludGVyZmFjZSBab2RCaWdJbnREZWYgZXh0ZW5kcyBab2RUeXBlRGVmIHtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RCaWdJbnQ7XG59XG5cbmV4cG9ydCBjbGFzcyBab2RCaWdJbnQgZXh0ZW5kcyBab2RUeXBlPGJpZ2ludCwgWm9kQmlnSW50RGVmPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTxiaWdpbnQ+IHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuYmlnaW50KSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUuYmlnaW50LFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICByZXR1cm4gT0soaW5wdXQuZGF0YSk7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gKHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtcyk6IFpvZEJpZ0ludCA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RCaWdJbnQoe1xuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RCaWdJbnQsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG59XG5cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICBab2RCb29sZWFuICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuZXhwb3J0IGludGVyZmFjZSBab2RCb29sZWFuRGVmIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kQm9vbGVhbjtcbn1cblxuZXhwb3J0IGNsYXNzIFpvZEJvb2xlYW4gZXh0ZW5kcyBab2RUeXBlPGJvb2xlYW4sIFpvZEJvb2xlYW5EZWY+IHtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPGJvb2xlYW4+IHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuYm9vbGVhbikge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLmJvb2xlYW4sXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGUgPSAocGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zKTogWm9kQm9vbGVhbiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RCb29sZWFuKHtcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kQm9vbGVhbixcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZERhdGUgICAgICAgIC8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgaW50ZXJmYWNlIFpvZERhdGVEZWYgZXh0ZW5kcyBab2RUeXBlRGVmIHtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2REYXRlO1xufVxuXG5leHBvcnQgY2xhc3MgWm9kRGF0ZSBleHRlbmRzIFpvZFR5cGU8RGF0ZSwgWm9kRGF0ZURlZj4ge1xuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuZGF0ZSkge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLmRhdGUsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIGlmIChpc05hTihpbnB1dC5kYXRhLmdldFRpbWUoKSkpIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF9kYXRlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICByZXR1cm4ge1xuICAgICAgc3RhdHVzOiBcInZhbGlkXCIsXG4gICAgICB2YWx1ZTogbmV3IERhdGUoKGlucHV0LmRhdGEgYXMgRGF0ZSkuZ2V0VGltZSgpKSxcbiAgICB9O1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IChwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXMpOiBab2REYXRlID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZERhdGUoe1xuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2REYXRlLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kVW5kZWZpbmVkICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuZXhwb3J0IGludGVyZmFjZSBab2RVbmRlZmluZWREZWYgZXh0ZW5kcyBab2RUeXBlRGVmIHtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RVbmRlZmluZWQ7XG59XG5cbmV4cG9ydCBjbGFzcyBab2RVbmRlZmluZWQgZXh0ZW5kcyBab2RUeXBlPHVuZGVmaW5lZCwgWm9kVW5kZWZpbmVkRGVmPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS51bmRlZmluZWQpIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS51bmRlZmluZWQsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxuICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXM7XG5cbiAgc3RhdGljIGNyZWF0ZSA9IChwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXMpOiBab2RVbmRlZmluZWQgPT4ge1xuICAgIHJldHVybiBuZXcgWm9kVW5kZWZpbmVkKHtcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kVW5kZWZpbmVkLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kTnVsbCAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kTnVsbERlZiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE51bGw7XG59XG5cbmV4cG9ydCBjbGFzcyBab2ROdWxsIGV4dGVuZHMgWm9kVHlwZTxudWxsLCBab2ROdWxsRGVmPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5udWxsKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUubnVsbCxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgcmV0dXJuIE9LKGlucHV0LmRhdGEpO1xuICB9XG4gIHN0YXRpYyBjcmVhdGUgPSAocGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zKTogWm9kTnVsbCA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2ROdWxsKHtcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTnVsbCxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZEFueSAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kQW55RGVmIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kQW55O1xufVxuXG5leHBvcnQgY2xhc3MgWm9kQW55IGV4dGVuZHMgWm9kVHlwZTxhbnksIFpvZEFueURlZj4ge1xuICAvLyB0byBwcmV2ZW50IGluc3RhbmNlcyBvZiBvdGhlciBjbGFzc2VzIGZyb20gZXh0ZW5kaW5nIFpvZEFueS4gdGhpcyBjYXVzZXMgaXNzdWVzIHdpdGggY2F0Y2hhbGwgaW4gWm9kT2JqZWN0LlxuICBfYW55OiB0cnVlID0gdHJ1ZTtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgcmV0dXJuIE9LKGlucHV0LmRhdGEpO1xuICB9XG4gIHN0YXRpYyBjcmVhdGUgPSAocGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zKTogWm9kQW55ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZEFueSh7XG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEFueSxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZFVua25vd24gICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgaW50ZXJmYWNlIFpvZFVua25vd25EZWYgZXh0ZW5kcyBab2RUeXBlRGVmIHtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RVbmtub3duO1xufVxuXG5leHBvcnQgY2xhc3MgWm9kVW5rbm93biBleHRlbmRzIFpvZFR5cGU8dW5rbm93biwgWm9kVW5rbm93bkRlZj4ge1xuICAvLyByZXF1aXJlZFxuICBfdW5rbm93bjogdHJ1ZSA9IHRydWU7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGUgPSAocGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zKTogWm9kVW5rbm93biA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RVbmtub3duKHtcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kVW5rbm93bixcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE5ldmVyICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuZXhwb3J0IGludGVyZmFjZSBab2ROZXZlckRlZiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE5ldmVyO1xufVxuXG5leHBvcnQgY2xhc3MgWm9kTmV2ZXIgZXh0ZW5kcyBab2RUeXBlPG5ldmVyLCBab2ROZXZlckRlZj4ge1xuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUubmV2ZXIsXG4gICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgfSk7XG4gICAgcmV0dXJuIElOVkFMSUQ7XG4gIH1cbiAgc3RhdGljIGNyZWF0ZSA9IChwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXMpOiBab2ROZXZlciA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2ROZXZlcih7XG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE5ldmVyLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kVm9pZCAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kVm9pZERlZiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFZvaWQ7XG59XG5cbmV4cG9ydCBjbGFzcyBab2RWb2lkIGV4dGVuZHMgWm9kVHlwZTx2b2lkLCBab2RWb2lkRGVmPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS51bmRlZmluZWQpIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS52b2lkLFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cbiAgICByZXR1cm4gT0soaW5wdXQuZGF0YSk7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gKHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtcyk6IFpvZFZvaWQgPT4ge1xuICAgIHJldHVybiBuZXcgWm9kVm9pZCh7XG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFZvaWQsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG59XG5cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICBab2RBcnJheSAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kQXJyYXlEZWY8VCBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RUeXBlQW55PlxuICBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB0eXBlOiBUO1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEFycmF5O1xuICBtaW5MZW5ndGg6IHsgdmFsdWU6IG51bWJlcjsgbWVzc2FnZT86IHN0cmluZyB9IHwgbnVsbDtcbiAgbWF4TGVuZ3RoOiB7IHZhbHVlOiBudW1iZXI7IG1lc3NhZ2U/OiBzdHJpbmcgfSB8IG51bGw7XG59XG5cbmV4cG9ydCB0eXBlIEFycmF5Q2FyZGluYWxpdHkgPSBcIm1hbnlcIiB8IFwiYXRsZWFzdG9uZVwiO1xudHlwZSBhcnJheU91dHB1dFR5cGU8XG4gIFQgZXh0ZW5kcyBab2RUeXBlQW55LFxuICBDYXJkaW5hbGl0eSBleHRlbmRzIEFycmF5Q2FyZGluYWxpdHkgPSBcIm1hbnlcIlxuPiA9IENhcmRpbmFsaXR5IGV4dGVuZHMgXCJhdGxlYXN0b25lXCJcbiAgPyBbVFtcIl9vdXRwdXRcIl0sIC4uLlRbXCJfb3V0cHV0XCJdW11dXG4gIDogVFtcIl9vdXRwdXRcIl1bXTtcblxuZXhwb3J0IGNsYXNzIFpvZEFycmF5PFxuICBUIGV4dGVuZHMgWm9kVHlwZUFueSxcbiAgQ2FyZGluYWxpdHkgZXh0ZW5kcyBBcnJheUNhcmRpbmFsaXR5ID0gXCJtYW55XCJcbj4gZXh0ZW5kcyBab2RUeXBlPFxuICBhcnJheU91dHB1dFR5cGU8VCwgQ2FyZGluYWxpdHk+LFxuICBab2RBcnJheURlZjxUPixcbiAgQ2FyZGluYWxpdHkgZXh0ZW5kcyBcImF0bGVhc3RvbmVcIlxuICAgID8gW1RbXCJfaW5wdXRcIl0sIC4uLlRbXCJfaW5wdXRcIl1bXV1cbiAgICA6IFRbXCJfaW5wdXRcIl1bXVxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHsgY3R4LCBzdGF0dXMgfSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG5cbiAgICBjb25zdCBkZWYgPSB0aGlzLl9kZWY7XG5cbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuYXJyYXkpIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5hcnJheSxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBpZiAoZGVmLm1pbkxlbmd0aCAhPT0gbnVsbCkge1xuICAgICAgaWYgKGN0eC5kYXRhLmxlbmd0aCA8IGRlZi5taW5MZW5ndGgudmFsdWUpIHtcbiAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19zbWFsbCxcbiAgICAgICAgICBtaW5pbXVtOiBkZWYubWluTGVuZ3RoLnZhbHVlLFxuICAgICAgICAgIHR5cGU6IFwiYXJyYXlcIixcbiAgICAgICAgICBpbmNsdXNpdmU6IHRydWUsXG4gICAgICAgICAgbWVzc2FnZTogZGVmLm1pbkxlbmd0aC5tZXNzYWdlLFxuICAgICAgICB9KTtcbiAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKGRlZi5tYXhMZW5ndGggIT09IG51bGwpIHtcbiAgICAgIGlmIChjdHguZGF0YS5sZW5ndGggPiBkZWYubWF4TGVuZ3RoLnZhbHVlKSB7XG4gICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS50b29fYmlnLFxuICAgICAgICAgIG1heGltdW06IGRlZi5tYXhMZW5ndGgudmFsdWUsXG4gICAgICAgICAgdHlwZTogXCJhcnJheVwiLFxuICAgICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgICBtZXNzYWdlOiBkZWYubWF4TGVuZ3RoLm1lc3NhZ2UsXG4gICAgICAgIH0pO1xuICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgcmV0dXJuIFByb21pc2UuYWxsKFxuICAgICAgICAoY3R4LmRhdGEgYXMgYW55W10pLm1hcCgoaXRlbSwgaSkgPT4ge1xuICAgICAgICAgIHJldHVybiBkZWYudHlwZS5fcGFyc2VBc3luYyhcbiAgICAgICAgICAgIG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCBpdGVtLCBjdHgucGF0aCwgaSlcbiAgICAgICAgICApO1xuICAgICAgICB9KVxuICAgICAgKS50aGVuKChyZXN1bHQpID0+IHtcbiAgICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlQXJyYXkoc3RhdHVzLCByZXN1bHQpO1xuICAgICAgfSk7XG4gICAgfVxuXG4gICAgY29uc3QgcmVzdWx0ID0gKGN0eC5kYXRhIGFzIGFueVtdKS5tYXAoKGl0ZW0sIGkpID0+IHtcbiAgICAgIHJldHVybiBkZWYudHlwZS5fcGFyc2VTeW5jKFxuICAgICAgICBuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgaXRlbSwgY3R4LnBhdGgsIGkpXG4gICAgICApO1xuICAgIH0pO1xuXG4gICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlQXJyYXkoc3RhdHVzLCByZXN1bHQpO1xuICB9XG5cbiAgZ2V0IGVsZW1lbnQoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi50eXBlO1xuICB9XG5cbiAgbWluKG1pbkxlbmd0aDogbnVtYmVyLCBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpOiB0aGlzIHtcbiAgICByZXR1cm4gbmV3IFpvZEFycmF5KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIG1pbkxlbmd0aDogeyB2YWx1ZTogbWluTGVuZ3RoLCBtZXNzYWdlOiBlcnJvclV0aWwudG9TdHJpbmcobWVzc2FnZSkgfSxcbiAgICB9KSBhcyBhbnk7XG4gIH1cblxuICBtYXgobWF4TGVuZ3RoOiBudW1iZXIsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSk6IHRoaXMge1xuICAgIHJldHVybiBuZXcgWm9kQXJyYXkoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgbWF4TGVuZ3RoOiB7IHZhbHVlOiBtYXhMZW5ndGgsIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSB9LFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIGxlbmd0aChsZW46IG51bWJlciwgbWVzc2FnZT86IGVycm9yVXRpbC5FcnJNZXNzYWdlKTogdGhpcyB7XG4gICAgcmV0dXJuIHRoaXMubWluKGxlbiwgbWVzc2FnZSkubWF4KGxlbiwgbWVzc2FnZSkgYXMgYW55O1xuICB9XG5cbiAgbm9uZW1wdHkobWVzc2FnZT86IGVycm9yVXRpbC5FcnJNZXNzYWdlKTogWm9kQXJyYXk8VCwgXCJhdGxlYXN0b25lXCI+IHtcbiAgICByZXR1cm4gdGhpcy5taW4oMSwgbWVzc2FnZSkgYXMgYW55O1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IDxUIGV4dGVuZHMgWm9kVHlwZUFueT4oXG4gICAgc2NoZW1hOiBULFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RBcnJheTxUPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RBcnJheSh7XG4gICAgICB0eXBlOiBzY2hlbWEsXG4gICAgICBtaW5MZW5ndGg6IG51bGwsXG4gICAgICBtYXhMZW5ndGg6IG51bGwsXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEFycmF5LFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG5leHBvcnQgdHlwZSBab2ROb25FbXB0eUFycmF5PFQgZXh0ZW5kcyBab2RUeXBlQW55PiA9IFpvZEFycmF5PFQsIFwiYXRsZWFzdG9uZVwiPjtcblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE9iamVjdCAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cblxuZXhwb3J0IG5hbWVzcGFjZSBvYmplY3RVdGlsIHtcbiAgZXhwb3J0IHR5cGUgTWVyZ2VTaGFwZXM8VSBleHRlbmRzIFpvZFJhd1NoYXBlLCBWIGV4dGVuZHMgWm9kUmF3U2hhcGU+ID0ge1xuICAgIFtrIGluIEV4Y2x1ZGU8a2V5b2YgVSwga2V5b2YgVj5dOiBVW2tdO1xuICB9ICYgVjtcblxuICB0eXBlIG9wdGlvbmFsS2V5czxUIGV4dGVuZHMgb2JqZWN0PiA9IHtcbiAgICBbayBpbiBrZXlvZiBUXTogdW5kZWZpbmVkIGV4dGVuZHMgVFtrXSA/IGsgOiBuZXZlcjtcbiAgfVtrZXlvZiBUXTtcblxuICB0eXBlIHJlcXVpcmVkS2V5czxUIGV4dGVuZHMgb2JqZWN0PiA9IHtcbiAgICBbayBpbiBrZXlvZiBUXTogdW5kZWZpbmVkIGV4dGVuZHMgVFtrXSA/IG5ldmVyIDogaztcbiAgfVtrZXlvZiBUXTtcblxuICBleHBvcnQgdHlwZSBhZGRRdWVzdGlvbk1hcmtzPFQgZXh0ZW5kcyBvYmplY3Q+ID0gUGFydGlhbDxcbiAgICBQaWNrPFQsIG9wdGlvbmFsS2V5czxUPj5cbiAgPiAmXG4gICAgUGljazxULCByZXF1aXJlZEtleXM8VD4+O1xuXG4gIGV4cG9ydCB0eXBlIGlkZW50aXR5PFQ+ID0gVDtcbiAgZXhwb3J0IHR5cGUgZmxhdHRlbjxUIGV4dGVuZHMgb2JqZWN0PiA9IGlkZW50aXR5PHsgW2sgaW4ga2V5b2YgVF06IFRba10gfT47XG5cbiAgZXhwb3J0IHR5cGUgbm9OZXZlcktleXM8VCBleHRlbmRzIFpvZFJhd1NoYXBlPiA9IHtcbiAgICBbayBpbiBrZXlvZiBUXTogW1Rba11dIGV4dGVuZHMgW25ldmVyXSA/IG5ldmVyIDogaztcbiAgfVtrZXlvZiBUXTtcblxuICBleHBvcnQgdHlwZSBub05ldmVyPFQgZXh0ZW5kcyBab2RSYXdTaGFwZT4gPSBpZGVudGl0eTx7XG4gICAgW2sgaW4gbm9OZXZlcktleXM8VD5dOiBrIGV4dGVuZHMga2V5b2YgVCA/IFRba10gOiBuZXZlcjtcbiAgfT47XG5cbiAgZXhwb3J0IGNvbnN0IG1lcmdlU2hhcGVzID0gPFUgZXh0ZW5kcyBab2RSYXdTaGFwZSwgVCBleHRlbmRzIFpvZFJhd1NoYXBlPihcbiAgICBmaXJzdDogVSxcbiAgICBzZWNvbmQ6IFRcbiAgKTogVCAmIFUgPT4ge1xuICAgIHJldHVybiB7XG4gICAgICAuLi5maXJzdCxcbiAgICAgIC4uLnNlY29uZCwgLy8gc2Vjb25kIG92ZXJ3cml0ZXMgZmlyc3RcbiAgICB9O1xuICB9O1xufVxuXG5leHBvcnQgdHlwZSBleHRlbmRTaGFwZTxBLCBCPiA9IE9taXQ8QSwga2V5b2YgQj4gJiBCO1xuXG5jb25zdCBBdWdtZW50RmFjdG9yeSA9XG4gIDxEZWYgZXh0ZW5kcyBab2RPYmplY3REZWY+KGRlZjogRGVmKSA9PlxuICA8QXVnbWVudGF0aW9uIGV4dGVuZHMgWm9kUmF3U2hhcGU+KFxuICAgIGF1Z21lbnRhdGlvbjogQXVnbWVudGF0aW9uXG4gICk6IFpvZE9iamVjdDxcbiAgICBleHRlbmRTaGFwZTxSZXR1cm5UeXBlPERlZltcInNoYXBlXCJdPiwgQXVnbWVudGF0aW9uPixcbiAgICBEZWZbXCJ1bmtub3duS2V5c1wiXSxcbiAgICBEZWZbXCJjYXRjaGFsbFwiXVxuICA+ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi5kZWYsXG4gICAgICBzaGFwZTogKCkgPT4gKHtcbiAgICAgICAgLi4uZGVmLnNoYXBlKCksXG4gICAgICAgIC4uLmF1Z21lbnRhdGlvbixcbiAgICAgIH0pLFxuICAgIH0pIGFzIGFueTtcbiAgfTtcblxudHlwZSBVbmtub3duS2V5c1BhcmFtID0gXCJwYXNzdGhyb3VnaFwiIHwgXCJzdHJpY3RcIiB8IFwic3RyaXBcIjtcblxuZXhwb3J0IGludGVyZmFjZSBab2RPYmplY3REZWY8XG4gIFQgZXh0ZW5kcyBab2RSYXdTaGFwZSA9IFpvZFJhd1NoYXBlLFxuICBVbmtub3duS2V5cyBleHRlbmRzIFVua25vd25LZXlzUGFyYW0gPSBVbmtub3duS2V5c1BhcmFtLFxuICBDYXRjaGFsbCBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RUeXBlQW55XG4+IGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kT2JqZWN0O1xuICBzaGFwZTogKCkgPT4gVDtcbiAgY2F0Y2hhbGw6IENhdGNoYWxsO1xuICB1bmtub3duS2V5czogVW5rbm93bktleXM7XG59XG5cbmV4cG9ydCB0eXBlIGJhc2VPYmplY3RPdXRwdXRUeXBlPFNoYXBlIGV4dGVuZHMgWm9kUmF3U2hhcGU+ID1cbiAgb2JqZWN0VXRpbC5mbGF0dGVuPFxuICAgIG9iamVjdFV0aWwuYWRkUXVlc3Rpb25NYXJrczx7XG4gICAgICBbayBpbiBrZXlvZiBTaGFwZV06IFNoYXBlW2tdW1wiX291dHB1dFwiXTtcbiAgICB9PlxuICA+O1xuXG5leHBvcnQgdHlwZSBvYmplY3RPdXRwdXRUeXBlPFxuICBTaGFwZSBleHRlbmRzIFpvZFJhd1NoYXBlLFxuICBDYXRjaGFsbCBleHRlbmRzIFpvZFR5cGVBbnlcbj4gPSBab2RUeXBlQW55IGV4dGVuZHMgQ2F0Y2hhbGxcbiAgPyBiYXNlT2JqZWN0T3V0cHV0VHlwZTxTaGFwZT5cbiAgOiBvYmplY3RVdGlsLmZsYXR0ZW48XG4gICAgICBiYXNlT2JqZWN0T3V0cHV0VHlwZTxTaGFwZT4gJiB7IFtrOiBzdHJpbmddOiBDYXRjaGFsbFtcIl9vdXRwdXRcIl0gfVxuICAgID47XG5cbmV4cG9ydCB0eXBlIGJhc2VPYmplY3RJbnB1dFR5cGU8U2hhcGUgZXh0ZW5kcyBab2RSYXdTaGFwZT4gPSBvYmplY3RVdGlsLmZsYXR0ZW48XG4gIG9iamVjdFV0aWwuYWRkUXVlc3Rpb25NYXJrczx7XG4gICAgW2sgaW4ga2V5b2YgU2hhcGVdOiBTaGFwZVtrXVtcIl9pbnB1dFwiXTtcbiAgfT5cbj47XG5cbmV4cG9ydCB0eXBlIG9iamVjdElucHV0VHlwZTxcbiAgU2hhcGUgZXh0ZW5kcyBab2RSYXdTaGFwZSxcbiAgQ2F0Y2hhbGwgZXh0ZW5kcyBab2RUeXBlQW55XG4+ID0gWm9kVHlwZUFueSBleHRlbmRzIENhdGNoYWxsXG4gID8gYmFzZU9iamVjdElucHV0VHlwZTxTaGFwZT5cbiAgOiBvYmplY3RVdGlsLmZsYXR0ZW48XG4gICAgICBiYXNlT2JqZWN0SW5wdXRUeXBlPFNoYXBlPiAmIHsgW2s6IHN0cmluZ106IENhdGNoYWxsW1wiX2lucHV0XCJdIH1cbiAgICA+O1xuXG50eXBlIGRlb3B0aW9uYWw8VCBleHRlbmRzIFpvZFR5cGVBbnk+ID0gVCBleHRlbmRzIFpvZE9wdGlvbmFsPGluZmVyIFU+XG4gID8gZGVvcHRpb25hbDxVPlxuICA6IFQ7XG5cbmV4cG9ydCB0eXBlIFNvbWVab2RPYmplY3QgPSBab2RPYmplY3Q8XG4gIFpvZFJhd1NoYXBlLFxuICBVbmtub3duS2V5c1BhcmFtLFxuICBab2RUeXBlQW55LFxuICBhbnksXG4gIGFueVxuPjtcblxuZnVuY3Rpb24gZGVlcFBhcnRpYWxpZnkoc2NoZW1hOiBab2RUeXBlQW55KTogYW55IHtcbiAgaWYgKHNjaGVtYSBpbnN0YW5jZW9mIFpvZE9iamVjdCkge1xuICAgIGNvbnN0IG5ld1NoYXBlOiBhbnkgPSB7fTtcblxuICAgIGZvciAoY29uc3Qga2V5IGluIHNjaGVtYS5zaGFwZSkge1xuICAgICAgY29uc3QgZmllbGRTY2hlbWEgPSBzY2hlbWEuc2hhcGVba2V5XTtcbiAgICAgIG5ld1NoYXBlW2tleV0gPSBab2RPcHRpb25hbC5jcmVhdGUoZGVlcFBhcnRpYWxpZnkoZmllbGRTY2hlbWEpKTtcbiAgICB9XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4uc2NoZW1hLl9kZWYsXG4gICAgICBzaGFwZTogKCkgPT4gbmV3U2hhcGUsXG4gICAgfSkgYXMgYW55O1xuICB9IGVsc2UgaWYgKHNjaGVtYSBpbnN0YW5jZW9mIFpvZEFycmF5KSB7XG4gICAgcmV0dXJuIFpvZEFycmF5LmNyZWF0ZShkZWVwUGFydGlhbGlmeShzY2hlbWEuZWxlbWVudCkpO1xuICB9IGVsc2UgaWYgKHNjaGVtYSBpbnN0YW5jZW9mIFpvZE9wdGlvbmFsKSB7XG4gICAgcmV0dXJuIFpvZE9wdGlvbmFsLmNyZWF0ZShkZWVwUGFydGlhbGlmeShzY2hlbWEudW53cmFwKCkpKTtcbiAgfSBlbHNlIGlmIChzY2hlbWEgaW5zdGFuY2VvZiBab2ROdWxsYWJsZSkge1xuICAgIHJldHVybiBab2ROdWxsYWJsZS5jcmVhdGUoZGVlcFBhcnRpYWxpZnkoc2NoZW1hLnVud3JhcCgpKSk7XG4gIH0gZWxzZSBpZiAoc2NoZW1hIGluc3RhbmNlb2YgWm9kVHVwbGUpIHtcbiAgICByZXR1cm4gWm9kVHVwbGUuY3JlYXRlKFxuICAgICAgc2NoZW1hLml0ZW1zLm1hcCgoaXRlbTogYW55KSA9PiBkZWVwUGFydGlhbGlmeShpdGVtKSlcbiAgICApO1xuICB9IGVsc2Uge1xuICAgIHJldHVybiBzY2hlbWE7XG4gIH1cbn1cblxuZXhwb3J0IGNsYXNzIFpvZE9iamVjdDxcbiAgVCBleHRlbmRzIFpvZFJhd1NoYXBlLFxuICBVbmtub3duS2V5cyBleHRlbmRzIFVua25vd25LZXlzUGFyYW0gPSBcInN0cmlwXCIsXG4gIENhdGNoYWxsIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnksXG4gIE91dHB1dCA9IG9iamVjdE91dHB1dFR5cGU8VCwgQ2F0Y2hhbGw+LFxuICBJbnB1dCA9IG9iamVjdElucHV0VHlwZTxULCBDYXRjaGFsbD5cbj4gZXh0ZW5kcyBab2RUeXBlPE91dHB1dCwgWm9kT2JqZWN0RGVmPFQsIFVua25vd25LZXlzLCBDYXRjaGFsbD4sIElucHV0PiB7XG4gIHJlYWRvbmx5IF9zaGFwZSE6IFQ7XG4gIHJlYWRvbmx5IF91bmtub3duS2V5cyE6IFVua25vd25LZXlzO1xuICByZWFkb25seSBfY2F0Y2hhbGwhOiBDYXRjaGFsbDtcbiAgcHJpdmF0ZSBfY2FjaGVkOiB7IHNoYXBlOiBUOyBrZXlzOiBzdHJpbmdbXSB9IHwgbnVsbCA9IG51bGw7XG5cbiAgX2dldENhY2hlZCgpOiB7IHNoYXBlOiBUOyBrZXlzOiBzdHJpbmdbXSB9IHtcbiAgICBpZiAodGhpcy5fY2FjaGVkICE9PSBudWxsKSByZXR1cm4gdGhpcy5fY2FjaGVkO1xuICAgIGNvbnN0IHNoYXBlID0gdGhpcy5fZGVmLnNoYXBlKCk7XG4gICAgY29uc3Qga2V5cyA9IHV0aWwub2JqZWN0S2V5cyhzaGFwZSk7XG4gICAgcmV0dXJuICh0aGlzLl9jYWNoZWQgPSB7IHNoYXBlLCBrZXlzIH0pO1xuICB9XG5cbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgY29uc3QgcGFyc2VkVHlwZSA9IHRoaXMuX2dldFR5cGUoaW5wdXQpO1xuICAgIGlmIChwYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLm9iamVjdCkge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm9iamVjdCxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBjb25zdCB7IHN0YXR1cywgY3R4IH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuXG4gICAgY29uc3QgeyBzaGFwZSwga2V5czogc2hhcGVLZXlzIH0gPSB0aGlzLl9nZXRDYWNoZWQoKTtcbiAgICBjb25zdCBleHRyYUtleXM6IHN0cmluZ1tdID0gW107XG4gICAgZm9yIChjb25zdCBrZXkgaW4gY3R4LmRhdGEpIHtcbiAgICAgIGlmICghc2hhcGVLZXlzLmluY2x1ZGVzKGtleSkpIHtcbiAgICAgICAgZXh0cmFLZXlzLnB1c2goa2V5KTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBjb25zdCBwYWlyczoge1xuICAgICAga2V5OiBQYXJzZVJldHVyblR5cGU8YW55PjtcbiAgICAgIHZhbHVlOiBQYXJzZVJldHVyblR5cGU8YW55PjtcbiAgICAgIGFsd2F5c1NldD86IGJvb2xlYW47XG4gICAgfVtdID0gW107XG4gICAgZm9yIChjb25zdCBrZXkgb2Ygc2hhcGVLZXlzKSB7XG4gICAgICBjb25zdCBrZXlWYWxpZGF0b3IgPSBzaGFwZVtrZXldO1xuICAgICAgY29uc3QgdmFsdWUgPSBjdHguZGF0YVtrZXldO1xuICAgICAgcGFpcnMucHVzaCh7XG4gICAgICAgIGtleTogeyBzdGF0dXM6IFwidmFsaWRcIiwgdmFsdWU6IGtleSB9LFxuICAgICAgICB2YWx1ZToga2V5VmFsaWRhdG9yLl9wYXJzZShcbiAgICAgICAgICBuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgdmFsdWUsIGN0eC5wYXRoLCBrZXkpXG4gICAgICAgICksXG4gICAgICAgIGFsd2F5c1NldDoga2V5IGluIGN0eC5kYXRhLFxuICAgICAgfSk7XG4gICAgfVxuXG4gICAgaWYgKHRoaXMuX2RlZi5jYXRjaGFsbCBpbnN0YW5jZW9mIFpvZE5ldmVyKSB7XG4gICAgICBjb25zdCB1bmtub3duS2V5cyA9IHRoaXMuX2RlZi51bmtub3duS2V5cztcblxuICAgICAgaWYgKHVua25vd25LZXlzID09PSBcInBhc3N0aHJvdWdoXCIpIHtcbiAgICAgICAgZm9yIChjb25zdCBrZXkgb2YgZXh0cmFLZXlzKSB7XG4gICAgICAgICAgcGFpcnMucHVzaCh7XG4gICAgICAgICAgICBrZXk6IHsgc3RhdHVzOiBcInZhbGlkXCIsIHZhbHVlOiBrZXkgfSxcbiAgICAgICAgICAgIHZhbHVlOiB7IHN0YXR1czogXCJ2YWxpZFwiLCB2YWx1ZTogY3R4LmRhdGFba2V5XSB9LFxuICAgICAgICAgIH0pO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKHVua25vd25LZXlzID09PSBcInN0cmljdFwiKSB7XG4gICAgICAgIGlmIChleHRyYUtleXMubGVuZ3RoID4gMCkge1xuICAgICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnVucmVjb2duaXplZF9rZXlzLFxuICAgICAgICAgICAga2V5czogZXh0cmFLZXlzLFxuICAgICAgICAgIH0pO1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG4gICAgICB9IGVsc2UgaWYgKHVua25vd25LZXlzID09PSBcInN0cmlwXCIpIHtcbiAgICAgIH0gZWxzZSB7XG4gICAgICAgIHRocm93IG5ldyBFcnJvcihgSW50ZXJuYWwgWm9kT2JqZWN0IGVycm9yOiBpbnZhbGlkIHVua25vd25LZXlzIHZhbHVlLmApO1xuICAgICAgfVxuICAgIH0gZWxzZSB7XG4gICAgICAvLyBydW4gY2F0Y2hhbGwgdmFsaWRhdGlvblxuICAgICAgY29uc3QgY2F0Y2hhbGwgPSB0aGlzLl9kZWYuY2F0Y2hhbGw7XG5cbiAgICAgIGZvciAoY29uc3Qga2V5IG9mIGV4dHJhS2V5cykge1xuICAgICAgICBjb25zdCB2YWx1ZSA9IGN0eC5kYXRhW2tleV07XG4gICAgICAgIHBhaXJzLnB1c2goe1xuICAgICAgICAgIGtleTogeyBzdGF0dXM6IFwidmFsaWRcIiwgdmFsdWU6IGtleSB9LFxuICAgICAgICAgIHZhbHVlOiBjYXRjaGFsbC5fcGFyc2UoXG4gICAgICAgICAgICBuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwgdmFsdWUsIGN0eC5wYXRoLCBrZXkpIC8vLCBjdHguY2hpbGQoa2V5KSwgdmFsdWUsIGdldFBhcnNlZFR5cGUodmFsdWUpXG4gICAgICAgICAgKSxcbiAgICAgICAgICBhbHdheXNTZXQ6IGtleSBpbiBjdHguZGF0YSxcbiAgICAgICAgfSk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBQcm9taXNlLnJlc29sdmUoKVxuICAgICAgICAudGhlbihhc3luYyAoKSA9PiB7XG4gICAgICAgICAgY29uc3Qgc3luY1BhaXJzOiBhbnlbXSA9IFtdO1xuICAgICAgICAgIGZvciAoY29uc3QgcGFpciBvZiBwYWlycykge1xuICAgICAgICAgICAgY29uc3Qga2V5ID0gYXdhaXQgcGFpci5rZXk7XG4gICAgICAgICAgICBzeW5jUGFpcnMucHVzaCh7XG4gICAgICAgICAgICAgIGtleSxcbiAgICAgICAgICAgICAgdmFsdWU6IGF3YWl0IHBhaXIudmFsdWUsXG4gICAgICAgICAgICAgIGFsd2F5c1NldDogcGFpci5hbHdheXNTZXQsXG4gICAgICAgICAgICB9KTtcbiAgICAgICAgICB9XG4gICAgICAgICAgcmV0dXJuIHN5bmNQYWlycztcbiAgICAgICAgfSlcbiAgICAgICAgLnRoZW4oKHN5bmNQYWlycykgPT4ge1xuICAgICAgICAgIHJldHVybiBQYXJzZVN0YXR1cy5tZXJnZU9iamVjdFN5bmMoc3RhdHVzLCBzeW5jUGFpcnMpO1xuICAgICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlT2JqZWN0U3luYyhzdGF0dXMsIHBhaXJzIGFzIGFueSk7XG4gICAgfVxuICB9XG5cbiAgZ2V0IHNoYXBlKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuc2hhcGUoKTtcbiAgfVxuXG4gIHN0cmljdChtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpOiBab2RPYmplY3Q8VCwgXCJzdHJpY3RcIiwgQ2F0Y2hhbGw+IHtcbiAgICBlcnJvclV0aWwuZXJyVG9PYmo7XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgdW5rbm93bktleXM6IFwic3RyaWN0XCIsXG4gICAgICAuLi4obWVzc2FnZSAhPT0gdW5kZWZpbmVkXG4gICAgICAgID8ge1xuICAgICAgICAgICAgZXJyb3JNYXA6IChpc3N1ZSwgY3R4KSA9PiB7XG4gICAgICAgICAgICAgIGNvbnN0IGRlZmF1bHRFcnJvciA9XG4gICAgICAgICAgICAgICAgdGhpcy5fZGVmLmVycm9yTWFwPy4oaXNzdWUsIGN0eCkubWVzc2FnZSA/PyBjdHguZGVmYXVsdEVycm9yO1xuICAgICAgICAgICAgICBpZiAoaXNzdWUuY29kZSA9PT0gXCJ1bnJlY29nbml6ZWRfa2V5c1wiKVxuICAgICAgICAgICAgICAgIHJldHVybiB7XG4gICAgICAgICAgICAgICAgICBtZXNzYWdlOiBlcnJvclV0aWwuZXJyVG9PYmoobWVzc2FnZSkubWVzc2FnZSA/PyBkZWZhdWx0RXJyb3IsXG4gICAgICAgICAgICAgICAgfTtcbiAgICAgICAgICAgICAgcmV0dXJuIHtcbiAgICAgICAgICAgICAgICBtZXNzYWdlOiBkZWZhdWx0RXJyb3IsXG4gICAgICAgICAgICAgIH07XG4gICAgICAgICAgICB9LFxuICAgICAgICAgIH1cbiAgICAgICAgOiB7fSksXG4gICAgfSkgYXMgYW55O1xuICB9XG5cbiAgc3RyaXAoKTogWm9kT2JqZWN0PFQsIFwic3RyaXBcIiwgQ2F0Y2hhbGw+IHtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICB1bmtub3duS2V5czogXCJzdHJpcFwiLFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIHBhc3N0aHJvdWdoKCk6IFpvZE9iamVjdDxULCBcInBhc3N0aHJvdWdoXCIsIENhdGNoYWxsPiB7XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgdW5rbm93bktleXM6IFwicGFzc3Rocm91Z2hcIixcbiAgICB9KSBhcyBhbnk7XG4gIH1cblxuICAvKipcbiAgICogQGRlcHJlY2F0ZWQgSW4gbW9zdCBjYXNlcywgdGhpcyBpcyBubyBsb25nZXIgbmVlZGVkIC0gdW5rbm93biBwcm9wZXJ0aWVzIGFyZSBub3cgc2lsZW50bHkgc3RyaXBwZWQuXG4gICAqIElmIHlvdSB3YW50IHRvIHBhc3MgdGhyb3VnaCB1bmtub3duIHByb3BlcnRpZXMsIHVzZSBgLnBhc3N0aHJvdWdoKClgIGluc3RlYWQuXG4gICAqL1xuICBub25zdHJpY3QgPSB0aGlzLnBhc3N0aHJvdWdoO1xuXG4gIGF1Z21lbnQgPSBBdWdtZW50RmFjdG9yeTxab2RPYmplY3REZWY8VCwgVW5rbm93bktleXMsIENhdGNoYWxsPj4odGhpcy5fZGVmKTtcbiAgZXh0ZW5kID0gQXVnbWVudEZhY3Rvcnk8Wm9kT2JqZWN0RGVmPFQsIFVua25vd25LZXlzLCBDYXRjaGFsbD4+KHRoaXMuX2RlZik7XG5cbiAgc2V0S2V5PEtleSBleHRlbmRzIHN0cmluZywgU2NoZW1hIGV4dGVuZHMgWm9kVHlwZUFueT4oXG4gICAga2V5OiBLZXksXG4gICAgc2NoZW1hOiBTY2hlbWFcbiAgKTogWm9kT2JqZWN0PFQgJiB7IFtrIGluIEtleV06IFNjaGVtYSB9LCBVbmtub3duS2V5cywgQ2F0Y2hhbGw+IHtcbiAgICByZXR1cm4gdGhpcy5hdWdtZW50KHsgW2tleV06IHNjaGVtYSB9KSBhcyBhbnk7XG4gIH1cblxuICAvKipcbiAgICogUHJpb3IgdG8gem9kQDEuMC4xMiB0aGVyZSB3YXMgYSBidWcgaW4gdGhlXG4gICAqIGluZmVycmVkIHR5cGUgb2YgbWVyZ2VkIG9iamVjdHMuIFBsZWFzZVxuICAgKiB1cGdyYWRlIGlmIHlvdSBhcmUgZXhwZXJpZW5jaW5nIGlzc3Vlcy5cbiAgICovXG4gIG1lcmdlPEluY29taW5nIGV4dGVuZHMgQW55Wm9kT2JqZWN0PihcbiAgICBtZXJnaW5nOiBJbmNvbWluZ1xuICApOiAvL1pvZE9iamVjdDxUICYgSW5jb21pbmdbXCJfc2hhcGVcIl0sIFVua25vd25LZXlzLCBDYXRjaGFsbD4gPSAobWVyZ2luZykgPT4ge1xuICBab2RPYmplY3Q8ZXh0ZW5kU2hhcGU8VCwgSW5jb21pbmdbXCJfc2hhcGVcIl0+LCBVbmtub3duS2V5cywgQ2F0Y2hhbGw+IHtcbiAgICAvLyBjb25zdCBtZXJnZWRTaGFwZSA9IG9iamVjdFV0aWwubWVyZ2VTaGFwZXMoXG4gICAgLy8gICB0aGlzLl9kZWYuc2hhcGUoKSxcbiAgICAvLyAgIG1lcmdpbmcuX2RlZi5zaGFwZSgpXG4gICAgLy8gKTtcbiAgICBjb25zdCBtZXJnZWQ6IGFueSA9IG5ldyBab2RPYmplY3Qoe1xuICAgICAgdW5rbm93bktleXM6IG1lcmdpbmcuX2RlZi51bmtub3duS2V5cyxcbiAgICAgIGNhdGNoYWxsOiBtZXJnaW5nLl9kZWYuY2F0Y2hhbGwsXG4gICAgICBzaGFwZTogKCkgPT5cbiAgICAgICAgb2JqZWN0VXRpbC5tZXJnZVNoYXBlcyh0aGlzLl9kZWYuc2hhcGUoKSwgbWVyZ2luZy5fZGVmLnNoYXBlKCkpLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RPYmplY3QsXG4gICAgfSkgYXMgYW55O1xuICAgIHJldHVybiBtZXJnZWQ7XG4gIH1cblxuICBjYXRjaGFsbDxJbmRleCBleHRlbmRzIFpvZFR5cGVBbnk+KFxuICAgIGluZGV4OiBJbmRleFxuICApOiBab2RPYmplY3Q8VCwgVW5rbm93bktleXMsIEluZGV4PiB7XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgY2F0Y2hhbGw6IGluZGV4LFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIHBpY2s8TWFzayBleHRlbmRzIHsgW2sgaW4ga2V5b2YgVF0/OiB0cnVlIH0+KFxuICAgIG1hc2s6IE1hc2tcbiAgKTogWm9kT2JqZWN0PFBpY2s8VCwgRXh0cmFjdDxrZXlvZiBULCBrZXlvZiBNYXNrPj4sIFVua25vd25LZXlzLCBDYXRjaGFsbD4ge1xuICAgIGNvbnN0IHNoYXBlOiBhbnkgPSB7fTtcbiAgICB1dGlsLm9iamVjdEtleXMobWFzaykubWFwKChrZXkpID0+IHtcbiAgICAgIC8vIG9ubHkgYWRkIHRvIHNoYXBlIGlmIGtleSBjb3JyZXNwb25kcyB0byBhbiBlbGVtZW50IG9mIHRoZSBjdXJyZW50IHNoYXBlXG4gICAgICBpZiAodGhpcy5zaGFwZVtrZXldKSBzaGFwZVtrZXldID0gdGhpcy5zaGFwZVtrZXldO1xuICAgIH0pO1xuICAgIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIHNoYXBlOiAoKSA9PiBzaGFwZSxcbiAgICB9KSBhcyBhbnk7XG4gIH1cblxuICBvbWl0PE1hc2sgZXh0ZW5kcyB7IFtrIGluIGtleW9mIFRdPzogdHJ1ZSB9PihcbiAgICBtYXNrOiBNYXNrXG4gICk6IFpvZE9iamVjdDxPbWl0PFQsIGtleW9mIE1hc2s+LCBVbmtub3duS2V5cywgQ2F0Y2hhbGw+IHtcbiAgICBjb25zdCBzaGFwZTogYW55ID0ge307XG4gICAgdXRpbC5vYmplY3RLZXlzKHRoaXMuc2hhcGUpLm1hcCgoa2V5KSA9PiB7XG4gICAgICBpZiAodXRpbC5vYmplY3RLZXlzKG1hc2spLmluZGV4T2Yoa2V5KSA9PT0gLTEpIHtcbiAgICAgICAgc2hhcGVba2V5XSA9IHRoaXMuc2hhcGVba2V5XTtcbiAgICAgIH1cbiAgICB9KTtcbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBzaGFwZTogKCkgPT4gc2hhcGUsXG4gICAgfSkgYXMgYW55O1xuICB9XG5cbiAgZGVlcFBhcnRpYWwoKTogcGFydGlhbFV0aWwuRGVlcFBhcnRpYWw8dGhpcz4ge1xuICAgIHJldHVybiBkZWVwUGFydGlhbGlmeSh0aGlzKSBhcyBhbnk7XG4gIH1cblxuICBwYXJ0aWFsKCk6IFpvZE9iamVjdDxcbiAgICB7IFtrIGluIGtleW9mIFRdOiBab2RPcHRpb25hbDxUW2tdPiB9LFxuICAgIFVua25vd25LZXlzLFxuICAgIENhdGNoYWxsXG4gID47XG4gIHBhcnRpYWw8TWFzayBleHRlbmRzIHsgW2sgaW4ga2V5b2YgVF0/OiB0cnVlIH0+KFxuICAgIG1hc2s6IE1hc2tcbiAgKTogWm9kT2JqZWN0PFxuICAgIG9iamVjdFV0aWwubm9OZXZlcjx7XG4gICAgICBbayBpbiBrZXlvZiBUXTogayBleHRlbmRzIGtleW9mIE1hc2sgPyBab2RPcHRpb25hbDxUW2tdPiA6IFRba107XG4gICAgfT4sXG4gICAgVW5rbm93bktleXMsXG4gICAgQ2F0Y2hhbGxcbiAgPjtcbiAgcGFydGlhbChtYXNrPzogYW55KSB7XG4gICAgY29uc3QgbmV3U2hhcGU6IGFueSA9IHt9O1xuICAgIGlmIChtYXNrKSB7XG4gICAgICB1dGlsLm9iamVjdEtleXModGhpcy5zaGFwZSkubWFwKChrZXkpID0+IHtcbiAgICAgICAgaWYgKHV0aWwub2JqZWN0S2V5cyhtYXNrKS5pbmRleE9mKGtleSkgPT09IC0xKSB7XG4gICAgICAgICAgbmV3U2hhcGVba2V5XSA9IHRoaXMuc2hhcGVba2V5XTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICBuZXdTaGFwZVtrZXldID0gdGhpcy5zaGFwZVtrZXldLm9wdGlvbmFsKCk7XG4gICAgICAgIH1cbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgICAuLi50aGlzLl9kZWYsXG4gICAgICAgIHNoYXBlOiAoKSA9PiBuZXdTaGFwZSxcbiAgICAgIH0pIGFzIGFueTtcbiAgICB9IGVsc2Uge1xuICAgICAgZm9yIChjb25zdCBrZXkgaW4gdGhpcy5zaGFwZSkge1xuICAgICAgICBjb25zdCBmaWVsZFNjaGVtYSA9IHRoaXMuc2hhcGVba2V5XTtcbiAgICAgICAgbmV3U2hhcGVba2V5XSA9IGZpZWxkU2NoZW1hLm9wdGlvbmFsKCk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgLi4udGhpcy5fZGVmLFxuICAgICAgc2hhcGU6ICgpID0+IG5ld1NoYXBlLFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIHJlcXVpcmVkKCk6IFpvZE9iamVjdDxcbiAgICB7IFtrIGluIGtleW9mIFRdOiBkZW9wdGlvbmFsPFRba10+IH0sXG4gICAgVW5rbm93bktleXMsXG4gICAgQ2F0Y2hhbGxcbiAgPiB7XG4gICAgY29uc3QgbmV3U2hhcGU6IGFueSA9IHt9O1xuICAgIGZvciAoY29uc3Qga2V5IGluIHRoaXMuc2hhcGUpIHtcbiAgICAgIGNvbnN0IGZpZWxkU2NoZW1hID0gdGhpcy5zaGFwZVtrZXldO1xuICAgICAgbGV0IG5ld0ZpZWxkID0gZmllbGRTY2hlbWE7XG4gICAgICB3aGlsZSAobmV3RmllbGQgaW5zdGFuY2VvZiBab2RPcHRpb25hbCkge1xuICAgICAgICBuZXdGaWVsZCA9IChuZXdGaWVsZCBhcyBab2RPcHRpb25hbDxhbnk+KS5fZGVmLmlubmVyVHlwZTtcbiAgICAgIH1cblxuICAgICAgbmV3U2hhcGVba2V5XSA9IG5ld0ZpZWxkO1xuICAgIH1cbiAgICByZXR1cm4gbmV3IFpvZE9iamVjdCh7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICBzaGFwZTogKCkgPT4gbmV3U2hhcGUsXG4gICAgfSkgYXMgYW55O1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IDxUIGV4dGVuZHMgWm9kUmF3U2hhcGU+KFxuICAgIHNoYXBlOiBULFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RPYmplY3Q8VD4gPT4ge1xuICAgIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICAgIHNoYXBlOiAoKSA9PiBzaGFwZSxcbiAgICAgIHVua25vd25LZXlzOiBcInN0cmlwXCIsXG4gICAgICBjYXRjaGFsbDogWm9kTmV2ZXIuY3JlYXRlKCksXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE9iamVjdCxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KSBhcyBhbnk7XG4gIH07XG5cbiAgc3RhdGljIHN0cmljdENyZWF0ZSA9IDxUIGV4dGVuZHMgWm9kUmF3U2hhcGU+KFxuICAgIHNoYXBlOiBULFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RPYmplY3Q8VCwgXCJzdHJpY3RcIj4gPT4ge1xuICAgIHJldHVybiBuZXcgWm9kT2JqZWN0KHtcbiAgICAgIHNoYXBlOiAoKSA9PiBzaGFwZSxcbiAgICAgIHVua25vd25LZXlzOiBcInN0cmljdFwiLFxuICAgICAgY2F0Y2hhbGw6IFpvZE5ldmVyLmNyZWF0ZSgpLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RPYmplY3QsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSkgYXMgYW55O1xuICB9O1xuXG4gIHN0YXRpYyBsYXp5Y3JlYXRlID0gPFQgZXh0ZW5kcyBab2RSYXdTaGFwZT4oXG4gICAgc2hhcGU6ICgpID0+IFQsXG4gICAgcGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zXG4gICk6IFpvZE9iamVjdDxUPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RPYmplY3Qoe1xuICAgICAgc2hhcGUsXG4gICAgICB1bmtub3duS2V5czogXCJzdHJpcFwiLFxuICAgICAgY2F0Y2hhbGw6IFpvZE5ldmVyLmNyZWF0ZSgpLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RPYmplY3QsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSkgYXMgYW55O1xuICB9O1xufVxuXG5leHBvcnQgdHlwZSBBbnlab2RPYmplY3QgPSBab2RPYmplY3Q8YW55LCBhbnksIGFueT47XG5cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICBab2RVbmlvbiAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbnR5cGUgWm9kVW5pb25PcHRpb25zID0gUmVhZG9ubHk8W1pvZFR5cGVBbnksIC4uLlpvZFR5cGVBbnlbXV0+O1xuZXhwb3J0IGludGVyZmFjZSBab2RVbmlvbkRlZjxcbiAgVCBleHRlbmRzIFpvZFVuaW9uT3B0aW9ucyA9IFJlYWRvbmx5PFxuICAgIFtab2RUeXBlQW55LCBab2RUeXBlQW55LCAuLi5ab2RUeXBlQW55W11dXG4gID5cbj4gZXh0ZW5kcyBab2RUeXBlRGVmIHtcbiAgb3B0aW9uczogVDtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RVbmlvbjtcbn1cblxuZXhwb3J0IGNsYXNzIFpvZFVuaW9uPFQgZXh0ZW5kcyBab2RVbmlvbk9wdGlvbnM+IGV4dGVuZHMgWm9kVHlwZTxcbiAgVFtudW1iZXJdW1wiX291dHB1dFwiXSxcbiAgWm9kVW5pb25EZWY8VD4sXG4gIFRbbnVtYmVyXVtcIl9pbnB1dFwiXVxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHsgY3R4IH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGNvbnN0IG9wdGlvbnMgPSB0aGlzLl9kZWYub3B0aW9ucztcblxuICAgIGZ1bmN0aW9uIGhhbmRsZVJlc3VsdHMoXG4gICAgICByZXN1bHRzOiB7IGN0eDogUGFyc2VDb250ZXh0OyByZXN1bHQ6IFN5bmNQYXJzZVJldHVyblR5cGU8YW55PiB9W11cbiAgICApIHtcbiAgICAgIC8vIHJldHVybiBmaXJzdCBpc3N1ZS1mcmVlIHZhbGlkYXRpb24gaWYgaXQgZXhpc3RzXG4gICAgICBmb3IgKGNvbnN0IHJlc3VsdCBvZiByZXN1bHRzKSB7XG4gICAgICAgIGlmIChyZXN1bHQucmVzdWx0LnN0YXR1cyA9PT0gXCJ2YWxpZFwiKSB7XG4gICAgICAgICAgcmV0dXJuIHJlc3VsdC5yZXN1bHQ7XG4gICAgICAgIH1cbiAgICAgIH1cblxuICAgICAgZm9yIChjb25zdCByZXN1bHQgb2YgcmVzdWx0cykge1xuICAgICAgICBpZiAocmVzdWx0LnJlc3VsdC5zdGF0dXMgPT09IFwiZGlydHlcIikge1xuICAgICAgICAgIC8vIGFkZCBpc3N1ZXMgZnJvbSBkaXJ0eSBvcHRpb25cblxuICAgICAgICAgIGN0eC5jb21tb24uaXNzdWVzLnB1c2goLi4ucmVzdWx0LmN0eC5jb21tb24uaXNzdWVzKTtcbiAgICAgICAgICByZXR1cm4gcmVzdWx0LnJlc3VsdDtcbiAgICAgICAgfVxuICAgICAgfVxuXG4gICAgICAvLyByZXR1cm4gaW52YWxpZFxuICAgICAgY29uc3QgdW5pb25FcnJvcnMgPSByZXN1bHRzLm1hcChcbiAgICAgICAgKHJlc3VsdCkgPT4gbmV3IFpvZEVycm9yKHJlc3VsdC5jdHguY29tbW9uLmlzc3VlcylcbiAgICAgICk7XG5cbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF91bmlvbixcbiAgICAgICAgdW5pb25FcnJvcnMsXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cblxuICAgIGlmIChjdHguY29tbW9uLmFzeW5jKSB7XG4gICAgICByZXR1cm4gUHJvbWlzZS5hbGwoXG4gICAgICAgIG9wdGlvbnMubWFwKGFzeW5jIChvcHRpb24pID0+IHtcbiAgICAgICAgICBjb25zdCBjaGlsZEN0eDogUGFyc2VDb250ZXh0ID0ge1xuICAgICAgICAgICAgLi4uY3R4LFxuICAgICAgICAgICAgY29tbW9uOiB7XG4gICAgICAgICAgICAgIC4uLmN0eC5jb21tb24sXG4gICAgICAgICAgICAgIGlzc3VlczogW10sXG4gICAgICAgICAgICB9LFxuICAgICAgICAgICAgcGFyZW50OiBudWxsLFxuICAgICAgICAgIH07XG4gICAgICAgICAgcmV0dXJuIHtcbiAgICAgICAgICAgIHJlc3VsdDogYXdhaXQgb3B0aW9uLl9wYXJzZUFzeW5jKHtcbiAgICAgICAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgICAgICBwYXJlbnQ6IGNoaWxkQ3R4LFxuICAgICAgICAgICAgfSksXG4gICAgICAgICAgICBjdHg6IGNoaWxkQ3R4LFxuICAgICAgICAgIH07XG4gICAgICAgIH0pXG4gICAgICApLnRoZW4oaGFuZGxlUmVzdWx0cyk7XG4gICAgfSBlbHNlIHtcbiAgICAgIGxldCBkaXJ0eTogdW5kZWZpbmVkIHwgeyByZXN1bHQ6IERJUlRZPGFueT47IGN0eDogUGFyc2VDb250ZXh0IH0gPVxuICAgICAgICB1bmRlZmluZWQ7XG4gICAgICBjb25zdCBpc3N1ZXM6IFpvZElzc3VlW11bXSA9IFtdO1xuICAgICAgZm9yIChjb25zdCBvcHRpb24gb2Ygb3B0aW9ucykge1xuICAgICAgICBjb25zdCBjaGlsZEN0eDogUGFyc2VDb250ZXh0ID0ge1xuICAgICAgICAgIC4uLmN0eCxcbiAgICAgICAgICBjb21tb246IHtcbiAgICAgICAgICAgIC4uLmN0eC5jb21tb24sXG4gICAgICAgICAgICBpc3N1ZXM6IFtdLFxuICAgICAgICAgIH0sXG4gICAgICAgICAgcGFyZW50OiBudWxsLFxuICAgICAgICB9O1xuICAgICAgICBjb25zdCByZXN1bHQgPSBvcHRpb24uX3BhcnNlU3luYyh7XG4gICAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgcGFyZW50OiBjaGlsZEN0eCxcbiAgICAgICAgfSk7XG5cbiAgICAgICAgaWYgKHJlc3VsdC5zdGF0dXMgPT09IFwidmFsaWRcIikge1xuICAgICAgICAgIHJldHVybiByZXN1bHQ7XG4gICAgICAgIH0gZWxzZSBpZiAocmVzdWx0LnN0YXR1cyA9PT0gXCJkaXJ0eVwiICYmICFkaXJ0eSkge1xuICAgICAgICAgIGRpcnR5ID0geyByZXN1bHQsIGN0eDogY2hpbGRDdHggfTtcbiAgICAgICAgfVxuXG4gICAgICAgIGlmIChjaGlsZEN0eC5jb21tb24uaXNzdWVzLmxlbmd0aCkge1xuICAgICAgICAgIGlzc3Vlcy5wdXNoKGNoaWxkQ3R4LmNvbW1vbi5pc3N1ZXMpO1xuICAgICAgICB9XG4gICAgICB9XG5cbiAgICAgIGlmIChkaXJ0eSkge1xuICAgICAgICBjdHguY29tbW9uLmlzc3Vlcy5wdXNoKC4uLmRpcnR5LmN0eC5jb21tb24uaXNzdWVzKTtcbiAgICAgICAgcmV0dXJuIGRpcnR5LnJlc3VsdDtcbiAgICAgIH1cblxuICAgICAgY29uc3QgdW5pb25FcnJvcnMgPSBpc3N1ZXMubWFwKChpc3N1ZXMpID0+IG5ldyBab2RFcnJvcihpc3N1ZXMpKTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF91bmlvbixcbiAgICAgICAgdW5pb25FcnJvcnMsXG4gICAgICB9KTtcblxuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICB9XG5cbiAgZ2V0IG9wdGlvbnMoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5vcHRpb25zO1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IDxcbiAgICBUIGV4dGVuZHMgUmVhZG9ubHk8W1pvZFR5cGVBbnksIFpvZFR5cGVBbnksIC4uLlpvZFR5cGVBbnlbXV0+XG4gID4oXG4gICAgdHlwZXM6IFQsXG4gICAgcGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zXG4gICk6IFpvZFVuaW9uPFQ+ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZFVuaW9uKHtcbiAgICAgIG9wdGlvbnM6IHR5cGVzLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RVbmlvbixcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZERpc2NyaW1pbmF0ZWRVbmlvbiAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cblxuZXhwb3J0IHR5cGUgWm9kRGlzY3JpbWluYXRlZFVuaW9uT3B0aW9uPFxuICBEaXNjcmltaW5hdG9yIGV4dGVuZHMgc3RyaW5nLFxuICBEaXNjcmltaW5hdG9yVmFsdWUgZXh0ZW5kcyBQcmltaXRpdmVcbj4gPSBab2RPYmplY3Q8XG4gIHsgW2tleSBpbiBEaXNjcmltaW5hdG9yXTogWm9kTGl0ZXJhbDxEaXNjcmltaW5hdG9yVmFsdWU+IH0gJiBab2RSYXdTaGFwZSxcbiAgYW55LFxuICBhbnlcbj47XG5cbmV4cG9ydCBpbnRlcmZhY2UgWm9kRGlzY3JpbWluYXRlZFVuaW9uRGVmPFxuICBEaXNjcmltaW5hdG9yIGV4dGVuZHMgc3RyaW5nLFxuICBEaXNjcmltaW5hdG9yVmFsdWUgZXh0ZW5kcyBQcmltaXRpdmUsXG4gIE9wdGlvbiBleHRlbmRzIFpvZERpc2NyaW1pbmF0ZWRVbmlvbk9wdGlvbjxEaXNjcmltaW5hdG9yLCBEaXNjcmltaW5hdG9yVmFsdWU+XG4+IGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIGRpc2NyaW1pbmF0b3I6IERpc2NyaW1pbmF0b3I7XG4gIG9wdGlvbnM6IE1hcDxEaXNjcmltaW5hdG9yVmFsdWUsIE9wdGlvbj47XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kRGlzY3JpbWluYXRlZFVuaW9uO1xufVxuXG5leHBvcnQgY2xhc3MgWm9kRGlzY3JpbWluYXRlZFVuaW9uPFxuICBEaXNjcmltaW5hdG9yIGV4dGVuZHMgc3RyaW5nLFxuICBEaXNjcmltaW5hdG9yVmFsdWUgZXh0ZW5kcyBQcmltaXRpdmUsXG4gIE9wdGlvbiBleHRlbmRzIFpvZERpc2NyaW1pbmF0ZWRVbmlvbk9wdGlvbjxEaXNjcmltaW5hdG9yLCBEaXNjcmltaW5hdG9yVmFsdWU+XG4+IGV4dGVuZHMgWm9kVHlwZTxcbiAgT3B0aW9uW1wiX291dHB1dFwiXSxcbiAgWm9kRGlzY3JpbWluYXRlZFVuaW9uRGVmPERpc2NyaW1pbmF0b3IsIERpc2NyaW1pbmF0b3JWYWx1ZSwgT3B0aW9uPixcbiAgT3B0aW9uW1wiX2lucHV0XCJdXG4+IHtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgY29uc3QgeyBjdHggfSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG5cbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUub2JqZWN0KSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgICAgZXhwZWN0ZWQ6IFpvZFBhcnNlZFR5cGUub2JqZWN0LFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cblxuICAgIGNvbnN0IGRpc2NyaW1pbmF0b3IgPSB0aGlzLmRpc2NyaW1pbmF0b3I7XG4gICAgY29uc3QgZGlzY3JpbWluYXRvclZhbHVlOiBEaXNjcmltaW5hdG9yVmFsdWUgPSBjdHguZGF0YVtkaXNjcmltaW5hdG9yXTtcbiAgICBjb25zdCBvcHRpb24gPSB0aGlzLm9wdGlvbnMuZ2V0KGRpc2NyaW1pbmF0b3JWYWx1ZSk7XG5cbiAgICBpZiAoIW9wdGlvbikge1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3VuaW9uX2Rpc2NyaW1pbmF0b3IsXG4gICAgICAgIG9wdGlvbnM6IHRoaXMudmFsaWREaXNjcmltaW5hdG9yVmFsdWVzLFxuICAgICAgICBwYXRoOiBbZGlzY3JpbWluYXRvcl0sXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cblxuICAgIGlmIChjdHguY29tbW9uLmFzeW5jKSB7XG4gICAgICByZXR1cm4gb3B0aW9uLl9wYXJzZUFzeW5jKHtcbiAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBwYXJlbnQ6IGN0eCxcbiAgICAgIH0pO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXR1cm4gb3B0aW9uLl9wYXJzZVN5bmMoe1xuICAgICAgICBkYXRhOiBjdHguZGF0YSxcbiAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgIHBhcmVudDogY3R4LFxuICAgICAgfSk7XG4gICAgfVxuICB9XG5cbiAgZ2V0IGRpc2NyaW1pbmF0b3IoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5kaXNjcmltaW5hdG9yO1xuICB9XG5cbiAgZ2V0IHZhbGlkRGlzY3JpbWluYXRvclZhbHVlcygpIHtcbiAgICByZXR1cm4gQXJyYXkuZnJvbSh0aGlzLm9wdGlvbnMua2V5cygpKTtcbiAgfVxuXG4gIGdldCBvcHRpb25zKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYub3B0aW9ucztcbiAgfVxuXG4gIC8qKlxuICAgKiBUaGUgY29uc3RydWN0b3Igb2YgdGhlIGRpc2NyaW1pbmF0ZWQgdW5pb24gc2NoZW1hLiBJdHMgYmVoYXZpb3VyIGlzIHZlcnkgc2ltaWxhciB0byB0aGF0IG9mIHRoZSBub3JtYWwgei51bmlvbigpIGNvbnN0cnVjdG9yLlxuICAgKiBIb3dldmVyLCBpdCBvbmx5IGFsbG93cyBhIHVuaW9uIG9mIG9iamVjdHMsIGFsbCBvZiB3aGljaCBuZWVkIHRvIHNoYXJlIGEgZGlzY3JpbWluYXRvciBwcm9wZXJ0eS4gVGhpcyBwcm9wZXJ0eSBtdXN0XG4gICAqIGhhdmUgYSBkaWZmZXJlbnQgdmFsdWUgZm9yIGVhY2ggb2JqZWN0IGluIHRoZSB1bmlvbi5cbiAgICogQHBhcmFtIGRpc2NyaW1pbmF0b3IgdGhlIG5hbWUgb2YgdGhlIGRpc2NyaW1pbmF0b3IgcHJvcGVydHlcbiAgICogQHBhcmFtIHR5cGVzIGFuIGFycmF5IG9mIG9iamVjdCBzY2hlbWFzXG4gICAqIEBwYXJhbSBwYXJhbXNcbiAgICovXG4gIHN0YXRpYyBjcmVhdGU8XG4gICAgRGlzY3JpbWluYXRvciBleHRlbmRzIHN0cmluZyxcbiAgICBEaXNjcmltaW5hdG9yVmFsdWUgZXh0ZW5kcyBQcmltaXRpdmUsXG4gICAgVHlwZXMgZXh0ZW5kcyBbXG4gICAgICBab2REaXNjcmltaW5hdGVkVW5pb25PcHRpb248RGlzY3JpbWluYXRvciwgRGlzY3JpbWluYXRvclZhbHVlPixcbiAgICAgIFpvZERpc2NyaW1pbmF0ZWRVbmlvbk9wdGlvbjxEaXNjcmltaW5hdG9yLCBEaXNjcmltaW5hdG9yVmFsdWU+LFxuICAgICAgLi4uWm9kRGlzY3JpbWluYXRlZFVuaW9uT3B0aW9uPERpc2NyaW1pbmF0b3IsIERpc2NyaW1pbmF0b3JWYWx1ZT5bXVxuICAgIF1cbiAgPihcbiAgICBkaXNjcmltaW5hdG9yOiBEaXNjcmltaW5hdG9yLFxuICAgIHR5cGVzOiBUeXBlcyxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kRGlzY3JpbWluYXRlZFVuaW9uPERpc2NyaW1pbmF0b3IsIERpc2NyaW1pbmF0b3JWYWx1ZSwgVHlwZXNbbnVtYmVyXT4ge1xuICAgIC8vIEdldCBhbGwgdGhlIHZhbGlkIGRpc2NyaW1pbmF0b3IgdmFsdWVzXG4gICAgY29uc3Qgb3B0aW9uczogTWFwPERpc2NyaW1pbmF0b3JWYWx1ZSwgVHlwZXNbbnVtYmVyXT4gPSBuZXcgTWFwKCk7XG5cbiAgICB0cnkge1xuICAgICAgdHlwZXMuZm9yRWFjaCgodHlwZSkgPT4ge1xuICAgICAgICBjb25zdCBkaXNjcmltaW5hdG9yVmFsdWUgPSB0eXBlLnNoYXBlW2Rpc2NyaW1pbmF0b3JdLnZhbHVlO1xuICAgICAgICBvcHRpb25zLnNldChkaXNjcmltaW5hdG9yVmFsdWUsIHR5cGUpO1xuICAgICAgfSk7XG4gICAgfSBjYXRjaCAoZSkge1xuICAgICAgdGhyb3cgbmV3IEVycm9yKFxuICAgICAgICBcIlRoZSBkaXNjcmltaW5hdG9yIHZhbHVlIGNvdWxkIG5vdCBiZSBleHRyYWN0ZWQgZnJvbSBhbGwgdGhlIHByb3ZpZGVkIHNjaGVtYXNcIlxuICAgICAgKTtcbiAgICB9XG5cbiAgICAvLyBBc3NlcnQgdGhhdCBhbGwgdGhlIGRpc2NyaW1pbmF0b3IgdmFsdWVzIGFyZSB1bmlxdWVcbiAgICBpZiAob3B0aW9ucy5zaXplICE9PSB0eXBlcy5sZW5ndGgpIHtcbiAgICAgIHRocm93IG5ldyBFcnJvcihcIlNvbWUgb2YgdGhlIGRpc2NyaW1pbmF0b3IgdmFsdWVzIGFyZSBub3QgdW5pcXVlXCIpO1xuICAgIH1cblxuICAgIHJldHVybiBuZXcgWm9kRGlzY3JpbWluYXRlZFVuaW9uPFxuICAgICAgRGlzY3JpbWluYXRvcixcbiAgICAgIERpc2NyaW1pbmF0b3JWYWx1ZSxcbiAgICAgIFR5cGVzW251bWJlcl1cbiAgICA+KHtcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kRGlzY3JpbWluYXRlZFVuaW9uLFxuICAgICAgZGlzY3JpbWluYXRvcixcbiAgICAgIG9wdGlvbnMsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH1cbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZEludGVyc2VjdGlvbiAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kSW50ZXJzZWN0aW9uRGVmPFxuICBUIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnksXG4gIFUgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueVxuPiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICBsZWZ0OiBUO1xuICByaWdodDogVTtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RJbnRlcnNlY3Rpb247XG59XG5cbmZ1bmN0aW9uIG1lcmdlVmFsdWVzKFxuICBhOiBhbnksXG4gIGI6IGFueVxuKTogeyB2YWxpZDogdHJ1ZTsgZGF0YTogYW55IH0gfCB7IHZhbGlkOiBmYWxzZSB9IHtcbiAgY29uc3QgYVR5cGUgPSBnZXRQYXJzZWRUeXBlKGEpO1xuICBjb25zdCBiVHlwZSA9IGdldFBhcnNlZFR5cGUoYik7XG5cbiAgaWYgKGEgPT09IGIpIHtcbiAgICByZXR1cm4geyB2YWxpZDogdHJ1ZSwgZGF0YTogYSB9O1xuICB9IGVsc2UgaWYgKGFUeXBlID09PSBab2RQYXJzZWRUeXBlLm9iamVjdCAmJiBiVHlwZSA9PT0gWm9kUGFyc2VkVHlwZS5vYmplY3QpIHtcbiAgICBjb25zdCBiS2V5cyA9IHV0aWwub2JqZWN0S2V5cyhiKTtcbiAgICBjb25zdCBzaGFyZWRLZXlzID0gdXRpbFxuICAgICAgLm9iamVjdEtleXMoYSlcbiAgICAgIC5maWx0ZXIoKGtleSkgPT4gYktleXMuaW5kZXhPZihrZXkpICE9PSAtMSk7XG5cbiAgICBjb25zdCBuZXdPYmo6IGFueSA9IHsgLi4uYSwgLi4uYiB9O1xuICAgIGZvciAoY29uc3Qga2V5IG9mIHNoYXJlZEtleXMpIHtcbiAgICAgIGNvbnN0IHNoYXJlZFZhbHVlID0gbWVyZ2VWYWx1ZXMoYVtrZXldLCBiW2tleV0pO1xuICAgICAgaWYgKCFzaGFyZWRWYWx1ZS52YWxpZCkge1xuICAgICAgICByZXR1cm4geyB2YWxpZDogZmFsc2UgfTtcbiAgICAgIH1cbiAgICAgIG5ld09ialtrZXldID0gc2hhcmVkVmFsdWUuZGF0YTtcbiAgICB9XG5cbiAgICByZXR1cm4geyB2YWxpZDogdHJ1ZSwgZGF0YTogbmV3T2JqIH07XG4gIH0gZWxzZSBpZiAoYVR5cGUgPT09IFpvZFBhcnNlZFR5cGUuYXJyYXkgJiYgYlR5cGUgPT09IFpvZFBhcnNlZFR5cGUuYXJyYXkpIHtcbiAgICBpZiAoYS5sZW5ndGggIT09IGIubGVuZ3RoKSB7XG4gICAgICByZXR1cm4geyB2YWxpZDogZmFsc2UgfTtcbiAgICB9XG5cbiAgICBjb25zdCBuZXdBcnJheSA9IFtdO1xuICAgIGZvciAobGV0IGluZGV4ID0gMDsgaW5kZXggPCBhLmxlbmd0aDsgaW5kZXgrKykge1xuICAgICAgY29uc3QgaXRlbUEgPSBhW2luZGV4XTtcbiAgICAgIGNvbnN0IGl0ZW1CID0gYltpbmRleF07XG4gICAgICBjb25zdCBzaGFyZWRWYWx1ZSA9IG1lcmdlVmFsdWVzKGl0ZW1BLCBpdGVtQik7XG5cbiAgICAgIGlmICghc2hhcmVkVmFsdWUudmFsaWQpIHtcbiAgICAgICAgcmV0dXJuIHsgdmFsaWQ6IGZhbHNlIH07XG4gICAgICB9XG5cbiAgICAgIG5ld0FycmF5LnB1c2goc2hhcmVkVmFsdWUuZGF0YSk7XG4gICAgfVxuXG4gICAgcmV0dXJuIHsgdmFsaWQ6IHRydWUsIGRhdGE6IG5ld0FycmF5IH07XG4gIH0gZWxzZSBpZiAoXG4gICAgYVR5cGUgPT09IFpvZFBhcnNlZFR5cGUuZGF0ZSAmJlxuICAgIGJUeXBlID09PSBab2RQYXJzZWRUeXBlLmRhdGUgJiZcbiAgICArYSA9PT0gK2JcbiAgKSB7XG4gICAgcmV0dXJuIHsgdmFsaWQ6IHRydWUsIGRhdGE6IGEgfTtcbiAgfSBlbHNlIHtcbiAgICByZXR1cm4geyB2YWxpZDogZmFsc2UgfTtcbiAgfVxufVxuXG5leHBvcnQgY2xhc3MgWm9kSW50ZXJzZWN0aW9uPFxuICBUIGV4dGVuZHMgWm9kVHlwZUFueSxcbiAgVSBleHRlbmRzIFpvZFR5cGVBbnlcbj4gZXh0ZW5kcyBab2RUeXBlPFxuICBUW1wiX291dHB1dFwiXSAmIFVbXCJfb3V0cHV0XCJdLFxuICBab2RJbnRlcnNlY3Rpb25EZWY8VCwgVT4sXG4gIFRbXCJfaW5wdXRcIl0gJiBVW1wiX2lucHV0XCJdXG4+IHtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgY29uc3QgeyBzdGF0dXMsIGN0eCB9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBjb25zdCBoYW5kbGVQYXJzZWQgPSAoXG4gICAgICBwYXJzZWRMZWZ0OiBTeW5jUGFyc2VSZXR1cm5UeXBlLFxuICAgICAgcGFyc2VkUmlnaHQ6IFN5bmNQYXJzZVJldHVyblR5cGVcbiAgICApOiBTeW5jUGFyc2VSZXR1cm5UeXBlPFQgJiBVPiA9PiB7XG4gICAgICBpZiAoaXNBYm9ydGVkKHBhcnNlZExlZnQpIHx8IGlzQWJvcnRlZChwYXJzZWRSaWdodCkpIHtcbiAgICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgICB9XG5cbiAgICAgIGNvbnN0IG1lcmdlZCA9IG1lcmdlVmFsdWVzKHBhcnNlZExlZnQudmFsdWUsIHBhcnNlZFJpZ2h0LnZhbHVlKTtcblxuICAgICAgaWYgKCFtZXJnZWQudmFsaWQpIHtcbiAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfaW50ZXJzZWN0aW9uX3R5cGVzLFxuICAgICAgICB9KTtcbiAgICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgICB9XG5cbiAgICAgIGlmIChpc0RpcnR5KHBhcnNlZExlZnQpIHx8IGlzRGlydHkocGFyc2VkUmlnaHQpKSB7XG4gICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgfVxuXG4gICAgICByZXR1cm4geyBzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IG1lcmdlZC5kYXRhIGFzIGFueSB9O1xuICAgIH07XG5cbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgcmV0dXJuIFByb21pc2UuYWxsKFtcbiAgICAgICAgdGhpcy5fZGVmLmxlZnQuX3BhcnNlQXN5bmMoe1xuICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgIHBhcmVudDogY3R4LFxuICAgICAgICB9KSxcbiAgICAgICAgdGhpcy5fZGVmLnJpZ2h0Ll9wYXJzZUFzeW5jKHtcbiAgICAgICAgICBkYXRhOiBjdHguZGF0YSxcbiAgICAgICAgICBwYXRoOiBjdHgucGF0aCxcbiAgICAgICAgICBwYXJlbnQ6IGN0eCxcbiAgICAgICAgfSksXG4gICAgICBdKS50aGVuKChbbGVmdCwgcmlnaHRdOiBhbnkpID0+IGhhbmRsZVBhcnNlZChsZWZ0LCByaWdodCkpO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXR1cm4gaGFuZGxlUGFyc2VkKFxuICAgICAgICB0aGlzLl9kZWYubGVmdC5fcGFyc2VTeW5jKHtcbiAgICAgICAgICBkYXRhOiBjdHguZGF0YSxcbiAgICAgICAgICBwYXRoOiBjdHgucGF0aCxcbiAgICAgICAgICBwYXJlbnQ6IGN0eCxcbiAgICAgICAgfSksXG4gICAgICAgIHRoaXMuX2RlZi5yaWdodC5fcGFyc2VTeW5jKHtcbiAgICAgICAgICBkYXRhOiBjdHguZGF0YSxcbiAgICAgICAgICBwYXRoOiBjdHgucGF0aCxcbiAgICAgICAgICBwYXJlbnQ6IGN0eCxcbiAgICAgICAgfSlcbiAgICAgICk7XG4gICAgfVxuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IDxUIGV4dGVuZHMgWm9kVHlwZUFueSwgVSBleHRlbmRzIFpvZFR5cGVBbnk+KFxuICAgIGxlZnQ6IFQsXG4gICAgcmlnaHQ6IFUsXG4gICAgcGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zXG4gICk6IFpvZEludGVyc2VjdGlvbjxULCBVPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RJbnRlcnNlY3Rpb24oe1xuICAgICAgbGVmdDogbGVmdCxcbiAgICAgIHJpZ2h0OiByaWdodCxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kSW50ZXJzZWN0aW9uLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kVHVwbGUgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgdHlwZSBab2RUdXBsZUl0ZW1zID0gW1pvZFR5cGVBbnksIC4uLlpvZFR5cGVBbnlbXV07XG5leHBvcnQgdHlwZSBBc3NlcnRBcnJheTxUPiA9IFQgZXh0ZW5kcyBhbnlbXSA/IFQgOiBuZXZlcjtcbmV4cG9ydCB0eXBlIE91dHB1dFR5cGVPZlR1cGxlPFQgZXh0ZW5kcyBab2RUdXBsZUl0ZW1zIHwgW10+ID0gQXNzZXJ0QXJyYXk8e1xuICBbayBpbiBrZXlvZiBUXTogVFtrXSBleHRlbmRzIFpvZFR5cGU8YW55LCBhbnk+ID8gVFtrXVtcIl9vdXRwdXRcIl0gOiBuZXZlcjtcbn0+O1xuZXhwb3J0IHR5cGUgT3V0cHV0VHlwZU9mVHVwbGVXaXRoUmVzdDxcbiAgVCBleHRlbmRzIFpvZFR1cGxlSXRlbXMgfCBbXSxcbiAgUmVzdCBleHRlbmRzIFpvZFR5cGVBbnkgfCBudWxsID0gbnVsbFxuPiA9IFJlc3QgZXh0ZW5kcyBab2RUeXBlQW55XG4gID8gWy4uLk91dHB1dFR5cGVPZlR1cGxlPFQ+LCAuLi5SZXN0W1wiX291dHB1dFwiXVtdXVxuICA6IE91dHB1dFR5cGVPZlR1cGxlPFQ+O1xuXG5leHBvcnQgdHlwZSBJbnB1dFR5cGVPZlR1cGxlPFQgZXh0ZW5kcyBab2RUdXBsZUl0ZW1zIHwgW10+ID0gQXNzZXJ0QXJyYXk8e1xuICBbayBpbiBrZXlvZiBUXTogVFtrXSBleHRlbmRzIFpvZFR5cGU8YW55LCBhbnk+ID8gVFtrXVtcIl9pbnB1dFwiXSA6IG5ldmVyO1xufT47XG5leHBvcnQgdHlwZSBJbnB1dFR5cGVPZlR1cGxlV2l0aFJlc3Q8XG4gIFQgZXh0ZW5kcyBab2RUdXBsZUl0ZW1zIHwgW10sXG4gIFJlc3QgZXh0ZW5kcyBab2RUeXBlQW55IHwgbnVsbCA9IG51bGxcbj4gPSBSZXN0IGV4dGVuZHMgWm9kVHlwZUFueVxuICA/IFsuLi5JbnB1dFR5cGVPZlR1cGxlPFQ+LCAuLi5SZXN0W1wiX2lucHV0XCJdW11dXG4gIDogSW5wdXRUeXBlT2ZUdXBsZTxUPjtcblxuZXhwb3J0IGludGVyZmFjZSBab2RUdXBsZURlZjxcbiAgVCBleHRlbmRzIFpvZFR1cGxlSXRlbXMgfCBbXSA9IFpvZFR1cGxlSXRlbXMsXG4gIFJlc3QgZXh0ZW5kcyBab2RUeXBlQW55IHwgbnVsbCA9IG51bGxcbj4gZXh0ZW5kcyBab2RUeXBlRGVmIHtcbiAgaXRlbXM6IFQ7XG4gIHJlc3Q6IFJlc3Q7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kVHVwbGU7XG59XG5cbmV4cG9ydCBjbGFzcyBab2RUdXBsZTxcbiAgVCBleHRlbmRzIFtab2RUeXBlQW55LCAuLi5ab2RUeXBlQW55W11dIHwgW10gPSBbWm9kVHlwZUFueSwgLi4uWm9kVHlwZUFueVtdXSxcbiAgUmVzdCBleHRlbmRzIFpvZFR5cGVBbnkgfCBudWxsID0gbnVsbFxuPiBleHRlbmRzIFpvZFR5cGU8XG4gIE91dHB1dFR5cGVPZlR1cGxlV2l0aFJlc3Q8VCwgUmVzdD4sXG4gIFpvZFR1cGxlRGVmPFQsIFJlc3Q+LFxuICBJbnB1dFR5cGVPZlR1cGxlV2l0aFJlc3Q8VCwgUmVzdD5cbj4ge1xuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBjb25zdCB7IHN0YXR1cywgY3R4IH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGlmIChjdHgucGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5hcnJheSkge1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLmFycmF5LFxuICAgICAgICByZWNlaXZlZDogY3R4LnBhcnNlZFR5cGUsXG4gICAgICB9KTtcbiAgICAgIHJldHVybiBJTlZBTElEO1xuICAgIH1cblxuICAgIGlmIChjdHguZGF0YS5sZW5ndGggPCB0aGlzLl9kZWYuaXRlbXMubGVuZ3RoKSB7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19zbWFsbCxcbiAgICAgICAgbWluaW11bTogdGhpcy5fZGVmLml0ZW1zLmxlbmd0aCxcbiAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICB0eXBlOiBcImFycmF5XCIsXG4gICAgICB9KTtcblxuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuXG4gICAgY29uc3QgcmVzdCA9IHRoaXMuX2RlZi5yZXN0O1xuXG4gICAgaWYgKCFyZXN0ICYmIGN0eC5kYXRhLmxlbmd0aCA+IHRoaXMuX2RlZi5pdGVtcy5sZW5ndGgpIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUudG9vX2JpZyxcbiAgICAgICAgbWF4aW11bTogdGhpcy5fZGVmLml0ZW1zLmxlbmd0aCxcbiAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICB0eXBlOiBcImFycmF5XCIsXG4gICAgICB9KTtcbiAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgIH1cblxuICAgIGNvbnN0IGl0ZW1zID0gKGN0eC5kYXRhIGFzIGFueVtdKVxuICAgICAgLm1hcCgoaXRlbSwgaXRlbUluZGV4KSA9PiB7XG4gICAgICAgIGNvbnN0IHNjaGVtYSA9IHRoaXMuX2RlZi5pdGVtc1tpdGVtSW5kZXhdIHx8IHRoaXMuX2RlZi5yZXN0O1xuICAgICAgICBpZiAoIXNjaGVtYSkgcmV0dXJuIG51bGwgYXMgYW55IGFzIFN5bmNQYXJzZVJldHVyblR5cGU8YW55PjtcbiAgICAgICAgcmV0dXJuIHNjaGVtYS5fcGFyc2UoXG4gICAgICAgICAgbmV3IFBhcnNlSW5wdXRMYXp5UGF0aChjdHgsIGl0ZW0sIGN0eC5wYXRoLCBpdGVtSW5kZXgpXG4gICAgICAgICk7XG4gICAgICB9KVxuICAgICAgLmZpbHRlcigoeCkgPT4gISF4KTsgLy8gZmlsdGVyIG51bGxzXG5cbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgcmV0dXJuIFByb21pc2UuYWxsKGl0ZW1zKS50aGVuKChyZXN1bHRzKSA9PiB7XG4gICAgICAgIHJldHVybiBQYXJzZVN0YXR1cy5tZXJnZUFycmF5KHN0YXR1cywgcmVzdWx0cyk7XG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgcmV0dXJuIFBhcnNlU3RhdHVzLm1lcmdlQXJyYXkoc3RhdHVzLCBpdGVtcyBhcyBTeW5jUGFyc2VSZXR1cm5UeXBlW10pO1xuICAgIH1cbiAgfVxuXG4gIGdldCBpdGVtcygpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLml0ZW1zO1xuICB9XG5cbiAgcmVzdDxSZXN0IGV4dGVuZHMgWm9kVHlwZUFueT4ocmVzdDogUmVzdCk6IFpvZFR1cGxlPFQsIFJlc3Q+IHtcbiAgICByZXR1cm4gbmV3IFpvZFR1cGxlKHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIHJlc3QsXG4gICAgfSk7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gPFQgZXh0ZW5kcyBbWm9kVHlwZUFueSwgLi4uWm9kVHlwZUFueVtdXSB8IFtdPihcbiAgICBzY2hlbWFzOiBULFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RUdXBsZTxULCBudWxsPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RUdXBsZSh7XG4gICAgICBpdGVtczogc2NoZW1hcyxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kVHVwbGUsXG4gICAgICByZXN0OiBudWxsLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kUmVjb3JkICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuZXhwb3J0IGludGVyZmFjZSBab2RSZWNvcmREZWY8XG4gIEtleSBleHRlbmRzIEtleVNjaGVtYSA9IFpvZFN0cmluZyxcbiAgVmFsdWUgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueVxuPiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB2YWx1ZVR5cGU6IFZhbHVlO1xuICBrZXlUeXBlOiBLZXk7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kUmVjb3JkO1xufVxuXG50eXBlIEtleVNjaGVtYSA9IFpvZFR5cGU8c3RyaW5nIHwgbnVtYmVyIHwgc3ltYm9sLCBhbnksIGFueT47XG50eXBlIFJlY29yZFR5cGU8SyBleHRlbmRzIHN0cmluZyB8IG51bWJlciB8IHN5bWJvbCwgVj4gPSBbc3RyaW5nXSBleHRlbmRzIFtLXVxuICA/IFJlY29yZDxLLCBWPlxuICA6IFtudW1iZXJdIGV4dGVuZHMgW0tdXG4gID8gUmVjb3JkPEssIFY+XG4gIDogW3N5bWJvbF0gZXh0ZW5kcyBbS11cbiAgPyBSZWNvcmQ8SywgVj5cbiAgOiBQYXJ0aWFsPFJlY29yZDxLLCBWPj47XG5leHBvcnQgY2xhc3MgWm9kUmVjb3JkPFxuICBLZXkgZXh0ZW5kcyBLZXlTY2hlbWEgPSBab2RTdHJpbmcsXG4gIFZhbHVlIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnlcbj4gZXh0ZW5kcyBab2RUeXBlPFxuICBSZWNvcmRUeXBlPEtleVtcIl9vdXRwdXRcIl0sIFZhbHVlW1wiX291dHB1dFwiXT4sXG4gIFpvZFJlY29yZERlZjxLZXksIFZhbHVlPixcbiAgUmVjb3JkVHlwZTxLZXlbXCJfaW5wdXRcIl0sIFZhbHVlW1wiX2lucHV0XCJdPlxuPiB7XG4gIGdldCBrZXlTY2hlbWEoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5rZXlUeXBlO1xuICB9XG4gIGdldCB2YWx1ZVNjaGVtYSgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnZhbHVlVHlwZTtcbiAgfVxuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBjb25zdCB7IHN0YXR1cywgY3R4IH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGlmIChjdHgucGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5vYmplY3QpIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5vYmplY3QsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuXG4gICAgY29uc3QgcGFpcnM6IHtcbiAgICAgIGtleTogUGFyc2VSZXR1cm5UeXBlPGFueT47XG4gICAgICB2YWx1ZTogUGFyc2VSZXR1cm5UeXBlPGFueT47XG4gICAgfVtdID0gW107XG5cbiAgICBjb25zdCBrZXlUeXBlID0gdGhpcy5fZGVmLmtleVR5cGU7XG4gICAgY29uc3QgdmFsdWVUeXBlID0gdGhpcy5fZGVmLnZhbHVlVHlwZTtcblxuICAgIGZvciAoY29uc3Qga2V5IGluIGN0eC5kYXRhKSB7XG4gICAgICBwYWlycy5wdXNoKHtcbiAgICAgICAga2V5OiBrZXlUeXBlLl9wYXJzZShuZXcgUGFyc2VJbnB1dExhenlQYXRoKGN0eCwga2V5LCBjdHgucGF0aCwga2V5KSksXG4gICAgICAgIHZhbHVlOiB2YWx1ZVR5cGUuX3BhcnNlKFxuICAgICAgICAgIG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCBjdHguZGF0YVtrZXldLCBjdHgucGF0aCwga2V5KVxuICAgICAgICApLFxuICAgICAgfSk7XG4gICAgfVxuXG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBQYXJzZVN0YXR1cy5tZXJnZU9iamVjdEFzeW5jKHN0YXR1cywgcGFpcnMpO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXR1cm4gUGFyc2VTdGF0dXMubWVyZ2VPYmplY3RTeW5jKHN0YXR1cywgcGFpcnMgYXMgYW55KTtcbiAgICB9XG4gIH1cblxuICBnZXQgZWxlbWVudCgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLnZhbHVlVHlwZTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGU8VmFsdWUgZXh0ZW5kcyBab2RUeXBlQW55PihcbiAgICB2YWx1ZVR5cGU6IFZhbHVlLFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RSZWNvcmQ8Wm9kU3RyaW5nLCBWYWx1ZT47XG4gIHN0YXRpYyBjcmVhdGU8S2V5cyBleHRlbmRzIEtleVNjaGVtYSwgVmFsdWUgZXh0ZW5kcyBab2RUeXBlQW55PihcbiAgICBrZXlTY2hlbWE6IEtleXMsXG4gICAgdmFsdWVUeXBlOiBWYWx1ZSxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kUmVjb3JkPEtleXMsIFZhbHVlPjtcbiAgc3RhdGljIGNyZWF0ZShmaXJzdDogYW55LCBzZWNvbmQ/OiBhbnksIHRoaXJkPzogYW55KTogWm9kUmVjb3JkPGFueSwgYW55PiB7XG4gICAgaWYgKHNlY29uZCBpbnN0YW5jZW9mIFpvZFR5cGUpIHtcbiAgICAgIHJldHVybiBuZXcgWm9kUmVjb3JkKHtcbiAgICAgICAga2V5VHlwZTogZmlyc3QsXG4gICAgICAgIHZhbHVlVHlwZTogc2Vjb25kLFxuICAgICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFJlY29yZCxcbiAgICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyh0aGlyZCksXG4gICAgICB9KTtcbiAgICB9XG5cbiAgICByZXR1cm4gbmV3IFpvZFJlY29yZCh7XG4gICAgICBrZXlUeXBlOiBab2RTdHJpbmcuY3JlYXRlKCksXG4gICAgICB2YWx1ZVR5cGU6IGZpcnN0LFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RSZWNvcmQsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHNlY29uZCksXG4gICAgfSk7XG4gIH1cbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE1hcCAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kTWFwRGVmPFxuICBLZXkgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueSxcbiAgVmFsdWUgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueVxuPiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB2YWx1ZVR5cGU6IFZhbHVlO1xuICBrZXlUeXBlOiBLZXk7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTWFwO1xufVxuXG5leHBvcnQgY2xhc3MgWm9kTWFwPFxuICBLZXkgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueSxcbiAgVmFsdWUgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueVxuPiBleHRlbmRzIFpvZFR5cGU8XG4gIE1hcDxLZXlbXCJfb3V0cHV0XCJdLCBWYWx1ZVtcIl9vdXRwdXRcIl0+LFxuICBab2RNYXBEZWY8S2V5LCBWYWx1ZT4sXG4gIE1hcDxLZXlbXCJfaW5wdXRcIl0sIFZhbHVlW1wiX2lucHV0XCJdPlxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHsgc3RhdHVzLCBjdHggfSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLm1hcCkge1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLm1hcCxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBjb25zdCBrZXlUeXBlID0gdGhpcy5fZGVmLmtleVR5cGU7XG4gICAgY29uc3QgdmFsdWVUeXBlID0gdGhpcy5fZGVmLnZhbHVlVHlwZTtcblxuICAgIGNvbnN0IHBhaXJzID0gWy4uLihjdHguZGF0YSBhcyBNYXA8dW5rbm93biwgdW5rbm93bj4pLmVudHJpZXMoKV0ubWFwKFxuICAgICAgKFtrZXksIHZhbHVlXSwgaW5kZXgpID0+IHtcbiAgICAgICAgcmV0dXJuIHtcbiAgICAgICAgICBrZXk6IGtleVR5cGUuX3BhcnNlKFxuICAgICAgICAgICAgbmV3IFBhcnNlSW5wdXRMYXp5UGF0aChjdHgsIGtleSwgY3R4LnBhdGgsIFtpbmRleCwgXCJrZXlcIl0pXG4gICAgICAgICAgKSxcbiAgICAgICAgICB2YWx1ZTogdmFsdWVUeXBlLl9wYXJzZShcbiAgICAgICAgICAgIG5ldyBQYXJzZUlucHV0TGF6eVBhdGgoY3R4LCB2YWx1ZSwgY3R4LnBhdGgsIFtpbmRleCwgXCJ2YWx1ZVwiXSlcbiAgICAgICAgICApLFxuICAgICAgICB9O1xuICAgICAgfVxuICAgICk7XG5cbiAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgY29uc3QgZmluYWxNYXAgPSBuZXcgTWFwKCk7XG4gICAgICByZXR1cm4gUHJvbWlzZS5yZXNvbHZlKCkudGhlbihhc3luYyAoKSA9PiB7XG4gICAgICAgIGZvciAoY29uc3QgcGFpciBvZiBwYWlycykge1xuICAgICAgICAgIGNvbnN0IGtleSA9IGF3YWl0IHBhaXIua2V5O1xuICAgICAgICAgIGNvbnN0IHZhbHVlID0gYXdhaXQgcGFpci52YWx1ZTtcbiAgICAgICAgICBpZiAoa2V5LnN0YXR1cyA9PT0gXCJhYm9ydGVkXCIgfHwgdmFsdWUuc3RhdHVzID09PSBcImFib3J0ZWRcIikge1xuICAgICAgICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgICAgICAgfVxuICAgICAgICAgIGlmIChrZXkuc3RhdHVzID09PSBcImRpcnR5XCIgfHwgdmFsdWUuc3RhdHVzID09PSBcImRpcnR5XCIpIHtcbiAgICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICAgIH1cblxuICAgICAgICAgIGZpbmFsTWFwLnNldChrZXkudmFsdWUsIHZhbHVlLnZhbHVlKTtcbiAgICAgICAgfVxuICAgICAgICByZXR1cm4geyBzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IGZpbmFsTWFwIH07XG4gICAgICB9KTtcbiAgICB9IGVsc2Uge1xuICAgICAgY29uc3QgZmluYWxNYXAgPSBuZXcgTWFwKCk7XG4gICAgICBmb3IgKGNvbnN0IHBhaXIgb2YgcGFpcnMpIHtcbiAgICAgICAgY29uc3Qga2V5ID0gcGFpci5rZXkgYXMgU3luY1BhcnNlUmV0dXJuVHlwZTtcbiAgICAgICAgY29uc3QgdmFsdWUgPSBwYWlyLnZhbHVlIGFzIFN5bmNQYXJzZVJldHVyblR5cGU7XG4gICAgICAgIGlmIChrZXkuc3RhdHVzID09PSBcImFib3J0ZWRcIiB8fCB2YWx1ZS5zdGF0dXMgPT09IFwiYWJvcnRlZFwiKSB7XG4gICAgICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgICAgIH1cbiAgICAgICAgaWYgKGtleS5zdGF0dXMgPT09IFwiZGlydHlcIiB8fCB2YWx1ZS5zdGF0dXMgPT09IFwiZGlydHlcIikge1xuICAgICAgICAgIHN0YXR1cy5kaXJ0eSgpO1xuICAgICAgICB9XG5cbiAgICAgICAgZmluYWxNYXAuc2V0KGtleS52YWx1ZSwgdmFsdWUudmFsdWUpO1xuICAgICAgfVxuICAgICAgcmV0dXJuIHsgc3RhdHVzOiBzdGF0dXMudmFsdWUsIHZhbHVlOiBmaW5hbE1hcCB9O1xuICAgIH1cbiAgfVxuICBzdGF0aWMgY3JlYXRlID0gPFxuICAgIEtleSBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RUeXBlQW55LFxuICAgIFZhbHVlIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnlcbiAgPihcbiAgICBrZXlUeXBlOiBLZXksXG4gICAgdmFsdWVUeXBlOiBWYWx1ZSxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kTWFwPEtleSwgVmFsdWU+ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZE1hcCh7XG4gICAgICB2YWx1ZVR5cGUsXG4gICAgICBrZXlUeXBlLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RNYXAsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG59XG5cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICBab2RTZXQgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgaW50ZXJmYWNlIFpvZFNldERlZjxWYWx1ZSBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RUeXBlQW55PlxuICBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB2YWx1ZVR5cGU6IFZhbHVlO1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZFNldDtcbiAgbWluU2l6ZTogeyB2YWx1ZTogbnVtYmVyOyBtZXNzYWdlPzogc3RyaW5nIH0gfCBudWxsO1xuICBtYXhTaXplOiB7IHZhbHVlOiBudW1iZXI7IG1lc3NhZ2U/OiBzdHJpbmcgfSB8IG51bGw7XG59XG5cbmV4cG9ydCBjbGFzcyBab2RTZXQ8VmFsdWUgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueT4gZXh0ZW5kcyBab2RUeXBlPFxuICBTZXQ8VmFsdWVbXCJfb3V0cHV0XCJdPixcbiAgWm9kU2V0RGVmPFZhbHVlPixcbiAgU2V0PFZhbHVlW1wiX2lucHV0XCJdPlxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHsgc3RhdHVzLCBjdHggfSA9IHRoaXMuX3Byb2Nlc3NJbnB1dFBhcmFtcyhpbnB1dCk7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLnNldCkge1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLnNldCxcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBjb25zdCBkZWYgPSB0aGlzLl9kZWY7XG5cbiAgICBpZiAoZGVmLm1pblNpemUgIT09IG51bGwpIHtcbiAgICAgIGlmIChjdHguZGF0YS5zaXplIDwgZGVmLm1pblNpemUudmFsdWUpIHtcbiAgICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLnRvb19zbWFsbCxcbiAgICAgICAgICBtaW5pbXVtOiBkZWYubWluU2l6ZS52YWx1ZSxcbiAgICAgICAgICB0eXBlOiBcInNldFwiLFxuICAgICAgICAgIGluY2x1c2l2ZTogdHJ1ZSxcbiAgICAgICAgICBtZXNzYWdlOiBkZWYubWluU2l6ZS5tZXNzYWdlLFxuICAgICAgICB9KTtcbiAgICAgICAgc3RhdHVzLmRpcnR5KCk7XG4gICAgICB9XG4gICAgfVxuXG4gICAgaWYgKGRlZi5tYXhTaXplICE9PSBudWxsKSB7XG4gICAgICBpZiAoY3R4LmRhdGEuc2l6ZSA+IGRlZi5tYXhTaXplLnZhbHVlKSB7XG4gICAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS50b29fYmlnLFxuICAgICAgICAgIG1heGltdW06IGRlZi5tYXhTaXplLnZhbHVlLFxuICAgICAgICAgIHR5cGU6IFwic2V0XCIsXG4gICAgICAgICAgaW5jbHVzaXZlOiB0cnVlLFxuICAgICAgICAgIG1lc3NhZ2U6IGRlZi5tYXhTaXplLm1lc3NhZ2UsXG4gICAgICAgIH0pO1xuICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICBjb25zdCB2YWx1ZVR5cGUgPSB0aGlzLl9kZWYudmFsdWVUeXBlO1xuXG4gICAgZnVuY3Rpb24gZmluYWxpemVTZXQoZWxlbWVudHM6IFN5bmNQYXJzZVJldHVyblR5cGU8YW55PltdKSB7XG4gICAgICBjb25zdCBwYXJzZWRTZXQgPSBuZXcgU2V0KCk7XG4gICAgICBmb3IgKGNvbnN0IGVsZW1lbnQgb2YgZWxlbWVudHMpIHtcbiAgICAgICAgaWYgKGVsZW1lbnQuc3RhdHVzID09PSBcImFib3J0ZWRcIikgcmV0dXJuIElOVkFMSUQ7XG4gICAgICAgIGlmIChlbGVtZW50LnN0YXR1cyA9PT0gXCJkaXJ0eVwiKSBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgcGFyc2VkU2V0LmFkZChlbGVtZW50LnZhbHVlKTtcbiAgICAgIH1cbiAgICAgIHJldHVybiB7IHN0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogcGFyc2VkU2V0IH07XG4gICAgfVxuXG4gICAgY29uc3QgZWxlbWVudHMgPSBbLi4uKGN0eC5kYXRhIGFzIFNldDx1bmtub3duPikudmFsdWVzKCldLm1hcCgoaXRlbSwgaSkgPT5cbiAgICAgIHZhbHVlVHlwZS5fcGFyc2UobmV3IFBhcnNlSW5wdXRMYXp5UGF0aChjdHgsIGl0ZW0sIGN0eC5wYXRoLCBpKSlcbiAgICApO1xuXG4gICAgaWYgKGN0eC5jb21tb24uYXN5bmMpIHtcbiAgICAgIHJldHVybiBQcm9taXNlLmFsbChlbGVtZW50cykudGhlbigoZWxlbWVudHMpID0+IGZpbmFsaXplU2V0KGVsZW1lbnRzKSk7XG4gICAgfSBlbHNlIHtcbiAgICAgIHJldHVybiBmaW5hbGl6ZVNldChlbGVtZW50cyBhcyBTeW5jUGFyc2VSZXR1cm5UeXBlW10pO1xuICAgIH1cbiAgfVxuXG4gIG1pbihtaW5TaXplOiBudW1iZXIsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSk6IHRoaXMge1xuICAgIHJldHVybiBuZXcgWm9kU2V0KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIG1pblNpemU6IHsgdmFsdWU6IG1pblNpemUsIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSB9LFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIG1heChtYXhTaXplOiBudW1iZXIsIG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSk6IHRoaXMge1xuICAgIHJldHVybiBuZXcgWm9kU2V0KHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIG1heFNpemU6IHsgdmFsdWU6IG1heFNpemUsIG1lc3NhZ2U6IGVycm9yVXRpbC50b1N0cmluZyhtZXNzYWdlKSB9LFxuICAgIH0pIGFzIGFueTtcbiAgfVxuXG4gIHNpemUoc2l6ZTogbnVtYmVyLCBtZXNzYWdlPzogZXJyb3JVdGlsLkVyck1lc3NhZ2UpOiB0aGlzIHtcbiAgICByZXR1cm4gdGhpcy5taW4oc2l6ZSwgbWVzc2FnZSkubWF4KHNpemUsIG1lc3NhZ2UpIGFzIGFueTtcbiAgfVxuXG4gIG5vbmVtcHR5KG1lc3NhZ2U/OiBlcnJvclV0aWwuRXJyTWVzc2FnZSk6IFpvZFNldDxWYWx1ZT4ge1xuICAgIHJldHVybiB0aGlzLm1pbigxLCBtZXNzYWdlKSBhcyBhbnk7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gPFZhbHVlIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnk+KFxuICAgIHZhbHVlVHlwZTogVmFsdWUsXG4gICAgcGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zXG4gICk6IFpvZFNldDxWYWx1ZT4gPT4ge1xuICAgIHJldHVybiBuZXcgWm9kU2V0KHtcbiAgICAgIHZhbHVlVHlwZSxcbiAgICAgIG1pblNpemU6IG51bGwsXG4gICAgICBtYXhTaXplOiBudWxsLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RTZXQsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG59XG5cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICBab2RGdW5jdGlvbiAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kRnVuY3Rpb25EZWY8XG4gIEFyZ3MgZXh0ZW5kcyBab2RUdXBsZTxhbnksIGFueT4gPSBab2RUdXBsZTxhbnksIGFueT4sXG4gIFJldHVybnMgZXh0ZW5kcyBab2RUeXBlQW55ID0gWm9kVHlwZUFueVxuPiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICBhcmdzOiBBcmdzO1xuICByZXR1cm5zOiBSZXR1cm5zO1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEZ1bmN0aW9uO1xufVxuXG5leHBvcnQgdHlwZSBPdXRlclR5cGVPZkZ1bmN0aW9uPFxuICBBcmdzIGV4dGVuZHMgWm9kVHVwbGU8YW55LCBhbnk+LFxuICBSZXR1cm5zIGV4dGVuZHMgWm9kVHlwZUFueVxuPiA9IEFyZ3NbXCJfaW5wdXRcIl0gZXh0ZW5kcyBBcnJheTxhbnk+XG4gID8gKC4uLmFyZ3M6IEFyZ3NbXCJfaW5wdXRcIl0pID0+IFJldHVybnNbXCJfb3V0cHV0XCJdXG4gIDogbmV2ZXI7XG5cbmV4cG9ydCB0eXBlIElubmVyVHlwZU9mRnVuY3Rpb248XG4gIEFyZ3MgZXh0ZW5kcyBab2RUdXBsZTxhbnksIGFueT4sXG4gIFJldHVybnMgZXh0ZW5kcyBab2RUeXBlQW55XG4+ID0gQXJnc1tcIl9vdXRwdXRcIl0gZXh0ZW5kcyBBcnJheTxhbnk+XG4gID8gKC4uLmFyZ3M6IEFyZ3NbXCJfb3V0cHV0XCJdKSA9PiBSZXR1cm5zW1wiX2lucHV0XCJdXG4gIDogbmV2ZXI7XG5cbmV4cG9ydCBjbGFzcyBab2RGdW5jdGlvbjxcbiAgQXJncyBleHRlbmRzIFpvZFR1cGxlPGFueSwgYW55PixcbiAgUmV0dXJucyBleHRlbmRzIFpvZFR5cGVBbnlcbj4gZXh0ZW5kcyBab2RUeXBlPFxuICBPdXRlclR5cGVPZkZ1bmN0aW9uPEFyZ3MsIFJldHVybnM+LFxuICBab2RGdW5jdGlvbkRlZjxBcmdzLCBSZXR1cm5zPixcbiAgSW5uZXJUeXBlT2ZGdW5jdGlvbjxBcmdzLCBSZXR1cm5zPlxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTxhbnk+IHtcbiAgICBjb25zdCB7IGN0eCB9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBpZiAoY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUuZnVuY3Rpb24pIHtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5mdW5jdGlvbixcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBmdW5jdGlvbiBtYWtlQXJnc0lzc3VlKGFyZ3M6IGFueSwgZXJyb3I6IFpvZEVycm9yKTogWm9kSXNzdWUge1xuICAgICAgcmV0dXJuIG1ha2VJc3N1ZSh7XG4gICAgICAgIGRhdGE6IGFyZ3MsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBlcnJvck1hcHM6IFtcbiAgICAgICAgICBjdHguY29tbW9uLmNvbnRleHR1YWxFcnJvck1hcCxcbiAgICAgICAgICBjdHguc2NoZW1hRXJyb3JNYXAsXG4gICAgICAgICAgb3ZlcnJpZGVFcnJvck1hcCxcbiAgICAgICAgICBkZWZhdWx0RXJyb3JNYXAsXG4gICAgICAgIF0uZmlsdGVyKCh4KSA9PiAhIXgpIGFzIFpvZEVycm9yTWFwW10sXG4gICAgICAgIGlzc3VlRGF0YToge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX2FyZ3VtZW50cyxcbiAgICAgICAgICBhcmd1bWVudHNFcnJvcjogZXJyb3IsXG4gICAgICAgIH0sXG4gICAgICB9KTtcbiAgICB9XG5cbiAgICBmdW5jdGlvbiBtYWtlUmV0dXJuc0lzc3VlKHJldHVybnM6IGFueSwgZXJyb3I6IFpvZEVycm9yKTogWm9kSXNzdWUge1xuICAgICAgcmV0dXJuIG1ha2VJc3N1ZSh7XG4gICAgICAgIGRhdGE6IHJldHVybnMsXG4gICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICBlcnJvck1hcHM6IFtcbiAgICAgICAgICBjdHguY29tbW9uLmNvbnRleHR1YWxFcnJvck1hcCxcbiAgICAgICAgICBjdHguc2NoZW1hRXJyb3JNYXAsXG4gICAgICAgICAgb3ZlcnJpZGVFcnJvck1hcCxcbiAgICAgICAgICBkZWZhdWx0RXJyb3JNYXAsXG4gICAgICAgIF0uZmlsdGVyKCh4KSA9PiAhIXgpIGFzIFpvZEVycm9yTWFwW10sXG4gICAgICAgIGlzc3VlRGF0YToge1xuICAgICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3JldHVybl90eXBlLFxuICAgICAgICAgIHJldHVyblR5cGVFcnJvcjogZXJyb3IsXG4gICAgICAgIH0sXG4gICAgICB9KTtcbiAgICB9XG5cbiAgICBjb25zdCBwYXJhbXMgPSB7IGVycm9yTWFwOiBjdHguY29tbW9uLmNvbnRleHR1YWxFcnJvck1hcCB9O1xuICAgIGNvbnN0IGZuID0gY3R4LmRhdGE7XG5cbiAgICBpZiAodGhpcy5fZGVmLnJldHVybnMgaW5zdGFuY2VvZiBab2RQcm9taXNlKSB7XG4gICAgICByZXR1cm4gT0soYXN5bmMgKC4uLmFyZ3M6IGFueVtdKSA9PiB7XG4gICAgICAgIGNvbnN0IGVycm9yID0gbmV3IFpvZEVycm9yKFtdKTtcbiAgICAgICAgY29uc3QgcGFyc2VkQXJncyA9IGF3YWl0IHRoaXMuX2RlZi5hcmdzXG4gICAgICAgICAgLnBhcnNlQXN5bmMoYXJncywgcGFyYW1zKVxuICAgICAgICAgIC5jYXRjaCgoZSkgPT4ge1xuICAgICAgICAgICAgZXJyb3IuYWRkSXNzdWUobWFrZUFyZ3NJc3N1ZShhcmdzLCBlKSk7XG4gICAgICAgICAgICB0aHJvdyBlcnJvcjtcbiAgICAgICAgICB9KTtcbiAgICAgICAgY29uc3QgcmVzdWx0ID0gYXdhaXQgZm4oLi4uKHBhcnNlZEFyZ3MgYXMgYW55KSk7XG4gICAgICAgIGNvbnN0IHBhcnNlZFJldHVybnMgPSBhd2FpdCAoXG4gICAgICAgICAgdGhpcy5fZGVmLnJldHVybnMgYXMgWm9kUHJvbWlzZTxab2RUeXBlQW55PlxuICAgICAgICApLl9kZWYudHlwZVxuICAgICAgICAgIC5wYXJzZUFzeW5jKHJlc3VsdCwgcGFyYW1zKVxuICAgICAgICAgIC5jYXRjaCgoZSkgPT4ge1xuICAgICAgICAgICAgZXJyb3IuYWRkSXNzdWUobWFrZVJldHVybnNJc3N1ZShyZXN1bHQsIGUpKTtcbiAgICAgICAgICAgIHRocm93IGVycm9yO1xuICAgICAgICAgIH0pO1xuICAgICAgICByZXR1cm4gcGFyc2VkUmV0dXJucztcbiAgICAgIH0pO1xuICAgIH0gZWxzZSB7XG4gICAgICByZXR1cm4gT0soKC4uLmFyZ3M6IGFueVtdKSA9PiB7XG4gICAgICAgIGNvbnN0IHBhcnNlZEFyZ3MgPSB0aGlzLl9kZWYuYXJncy5zYWZlUGFyc2UoYXJncywgcGFyYW1zKTtcbiAgICAgICAgaWYgKCFwYXJzZWRBcmdzLnN1Y2Nlc3MpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgWm9kRXJyb3IoW21ha2VBcmdzSXNzdWUoYXJncywgcGFyc2VkQXJncy5lcnJvcildKTtcbiAgICAgICAgfVxuICAgICAgICBjb25zdCByZXN1bHQgPSBmbiguLi4ocGFyc2VkQXJncy5kYXRhIGFzIGFueSkpO1xuICAgICAgICBjb25zdCBwYXJzZWRSZXR1cm5zID0gdGhpcy5fZGVmLnJldHVybnMuc2FmZVBhcnNlKHJlc3VsdCwgcGFyYW1zKTtcbiAgICAgICAgaWYgKCFwYXJzZWRSZXR1cm5zLnN1Y2Nlc3MpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgWm9kRXJyb3IoW21ha2VSZXR1cm5zSXNzdWUocmVzdWx0LCBwYXJzZWRSZXR1cm5zLmVycm9yKV0pO1xuICAgICAgICB9XG4gICAgICAgIHJldHVybiBwYXJzZWRSZXR1cm5zLmRhdGE7XG4gICAgICB9KSBhcyBhbnk7XG4gICAgfVxuICB9XG5cbiAgcGFyYW1ldGVycygpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLmFyZ3M7XG4gIH1cblxuICByZXR1cm5UeXBlKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYucmV0dXJucztcbiAgfVxuXG4gIGFyZ3M8SXRlbXMgZXh0ZW5kcyBQYXJhbWV0ZXJzPHR5cGVvZiBab2RUdXBsZVtcImNyZWF0ZVwiXT5bMF0+KFxuICAgIC4uLml0ZW1zOiBJdGVtc1xuICApOiBab2RGdW5jdGlvbjxab2RUdXBsZTxJdGVtcywgWm9kVW5rbm93bj4sIFJldHVybnM+IHtcbiAgICByZXR1cm4gbmV3IFpvZEZ1bmN0aW9uKHtcbiAgICAgIC4uLnRoaXMuX2RlZixcbiAgICAgIGFyZ3M6IFpvZFR1cGxlLmNyZWF0ZShpdGVtcykucmVzdChab2RVbmtub3duLmNyZWF0ZSgpKSBhcyBhbnksXG4gICAgfSk7XG4gIH1cblxuICByZXR1cm5zPE5ld1JldHVyblR5cGUgZXh0ZW5kcyBab2RUeXBlPGFueSwgYW55Pj4oXG4gICAgcmV0dXJuVHlwZTogTmV3UmV0dXJuVHlwZVxuICApOiBab2RGdW5jdGlvbjxBcmdzLCBOZXdSZXR1cm5UeXBlPiB7XG4gICAgcmV0dXJuIG5ldyBab2RGdW5jdGlvbih7XG4gICAgICAuLi50aGlzLl9kZWYsXG4gICAgICByZXR1cm5zOiByZXR1cm5UeXBlLFxuICAgIH0pO1xuICB9XG5cbiAgaW1wbGVtZW50PEYgZXh0ZW5kcyBJbm5lclR5cGVPZkZ1bmN0aW9uPEFyZ3MsIFJldHVybnM+PihmdW5jOiBGKTogRiB7XG4gICAgY29uc3QgdmFsaWRhdGVkRnVuYyA9IHRoaXMucGFyc2UoZnVuYyk7XG4gICAgcmV0dXJuIHZhbGlkYXRlZEZ1bmMgYXMgYW55O1xuICB9XG5cbiAgc3RyaWN0SW1wbGVtZW50KFxuICAgIGZ1bmM6IElubmVyVHlwZU9mRnVuY3Rpb248QXJncywgUmV0dXJucz5cbiAgKTogSW5uZXJUeXBlT2ZGdW5jdGlvbjxBcmdzLCBSZXR1cm5zPiB7XG4gICAgY29uc3QgdmFsaWRhdGVkRnVuYyA9IHRoaXMucGFyc2UoZnVuYyk7XG4gICAgcmV0dXJuIHZhbGlkYXRlZEZ1bmMgYXMgYW55O1xuICB9XG5cbiAgdmFsaWRhdGUgPSB0aGlzLmltcGxlbWVudDtcblxuICBzdGF0aWMgY3JlYXRlID0gPFxuICAgIFQgZXh0ZW5kcyBab2RUdXBsZTxhbnksIGFueT4gPSBab2RUdXBsZTxbXSwgWm9kVW5rbm93bj4sXG4gICAgVSBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RVbmtub3duXG4gID4oXG4gICAgYXJncz86IFQsXG4gICAgcmV0dXJucz86IFUsXG4gICAgcGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zXG4gICk6IFpvZEZ1bmN0aW9uPFQsIFU+ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZEZ1bmN0aW9uKHtcbiAgICAgIGFyZ3M6IChhcmdzXG4gICAgICAgID8gYXJncy5yZXN0KFpvZFVua25vd24uY3JlYXRlKCkpXG4gICAgICAgIDogWm9kVHVwbGUuY3JlYXRlKFtdKS5yZXN0KFpvZFVua25vd24uY3JlYXRlKCkpKSBhcyBhbnksXG4gICAgICByZXR1cm5zOiByZXR1cm5zIHx8IFpvZFVua25vd24uY3JlYXRlKCksXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEZ1bmN0aW9uLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pIGFzIGFueTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZExhenkgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgaW50ZXJmYWNlIFpvZExhenlEZWY8VCBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RUeXBlQW55PlxuICBleHRlbmRzIFpvZFR5cGVEZWYge1xuICBnZXR0ZXI6ICgpID0+IFQ7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kTGF6eTtcbn1cblxuZXhwb3J0IGNsYXNzIFpvZExhenk8VCBleHRlbmRzIFpvZFR5cGVBbnk+IGV4dGVuZHMgWm9kVHlwZTxcbiAgb3V0cHV0PFQ+LFxuICBab2RMYXp5RGVmPFQ+LFxuICBpbnB1dDxUPlxuPiB7XG4gIGdldCBzY2hlbWEoKTogVCB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5nZXR0ZXIoKTtcbiAgfVxuXG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHsgY3R4IH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGNvbnN0IGxhenlTY2hlbWEgPSB0aGlzLl9kZWYuZ2V0dGVyKCk7XG4gICAgcmV0dXJuIGxhenlTY2hlbWEuX3BhcnNlKHsgZGF0YTogY3R4LmRhdGEsIHBhdGg6IGN0eC5wYXRoLCBwYXJlbnQ6IGN0eCB9KTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGUgPSA8VCBleHRlbmRzIFpvZFR5cGVBbnk+KFxuICAgIGdldHRlcjogKCkgPT4gVCxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kTGF6eTxUPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RMYXp5KHtcbiAgICAgIGdldHRlcjogZ2V0dGVyLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RMYXp5LFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kTGl0ZXJhbCAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kTGl0ZXJhbERlZjxUID0gYW55PiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB2YWx1ZTogVDtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RMaXRlcmFsO1xufVxuXG5leHBvcnQgY2xhc3MgWm9kTGl0ZXJhbDxUPiBleHRlbmRzIFpvZFR5cGU8VCwgWm9kTGl0ZXJhbERlZjxUPj4ge1xuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBpZiAoaW5wdXQuZGF0YSAhPT0gdGhpcy5fZGVmLnZhbHVlKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfbGl0ZXJhbCxcbiAgICAgICAgZXhwZWN0ZWQ6IHRoaXMuX2RlZi52YWx1ZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiB7IHN0YXR1czogXCJ2YWxpZFwiLCB2YWx1ZTogaW5wdXQuZGF0YSB9O1xuICB9XG5cbiAgZ2V0IHZhbHVlKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYudmFsdWU7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gPFQgZXh0ZW5kcyBQcmltaXRpdmU+KFxuICAgIHZhbHVlOiBULFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RMaXRlcmFsPFQ+ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZExpdGVyYWwoe1xuICAgICAgdmFsdWU6IHZhbHVlLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RMaXRlcmFsLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kRW51bSAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCB0eXBlIEFycmF5S2V5cyA9IGtleW9mIGFueVtdO1xuZXhwb3J0IHR5cGUgSW5kaWNlczxUPiA9IEV4Y2x1ZGU8a2V5b2YgVCwgQXJyYXlLZXlzPjtcblxudHlwZSBFbnVtVmFsdWVzID0gW3N0cmluZywgLi4uc3RyaW5nW11dO1xuXG50eXBlIFZhbHVlczxUIGV4dGVuZHMgRW51bVZhbHVlcz4gPSB7XG4gIFtrIGluIFRbbnVtYmVyXV06IGs7XG59O1xuXG5leHBvcnQgaW50ZXJmYWNlIFpvZEVudW1EZWY8VCBleHRlbmRzIEVudW1WYWx1ZXMgPSBFbnVtVmFsdWVzPlxuICBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB2YWx1ZXM6IFQ7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kRW51bTtcbn1cblxudHlwZSBXcml0ZWFibGU8VD4gPSB7IC1yZWFkb25seSBbUCBpbiBrZXlvZiBUXTogVFtQXSB9O1xuXG5mdW5jdGlvbiBjcmVhdGVab2RFbnVtPFUgZXh0ZW5kcyBzdHJpbmcsIFQgZXh0ZW5kcyBSZWFkb25seTxbVSwgLi4uVVtdXT4+KFxuICB2YWx1ZXM6IFQsXG4gIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuKTogWm9kRW51bTxXcml0ZWFibGU8VD4+O1xuZnVuY3Rpb24gY3JlYXRlWm9kRW51bTxVIGV4dGVuZHMgc3RyaW5nLCBUIGV4dGVuZHMgW1UsIC4uLlVbXV0+KFxuICB2YWx1ZXM6IFQsXG4gIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuKTogWm9kRW51bTxUPjtcbmZ1bmN0aW9uIGNyZWF0ZVpvZEVudW0odmFsdWVzOiBhbnksIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtcykge1xuICByZXR1cm4gbmV3IFpvZEVudW0oe1xuICAgIHZhbHVlczogdmFsdWVzIGFzIGFueSxcbiAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEVudW0sXG4gICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICB9KSBhcyBhbnk7XG59XG5cbmV4cG9ydCBjbGFzcyBab2RFbnVtPFQgZXh0ZW5kcyBbc3RyaW5nLCAuLi5zdHJpbmdbXV0+IGV4dGVuZHMgWm9kVHlwZTxcbiAgVFtudW1iZXJdLFxuICBab2RFbnVtRGVmPFQ+XG4+IHtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgaWYgKHR5cGVvZiBpbnB1dC5kYXRhICE9PSBcInN0cmluZ1wiKSB7XG4gICAgICBjb25zdCBjdHggPSB0aGlzLl9nZXRPclJldHVybkN0eChpbnB1dCk7XG4gICAgICBjb25zdCBleHBlY3RlZFZhbHVlcyA9IHRoaXMuX2RlZi52YWx1ZXM7XG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgZXhwZWN0ZWQ6IHV0aWwuam9pblZhbHVlcyhleHBlY3RlZFZhbHVlcykgYXMgXCJzdHJpbmdcIixcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5wYXJzZWRUeXBlLFxuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG5cbiAgICBpZiAodGhpcy5fZGVmLnZhbHVlcy5pbmRleE9mKGlucHV0LmRhdGEpID09PSAtMSkge1xuICAgICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgICAgY29uc3QgZXhwZWN0ZWRWYWx1ZXMgPSB0aGlzLl9kZWYudmFsdWVzO1xuXG4gICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIHtcbiAgICAgICAgcmVjZWl2ZWQ6IGN0eC5kYXRhLFxuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF9lbnVtX3ZhbHVlLFxuICAgICAgICBvcHRpb25zOiBleHBlY3RlZFZhbHVlcyxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuICAgIHJldHVybiBPSyhpbnB1dC5kYXRhKTtcbiAgfVxuXG4gIGdldCBvcHRpb25zKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYudmFsdWVzO1xuICB9XG5cbiAgZ2V0IGVudW0oKTogVmFsdWVzPFQ+IHtcbiAgICBjb25zdCBlbnVtVmFsdWVzOiBhbnkgPSB7fTtcbiAgICBmb3IgKGNvbnN0IHZhbCBvZiB0aGlzLl9kZWYudmFsdWVzKSB7XG4gICAgICBlbnVtVmFsdWVzW3ZhbF0gPSB2YWw7XG4gICAgfVxuICAgIHJldHVybiBlbnVtVmFsdWVzIGFzIGFueTtcbiAgfVxuXG4gIGdldCBWYWx1ZXMoKTogVmFsdWVzPFQ+IHtcbiAgICBjb25zdCBlbnVtVmFsdWVzOiBhbnkgPSB7fTtcbiAgICBmb3IgKGNvbnN0IHZhbCBvZiB0aGlzLl9kZWYudmFsdWVzKSB7XG4gICAgICBlbnVtVmFsdWVzW3ZhbF0gPSB2YWw7XG4gICAgfVxuICAgIHJldHVybiBlbnVtVmFsdWVzIGFzIGFueTtcbiAgfVxuXG4gIGdldCBFbnVtKCk6IFZhbHVlczxUPiB7XG4gICAgY29uc3QgZW51bVZhbHVlczogYW55ID0ge307XG4gICAgZm9yIChjb25zdCB2YWwgb2YgdGhpcy5fZGVmLnZhbHVlcykge1xuICAgICAgZW51bVZhbHVlc1t2YWxdID0gdmFsO1xuICAgIH1cbiAgICByZXR1cm4gZW51bVZhbHVlcyBhcyBhbnk7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gY3JlYXRlWm9kRW51bTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE5hdGl2ZUVudW0gICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgaW50ZXJmYWNlIFpvZE5hdGl2ZUVudW1EZWY8VCBleHRlbmRzIEVudW1MaWtlID0gRW51bUxpa2U+XG4gIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIHZhbHVlczogVDtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROYXRpdmVFbnVtO1xufVxuXG50eXBlIEVudW1MaWtlID0geyBbazogc3RyaW5nXTogc3RyaW5nIHwgbnVtYmVyOyBbbnU6IG51bWJlcl06IHN0cmluZyB9O1xuXG5leHBvcnQgY2xhc3MgWm9kTmF0aXZlRW51bTxUIGV4dGVuZHMgRW51bUxpa2U+IGV4dGVuZHMgWm9kVHlwZTxcbiAgVFtrZXlvZiBUXSxcbiAgWm9kTmF0aXZlRW51bURlZjxUPlxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTxUW2tleW9mIFRdPiB7XG4gICAgY29uc3QgbmF0aXZlRW51bVZhbHVlcyA9IHV0aWwuZ2V0VmFsaWRFbnVtVmFsdWVzKHRoaXMuX2RlZi52YWx1ZXMpO1xuXG4gICAgY29uc3QgY3R4ID0gdGhpcy5fZ2V0T3JSZXR1cm5DdHgoaW5wdXQpO1xuICAgIGlmIChcbiAgICAgIGN0eC5wYXJzZWRUeXBlICE9PSBab2RQYXJzZWRUeXBlLnN0cmluZyAmJlxuICAgICAgY3R4LnBhcnNlZFR5cGUgIT09IFpvZFBhcnNlZFR5cGUubnVtYmVyXG4gICAgKSB7XG4gICAgICBjb25zdCBleHBlY3RlZFZhbHVlcyA9IHV0aWwub2JqZWN0VmFsdWVzKG5hdGl2ZUVudW1WYWx1ZXMpO1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGV4cGVjdGVkOiB1dGlsLmpvaW5WYWx1ZXMoZXhwZWN0ZWRWYWx1ZXMpIGFzIFwic3RyaW5nXCIsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgICAgY29kZTogWm9kSXNzdWVDb2RlLmludmFsaWRfdHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuXG4gICAgaWYgKG5hdGl2ZUVudW1WYWx1ZXMuaW5kZXhPZihpbnB1dC5kYXRhKSA9PT0gLTEpIHtcbiAgICAgIGNvbnN0IGV4cGVjdGVkVmFsdWVzID0gdXRpbC5vYmplY3RWYWx1ZXMobmF0aXZlRW51bVZhbHVlcyk7XG5cbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICByZWNlaXZlZDogY3R4LmRhdGEsXG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX2VudW1fdmFsdWUsXG4gICAgICAgIG9wdGlvbnM6IGV4cGVjdGVkVmFsdWVzLFxuICAgICAgfSk7XG4gICAgICByZXR1cm4gSU5WQUxJRDtcbiAgICB9XG4gICAgcmV0dXJuIE9LKGlucHV0LmRhdGEgYXMgYW55KTtcbiAgfVxuXG4gIGdldCBlbnVtKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYudmFsdWVzO1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IDxUIGV4dGVuZHMgRW51bUxpa2U+KFxuICAgIHZhbHVlczogVCxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kTmF0aXZlRW51bTxUPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2ROYXRpdmVFbnVtKHtcbiAgICAgIHZhbHVlczogdmFsdWVzLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROYXRpdmVFbnVtLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pO1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kUHJvbWlzZSAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbmV4cG9ydCBpbnRlcmZhY2UgWm9kUHJvbWlzZURlZjxUIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnk+XG4gIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIHR5cGU6IFQ7XG4gIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kUHJvbWlzZTtcbn1cblxuZXhwb3J0IGNsYXNzIFpvZFByb21pc2U8VCBleHRlbmRzIFpvZFR5cGVBbnk+IGV4dGVuZHMgWm9kVHlwZTxcbiAgUHJvbWlzZTxUW1wiX291dHB1dFwiXT4sXG4gIFpvZFByb21pc2VEZWY8VD4sXG4gIFByb21pc2U8VFtcIl9pbnB1dFwiXT5cbj4ge1xuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBjb25zdCB7IGN0eCB9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcbiAgICBpZiAoXG4gICAgICBjdHgucGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5wcm9taXNlICYmXG4gICAgICBjdHguY29tbW9uLmFzeW5jID09PSBmYWxzZVxuICAgICkge1xuICAgICAgYWRkSXNzdWVUb0NvbnRleHQoY3R4LCB7XG4gICAgICAgIGNvZGU6IFpvZElzc3VlQ29kZS5pbnZhbGlkX3R5cGUsXG4gICAgICAgIGV4cGVjdGVkOiBab2RQYXJzZWRUeXBlLnByb21pc2UsXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuXG4gICAgY29uc3QgcHJvbWlzaWZpZWQgPVxuICAgICAgY3R4LnBhcnNlZFR5cGUgPT09IFpvZFBhcnNlZFR5cGUucHJvbWlzZVxuICAgICAgICA/IGN0eC5kYXRhXG4gICAgICAgIDogUHJvbWlzZS5yZXNvbHZlKGN0eC5kYXRhKTtcblxuICAgIHJldHVybiBPSyhcbiAgICAgIHByb21pc2lmaWVkLnRoZW4oKGRhdGE6IGFueSkgPT4ge1xuICAgICAgICByZXR1cm4gdGhpcy5fZGVmLnR5cGUucGFyc2VBc3luYyhkYXRhLCB7XG4gICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgZXJyb3JNYXA6IGN0eC5jb21tb24uY29udGV4dHVhbEVycm9yTWFwLFxuICAgICAgICB9KTtcbiAgICAgIH0pXG4gICAgKTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGUgPSA8VCBleHRlbmRzIFpvZFR5cGVBbnk+KFxuICAgIHNjaGVtYTogVCxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kUHJvbWlzZTxUPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RQcm9taXNlKHtcbiAgICAgIHR5cGU6IHNjaGVtYSxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kUHJvbWlzZSxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgWm9kRWZmZWN0cyAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuXG5leHBvcnQgdHlwZSBSZWZpbmVtZW50PFQ+ID0gKGFyZzogVCwgY3R4OiBSZWZpbmVtZW50Q3R4KSA9PiBhbnk7XG5leHBvcnQgdHlwZSBTdXBlclJlZmluZW1lbnQ8VD4gPSAoYXJnOiBULCBjdHg6IFJlZmluZW1lbnRDdHgpID0+IHZvaWQ7XG5cbmV4cG9ydCB0eXBlIFJlZmluZW1lbnRFZmZlY3Q8VD4gPSB7XG4gIHR5cGU6IFwicmVmaW5lbWVudFwiO1xuICByZWZpbmVtZW50OiAoYXJnOiBULCBjdHg6IFJlZmluZW1lbnRDdHgpID0+IGFueTtcbn07XG5leHBvcnQgdHlwZSBUcmFuc2Zvcm1FZmZlY3Q8VD4gPSB7XG4gIHR5cGU6IFwidHJhbnNmb3JtXCI7XG4gIHRyYW5zZm9ybTogKGFyZzogVCwgY3R4OiBSZWZpbmVtZW50Q3R4KSA9PiBhbnk7XG59O1xuZXhwb3J0IHR5cGUgUHJlcHJvY2Vzc0VmZmVjdDxUPiA9IHtcbiAgdHlwZTogXCJwcmVwcm9jZXNzXCI7XG4gIHRyYW5zZm9ybTogKGFyZzogVCkgPT4gYW55O1xufTtcbmV4cG9ydCB0eXBlIEVmZmVjdDxUPiA9XG4gIHwgUmVmaW5lbWVudEVmZmVjdDxUPlxuICB8IFRyYW5zZm9ybUVmZmVjdDxUPlxuICB8IFByZXByb2Nlc3NFZmZlY3Q8VD47XG5cbmV4cG9ydCBpbnRlcmZhY2UgWm9kRWZmZWN0c0RlZjxUIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnk+XG4gIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIHNjaGVtYTogVDtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RFZmZlY3RzO1xuICBlZmZlY3Q6IEVmZmVjdDxhbnk+O1xufVxuXG5leHBvcnQgY2xhc3MgWm9kRWZmZWN0czxcbiAgVCBleHRlbmRzIFpvZFR5cGVBbnksXG4gIE91dHB1dCA9IFRbXCJfb3V0cHV0XCJdLFxuICBJbnB1dCA9IFRbXCJfaW5wdXRcIl1cbj4gZXh0ZW5kcyBab2RUeXBlPE91dHB1dCwgWm9kRWZmZWN0c0RlZjxUPiwgSW5wdXQ+IHtcbiAgaW5uZXJUeXBlKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuc2NoZW1hO1xuICB9XG5cbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgY29uc3QgeyBzdGF0dXMsIGN0eCB9ID0gdGhpcy5fcHJvY2Vzc0lucHV0UGFyYW1zKGlucHV0KTtcblxuICAgIGNvbnN0IGVmZmVjdCA9IHRoaXMuX2RlZi5lZmZlY3QgfHwgbnVsbDtcblxuICAgIGlmIChlZmZlY3QudHlwZSA9PT0gXCJwcmVwcm9jZXNzXCIpIHtcbiAgICAgIGNvbnN0IHByb2Nlc3NlZCA9IGVmZmVjdC50cmFuc2Zvcm0oY3R4LmRhdGEpO1xuXG4gICAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgICByZXR1cm4gUHJvbWlzZS5yZXNvbHZlKHByb2Nlc3NlZCkudGhlbigocHJvY2Vzc2VkKSA9PiB7XG4gICAgICAgICAgcmV0dXJuIHRoaXMuX2RlZi5zY2hlbWEuX3BhcnNlQXN5bmMoe1xuICAgICAgICAgICAgZGF0YTogcHJvY2Vzc2VkLFxuICAgICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgICBwYXJlbnQ6IGN0eCxcbiAgICAgICAgICB9KTtcbiAgICAgICAgfSk7XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gdGhpcy5fZGVmLnNjaGVtYS5fcGFyc2VTeW5jKHtcbiAgICAgICAgICBkYXRhOiBwcm9jZXNzZWQsXG4gICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgcGFyZW50OiBjdHgsXG4gICAgICAgIH0pO1xuICAgICAgfVxuICAgIH1cblxuICAgIGNvbnN0IGNoZWNrQ3R4OiBSZWZpbmVtZW50Q3R4ID0ge1xuICAgICAgYWRkSXNzdWU6IChhcmc6IElzc3VlRGF0YSkgPT4ge1xuICAgICAgICBhZGRJc3N1ZVRvQ29udGV4dChjdHgsIGFyZyk7XG4gICAgICAgIGlmIChhcmcuZmF0YWwpIHtcbiAgICAgICAgICBzdGF0dXMuYWJvcnQoKTtcbiAgICAgICAgfSBlbHNlIHtcbiAgICAgICAgICBzdGF0dXMuZGlydHkoKTtcbiAgICAgICAgfVxuICAgICAgfSxcbiAgICAgIGdldCBwYXRoKCkge1xuICAgICAgICByZXR1cm4gY3R4LnBhdGg7XG4gICAgICB9LFxuICAgIH07XG5cbiAgICBjaGVja0N0eC5hZGRJc3N1ZSA9IGNoZWNrQ3R4LmFkZElzc3VlLmJpbmQoY2hlY2tDdHgpO1xuICAgIGlmIChlZmZlY3QudHlwZSA9PT0gXCJyZWZpbmVtZW50XCIpIHtcbiAgICAgIGNvbnN0IGV4ZWN1dGVSZWZpbmVtZW50ID0gKFxuICAgICAgICBhY2M6IHVua25vd25cbiAgICAgICAgLy8gZWZmZWN0OiBSZWZpbmVtZW50RWZmZWN0PGFueT5cbiAgICAgICk6IGFueSA9PiB7XG4gICAgICAgIGNvbnN0IHJlc3VsdCA9IGVmZmVjdC5yZWZpbmVtZW50KGFjYywgY2hlY2tDdHgpO1xuICAgICAgICBpZiAoY3R4LmNvbW1vbi5hc3luYykge1xuICAgICAgICAgIHJldHVybiBQcm9taXNlLnJlc29sdmUocmVzdWx0KTtcbiAgICAgICAgfVxuICAgICAgICBpZiAocmVzdWx0IGluc3RhbmNlb2YgUHJvbWlzZSkge1xuICAgICAgICAgIHRocm93IG5ldyBFcnJvcihcbiAgICAgICAgICAgIFwiQXN5bmMgcmVmaW5lbWVudCBlbmNvdW50ZXJlZCBkdXJpbmcgc3luY2hyb25vdXMgcGFyc2Ugb3BlcmF0aW9uLiBVc2UgLnBhcnNlQXN5bmMgaW5zdGVhZC5cIlxuICAgICAgICAgICk7XG4gICAgICAgIH1cbiAgICAgICAgcmV0dXJuIGFjYztcbiAgICAgIH07XG5cbiAgICAgIGlmIChjdHguY29tbW9uLmFzeW5jID09PSBmYWxzZSkge1xuICAgICAgICBjb25zdCBpbm5lciA9IHRoaXMuX2RlZi5zY2hlbWEuX3BhcnNlU3luYyh7XG4gICAgICAgICAgZGF0YTogY3R4LmRhdGEsXG4gICAgICAgICAgcGF0aDogY3R4LnBhdGgsXG4gICAgICAgICAgcGFyZW50OiBjdHgsXG4gICAgICAgIH0pO1xuICAgICAgICBpZiAoaW5uZXIuc3RhdHVzID09PSBcImFib3J0ZWRcIikgcmV0dXJuIElOVkFMSUQ7XG4gICAgICAgIGlmIChpbm5lci5zdGF0dXMgPT09IFwiZGlydHlcIikgc3RhdHVzLmRpcnR5KCk7XG5cbiAgICAgICAgLy8gcmV0dXJuIHZhbHVlIGlzIGlnbm9yZWRcbiAgICAgICAgZXhlY3V0ZVJlZmluZW1lbnQoaW5uZXIudmFsdWUpO1xuICAgICAgICByZXR1cm4geyBzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IGlubmVyLnZhbHVlIH07XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gdGhpcy5fZGVmLnNjaGVtYVxuICAgICAgICAgIC5fcGFyc2VBc3luYyh7IGRhdGE6IGN0eC5kYXRhLCBwYXRoOiBjdHgucGF0aCwgcGFyZW50OiBjdHggfSlcbiAgICAgICAgICAudGhlbigoaW5uZXIpID0+IHtcbiAgICAgICAgICAgIGlmIChpbm5lci5zdGF0dXMgPT09IFwiYWJvcnRlZFwiKSByZXR1cm4gSU5WQUxJRDtcbiAgICAgICAgICAgIGlmIChpbm5lci5zdGF0dXMgPT09IFwiZGlydHlcIikgc3RhdHVzLmRpcnR5KCk7XG5cbiAgICAgICAgICAgIHJldHVybiBleGVjdXRlUmVmaW5lbWVudChpbm5lci52YWx1ZSkudGhlbigoKSA9PiB7XG4gICAgICAgICAgICAgIHJldHVybiB7IHN0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogaW5uZXIudmFsdWUgfTtcbiAgICAgICAgICAgIH0pO1xuICAgICAgICAgIH0pO1xuICAgICAgfVxuICAgIH1cblxuICAgIGlmIChlZmZlY3QudHlwZSA9PT0gXCJ0cmFuc2Zvcm1cIikge1xuICAgICAgaWYgKGN0eC5jb21tb24uYXN5bmMgPT09IGZhbHNlKSB7XG4gICAgICAgIGNvbnN0IGJhc2UgPSB0aGlzLl9kZWYuc2NoZW1hLl9wYXJzZVN5bmMoe1xuICAgICAgICAgIGRhdGE6IGN0eC5kYXRhLFxuICAgICAgICAgIHBhdGg6IGN0eC5wYXRoLFxuICAgICAgICAgIHBhcmVudDogY3R4LFxuICAgICAgICB9KTtcbiAgICAgICAgLy8gaWYgKGJhc2Uuc3RhdHVzID09PSBcImFib3J0ZWRcIikgcmV0dXJuIElOVkFMSUQ7XG4gICAgICAgIC8vIGlmIChiYXNlLnN0YXR1cyA9PT0gXCJkaXJ0eVwiKSB7XG4gICAgICAgIC8vICAgcmV0dXJuIHsgc3RhdHVzOiBcImRpcnR5XCIsIHZhbHVlOiBiYXNlLnZhbHVlIH07XG4gICAgICAgIC8vIH1cbiAgICAgICAgaWYgKCFpc1ZhbGlkKGJhc2UpKSByZXR1cm4gYmFzZTtcblxuICAgICAgICBjb25zdCByZXN1bHQgPSBlZmZlY3QudHJhbnNmb3JtKGJhc2UudmFsdWUsIGNoZWNrQ3R4KTtcbiAgICAgICAgaWYgKHJlc3VsdCBpbnN0YW5jZW9mIFByb21pc2UpIHtcbiAgICAgICAgICB0aHJvdyBuZXcgRXJyb3IoXG4gICAgICAgICAgICBgQXN5bmNocm9ub3VzIHRyYW5zZm9ybSBlbmNvdW50ZXJlZCBkdXJpbmcgc3luY2hyb25vdXMgcGFyc2Ugb3BlcmF0aW9uLiBVc2UgLnBhcnNlQXN5bmMgaW5zdGVhZC5gXG4gICAgICAgICAgKTtcbiAgICAgICAgfVxuXG4gICAgICAgIHJldHVybiB7IHN0YXR1czogc3RhdHVzLnZhbHVlLCB2YWx1ZTogcmVzdWx0IH07XG4gICAgICB9IGVsc2Uge1xuICAgICAgICByZXR1cm4gdGhpcy5fZGVmLnNjaGVtYVxuICAgICAgICAgIC5fcGFyc2VBc3luYyh7IGRhdGE6IGN0eC5kYXRhLCBwYXRoOiBjdHgucGF0aCwgcGFyZW50OiBjdHggfSlcbiAgICAgICAgICAudGhlbigoYmFzZSkgPT4ge1xuICAgICAgICAgICAgaWYgKCFpc1ZhbGlkKGJhc2UpKSByZXR1cm4gYmFzZTtcbiAgICAgICAgICAgIC8vIGlmIChiYXNlLnN0YXR1cyA9PT0gXCJhYm9ydGVkXCIpIHJldHVybiBJTlZBTElEO1xuICAgICAgICAgICAgLy8gaWYgKGJhc2Uuc3RhdHVzID09PSBcImRpcnR5XCIpIHtcbiAgICAgICAgICAgIC8vICAgcmV0dXJuIHsgc3RhdHVzOiBcImRpcnR5XCIsIHZhbHVlOiBiYXNlLnZhbHVlIH07XG4gICAgICAgICAgICAvLyB9XG4gICAgICAgICAgICByZXR1cm4gUHJvbWlzZS5yZXNvbHZlKGVmZmVjdC50cmFuc2Zvcm0oYmFzZS52YWx1ZSwgY2hlY2tDdHgpKS50aGVuKFxuICAgICAgICAgICAgICAocmVzdWx0KSA9PiAoeyBzdGF0dXM6IHN0YXR1cy52YWx1ZSwgdmFsdWU6IHJlc3VsdCB9KVxuICAgICAgICAgICAgKTtcbiAgICAgICAgICB9KTtcbiAgICAgIH1cbiAgICB9XG5cbiAgICB1dGlsLmFzc2VydE5ldmVyKGVmZmVjdCk7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gPEkgZXh0ZW5kcyBab2RUeXBlQW55PihcbiAgICBzY2hlbWE6IEksXG4gICAgZWZmZWN0OiBFZmZlY3Q8SVtcIl9vdXRwdXRcIl0+LFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RFZmZlY3RzPEksIElbXCJfb3V0cHV0XCJdPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RFZmZlY3RzKHtcbiAgICAgIHNjaGVtYSxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kRWZmZWN0cyxcbiAgICAgIGVmZmVjdCxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KTtcbiAgfTtcblxuICBzdGF0aWMgY3JlYXRlV2l0aFByZXByb2Nlc3MgPSA8SSBleHRlbmRzIFpvZFR5cGVBbnk+KFxuICAgIHByZXByb2Nlc3M6IChhcmc6IHVua25vd24pID0+IHVua25vd24sXG4gICAgc2NoZW1hOiBJLFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RFZmZlY3RzPEksIElbXCJfb3V0cHV0XCJdPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RFZmZlY3RzKHtcbiAgICAgIHNjaGVtYSxcbiAgICAgIGVmZmVjdDogeyB0eXBlOiBcInByZXByb2Nlc3NcIiwgdHJhbnNmb3JtOiBwcmVwcm9jZXNzIH0sXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZEVmZmVjdHMsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG59XG5cbmV4cG9ydCB7IFpvZEVmZmVjdHMgYXMgWm9kVHJhbnNmb3JtZXIgfTtcblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE9wdGlvbmFsICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuZXhwb3J0IGludGVyZmFjZSBab2RPcHRpb25hbERlZjxUIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnk+XG4gIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIGlubmVyVHlwZTogVDtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2RPcHRpb25hbDtcbn1cblxuZXhwb3J0IHR5cGUgWm9kT3B0aW9uYWxUeXBlPFQgZXh0ZW5kcyBab2RUeXBlQW55PiA9IFpvZE9wdGlvbmFsPFQ+O1xuXG5leHBvcnQgY2xhc3MgWm9kT3B0aW9uYWw8VCBleHRlbmRzIFpvZFR5cGVBbnk+IGV4dGVuZHMgWm9kVHlwZTxcbiAgVFtcIl9vdXRwdXRcIl0gfCB1bmRlZmluZWQsXG4gIFpvZE9wdGlvbmFsRGVmPFQ+LFxuICBUW1wiX2lucHV0XCJdIHwgdW5kZWZpbmVkXG4+IHtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPHRoaXNbXCJfb3V0cHV0XCJdPiB7XG4gICAgY29uc3QgcGFyc2VkVHlwZSA9IHRoaXMuX2dldFR5cGUoaW5wdXQpO1xuICAgIGlmIChwYXJzZWRUeXBlID09PSBab2RQYXJzZWRUeXBlLnVuZGVmaW5lZCkge1xuICAgICAgcmV0dXJuIE9LKHVuZGVmaW5lZCk7XG4gICAgfVxuICAgIHJldHVybiB0aGlzLl9kZWYuaW5uZXJUeXBlLl9wYXJzZShpbnB1dCk7XG4gIH1cblxuICB1bndyYXAoKSB7XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5pbm5lclR5cGU7XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gPFQgZXh0ZW5kcyBab2RUeXBlQW55PihcbiAgICB0eXBlOiBULFxuICAgIHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtc1xuICApOiBab2RPcHRpb25hbDxUPiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2RPcHRpb25hbCh7XG4gICAgICBpbm5lclR5cGU6IHR5cGUsXG4gICAgICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE9wdGlvbmFsLFxuICAgICAgLi4ucHJvY2Vzc0NyZWF0ZVBhcmFtcyhwYXJhbXMpLFxuICAgIH0pIGFzIGFueTtcbiAgfTtcbn1cblxuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgIFpvZE51bGxhYmxlICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuZXhwb3J0IGludGVyZmFjZSBab2ROdWxsYWJsZURlZjxUIGV4dGVuZHMgWm9kVHlwZUFueSA9IFpvZFR5cGVBbnk+XG4gIGV4dGVuZHMgWm9kVHlwZURlZiB7XG4gIGlubmVyVHlwZTogVDtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROdWxsYWJsZTtcbn1cblxuZXhwb3J0IHR5cGUgWm9kTnVsbGFibGVUeXBlPFQgZXh0ZW5kcyBab2RUeXBlQW55PiA9IFpvZE51bGxhYmxlPFQ+O1xuXG5leHBvcnQgY2xhc3MgWm9kTnVsbGFibGU8VCBleHRlbmRzIFpvZFR5cGVBbnk+IGV4dGVuZHMgWm9kVHlwZTxcbiAgVFtcIl9vdXRwdXRcIl0gfCBudWxsLFxuICBab2ROdWxsYWJsZURlZjxUPixcbiAgVFtcIl9pbnB1dFwiXSB8IG51bGxcbj4ge1xuICBfcGFyc2UoaW5wdXQ6IFBhcnNlSW5wdXQpOiBQYXJzZVJldHVyblR5cGU8dGhpc1tcIl9vdXRwdXRcIl0+IHtcbiAgICBjb25zdCBwYXJzZWRUeXBlID0gdGhpcy5fZ2V0VHlwZShpbnB1dCk7XG4gICAgaWYgKHBhcnNlZFR5cGUgPT09IFpvZFBhcnNlZFR5cGUubnVsbCkge1xuICAgICAgcmV0dXJuIE9LKG51bGwpO1xuICAgIH1cbiAgICByZXR1cm4gdGhpcy5fZGVmLmlubmVyVHlwZS5fcGFyc2UoaW5wdXQpO1xuICB9XG5cbiAgdW53cmFwKCkge1xuICAgIHJldHVybiB0aGlzLl9kZWYuaW5uZXJUeXBlO1xuICB9XG5cbiAgc3RhdGljIGNyZWF0ZSA9IDxUIGV4dGVuZHMgWm9kVHlwZUFueT4oXG4gICAgdHlwZTogVCxcbiAgICBwYXJhbXM/OiBSYXdDcmVhdGVQYXJhbXNcbiAgKTogWm9kTnVsbGFibGU8VD4gPT4ge1xuICAgIHJldHVybiBuZXcgWm9kTnVsbGFibGUoe1xuICAgICAgaW5uZXJUeXBlOiB0eXBlLFxuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROdWxsYWJsZSxcbiAgICAgIC4uLnByb2Nlc3NDcmVhdGVQYXJhbXMocGFyYW1zKSxcbiAgICB9KSBhcyBhbnk7XG4gIH07XG59XG5cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgWm9kRGVmYXVsdCAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgICAgICAgICAgICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG5leHBvcnQgaW50ZXJmYWNlIFpvZERlZmF1bHREZWY8VCBleHRlbmRzIFpvZFR5cGVBbnkgPSBab2RUeXBlQW55PlxuICBleHRlbmRzIFpvZFR5cGVEZWYge1xuICBpbm5lclR5cGU6IFQ7XG4gIGRlZmF1bHRWYWx1ZTogKCkgPT4gdXRpbC5ub1VuZGVmaW5lZDxUW1wiX2lucHV0XCJdPjtcbiAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2REZWZhdWx0O1xufVxuXG5leHBvcnQgY2xhc3MgWm9kRGVmYXVsdDxUIGV4dGVuZHMgWm9kVHlwZUFueT4gZXh0ZW5kcyBab2RUeXBlPFxuICB1dGlsLm5vVW5kZWZpbmVkPFRbXCJfb3V0cHV0XCJdPixcbiAgWm9kRGVmYXVsdERlZjxUPixcbiAgVFtcIl9pbnB1dFwiXSB8IHVuZGVmaW5lZFxuPiB7XG4gIF9wYXJzZShpbnB1dDogUGFyc2VJbnB1dCk6IFBhcnNlUmV0dXJuVHlwZTx0aGlzW1wiX291dHB1dFwiXT4ge1xuICAgIGNvbnN0IHsgY3R4IH0gPSB0aGlzLl9wcm9jZXNzSW5wdXRQYXJhbXMoaW5wdXQpO1xuICAgIGxldCBkYXRhID0gY3R4LmRhdGE7XG4gICAgaWYgKGN0eC5wYXJzZWRUeXBlID09PSBab2RQYXJzZWRUeXBlLnVuZGVmaW5lZCkge1xuICAgICAgZGF0YSA9IHRoaXMuX2RlZi5kZWZhdWx0VmFsdWUoKTtcbiAgICB9XG4gICAgcmV0dXJuIHRoaXMuX2RlZi5pbm5lclR5cGUuX3BhcnNlKHtcbiAgICAgIGRhdGEsXG4gICAgICBwYXRoOiBjdHgucGF0aCxcbiAgICAgIHBhcmVudDogY3R4LFxuICAgIH0pO1xuICB9XG5cbiAgcmVtb3ZlRGVmYXVsdCgpIHtcbiAgICByZXR1cm4gdGhpcy5fZGVmLmlubmVyVHlwZTtcbiAgfVxuXG4gIHN0YXRpYyBjcmVhdGUgPSA8VCBleHRlbmRzIFpvZFR5cGVBbnk+KFxuICAgIHR5cGU6IFQsXG4gICAgcGFyYW1zPzogUmF3Q3JlYXRlUGFyYW1zXG4gICk6IFpvZE9wdGlvbmFsPFQ+ID0+IHtcbiAgICByZXR1cm4gbmV3IFpvZE9wdGlvbmFsKHtcbiAgICAgIGlubmVyVHlwZTogdHlwZSxcbiAgICAgIHR5cGVOYW1lOiBab2RGaXJzdFBhcnR5VHlwZUtpbmQuWm9kT3B0aW9uYWwsXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSkgYXMgYW55O1xuICB9O1xufVxuXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy9cbi8vLy8vLy8vLy8gICAgICAgICAgICAgICAgICAgICAvLy8vLy8vLy8vXG4vLy8vLy8vLy8vICAgICAgWm9kTmFOICAgICAgICAgLy8vLy8vLy8vL1xuLy8vLy8vLy8vLyAgICAgICAgICAgICAgICAgICAgIC8vLy8vLy8vLy9cbi8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vXG4vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vL1xuXG5leHBvcnQgaW50ZXJmYWNlIFpvZE5hTkRlZiBleHRlbmRzIFpvZFR5cGVEZWYge1xuICB0eXBlTmFtZTogWm9kRmlyc3RQYXJ0eVR5cGVLaW5kLlpvZE5hTjtcbn1cblxuZXhwb3J0IGNsYXNzIFpvZE5hTiBleHRlbmRzIFpvZFR5cGU8bnVtYmVyLCBab2ROYU5EZWY+IHtcbiAgX3BhcnNlKGlucHV0OiBQYXJzZUlucHV0KTogUGFyc2VSZXR1cm5UeXBlPGFueT4ge1xuICAgIGNvbnN0IHBhcnNlZFR5cGUgPSB0aGlzLl9nZXRUeXBlKGlucHV0KTtcbiAgICBpZiAocGFyc2VkVHlwZSAhPT0gWm9kUGFyc2VkVHlwZS5uYW4pIHtcbiAgICAgIGNvbnN0IGN0eCA9IHRoaXMuX2dldE9yUmV0dXJuQ3R4KGlucHV0KTtcbiAgICAgIGFkZElzc3VlVG9Db250ZXh0KGN0eCwge1xuICAgICAgICBjb2RlOiBab2RJc3N1ZUNvZGUuaW52YWxpZF90eXBlLFxuICAgICAgICBleHBlY3RlZDogWm9kUGFyc2VkVHlwZS5uYW4sXG4gICAgICAgIHJlY2VpdmVkOiBjdHgucGFyc2VkVHlwZSxcbiAgICAgIH0pO1xuICAgICAgcmV0dXJuIElOVkFMSUQ7XG4gICAgfVxuXG4gICAgcmV0dXJuIHsgc3RhdHVzOiBcInZhbGlkXCIsIHZhbHVlOiBpbnB1dC5kYXRhIH07XG4gIH1cblxuICBzdGF0aWMgY3JlYXRlID0gKHBhcmFtcz86IFJhd0NyZWF0ZVBhcmFtcyk6IFpvZE5hTiA9PiB7XG4gICAgcmV0dXJuIG5ldyBab2ROYU4oe1xuICAgICAgdHlwZU5hbWU6IFpvZEZpcnN0UGFydHlUeXBlS2luZC5ab2ROYU4sXG4gICAgICAuLi5wcm9jZXNzQ3JlYXRlUGFyYW1zKHBhcmFtcyksXG4gICAgfSk7XG4gIH07XG59XG5cbmV4cG9ydCBjb25zdCBjdXN0b20gPSA8VD4oXG4gIGNoZWNrPzogKGRhdGE6IHVua25vd24pID0+IGFueSxcbiAgcGFyYW1zOiBQYXJhbWV0ZXJzPFpvZFR5cGVBbnlbXCJyZWZpbmVcIl0+WzFdID0ge30sXG4gIGZhdGFsPzogYm9vbGVhblxuKTogWm9kVHlwZTxUPiA9PiB7XG4gIGlmIChjaGVjaylcbiAgICByZXR1cm4gWm9kQW55LmNyZWF0ZSgpLnN1cGVyUmVmaW5lKChkYXRhLCBjdHgpID0+IHtcbiAgICAgIGlmICghY2hlY2soZGF0YSkpIHtcbiAgICAgICAgY29uc3QgcCA9IHR5cGVvZiBwYXJhbXMgPT09IFwiZnVuY3Rpb25cIiA/IHBhcmFtcyhkYXRhKSA6IHBhcmFtcztcbiAgICAgICAgY29uc3QgcDIgPSB0eXBlb2YgcCA9PT0gXCJzdHJpbmdcIiA/IHsgbWVzc2FnZTogcCB9IDogcDtcbiAgICAgICAgY3R4LmFkZElzc3VlKHsgY29kZTogXCJjdXN0b21cIiwgLi4ucDIsIGZhdGFsIH0pO1xuICAgICAgfVxuICAgIH0pO1xuICByZXR1cm4gWm9kQW55LmNyZWF0ZSgpO1xufTtcblxuZXhwb3J0IHsgWm9kVHlwZSBhcyBTY2hlbWEsIFpvZFR5cGUgYXMgWm9kU2NoZW1hIH07XG5cbmV4cG9ydCBjb25zdCBsYXRlID0ge1xuICBvYmplY3Q6IFpvZE9iamVjdC5sYXp5Y3JlYXRlLFxufTtcblxuZXhwb3J0IGVudW0gWm9kRmlyc3RQYXJ0eVR5cGVLaW5kIHtcbiAgWm9kU3RyaW5nID0gXCJab2RTdHJpbmdcIixcbiAgWm9kTnVtYmVyID0gXCJab2ROdW1iZXJcIixcbiAgWm9kTmFOID0gXCJab2ROYU5cIixcbiAgWm9kQmlnSW50ID0gXCJab2RCaWdJbnRcIixcbiAgWm9kQm9vbGVhbiA9IFwiWm9kQm9vbGVhblwiLFxuICBab2REYXRlID0gXCJab2REYXRlXCIsXG4gIFpvZFVuZGVmaW5lZCA9IFwiWm9kVW5kZWZpbmVkXCIsXG4gIFpvZE51bGwgPSBcIlpvZE51bGxcIixcbiAgWm9kQW55ID0gXCJab2RBbnlcIixcbiAgWm9kVW5rbm93biA9IFwiWm9kVW5rbm93blwiLFxuICBab2ROZXZlciA9IFwiWm9kTmV2ZXJcIixcbiAgWm9kVm9pZCA9IFwiWm9kVm9pZFwiLFxuICBab2RBcnJheSA9IFwiWm9kQXJyYXlcIixcbiAgWm9kT2JqZWN0ID0gXCJab2RPYmplY3RcIixcbiAgWm9kVW5pb24gPSBcIlpvZFVuaW9uXCIsXG4gIFpvZERpc2NyaW1pbmF0ZWRVbmlvbiA9IFwiWm9kRGlzY3JpbWluYXRlZFVuaW9uXCIsXG4gIFpvZEludGVyc2VjdGlvbiA9IFwiWm9kSW50ZXJzZWN0aW9uXCIsXG4gIFpvZFR1cGxlID0gXCJab2RUdXBsZVwiLFxuICBab2RSZWNvcmQgPSBcIlpvZFJlY29yZFwiLFxuICBab2RNYXAgPSBcIlpvZE1hcFwiLFxuICBab2RTZXQgPSBcIlpvZFNldFwiLFxuICBab2RGdW5jdGlvbiA9IFwiWm9kRnVuY3Rpb25cIixcbiAgWm9kTGF6eSA9IFwiWm9kTGF6eVwiLFxuICBab2RMaXRlcmFsID0gXCJab2RMaXRlcmFsXCIsXG4gIFpvZEVudW0gPSBcIlpvZEVudW1cIixcbiAgWm9kRWZmZWN0cyA9IFwiWm9kRWZmZWN0c1wiLFxuICBab2ROYXRpdmVFbnVtID0gXCJab2ROYXRpdmVFbnVtXCIsXG4gIFpvZE9wdGlvbmFsID0gXCJab2RPcHRpb25hbFwiLFxuICBab2ROdWxsYWJsZSA9IFwiWm9kTnVsbGFibGVcIixcbiAgWm9kRGVmYXVsdCA9IFwiWm9kRGVmYXVsdFwiLFxuICBab2RQcm9taXNlID0gXCJab2RQcm9taXNlXCIsXG59XG5leHBvcnQgdHlwZSBab2RGaXJzdFBhcnR5U2NoZW1hVHlwZXMgPVxuICB8IFpvZFN0cmluZ1xuICB8IFpvZE51bWJlclxuICB8IFpvZE5hTlxuICB8IFpvZEJpZ0ludFxuICB8IFpvZEJvb2xlYW5cbiAgfCBab2REYXRlXG4gIHwgWm9kVW5kZWZpbmVkXG4gIHwgWm9kTnVsbFxuICB8IFpvZEFueVxuICB8IFpvZFVua25vd25cbiAgfCBab2ROZXZlclxuICB8IFpvZFZvaWRcbiAgfCBab2RBcnJheTxhbnksIGFueT5cbiAgfCBab2RPYmplY3Q8YW55LCBhbnksIGFueSwgYW55LCBhbnk+XG4gIHwgWm9kVW5pb248YW55PlxuICB8IFpvZERpc2NyaW1pbmF0ZWRVbmlvbjxhbnksIGFueSwgYW55PlxuICB8IFpvZEludGVyc2VjdGlvbjxhbnksIGFueT5cbiAgfCBab2RUdXBsZTxhbnksIGFueT5cbiAgfCBab2RSZWNvcmQ8YW55LCBhbnk+XG4gIHwgWm9kTWFwPGFueT5cbiAgfCBab2RTZXQ8YW55PlxuICB8IFpvZEZ1bmN0aW9uPGFueSwgYW55PlxuICB8IFpvZExhenk8YW55PlxuICB8IFpvZExpdGVyYWw8YW55PlxuICB8IFpvZEVudW08YW55PlxuICB8IFpvZEVmZmVjdHM8YW55LCBhbnksIGFueT5cbiAgfCBab2ROYXRpdmVFbnVtPGFueT5cbiAgfCBab2RPcHRpb25hbDxhbnk+XG4gIHwgWm9kTnVsbGFibGU8YW55PlxuICB8IFpvZERlZmF1bHQ8YW55PlxuICB8IFpvZFByb21pc2U8YW55PjtcblxuY29uc3QgaW5zdGFuY2VPZlR5cGUgPSA8VCBleHRlbmRzIG5ldyAoLi4uYXJnczogYW55W10pID0+IGFueT4oXG4gIGNsczogVCxcbiAgcGFyYW1zOiBQYXJhbWV0ZXJzPFpvZFR5cGVBbnlbXCJyZWZpbmVcIl0+WzFdID0ge1xuICAgIG1lc3NhZ2U6IGBJbnB1dCBub3QgaW5zdGFuY2Ugb2YgJHtjbHMubmFtZX1gLFxuICB9XG4pID0+IGN1c3RvbTxJbnN0YW5jZVR5cGU8VD4+KChkYXRhKSA9PiBkYXRhIGluc3RhbmNlb2YgY2xzLCBwYXJhbXMsIHRydWUpO1xuXG5jb25zdCBzdHJpbmdUeXBlID0gWm9kU3RyaW5nLmNyZWF0ZTtcbmNvbnN0IG51bWJlclR5cGUgPSBab2ROdW1iZXIuY3JlYXRlO1xuY29uc3QgbmFuVHlwZSA9IFpvZE5hTi5jcmVhdGU7XG5jb25zdCBiaWdJbnRUeXBlID0gWm9kQmlnSW50LmNyZWF0ZTtcbmNvbnN0IGJvb2xlYW5UeXBlID0gWm9kQm9vbGVhbi5jcmVhdGU7XG5jb25zdCBkYXRlVHlwZSA9IFpvZERhdGUuY3JlYXRlO1xuY29uc3QgdW5kZWZpbmVkVHlwZSA9IFpvZFVuZGVmaW5lZC5jcmVhdGU7XG5jb25zdCBudWxsVHlwZSA9IFpvZE51bGwuY3JlYXRlO1xuY29uc3QgYW55VHlwZSA9IFpvZEFueS5jcmVhdGU7XG5jb25zdCB1bmtub3duVHlwZSA9IFpvZFVua25vd24uY3JlYXRlO1xuY29uc3QgbmV2ZXJUeXBlID0gWm9kTmV2ZXIuY3JlYXRlO1xuY29uc3Qgdm9pZFR5cGUgPSBab2RWb2lkLmNyZWF0ZTtcbmNvbnN0IGFycmF5VHlwZSA9IFpvZEFycmF5LmNyZWF0ZTtcbmNvbnN0IG9iamVjdFR5cGUgPSBab2RPYmplY3QuY3JlYXRlO1xuY29uc3Qgc3RyaWN0T2JqZWN0VHlwZSA9IFpvZE9iamVjdC5zdHJpY3RDcmVhdGU7XG5jb25zdCB1bmlvblR5cGUgPSBab2RVbmlvbi5jcmVhdGU7XG5jb25zdCBkaXNjcmltaW5hdGVkVW5pb25UeXBlID0gWm9kRGlzY3JpbWluYXRlZFVuaW9uLmNyZWF0ZTtcbmNvbnN0IGludGVyc2VjdGlvblR5cGUgPSBab2RJbnRlcnNlY3Rpb24uY3JlYXRlO1xuY29uc3QgdHVwbGVUeXBlID0gWm9kVHVwbGUuY3JlYXRlO1xuY29uc3QgcmVjb3JkVHlwZSA9IFpvZFJlY29yZC5jcmVhdGU7XG5jb25zdCBtYXBUeXBlID0gWm9kTWFwLmNyZWF0ZTtcbmNvbnN0IHNldFR5cGUgPSBab2RTZXQuY3JlYXRlO1xuY29uc3QgZnVuY3Rpb25UeXBlID0gWm9kRnVuY3Rpb24uY3JlYXRlO1xuY29uc3QgbGF6eVR5cGUgPSBab2RMYXp5LmNyZWF0ZTtcbmNvbnN0IGxpdGVyYWxUeXBlID0gWm9kTGl0ZXJhbC5jcmVhdGU7XG5jb25zdCBlbnVtVHlwZSA9IFpvZEVudW0uY3JlYXRlO1xuY29uc3QgbmF0aXZlRW51bVR5cGUgPSBab2ROYXRpdmVFbnVtLmNyZWF0ZTtcbmNvbnN0IHByb21pc2VUeXBlID0gWm9kUHJvbWlzZS5jcmVhdGU7XG5jb25zdCBlZmZlY3RzVHlwZSA9IFpvZEVmZmVjdHMuY3JlYXRlO1xuY29uc3Qgb3B0aW9uYWxUeXBlID0gWm9kT3B0aW9uYWwuY3JlYXRlO1xuY29uc3QgbnVsbGFibGVUeXBlID0gWm9kTnVsbGFibGUuY3JlYXRlO1xuY29uc3QgcHJlcHJvY2Vzc1R5cGUgPSBab2RFZmZlY3RzLmNyZWF0ZVdpdGhQcmVwcm9jZXNzO1xuY29uc3Qgb3N0cmluZyA9ICgpID0+IHN0cmluZ1R5cGUoKS5vcHRpb25hbCgpO1xuY29uc3Qgb251bWJlciA9ICgpID0+IG51bWJlclR5cGUoKS5vcHRpb25hbCgpO1xuY29uc3Qgb2Jvb2xlYW4gPSAoKSA9PiBib29sZWFuVHlwZSgpLm9wdGlvbmFsKCk7XG5cbmV4cG9ydCB7XG4gIGFueVR5cGUgYXMgYW55LFxuICBhcnJheVR5cGUgYXMgYXJyYXksXG4gIGJpZ0ludFR5cGUgYXMgYmlnaW50LFxuICBib29sZWFuVHlwZSBhcyBib29sZWFuLFxuICBkYXRlVHlwZSBhcyBkYXRlLFxuICBkaXNjcmltaW5hdGVkVW5pb25UeXBlIGFzIGRpc2NyaW1pbmF0ZWRVbmlvbixcbiAgZWZmZWN0c1R5cGUgYXMgZWZmZWN0LFxuICBlbnVtVHlwZSBhcyBlbnVtLFxuICBmdW5jdGlvblR5cGUgYXMgZnVuY3Rpb24sXG4gIGluc3RhbmNlT2ZUeXBlIGFzIGluc3RhbmNlb2YsXG4gIGludGVyc2VjdGlvblR5cGUgYXMgaW50ZXJzZWN0aW9uLFxuICBsYXp5VHlwZSBhcyBsYXp5LFxuICBsaXRlcmFsVHlwZSBhcyBsaXRlcmFsLFxuICBtYXBUeXBlIGFzIG1hcCxcbiAgbmFuVHlwZSBhcyBuYW4sXG4gIG5hdGl2ZUVudW1UeXBlIGFzIG5hdGl2ZUVudW0sXG4gIG5ldmVyVHlwZSBhcyBuZXZlcixcbiAgbnVsbFR5cGUgYXMgbnVsbCxcbiAgbnVsbGFibGVUeXBlIGFzIG51bGxhYmxlLFxuICBudW1iZXJUeXBlIGFzIG51bWJlcixcbiAgb2JqZWN0VHlwZSBhcyBvYmplY3QsXG4gIG9ib29sZWFuLFxuICBvbnVtYmVyLFxuICBvcHRpb25hbFR5cGUgYXMgb3B0aW9uYWwsXG4gIG9zdHJpbmcsXG4gIHByZXByb2Nlc3NUeXBlIGFzIHByZXByb2Nlc3MsXG4gIHByb21pc2VUeXBlIGFzIHByb21pc2UsXG4gIHJlY29yZFR5cGUgYXMgcmVjb3JkLFxuICBzZXRUeXBlIGFzIHNldCxcbiAgc3RyaWN0T2JqZWN0VHlwZSBhcyBzdHJpY3RPYmplY3QsXG4gIHN0cmluZ1R5cGUgYXMgc3RyaW5nLFxuICBlZmZlY3RzVHlwZSBhcyB0cmFuc2Zvcm1lcixcbiAgdHVwbGVUeXBlIGFzIHR1cGxlLFxuICB1bmRlZmluZWRUeXBlIGFzIHVuZGVmaW5lZCxcbiAgdW5pb25UeXBlIGFzIHVuaW9uLFxuICB1bmtub3duVHlwZSBhcyB1bmtub3duLFxuICB2b2lkVHlwZSBhcyB2b2lkLFxufTtcbiJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxTQUFTLFNBQVMsUUFBUSx3QkFBd0IsQ0FBQztBQUNuRCxTQUNFLGlCQUFpQixFQUdqQixPQUFPLEVBQ1AsU0FBUyxFQUNULE9BQU8sRUFDUCxPQUFPLEVBQ1AsT0FBTyxFQUNQLFNBQVMsRUFDVCxFQUFFLEVBTUYsV0FBVyxRQUVOLHdCQUF3QixDQUFDO0FBR2hDLFNBQVMsYUFBYSxFQUFFLElBQUksRUFBRSxhQUFhLFFBQVEsbUJBQW1CLENBQUM7QUFDdkUsU0FDRSxlQUFlLEVBRWYsZ0JBQWdCLEVBR2hCLFFBQVEsRUFHUixZQUFZLFFBQ1AsZUFBZSxDQUFDO0FBMkJ2QixNQUFNLGtCQUFrQjtJQUN0QixNQUFNLENBQWU7SUFDckIsSUFBSSxDQUFNO0lBQ1YsS0FBSyxDQUFZO0lBQ2pCLElBQUksQ0FBd0M7SUFDNUMsWUFDRSxNQUFvQixFQUNwQixLQUFVLEVBQ1YsSUFBZSxFQUNmLEdBQTBDLENBQzFDO1FBQ0EsSUFBSSxDQUFDLE1BQU0sR0FBRyxNQUFNLENBQUM7UUFDckIsSUFBSSxDQUFDLElBQUksR0FBRyxLQUFLLENBQUM7UUFDbEIsSUFBSSxDQUFDLEtBQUssR0FBRyxJQUFJLENBQUM7UUFDbEIsSUFBSSxDQUFDLElBQUksR0FBRyxHQUFHLENBQUM7S0FDakI7SUFDRCxJQUFJLElBQUksR0FBRztRQUNULE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxNQUFNLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO0tBQ3JDO0NBQ0Y7QUFFRCxNQUFNLFlBQVksR0FBRyxDQUNuQixHQUFpQixFQUNqQixNQUFtQyxHQUdhO0lBQ2hELElBQUksT0FBTyxDQUFDLE1BQU0sQ0FBQyxFQUFFO1FBQ25CLE9BQU87WUFBRSxPQUFPLEVBQUUsSUFBSTtZQUFFLElBQUks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