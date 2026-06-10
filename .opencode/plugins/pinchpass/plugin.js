import { tool } from "@opencode-ai/plugin/tool";
import { execSync } from "child_process";
export default async () => {
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
                    const cmd = ["pinchpass", "request", ...args.names, "-json"];
                    if (args.tunnel)
                        cmd.push("-tunnel");
                    if (args.note)
                        cmd.push("-note", args.note);
                    if (args.ttl)
                        cmd.push("-ttl", String(args.ttl));
                    try {
                        const stdout = execSync(cmd.join(" "), {
                            encoding: "utf-8",
                            timeout: (args.ttl ?? 30) * 60 * 1000,
                        });
                        return stdout;
                    }
                    catch (err) {
                        const msg = err instanceof Error ? err.message : String(err);
                        return `Failed to generate secret request: ${msg}`;
                    }
                },
            }),
        },
    };
};
