"use client";

import React from "react";
import Image from "next/image";
import Link from "next/link";
import { Play, Info } from "lucide-react";
import { motion } from "framer-motion";

interface HeroProps {
  content: any;
}

export const Hero = ({ content }: HeroProps) => {
  if (!content) return null;

  return (
    <section className="relative w-full h-[85vh] flex items-center overflow-hidden">
      {/* Background Image with Gradient Overlay */}
      <div className="absolute inset-0">
        <Image
          src={content.assets?.thumbnail || "/icon.png"}
          alt={content.title}
          fill
          className="object-cover object-top"
          priority
        />
        <div className="absolute inset-0 bg-gradient-to-r from-black via-black/60 to-transparent" />
        <div className="absolute inset-x-0 bottom-0 h-40 bg-gradient-to-t from-black to-transparent" />
      </div>

      {/* Content */}
      <div className="relative z-10 px-4 md:px-12 max-w-2xl">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8 }}
        >
          <span className="inline-block px-3 py-1 rounded-full bg-primary/20 text-primary text-xs font-bold tracking-widest uppercase mb-4 border border-primary/30">
            Featured Content
          </span>
          <h1 className="text-5xl md:text-7xl font-bold mb-4 tracking-tight">
            {content.title}
          </h1>
          <p className="text-lg text-foreground/80 mb-8 line-clamp-3 md:line-clamp-none max-w-lg">
            {content.description}
          </p>

          <div className="flex flex-wrap gap-4">
            <Link href={`/videos/watch/${content.short_id}`}>
              <button className="flex items-center gap-2 px-8 py-3 bg-primary hover:bg-primary/90 text-white rounded-full font-bold transition-all transform hover:scale-105 active:scale-95 shadow-lg shadow-primary/20 cursor-pointer">
                <Play className="w-5 h-5 fill-current" />
                Play Now
              </button>
            </Link>
            <Link href={`/videos/${content.short_id}`}>
              <button className="flex items-center gap-2 px-8 py-3 glass hover:bg-white/20 text-white rounded-full font-bold transition-all transform hover:scale-105 active:scale-95 cursor-pointer">
                <Info className="w-5 h-5" />
                More Info
              </button>
            </Link>
          </div>
        </motion.div>
      </div>

      {/* Video Preview Overlay (Placeholder for future) */}
      <div className="hidden lg:block absolute right-12 bottom-12 w-64 aspect-video rounded-xl overflow-hidden glass p-1 animate-pulse">
        <div className="w-full h-full bg-white/5 rounded-lg flex items-center justify-center">
            <span className="text-xs text-white/30 truncate px-4">Live Preview Buffering...</span>
        </div>
      </div>
    </section>
  );
};
