"use client";

import { motion } from "framer-motion";

const withoutVibeCloud = [
  "Figure out which CLIs to install",
  "Authenticate with each provider separately",
  "Configure Vercel project settings",
  "Set up Supabase and run migrations",
  "Wire environment variables across services",
  "Debug deployment errors in each platform",
  "Coordinate deploy order manually",
  "Hope nothing broke",
];

export function Comparison() {
  return (
    <div className="grid md:grid-cols-2 gap-6 max-w-4xl mx-auto">
      <motion.div
        initial={{ opacity: 0, x: -20 }}
        whileInView={{ opacity: 1, x: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5 }}
        className="rounded-2xl border border-red-500/20 bg-red-500/[0.03] p-8"
      >
        <div className="text-red-400 font-semibold text-sm uppercase tracking-wider mb-1">
          Without VibeCloud
        </div>
        <div className="text-white/30 text-xs mb-6">~2-4 hours of yak shaving</div>
        <ul className="space-y-3">
          {withoutVibeCloud.map((item, i) => (
            <li
              key={i}
              className="flex items-start gap-3 text-sm text-white/40"
            >
              <span className="text-red-500/60 mt-0.5 flex-shrink-0">
                &times;
              </span>
              {item}
            </li>
          ))}
        </ul>
      </motion.div>

      <motion.div
        initial={{ opacity: 0, x: 20 }}
        whileInView={{ opacity: 1, x: 0 }}
        viewport={{ once: true }}
        transition={{ duration: 0.5, delay: 0.1 }}
        className="rounded-2xl border border-green-500/20 bg-green-500/[0.03] p-8"
      >
        <div className="text-green-400 font-semibold text-sm uppercase tracking-wider mb-1">
          With VibeCloud
        </div>
        <div className="text-white/30 text-xs mb-6">~2 minutes</div>
        <ul className="space-y-4">
          <li className="flex items-start gap-3 text-sm">
            <span className="text-green-500/80 mt-0.5 flex-shrink-0">1.</span>
            <div>
              <div className="text-white/70 mb-1">Install the CLI</div>
              <code className="text-green-400 font-mono text-xs">curl -fsSL https://vibecloudai.com/install | sh</code>
            </div>
          </li>
          <li className="flex items-start gap-3 text-sm">
            <span className="text-green-500/80 mt-0.5 flex-shrink-0">2.</span>
            <div>
              <div className="text-white/70 mb-1">Tell Claude to deploy</div>
              <span className="text-indigo-400 text-xs">&quot;Use vibecloud CLI to deploy my app to production&quot;</span>
            </div>
          </li>
        </ul>
        <div className="mt-8 pt-6 border-t border-green-500/10">
          <div className="text-white/30 text-xs">Claude handles the rest. You&apos;re live.</div>
        </div>
      </motion.div>
    </div>
  );
}
