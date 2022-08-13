// Copyright © 2020-2022 Truestamp Inc. All rights reserved.
import { crypto } from "https://deno.land/std@0.145.0/crypto/mod.ts";
import { parse } from "https://deno.land/std@0.145.0/path/mod.ts";
import { copy, readAllSync } from "https://deno.land/std@0.145.0/streams/conversion.ts";
import { sleep } from "https://deno.land/x/sleep@v1.2.1/mod.ts";
import { decode, validate } from "https://deno.land/x/djwt@v2.7/mod.ts";
export { appPaths, colors, Command, CompletionsCommand, Conf, copy, createTruestampClient, crypto, DB, decode, decodeUnsafely, deleteTokensInConfig, EnumType, getAccessTokenWithPrompts, getConfigAccessToken, getConfigForEnv, getConfigIdTokenPayload, getConfigKeyForEnv, getConfigRefreshToken, HelpCommand, parse, readAllSync, setConfigKeyForEnv, sleep, Truestamp, validate, ValidationError, verify, Table };
import { DB } from "https://deno.land/x/sqlite@v3.4.0/mod.ts";
import { colors } from "https://deno.land/x/cliffy@v0.24.2/ansi/colors.ts";
import { CompletionsCommand } from "https://deno.land/x/cliffy@v0.24.2/command/completions/mod.ts";
import { HelpCommand } from "https://deno.land/x/cliffy@v0.24.2/command/help/mod.ts";
import { Command, EnumType, ValidationError } from "https://deno.land/x/cliffy@v0.24.2/command/mod.ts";
import Conf from "https://raw.githubusercontent.com/truestamp/deno-conf/v1.0.6/mod.ts";
import { appPaths } from "https://raw.githubusercontent.com/truestamp/deno-app-paths/v1.1.0/mod.ts";
import { getConfigForEnv, getConfigKeyForEnv, setConfigKeyForEnv } from "./config.ts";
import { deleteTokensInConfig, getAccessTokenWithPrompts, getConfigAccessToken, getConfigIdTokenPayload, getConfigRefreshToken } from "./auth.ts";
// See : https://www.skypack.dev/view/@truestamp/truestamp-js
// See SkyPack : https://docs.skypack.dev/skypack-cdn/api-reference/lookup-urls
import { decodeUnsafely } from "https://cdn.skypack.dev/@truestamp/id@~v1.3.1?dts";
import Truestamp from "https://cdn.skypack.dev/@truestamp/truestamp-js@~v0.13.1?dts";
import { verify } from "https://cdn.skypack.dev/@truestamp/verify@~v0.3.0?dts";
import { createTruestampClient } from "./truestamp.ts";
import { Table } from "https://deno.land/x/cliffy@v0.24.2/table/mod.ts";
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImZpbGU6Ly8vVXNlcnMvZ2xlbm4vc3JjL2dpdGh1Yi5jb20vdHJ1ZXN0YW1wL3RydWVzdGFtcC1jbGkvc3JjL2RlcHMudHMiXSwic291cmNlc0NvbnRlbnQiOlsiLy8gQ29weXJpZ2h0IMKpIDIwMjAtMjAyMiBUcnVlc3RhbXAgSW5jLiBBbGwgcmlnaHRzIHJlc2VydmVkLlxuXG5pbXBvcnQgeyBjcnlwdG8gfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQvc3RkQDAuMTQ1LjAvY3J5cHRvL21vZC50c1wiO1xuaW1wb3J0IHsgcGFyc2UgfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQvc3RkQDAuMTQ1LjAvcGF0aC9tb2QudHNcIjtcbmltcG9ydCB7XG4gIGNvcHksXG4gIHJlYWRBbGxTeW5jXG59IGZyb20gXCJodHRwczovL2Rlbm8ubGFuZC9zdGRAMC4xNDUuMC9zdHJlYW1zL2NvbnZlcnNpb24udHNcIjtcblxuaW1wb3J0IHsgc2xlZXAgfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQveC9zbGVlcEB2MS4yLjEvbW9kLnRzXCI7XG5cbmltcG9ydCB7IGRlY29kZSwgdmFsaWRhdGUgfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQveC9kand0QHYyLjcvbW9kLnRzXCI7XG5leHBvcnQgdHlwZSB7IElUeXBlSW5mbyB9IGZyb20gXCJodHRwczovL2Rlbm8ubGFuZC94L2NsaWZmeUB2MC4yNC4yL2ZsYWdzL21vZC50c1wiO1xuZXhwb3J0IHR5cGUgeyBQYXlsb2FkIH0gZnJvbSBcImh0dHBzOi8vZGVuby5sYW5kL3gvZGp3dEB2Mi43L21vZC50c1wiO1xuZXhwb3J0IHR5cGUgeyBSb3cgfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQveC9zcWxpdGVAdjMuNC4wL21vZC50c1wiO1xuZXhwb3J0IHR5cGUge1xuICBKc29uLFxuICBTdG9yZVR5cGVcbn0gZnJvbSBcImh0dHBzOi8vcmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbS90cnVlc3RhbXAvZGVuby1jb25mL3YxLjAuNi9tb2QudHNcIjtcbmV4cG9ydCB7XG4gIGFwcFBhdGhzLFxuICBjb2xvcnMsXG4gIENvbW1hbmQsXG4gIENvbXBsZXRpb25zQ29tbWFuZCxcbiAgQ29uZixcbiAgY29weSxcbiAgY3JlYXRlVHJ1ZXN0YW1wQ2xpZW50LFxuICBjcnlwdG8sXG4gIERCLFxuICBkZWNvZGUsXG4gIGRlY29kZVVuc2FmZWx5LFxuICBkZWxldGVUb2tlbnNJbkNvbmZpZyxcbiAgRW51bVR5cGUsXG4gIGdldEFjY2Vzc1Rva2VuV2l0aFByb21wdHMsXG4gIGdldENvbmZpZ0FjY2Vzc1Rva2VuLFxuICBnZXRDb25maWdGb3JFbnYsXG4gIGdldENvbmZpZ0lkVG9rZW5QYXlsb2FkLFxuICBnZXRDb25maWdLZXlGb3JFbnYsXG4gIGdldENvbmZpZ1JlZnJlc2hUb2tlbixcbiAgSGVscENvbW1hbmQsXG4gIHBhcnNlLFxuICByZWFkQWxsU3luYyxcbiAgc2V0Q29uZmlnS2V5Rm9yRW52LFxuICBzbGVlcCxcbiAgVHJ1ZXN0YW1wLFxuICB2YWxpZGF0ZSxcbiAgVmFsaWRhdGlvbkVycm9yLFxuICB2ZXJpZnksXG4gIFRhYmxlLFxufTtcblxuXG5pbXBvcnQgeyBEQiB9IGZyb20gXCJodHRwczovL2Rlbm8ubGFuZC94L3NxbGl0ZUB2My40LjAvbW9kLnRzXCI7XG5cbmltcG9ydCB7IGNvbG9ycyB9IGZyb20gXCJodHRwczovL2Rlbm8ubGFuZC94L2NsaWZmeUB2MC4yNC4yL2Fuc2kvY29sb3JzLnRzXCI7XG5pbXBvcnQgeyBDb21wbGV0aW9uc0NvbW1hbmQgfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQveC9jbGlmZnlAdjAuMjQuMi9jb21tYW5kL2NvbXBsZXRpb25zL21vZC50c1wiO1xuaW1wb3J0IHsgSGVscENvbW1hbmQgfSBmcm9tIFwiaHR0cHM6Ly9kZW5vLmxhbmQveC9jbGlmZnlAdjAuMjQuMi9jb21tYW5kL2hlbHAvbW9kLnRzXCI7XG5pbXBvcnQge1xuICBDb21tYW5kLFxuICBFbnVtVHlwZSxcbiAgVmFsaWRhdGlvbkVycm9yXG59IGZyb20gXCJodHRwczovL2Rlbm8ubGFuZC94L2NsaWZmeUB2MC4yNC4yL2NvbW1hbmQvbW9kLnRzXCI7XG5cbmltcG9ydCBDb25mIGZyb20gXCJodHRwczovL3Jhdy5naXRodWJ1c2VyY29udGVudC5jb20vdHJ1ZXN0YW1wL2Rlbm8tY29uZi92MS4wLjYvbW9kLnRzXCI7XG5cbmltcG9ydCB7IGFwcFBhdGhzIH0gZnJvbSBcImh0dHBzOi8vcmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbS90cnVlc3RhbXAvZGVuby1hcHAtcGF0aHMvdjEuMS4wL21vZC50c1wiO1xuXG5pbXBvcnQge1xuICBnZXRDb25maWdGb3JFbnYsXG4gIGdldENvbmZpZ0tleUZvckVudixcbiAgc2V0Q29uZmlnS2V5Rm9yRW52XG59IGZyb20gXCIuL2NvbmZpZy50c1wiO1xuXG5pbXBvcnQge1xuICBkZWxldGVUb2tlbnNJbkNvbmZpZyxcbiAgZ2V0QWNjZXNzVG9rZW5XaXRoUHJvbXB0cyxcbiAgZ2V0Q29uZmlnQWNjZXNzVG9rZW4sXG4gIGdldENvbmZpZ0lkVG9rZW5QYXlsb2FkLFxuICBnZXRDb25maWdSZWZyZXNoVG9rZW5cbn0gZnJvbSBcIi4vYXV0aC50c1wiO1xuXG4vLyBTZWUgOiBodHRwczovL3d3dy5za3lwYWNrLmRldi92aWV3L0B0cnVlc3RhbXAvdHJ1ZXN0YW1wLWpzXG4vLyBTZWUgU2t5UGFjayA6IGh0dHBzOi8vZG9jcy5za3lwYWNrLmRldi9za3lwYWNrLWNkbi9hcGktcmVmZXJlbmNlL2xvb2t1cC11cmxzXG5pbXBvcnQgeyBkZWNvZGVVbnNhZmVseSB9IGZyb20gXCJodHRwczovL2Nkbi5za3lwYWNrLmRldi9AdHJ1ZXN0YW1wL2lkQH52MS4zLjE/ZHRzXCI7XG5pbXBvcnQgVHJ1ZXN0YW1wIGZyb20gXCJodHRwczovL2Nkbi5za3lwYWNrLmRldi9AdHJ1ZXN0YW1wL3RydWVzdGFtcC1qc0B+djAuMTMuMT9kdHNcIjtcbmltcG9ydCB7IHZlcmlmeSB9IGZyb20gXCJodHRwczovL2Nkbi5za3lwYWNrLmRldi9AdHJ1ZXN0YW1wL3ZlcmlmeUB+djAuMy4wP2R0c1wiO1xuaW1wb3J0IHsgY3JlYXRlVHJ1ZXN0YW1wQ2xpZW50IH0gZnJvbSBcIi4vdHJ1ZXN0YW1wLnRzXCI7XG5cbmltcG9ydCB7IFRhYmxlIH0gZnJvbSBcImh0dHBzOi8vZGVuby5sYW5kL3gvY2xpZmZ5QHYwLjI0LjIvdGFibGUvbW9kLnRzXCI7XG4iXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsNERBQTREO0FBRTVELFNBQVMsTUFBTSxRQUFRLDZDQUE2QyxDQUFDO0FBQ3JFLFNBQVMsS0FBSyxRQUFRLDJDQUEyQyxDQUFDO0FBQ2xFLFNBQ0UsSUFBSSxFQUNKLFdBQVcsUUFDTixxREFBcUQsQ0FBQztBQUU3RCxTQUFTLEtBQUssUUFBUSx5Q0FBeUMsQ0FBQztBQUVoRSxTQUFTLE1BQU0sRUFBRSxRQUFRLFFBQVEsc0NBQXNDLENBQUM7QUFReEUsU0FDRSxRQUFRLEVBQ1IsTUFBTSxFQUNOLE9BQU8sRUFDUCxrQkFBa0IsRUFDbEIsSUFBSSxFQUNKLElBQUksRUFDSixxQkFBcUIsRUFDckIsTUFBTSxFQUNOLEVBQUUsRUFDRixNQUFNLEVBQ04sY0FBYyxFQUNkLG9CQUFvQixFQUNwQixRQUFRLEVBQ1IseUJBQXlCLEVBQ3pCLG9CQUFvQixFQUNwQixlQUFlLEVBQ2YsdUJBQXVCLEVBQ3ZCLGtCQUFrQixFQUNsQixxQkFBcUIsRUFDckIsV0FBVyxFQUNYLEtBQUssRUFDTCxXQUFXLEVBQ1gsa0JBQWtCLEVBQ2xCLEtBQUssRUFDTCxTQUFTLEVBQ1QsUUFBUSxFQUNSLGVBQWUsRUFDZixNQUFNLEVBQ04sS0FBSyxHQUNMO0FBR0YsU0FBUyxFQUFFLFFBQVEsMENBQTBDLENBQUM7QUFFOUQsU0FBUyxNQUFNLFFBQVEsbURBQW1ELENBQUM7QUFDM0UsU0FBUyxrQkFBa0IsUUFBUSwrREFBK0QsQ0FBQztBQUNuRyxTQUFTLFdBQVcsUUFBUSx3REFBd0QsQ0FBQztBQUNyRixTQUNFLE9BQU8sRUFDUCxRQUFRLEVBQ1IsZUFBZSxRQUNWLG1EQUFtRCxDQUFDO0FBRTNELE9BQU8sSUFBSSxNQUFNLHFFQUFxRSxDQUFDO0FBRXZGLFNBQVMsUUFBUSxRQUFRLDBFQUEwRSxDQUFDO0FBRXBHLFNBQ0UsZUFBZSxFQUNmLGtCQUFrQixFQUNsQixrQkFBa0IsUUFDYixhQUFhLENBQUM7QUFFckIsU0FDRSxvQkFBb0IsRUFDcEIseUJBQXlCLEVBQ3pCLG9CQUFvQixFQUNwQix1QkFBdUIsRUFDdkIscUJBQXFCLFFBQ2hCLFdBQVcsQ0FBQztBQUVuQiw2REFBNkQ7QUFDN0QsK0VBQStFO0FBQy9FLFNBQVMsY0FBYyxRQUFRLG1EQUFtRCxDQUFDO0FBQ25GLE9BQU8sU0FBUyxNQUFNLDhEQUE4RCxDQUFDO0FBQ3JGLFNBQVMsTUFBTSxRQUFRLHVEQUF1RCxDQUFDO0FBQy9FLFNBQVMscUJBQXFCLFFBQVEsZ0JBQWdCLENBQUM7QUFFdkQsU0FBUyxLQUFLLFFBQVEsaURBQWlELENBQUMifQ==