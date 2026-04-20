"use client";

import { motion } from "framer-motion";
import type { ReactNode } from "react";

export function FeatureCard({
  icon,
  title,
  description,
  index,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  index: number;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-50px" }}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      className="group relative rounded-2xl border border-white/[0.08] bg-white/[0.02] p-8 hover:border-indigo-500/30 hover:bg-white/[0.04] transition-all duration-300"
    >
      <div className="mb-4 inline-flex items-center justify-center w-12 h-12 rounded-xl bg-indigo-500/10 text-indigo-400 group-hover:bg-indigo-500/20 transition-colors">
        {icon}
      </div>
      <h3 className="text-lg font-semibold text-white mb-2">{title}</h3>
      <p className="text-white/50 leading-relaxed text-sm">{description}</p>
    </motion.div>
  );
}
