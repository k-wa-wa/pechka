"use client";

import React from "react";
import Image from "next/image";
import Link from "next/link";
import { Navbar } from "@/components/layout/Navbar";
import { Button } from "@/components/ui/Button";
import { Play, Share2, Plus, Star, Eye } from "lucide-react";
import { motion } from "framer-motion";

interface DetailViewProps {
  content: any;
}

export const DetailView = ({ content }: DetailViewProps) => {
  const isVideo = content.type === "video" || content.type === "vr360";
  const isVR = content.type === "vr360";
  const isImage = content.type === "image_gallery";

  return (
    <main className="min-h-screen bg-black">
      <Navbar />

      {/* Hero Section */}
      <div className="relative h-[60vh] md:h-[75vh]">
        <Image
          src={content.assets?.thumbnail || "/icon.png"}
          alt={content.title}
          fill
          className="object-cover object-top brightness-[0.4]"
          priority
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black via-transparent to-black/20" />
        
        <div className="absolute inset-x-0 bottom-0 px-4 md:px-12 pb-12 flex flex-col md:flex-row items-end gap-8">
          {/* Metadata */}
          <div className="flex-1 space-y-4">
             <motion.div
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
             >
                <div className="flex items-center gap-3 text-primary font-bold text-sm tracking-widest uppercase mb-2">
                    <span>{content.metadata?.year || "2026"}</span>
                    <span className="w-1 h-1 rounded-full bg-white/20" />
                    <span>{content.metadata?.duration_seconds ? `${Math.floor(content.metadata.duration_seconds / 60)} min` : "N/A"}</span>
                    <span className="w-1 h-1 rounded-full bg-white/20" />
                    <span className="px-1.5 py-0.5 rounded border border-white/20 text-[10px] flex items-center gap-1">
                        <Star className="w-2 h-2 fill-primary text-primary" />
                        {content.rating?.toFixed(1) || "5.0"}
                    </span>
                </div>
                <h1 className="text-4xl md:text-6xl font-black tracking-tight mb-4">{content.title}</h1>
                <p className="text-foreground/70 max-w-2xl text-lg leading-relaxed">{content.description}</p>
             </motion.div>

             <div className="flex flex-wrap gap-4 pt-4">
                {isVideo && (
                    <Link href={`/videos/watch/${content.short_id}`} className="cursor-pointer">
                        <Button size="lg" className="px-10">
                            <Play className="w-5 h-5 mr-2 fill-current" />
                            {isVR ? "Enter VR World" : "Play Now"}
                        </Button>
                    </Link>
                )}
                {isImage && (
                    <Button size="lg" className="px-10 cursor-pointer">
                        <Eye className="w-5 h-5 mr-2" />
                        Open Gallery
                    </Button>
                )}
                <Button variant="secondary" size="lg" disabled className="opacity-50 cursor-not-allowed">
                    <Plus className="w-5 h-5 mr-2" />
                    My List
                </Button>
                <Button variant="outline" size="lg" className="w-12 h-12 p-0">
                    <Share2 className="w-5 h-5" />
                </Button>
             </div>
          </div>
        </div>
      </div>

      {/* Details Area */}
      <div className="px-4 md:px-12 py-16 grid grid-cols-1 lg:grid-cols-3 gap-12 border-t border-white/5">
        <div className="lg:col-span-2 space-y-12">
           <section>
              <h2 className="text-xl font-bold mb-6 flex items-center gap-2">
                About this {content.type}
                <span className="w-8 h-px bg-primary/50" />
              </h2>
              <p className="text-foreground/60 leading-loose">
                  {content.description} This {content.type} is delivered using our premium edge infrastructure.
              </p>
           </section>

           <section>
              <h2 className="text-xl font-bold mb-6 flex items-center gap-2">
                Technical info
                <span className="w-8 h-px bg-primary/50" />
              </h2>
              <div className="grid grid-cols-2 md:grid-cols-3 gap-6">
                 <div>
                    <label className="text-[10px] uppercase font-bold text-foreground/30 block mb-1">Type</label>
                    <span className="text-sm font-medium uppercase tracking-wider">{content.type}</span>
                 </div>
                 {content.metadata?.director && (
                    <div>
                        <label className="text-[10px] uppercase font-bold text-foreground/30 block mb-1">Director</label>
                        <span className="text-sm font-medium">{content.metadata.director}</span>
                    </div>
                 )}
                 {content.tags && content.tags.length > 0 && (
                    <div className="col-span-2 md:col-span-3">
                        <label className="text-[10px] uppercase font-bold text-foreground/30 block mb-2">Tags</label>
                        <div className="flex flex-wrap gap-2">
                            {content.tags.map((tag: string) => (
                                <span key={tag} className="px-3 py-1 bg-white/5 border border-white/10 rounded-full text-xs text-foreground/60 hover:text-primary hover:border-primary/50 transition-colors cursor-pointer">
                                    #{tag}
                                </span>
                            ))}
                        </div>
                    </div>
                 )}
              </div>
           </section>
        </div>

        {/* Sidebar */}
        <div className="space-y-8">
           <div className="glass p-6 rounded-2xl">
              <h3 className="font-bold mb-4 flex items-center justify-between">
                User Rating
                <div className="flex items-center text-primary text-xs">
                    <Star className="w-3 h-3 fill-current mr-1" />
                    {content.rating?.toFixed(1) || "5.0"}/5.0
                </div>
              </h3>
              <div className="space-y-4">
                 <div className="h-1.5 w-full bg-white/5 rounded-full overflow-hidden">
                    <div 
                        className="h-full bg-primary shadow-[0_0_8px_rgba(255,87,34,0.5)]" 
                        style={{ width: `${(content.rating || 5.0) * 20}%` }}
                    />
                 </div>
                 <p className="text-[10px] text-foreground/40 text-center uppercase tracking-widest font-bold">Based on community feedback</p>
              </div>
           </div>
        </div>
      </div>
    </main>
  );
};
