"use client";

import React from "react";
import Image from "next/image";
import Link from "next/link";
import { Play, Eye, Glasses, Star } from "lucide-react";
import { motion } from "framer-motion";
import { cn } from "@/lib/utils";

interface ContentCardProps {
  id: string;
  title: string;
  thumbnailUrl?: string;
  type: "video" | "vr360" | "image_gallery" | string;
  rating?: number;
  className?: string;
}

export const ContentCard = ({ id, title, thumbnailUrl, type, rating, className }: ContentCardProps) => {
  const getIcon = () => {
    switch (type) {
      case "vr360": return <Glasses className="w-6 h-6 fill-current text-white" />;
      case "image_gallery": return <Eye className="w-6 h-6 fill-current text-white" />;
      default: return <Play className="w-6 h-6 fill-current text-white" />;
    }
  };

  return (
    <Link href={`/videos/${id}`} className={cn("group block w-full", className)}>
      <div className="space-y-3">
        <motion.div 
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.98 }}
          className="relative aspect-video rounded-lg overflow-hidden border border-white/5 bg-black transition-all group-hover:border-primary/50 group-hover:shadow-lg group-hover:shadow-primary/10"
        >
          {thumbnailUrl ? (
            <Image
              src={thumbnailUrl}
              alt={title}
              fill
              unoptimized
              className="object-cover transition-transform duration-500 group-hover:scale-110"
            />
          ) : (
             <div className="absolute inset-0 bg-black flex items-center justify-center opacity-40">
                {getIcon()}
             </div>
          )}
          
          {/* Rating Badge */}
          {rating && (
            <div className="absolute top-2 right-2 z-10 bg-black/60 backdrop-blur-md px-1.5 py-0.5 rounded text-[10px] font-bold flex items-center gap-1 border border-white/10 group-hover:border-primary/50 transition-colors">
              <Star className="w-3 h-3 text-primary fill-primary" />
              {rating.toFixed(1)}
            </div>
          )}
          
          {/* Hover Overlay - Only Play Icon now */}
          <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex flex-col items-center justify-center p-4">
              <div className="w-12 h-12 rounded-full bg-primary flex items-center justify-center transform translate-y-4 group-hover:translate-y-0 transition-transform">
                  {getIcon()}
              </div>
          </div>
        </motion.div>

        {/* Permanent Title and Subtitle */}
        <div className="px-1">
          <h3 className="text-sm font-bold leading-tight line-clamp-1 group-hover:text-primary transition-colors">{title}</h3>
          <span className="text-[10px] uppercase tracking-widest text-foreground/40 font-bold mt-1 block">
              {type === "vr360" ? "360° VR" : type === "image_gallery" ? "Gallery" : "Video"}
          </span>
        </div>
      </div>
    </Link>
  );
};
