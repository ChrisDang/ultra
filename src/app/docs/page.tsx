import { Terminal, Download, Box, CheckCircle } from "lucide-react";
import { CopyButton } from "@/components/copy-button";

const INSTALL_CMD = "curl -fsSL https://vibecloudai.com/install | sh";
const PIN_CMD = "VIBECLOUD_VERSION=v0.1.0 curl -fsSL https://vibecloudai.com/install | sh";

const commands = [
  {
    command: "init",
    description: "Initialize a VibeCloud project in the current directory",
  },
  {
    command: "login",
    description: "Authenticate with provider CLIs (Vercel, Supabase, Expo)",
  },
  {
    command: "doctor",
    description:
      "Preflight check — verify CLIs, auth, and project linkage",
  },
  {
    command: "deploy",
    description: "Deploy the current project",
  },
  {
    command: "status",
    description: "Show project status from each provider",
  },
  {
    command: "logs",
    description: "Fetch logs for the current project",
  },
  {
    command: "explain",
    description:
      "Show full project state across all providers in one view",
  },
];

const platforms = [
  { os: "macOS", arch: "Intel (amd64), Apple Silicon (arm64)" },
  { os: "Linux", arch: "amd64, arm64" },
];

export default function DocsPage() {
  return (
    <main className="pt-24 pb-32 px-6">
      <div className="max-w-3xl mx-auto">
        {/* Header */}
        <div className="mb-16">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-indigo-500/20 bg-indigo-500/[0.06] text-indigo-400 text-xs font-medium mb-6">
            <Download className="w-3 h-3" />
            Installation guide
          </div>
          <h1 className="text-4xl sm:text-5xl font-bold tracking-tight mb-4">
            Install VibeCloud CLI
          </h1>
          <p className="text-white/40 text-lg">
            Get up and running in under a minute.
          </p>
        </div>

        {/* Quick install */}
        <section className="mb-16">
          <h2 className="text-xl font-semibold mb-2 flex items-center gap-2">
            <Terminal className="w-5 h-5 text-indigo-400" />
            Quick install (macOS / Linux)
          </h2>
          <p className="text-white/40 text-sm mb-4">
            This auto-detects your OS and architecture, downloads the latest
            release, and installs the{" "}
            <code className="text-indigo-400 bg-indigo-500/10 px-1.5 py-0.5 rounded text-xs">
              vibecloud
            </code>{" "}
            binary to{" "}
            <code className="text-white/60 bg-white/[0.06] px-1.5 py-0.5 rounded text-xs">
              /usr/local/bin
            </code>
            .
          </p>
          <div className="rounded-xl border border-white/[0.08] bg-black/40 p-4 font-mono text-sm text-green-400 flex items-center justify-between gap-3">
            <code className="overflow-x-auto">{INSTALL_CMD}</code>
            <CopyButton text={INSTALL_CMD} />
          </div>
        </section>

        {/* Pin version */}
        <section className="mb-16">
          <h2 className="text-xl font-semibold mb-2">Pin a specific version</h2>
          <div className="rounded-xl border border-white/[0.08] bg-black/40 p-4 font-mono text-sm text-green-400 flex items-center justify-between gap-3">
            <code className="overflow-x-auto">{PIN_CMD}</code>
            <CopyButton text={PIN_CMD} />
          </div>
        </section>

        {/* Upgrade */}
        <section className="mb-16">
          <h2 className="text-xl font-semibold mb-2">Upgrade</h2>
          <p className="text-white/40 text-sm">
            Re-run the install command — it always fetches the latest release.
          </p>
        </section>

        {/* Verify */}
        <section className="mb-16">
          <h2 className="text-xl font-semibold mb-2 flex items-center gap-2">
            <CheckCircle className="w-5 h-5 text-green-400" />
            Verify installation
          </h2>
          <div className="rounded-xl border border-white/[0.08] bg-black/40 p-4 font-mono text-sm text-green-400 flex items-center justify-between gap-3">
            <code>vibecloud version</code>
            <CopyButton text="vibecloud version" />
          </div>
        </section>

        {/* Supported platforms */}
        <section className="mb-16">
          <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
            <Box className="w-5 h-5 text-indigo-400" />
            Supported platforms
          </h2>
          <div className="rounded-xl border border-white/[0.08] overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-white/[0.08] bg-white/[0.02]">
                  <th className="text-left px-6 py-3 text-white/50 font-medium">
                    OS
                  </th>
                  <th className="text-left px-6 py-3 text-white/50 font-medium">
                    Architecture
                  </th>
                </tr>
              </thead>
              <tbody>
                {platforms.map((p, i) => (
                  <tr
                    key={i}
                    className="border-b border-white/[0.04] last:border-0"
                  >
                    <td className="px-6 py-3 text-white/70 font-mono">
                      {p.os}
                    </td>
                    <td className="px-6 py-3 text-white/50">{p.arch}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        {/* Get started */}
        <section className="mb-16">
          <h2 className="text-xl font-semibold mb-4">Get started</h2>
          <div className="space-y-3">
            {[
              {
                cmd: "vibecloud doctor",
                desc: "Check that provider CLIs are installed and authenticated",
              },
              {
                cmd: "vibecloud init",
                desc: "Initialize a project in the current directory",
              },
              {
                cmd: "vibecloud deploy",
                desc: "Deploy across all detected providers",
              },
            ].map((item, i) => (
              <div
                key={i}
                className="flex items-center gap-4 rounded-xl border border-white/[0.06] bg-white/[0.02] p-4"
              >
                <code className="text-green-400 font-mono text-sm whitespace-nowrap flex-shrink-0">
                  $ {item.cmd}
                </code>
                <span className="text-white/40 text-sm flex-1">{item.desc}</span>
                <CopyButton text={item.cmd} />
              </div>
            ))}
          </div>
        </section>

        {/* All commands */}
        <section>
          <h2 className="text-xl font-semibold mb-4">Available commands</h2>
          <div className="rounded-xl border border-white/[0.08] overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-white/[0.08] bg-white/[0.02]">
                  <th className="text-left px-6 py-3 text-white/50 font-medium">
                    Command
                  </th>
                  <th className="text-left px-6 py-3 text-white/50 font-medium">
                    Description
                  </th>
                </tr>
              </thead>
              <tbody>
                {commands.map((c, i) => (
                  <tr
                    key={i}
                    className="border-b border-white/[0.04] last:border-0"
                  >
                    <td className="px-6 py-3 font-mono text-indigo-400">
                      {c.command}
                    </td>
                    <td className="px-6 py-3 text-white/50">
                      {c.description}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </main>
  );
}
