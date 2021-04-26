export function verify(algorithm, jwtAlg) {
    return Array.isArray(algorithm)
        ? algorithm.includes(jwtAlg)
        : algorithm === jwtAlg;
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiYWxnb3JpdGhtLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiYWxnb3JpdGhtLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQWdCQSxNQUFNLFVBQVUsTUFBTSxDQUFDLFNBQXlCLEVBQUUsTUFBYztJQUM5RCxPQUFPLEtBQUssQ0FBQyxPQUFPLENBQUMsU0FBUyxDQUFDO1FBQzdCLENBQUMsQ0FBRSxTQUFzQixDQUFDLFFBQVEsQ0FBQyxNQUFNLENBQUM7UUFDMUMsQ0FBQyxDQUFDLFNBQVMsS0FBSyxNQUFNLENBQUM7QUFDM0IsQ0FBQyJ9