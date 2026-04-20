import { Comparison } from "@/components/comparison";
import { CopyButton } from "@/components/copy-button";
import { FeatureCard } from "@/components/feature-card";
import { RotatingWords } from "@/components/rotating-words";
import { Terminal } from "@/components/terminal";
import {
  ArrowRight,
  Cpu,
  Download,
  Layers,
  MessageSquare,
  Rocket,
  Shield,
  Terminal as TerminalIcon,
  Zap,
} from "lucide-react";

const features = [
  {
    icon: <Zap className="w-5 h-5" />,
    title: "Auto-detect your stack",
    description:
      "VibeCloud works with any stack — Next.js, React, Expo, Python, Go, static sites, and more. Just point it at your project and go.",
  },
  {
    icon: <Layers className="w-5 h-5" />,
    title: "One command deploys everything",
    description:
      "Supabase migrations, Vercel frontend, Expo builds — all deployed in the right order. VibeCloud enables Claude to handle it all for you.",
  },
  {
    icon: <Cpu className="w-5 h-5" />,
    title: "Built for AI agents",
    description:
      "Seamless and intelligent. The only deployment CLI designed from day one to work with Claude Code.",
  },
  {
    icon: <TerminalIcon className="w-5 h-5" />,
    title: "Claude does the work",
    description:
      "You don't run vibecloud commands yourself. Tell Claude to deploy and it handles init, auth, linking, and deployment automatically.",
  },
  {
    icon: <Shield className="w-5 h-5" />,
    title: "Preflight checks built in",
    description:
      "Claude runs vibecloud doctor before deploying — checking CLIs, auth, and project linkage. Issues get fixed before they become errors.",
  },
  {
    icon: <Rocket className="w-5 h-5" />,
    title: "Zero config required",
    description:
      "No YAML files, no build configs, no environment variable wiring. VibeCloud figures it out from your project structure.",
  },
];

export default function Home() {
  return (
    <main className="relative">
      {/* ── Hero ── */}
      <section className="relative overflow-hidden bg-grid">
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[600px] bg-indigo-500/[0.07] rounded-full blur-[120px] pointer-events-none" />

        <div className="relative max-w-5xl mx-auto px-6 pt-32 pb-24 text-center">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full border border-indigo-500/20 bg-indigo-500/[0.06] text-indigo-400 text-xs font-medium mb-8">
            <span className="w-1.5 h-1.5 rounded-full bg-indigo-400 animate-pulse" />
            Deploy in minutes, not hours
          </div>

          <h1 className="text-5xl sm:text-6xl lg:text-7xl font-bold tracking-tight leading-[1.1] mb-6">
            You built the app.
            <br />
            <span className="text-gradient">Now ship it.</span>
          </h1>

          <p className="text-lg sm:text-xl max-w-2xl mx-auto mb-4 leading-relaxed text-white/50">
            Install the CLI, tell Claude to deploy. That&apos;s it.
          </p>
          <div className="text-lg sm:text-xl max-w-2xl mx-auto mb-12">
            <RotatingWords />
          </div>

          <div className="flex items-center justify-center mb-20">
            <a
              href="/docs"
              className="inline-flex items-center gap-2 px-8 py-3.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition-colors text-sm"
            >
              Get started
              <ArrowRight className="w-4 h-4" />
            </a>
          </div>

          <Terminal />
        </div>
      </section>

      {/* ── Problem / Solution ── */}
      <section className="py-28 px-6">
        <div className="max-w-5xl mx-auto text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Deployments shouldn&apos;t be the hard part
          </h2>
          <p className="text-white/40 max-w-xl mx-auto text-lg">
            You used Claude to build an entire app in an afternoon. Then you
            spent the rest of the day trying to deploy it.
          </p>
        </div>

        <Comparison />
      </section>

      {/* ── How it works (2 steps) ── */}
      <section className="py-28 px-6 bg-grid">
        <div className="max-w-4xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              Two steps. You&apos;re live.
            </h2>
            <p className="text-white/40 max-w-lg mx-auto">
              Install the CLI, then let Claude handle the rest.
            </p>
          </div>

          <div className="grid md:grid-cols-2 gap-6">
            {/* Step 1 */}
            <div className="rounded-2xl border border-white/[0.08] bg-white/[0.02] p-8">
              <div className="flex items-center gap-3 mb-6">
                <div className="w-10 h-10 rounded-full bg-indigo-500/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400 font-mono text-sm font-bold">
                  1
                </div>
                <h3 className="text-lg font-semibold text-white">Install the CLI</h3>
              </div>
              <p className="text-white/40 text-sm mb-5">
                One command. Takes 10 seconds. It also auto-installs any provider CLIs you&apos;re missing.
              </p>
              <div className="rounded-lg bg-black/40 border border-white/[0.06] px-4 py-3 font-mono text-sm text-green-400 flex items-center gap-3">
                <Download className="w-4 h-4 text-white/20 flex-shrink-0" />
                <span className="overflow-x-auto flex-1">curl -fsSL https://vibecloudai.com/install | sh</span>
                <CopyButton text="curl -fsSL https://vibecloudai.com/install | sh" />
              </div>
            </div>

            {/* Step 2 */}
            <div className="rounded-2xl border border-indigo-500/20 bg-indigo-500/[0.03] p-8">
              <div className="flex items-center gap-3 mb-6">
                <div className="w-10 h-10 rounded-full bg-indigo-500/20 border border-indigo-500/30 flex items-center justify-center text-indigo-400 font-mono text-sm font-bold">
                  2
                </div>
                <h3 className="text-lg font-semibold text-white">Tell Claude to deploy</h3>
              </div>
              <p className="text-white/40 text-sm mb-5">
                That&apos;s it. Claude runs vibecloud commands, reads the structured output, and handles everything — init, auth, linking, deployment.
              </p>
              <div className="rounded-lg bg-black/40 border border-indigo-500/10 px-4 py-3 font-mono text-sm text-indigo-300 flex items-center gap-3">
                <MessageSquare className="w-4 h-4 text-white/20 flex-shrink-0" />
                <span className="flex-1">Use vibecloud CLI to deploy my app to production</span>
                <CopyButton text="Use vibecloud CLI to deploy my app to production" />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* ── Features ── */}
      <section className="py-28 px-6">
        <div className="max-w-5xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              Everything you need to go live
            </h2>
            <p className="text-white/40 max-w-lg mx-auto">
              VibeCloud handles the boring parts so you can focus on building.
            </p>
          </div>

          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
            {features.map((f, i) => (
              <FeatureCard key={i} index={i} {...f} />
            ))}
          </div>
        </div>
      </section>

      {/* ── AI-native callout ── */}
      <section className="py-28 px-6 bg-grid">
        <div className="max-w-3xl mx-auto text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-indigo-500/10 mb-8">
            <Cpu className="w-8 h-8 text-indigo-400" />
          </div>
          <h2 className="text-3xl sm:text-4xl font-bold mb-6">
            Designed for AI-first workflows
          </h2>
          <p className="text-white/50 text-lg leading-relaxed max-w-xl mx-auto">
            VibeCloud is built from the ground up to work with Claude Code.
            Every command is optimized for AI comprehension — Claude understands
            what happened, what went wrong, and what to do next. You just say
            &quot;deploy&quot; and the rest is handled.
          </p>
        </div>
      </section>

      {/* ── Final CTA ── */}
      <section className="py-32 px-6">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-4xl sm:text-5xl font-bold mb-6">
            Stop configuring.
            <br />
            <span className="text-gradient">Start shipping.</span>
          </h2>
          <p className="text-white/40 text-lg mb-10 max-w-md mx-auto">
            Install the CLI. Tell Claude to deploy. You&apos;re live.
          </p>
          <div className="flex items-center justify-center">
            <a
              href="/docs"
              className="inline-flex items-center gap-2 px-10 py-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition-colors"
            >
              Get started
              <ArrowRight className="w-4 h-4" />
            </a>
          </div>
          <p className="mt-6 text-white/20 text-sm font-mono">
            curl -fsSL https://vibecloudai.com/install | sh
          </p>
        </div>
      </section>

      {/* ── Footer ── */}
      <footer className="border-t border-white/[0.06] py-8 px-6">
        <div className="max-w-5xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-white/30">
          <div className="font-medium text-white/50">VibeCloud</div>
          <span>Built for vibe coders</span>
        </div>
      </footer>
    </main>
  );
}
