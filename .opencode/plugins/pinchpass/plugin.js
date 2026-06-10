import { tool } from "@opencode-ai/plugin/tool";
export default async ({ $ }) => {
    return {
        tool: {
            request_secret: tool({
                description: "Generate a one-time E2E-encrypted secret request link. " +
                    "Use when you need to collect sensitive values (API keys, tokens, passwords) from the user. " +
                    "The tool starts a local server, prints a link, and blocks until the user submits or the TTL expires. " +
                    "On success the secret is saved to .env and the tool returns the result.",
                args: {
                    names: tool.schema
                        .array(tool.schema.string())
                        .min(1)
                        .describe("Secret name(s) to collect, e.g. ['GEMINI_API_KEY']"),
                    tunnel: tool.schema
                        .boolean()
                        .optional()
                        .describe("Create a public URL via bore.pub tunnel (default: false)"),
                    note: tool.schema
                        .string()
                        .optional()
                        .describe("Description shown on the form to help the user"),
                    ttl: tool.schema
                        .number()
                        .optional()
                        .describe("Minutes until the link expires (default: 30)"),
                },
                async execute(args) {
                    const pieces = ["pinchpass", "request", ...args.names, "-json"];
                    if (args.tunnel)
                        pieces.push("-tunnel");
                    if (args.note)
                        pieces.push("-note", args.note);
                    if (args.ttl)
                        pieces.push("-ttl", String(args.ttl));
                    try {
                        const result = await $.raw(pieces);
                        return result.stdout || result.text || String(result);
                    }
                    catch (err) {
                        const msg = err instanceof Error ? err.message : String(err);
                        const stderr = err && typeof err === "object" && "stderr" in err
                            ? err.stderr
                            : "";
                        return `Failed: ${msg}${stderr ? "\n" + stderr : ""}`;
                    }
                },
            }),
        },
    };
};
