"use client";

import React, { useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import { Play, Pause, Volume2, VolumeX, Maximize, ChevronLeft } from "lucide-react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";

interface VideoPlayerProps {
  src: string;
}

export const VideoPlayer = ({ src }: VideoPlayerProps) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [progress, setProgress] = useState(0);
  const [showControls, setShowControls] = useState(true);
  const router = useRouter();
  const controlsTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    if (Hls.isSupported()) {
      const hls = new Hls();
      hls.loadSource(src);
      hls.attachMedia(video);
      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().catch(err => console.log("Autoplay blocked:", err));
        setIsPlaying(true);
      });
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = src;
      video.play().catch(err => console.log("Autoplay blocked:", err));
      setIsPlaying(true);
    }

    const updateProgress = () => {
      setProgress((video.currentTime / video.duration) * 100);
    };

    video.addEventListener("timeupdate", updateProgress);
    return () => video.removeEventListener("timeupdate", updateProgress);
  }, [src]);

  const togglePlay = () => {
    if (videoRef.current?.paused) {
      videoRef.current.play();
      setIsPlaying(true);
    } else {
      videoRef.current?.pause();
      setIsPlaying(false);
    }
  };

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  const handleMouseMove = () => {
    setShowControls(true);
    if (controlsTimeoutRef.current) clearTimeout(controlsTimeoutRef.current);
    controlsTimeoutRef.current = setTimeout(() => {
      if (isPlaying) setShowControls(false);
    }, 3000);
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseFloat(e.target.value);
    if (videoRef.current) {
      videoRef.current.currentTime = (val / 100) * videoRef.current.duration;
      setProgress(val);
    }
  };

  const toggleFullscreen = () => {
    if (containerRef.current?.requestFullscreen) {
      containerRef.current.requestFullscreen();
    }
  };

  return (
    <div 
      ref={containerRef}
      className="relative w-full h-full bg-black flex items-center justify-center group"
      onMouseMove={handleMouseMove}
      onClick={togglePlay}
    >
      <video
        ref={videoRef}
        className="w-full max-h-screen"
        playsInline
      />

      {/* Overlays */}
      <div className={cn(
        "absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/40 transition-opacity duration-500",
        showControls ? "opacity-100" : "opacity-0 cursor-none"
      )} onClick={(e) => e.stopPropagation()}>
        
        {/* Top: Back Button */}
        <div className="absolute top-0 left-0 p-8">
           <button 
             onClick={() => router.back()}
             className="flex items-center gap-2 text-white/50 hover:text-white transition-colors"
           >
             <ChevronLeft />
             Back
           </button>
        </div>

        {/* Bottom: Controls */}
        <div className="absolute bottom-0 inset-x-0 p-8 space-y-4">
           {/* Progress Bar */}
           <div className="group/progress relative h-1 bg-white/10 rounded-full cursor-pointer">
              <input 
                type="range"
                value={progress}
                onChange={handleSeek}
                className="absolute inset-x-0 -top-2 w-full h-6 opacity-0 z-10 cursor-pointer"
              />
              <div 
                className="absolute top-0 left-0 h-full bg-primary rounded-full" 
                style={{ width: `${progress}%` }}
              />
              <div 
                className="absolute top-1/2 -translate-y-1/2 w-4 h-4 bg-primary rounded-full opacity-0 group-hover/progress:opacity-100 transition-opacity shadow-[0_0_10px_rgba(255,87,34,0.8)]"
                style={{ left: `${progress}%`, marginLeft: "-8px" }}
              />
           </div>

           <div className="flex items-center justify-between">
              <div className="flex items-center gap-6">
                <button onClick={togglePlay} className="hover:text-primary transition-colors">
                    {isPlaying ? <Pause fill="currentColor" /> : <Play fill="currentColor" />}
                </button>
                <div className="flex items-center gap-2 group/volume">
                    <button onClick={toggleMute} className="hover:text-primary transition-colors">
                        {isMuted ? <VolumeX /> : <Volume2 />}
                    </button>
                    <input 
                        type="range" 
                        className="w-0 group-hover/volume:w-20 transition-all accent-primary h-1" 
                        min="0" max="1" step="0.1" 
                        onChange={(e) => {
                            const v = parseFloat(e.target.value);
                            if (videoRef.current) videoRef.current.volume = v;
                            setVolume(v);
                        }}
                    />
                </div>
              </div>

              <div className="flex items-center gap-6">
                <button onClick={toggleFullscreen} className="hover:text-primary transition-colors">
                    <Maximize className="w-5 h-5" />
                </button>
              </div>
           </div>
        </div>
      </div>
    </div>
  );
};
