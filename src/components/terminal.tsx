"use client";

import { motion } from "framer-motion";
import { useEffect, useState } from "react";

const lines = [
  { text: "> Use vibecloud CLI to deploy my app to production", type: "user" as const, delay: 0 },
  { text: "Running vibecloud doctor...", type: "claude" as const, delay: 800 },
  { text: "Running vibecloud init...", type: "claude" as const, delay: 1400 },
  {
    text: "Detected: Next.js + Supabase. Linking providers...",
    type: "output" as const,
    delay: 2000,
  },
  { text: "Running vibecloud deploy --prod...", type: "claude" as const, delay: 2800 },
  {
    text: '{"success":true,"message":"Deployed to production","data":{"url":"https://myapp.vercel.app"}}',
    type: "output" as const,
    delay: 3600,
  },
  {
    text: "Your app is live at https://myapp.vercel.app",
    type: "claude" as const,
    delay: 4400,
  },
];

export function Terminal() {
  const [visibleLines, setVisibleLines] = useState(0);

  useEffect(() => {
    const timers = lines.map((line, i) =>
      setTimeout(() => setVisibleLines(i + 1), line.delay + 500)
    );
    return () => timers.forEach(clearTimeout);
  }, []);

  return (
    <div className="w-full max-w-2xl mx-auto rounded-xl border border-white/10 bg-black/60 backdrop-blur-sm shadow-2xl overflow-hidden">
      {/* Title bar */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-white/10 bg-white/[0.02]">
        <div className="w-3 h-3 rounded-full bg-red-500/80" />
        <div className="w-3 h-3 rounded-full bg-yellow-500/80" />
        <div className="w-3 h-3 rounded-full bg-green-500/80" />
        <span className="ml-2 text-xs text-white/40 font-mono">
          Claude Code — ~/my-app
        </span>
      </div>

      {/* Terminal body */}
      <div className="p-5 font-mono text-sm leading-relaxed min-h-[220px]">
        {lines.slice(0, visibleLines).map((line, i) => (
          <motion.div
            key={i}
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
            className={
              line.type === "user"
                ? "text-white font-medium mb-1"
                : line.type === "claude"
                  ? "text-indigo-400 text-xs mb-1"
                  : "text-white/40 text-xs break-all mb-1"
            }
          >
            {line.text}
          </motion.div>
        ))}
        {visibleLines < lines.length && (
          <span className="text-indigo-400 cursor-blink">_</span>
        )}
      </div>
    </div>
  );
}
