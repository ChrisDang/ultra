"use client";

import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";

const words = [
  "database migrations",
  "frontend deployments",
  "environment variables",
  "DNS configuration",
  "monorepo builds",
  "container orchestration",
  "CI/CD pipelines",
  "production debugging",
  "log drains",
  "YAML files",
  "build configs",
  "deploy rollbacks",
  "port forwarding",
  "dependency hell",
  "Docker volumes",
  "staging vs production",
  "cold start optimization",
  "edge function routing",
  "serverless functions",
  "mobile builds",
];

export function RotatingWords() {
  const [index, setIndex] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setIndex((i) => (i + 1) % words.length);
    }, 2000);
    return () => clearInterval(interval);
  }, []);

  return (
    <span>
      <span className="text-white/50">Claude + VibeCloud handle </span>
      <span className="relative inline-block border-b-2 border-indigo-500/40 text-center align-baseline">
        {/* Invisible sizer — longest phrase sets the width */}
        <span className="invisible font-semibold whitespace-nowrap">
          {words.reduce((a, b) => (a.length >= b.length ? a : b))}
        </span>
        <AnimatePresence mode="wait">
          <motion.span
            key={words[index]}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.25 }}
            className="text-gradient font-semibold whitespace-nowrap absolute inset-0 flex items-center justify-center"
          >
            {words[index]}
          </motion.span>
        </AnimatePresence>
      </span>
    </span>
  );
}
