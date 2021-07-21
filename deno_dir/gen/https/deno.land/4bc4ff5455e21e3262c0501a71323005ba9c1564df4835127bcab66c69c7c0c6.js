import Transform from "./transform.ts";
export default class PassThrough extends Transform {
    constructor(options) {
        super(options);
    }
    _transform(chunk, _encoding, cb) {
        cb(null, chunk);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoicGFzc3Rocm91Z2guanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJwYXNzdGhyb3VnaC50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFDQSxPQUFPLFNBQVMsTUFBTSxnQkFBZ0IsQ0FBQztBQUl2QyxNQUFNLENBQUMsT0FBTyxPQUFPLFdBQVksU0FBUSxTQUFTO0lBQ2hELFlBQVksT0FBMEI7UUFDcEMsS0FBSyxDQUFDLE9BQU8sQ0FBQyxDQUFDO0lBQ2pCLENBQUM7SUFFRCxVQUFVLENBRVIsS0FBVSxFQUNWLFNBQW9CLEVBRXBCLEVBQThDO1FBRTlDLEVBQUUsQ0FBQyxJQUFJLEVBQUUsS0FBSyxDQUFDLENBQUM7SUFDbEIsQ0FBQztDQUNGIn0=