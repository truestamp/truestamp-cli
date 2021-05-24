import { HttpRequest, QueryParameterBag } from "../types/mod.ts";

/**
 * @internal
 */
export function cloneRequest({ headers, query, ...rest }: HttpRequest): HttpRequest {
  return {
    ...rest,
    headers: { ...headers },
    query: query ? cloneQuery(query) : undefined,
  };
}

function cloneQuery(query: QueryParameterBag): QueryParameterBag {
  return Object.keys(query).reduce((carry: QueryParameterBag, paramName: string) => {
    const param = query[paramName];
    return {
      ...carry,
      [paramName]: Array.isArray(param) ? [...param] : param,
    };
  }, {});
}
