"use client";

import React, { useEffect, useState } from "react";
import { VideoPlayer } from "@/components/video/VideoPlayer";
import { ChevronLeft, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";

export default function WatchPage({ params }: { params: Promise<{ short_id: string }> }) {
  const { short_id } = React.use(params);
  const [content, setContent] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    const fetchContent = async () => {
      try {
        const res = await fetch(`/api/catalog/v1/catalog/contents/${short_id}`);
        const data = await res.json();
        setContent(data);
      } catch (error) {
        console.error("Failed to fetch content details:", error);
      } finally {
        setLoading(false);
      }
    };
    fetchContent();
  }, [short_id]);

  if (loading) {
    return (
      <div className="w-screen h-screen bg-black flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-primary animate-spin" />
      </div>
    );
  }

  const videoSrc = content?.assets?.hls_master;

  if (!videoSrc) {
    return (
      <div className="w-screen h-screen bg-black flex flex-col items-center justify-center p-4">
        <h1 className="text-xl font-bold mb-4">Content Not Found</h1>
        <button 
          onClick={() => router.back()}
          className="flex items-center gap-2 text-primary hover:underline"
        >
          <ChevronLeft /> Go Back
        </button>
      </div>
    );
  }

  return (
    <main className="w-screen h-screen bg-black overflow-hidden relative">
      <VideoPlayer src={videoSrc} />
      
      {/* Fallback/Overlay if video fails or for demo */}
      <div className="absolute bottom-6 left-1/2 -translate-x-1/2 text-white/10 text-[8px] pointer-events-none uppercase tracking-widest">
          Source: {videoSrc}
      </div>
    </main>
  );
}
