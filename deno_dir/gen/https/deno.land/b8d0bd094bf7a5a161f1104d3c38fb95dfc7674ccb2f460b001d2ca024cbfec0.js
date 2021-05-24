export function resolveUserAgentConfig(input) {
    return {
        ...input,
        customUserAgent: typeof input.customUserAgent === "string" ? [[input.customUserAgent]] : input.customUserAgent,
    };
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiY29uZmlndXJhdGlvbnMuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyJjb25maWd1cmF0aW9ucy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUEwQkEsTUFBTSxVQUFVLHNCQUFzQixDQUNwQyxLQUFvRDtJQUVwRCxPQUFPO1FBQ0wsR0FBRyxLQUFLO1FBQ1IsZUFBZSxFQUFFLE9BQU8sS0FBSyxDQUFDLGVBQWUsS0FBSyxRQUFRLENBQUMsQ0FBQyxDQUFDLENBQUMsQ0FBQyxLQUFLLENBQUMsZUFBZSxDQUFDLENBQUMsQ0FBQyxDQUFDLENBQUMsS0FBSyxDQUFDLGVBQWU7S0FDL0csQ0FBQztBQUNKLENBQUMifQ==