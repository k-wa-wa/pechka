"use client";

import React, { useEffect, useRef, useState, useCallback } from "react";
import Hls from "hls.js";
import { Play, Pause, Volume2, VolumeX, Maximize, ChevronLeft, Settings, ChevronsLeft, ChevronsRight, RotateCcw, RotateCw } from "lucide-react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { cn } from "@/lib/utils";

interface VideoPlayerProps {
  src: string;
  backUrl?: string;
}

export const VideoPlayer = ({ src, backUrl }: VideoPlayerProps) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [volume, setVolume] = useState(1);
  const [isMuted, setIsMuted] = useState(false);
  const [progress, setProgress] = useState(0);
  const [showControls, setShowControls] = useState(true);
  const router = useRouter();
  const controlsTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const [skipDuration, setSkipDuration] = useState(10);
  const [showSettings, setShowSettings] = useState(false);
  const [skipIndicator, setSkipIndicator] = useState<{ show: boolean; type: 'forward' | 'backward' } | null>(null);
  const skipTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const lastTapTimeRef = useRef<number>(0);
  const tapTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    const saved = localStorage.getItem('pechka_skip_duration');
    if (saved) setSkipDuration(Number(saved));
  }, []);

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
      if (video && video.duration > 0) {
        setProgress((video.currentTime / video.duration) * 100);
      }
    };

    video.addEventListener("timeupdate", updateProgress);
    video.addEventListener("loadedmetadata", updateProgress);
    return () => {
      video.removeEventListener("timeupdate", updateProgress);
      video.removeEventListener("loadedmetadata", updateProgress);
    };
  }, [src]);

  const togglePlay = useCallback(() => {
    if (videoRef.current?.paused) {
      videoRef.current.play();
      setIsPlaying(true);
    } else {
      videoRef.current?.pause();
      setIsPlaying(false);
    }
  }, []);

  const handleSkip = useCallback((direction: 'forward' | 'backward') => {
    if (videoRef.current && videoRef.current.duration > 0) {
      const delta = direction === 'forward' ? skipDuration : -skipDuration;
      videoRef.current.currentTime = Math.max(0, Math.min(videoRef.current.currentTime + delta, videoRef.current.duration));
      setProgress((videoRef.current.currentTime / videoRef.current.duration) * 100);
      
      setSkipIndicator({ show: true, type: direction });
      if (skipTimeoutRef.current) clearTimeout(skipTimeoutRef.current);
      skipTimeoutRef.current = setTimeout(() => setSkipIndicator(null), 500);
    }
  }, [skipDuration]);

  const toggleMute = useCallback(() => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  }, [isMuted]);

  const toggleFullscreen = useCallback(() => {
    if (!containerRef.current) return;

    const doc = document as any;
    const container = containerRef.current as any;

    const isFullscreen = doc.fullscreenElement || 
                         doc.webkitFullscreenElement || 
                         doc.mozFullScreenElement || 
                         doc.msFullscreenElement;

    if (!isFullscreen) {
      if (container.requestFullscreen) {
        container.requestFullscreen();
      } else if (container.webkitRequestFullscreen) {
        container.webkitRequestFullscreen();
      } else if (container.mozRequestFullScreen) {
        container.mozRequestFullScreen();
      } else if (container.msRequestFullscreen) {
        container.msRequestFullscreen();
      } else if (videoRef.current && (videoRef.current as any).webkitEnterFullscreen) {
        // Fallback for iPhone Safari
        (videoRef.current as any).webkitEnterFullscreen();
      }
    } else {
      if (doc.exitFullscreen) {
        doc.exitFullscreen();
      } else if (doc.webkitExitFullscreen) {
        doc.webkitExitFullscreen();
      } else if (doc.mozCancelFullScreen) {
        doc.mozCancelFullScreen();
      } else if (doc.msExitFullscreen) {
        doc.msExitFullscreen();
      }
    }
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (document.activeElement?.tagName === 'INPUT' || document.activeElement?.tagName === 'TEXTAREA') return;

      switch(e.key.toLowerCase()) {
        case ' ':
          e.preventDefault();
          togglePlay();
          break;
        case 'arrowleft':
          e.preventDefault();
          handleSkip('backward');
          break;
        case 'arrowright':
          e.preventDefault();
          handleSkip('forward');
          break;
        case 'arrowup':
          e.preventDefault();
          if (videoRef.current) {
            videoRef.current.volume = Math.min(1, videoRef.current.volume + 0.1);
            setVolume(videoRef.current.volume);
          }
          break;
        case 'arrowdown':
          e.preventDefault();
          if (videoRef.current) {
            videoRef.current.volume = Math.max(0, videoRef.current.volume - 0.1);
            setVolume(videoRef.current.volume);
          }
          break;
        case 'f':
          e.preventDefault();
          toggleFullscreen();
          break;
        case 'm':
          e.preventDefault();
          toggleMute();
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [togglePlay, handleSkip, toggleFullscreen, toggleMute]);

  const handleDurationChange = (val: number) => {
    setSkipDuration(val);
    localStorage.setItem('pechka_skip_duration', val.toString());
    setShowSettings(false);
  };

  const processTap = (clientX: number, target: HTMLElement) => {
    const currentTime = Date.now();
    const timeDiff = currentTime - lastTapTimeRef.current;
    
    if (timeDiff > 0 && timeDiff < 300) {
      // Double tap detected
      if (tapTimeoutRef.current) clearTimeout(tapTimeoutRef.current);
      const rect = target.getBoundingClientRect();
      const clickX = clientX - rect.left;
      
      const isLeftSide = clickX < rect.width / 2;
      handleSkip(isLeftSide ? 'backward' : 'forward');
      
      lastTapTimeRef.current = 0;
    } else {
      // Potential single tap
      lastTapTimeRef.current = currentTime;
      if (tapTimeoutRef.current) clearTimeout(tapTimeoutRef.current);
      tapTimeoutRef.current = setTimeout(() => {
        togglePlay();
        lastTapTimeRef.current = 0;
      }, 300);
    }
  };

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    // Only process if it's the primary pointer (mouse left click or touch)
    if (e.button !== 0 && e.pointerType === 'mouse') return;
    
    // Ignore interactive elements
    const target = e.target as HTMLElement;
    if (target.closest('button') || target.closest('input') || target.closest('a') || target.closest('.pointer-events-auto')) {
      // If it's an interactive element, we stop propagation here to be safe
      // but only if we want to ensure the background doesn't detect it as a tap.
      return;
    }

    processTap(e.clientX, e.currentTarget);
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

  return (
    <div 
      ref={containerRef}
      className="relative w-full h-[100svh] bg-black flex items-center justify-center group overflow-hidden"
      style={{ touchAction: 'manipulation' }}
      onMouseMove={handleMouseMove}
      onPointerDown={handlePointerDown}
    >
      <video
        ref={videoRef}
        className="w-full h-full object-contain pointer-events-none"
        playsInline
      />


      {/* Skip Indicator */}
      {skipIndicator?.show && (
        <div className={cn(
          "absolute top-1/2 -translate-y-1/2 flex flex-col items-center justify-center p-6 md:p-8 rounded-full bg-black/60 text-white pointer-events-none animate-in fade-in zoom-in duration-200 z-[60]",
          skipIndicator.type === 'backward' ? "left-1/4 -translate-x-1/2" : "right-1/4 translate-x-1/2"
        )}>
          {skipIndicator.type === 'backward' ? <ChevronsLeft className="w-8 h-8 md:w-12 md:h-12" /> : <ChevronsRight className="w-8 h-8 md:w-12 md:h-12" />}
          <span className="text-sm md:text-base font-bold mt-2">{skipDuration}s</span>
        </div>
      )}

      {/* Overlays */}
      <div 
        className={cn(
          "absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/40 transition-opacity duration-500 z-50 pointer-events-none",
          showControls ? "opacity-100" : "opacity-0 cursor-none"
        )} 
      >
        
        {/* Top: Back Button */}
        <div className="absolute top-0 left-0 p-4 md:p-8 pt-[env(safe-area-inset-top,2rem)] pointer-events-auto">
           {backUrl ? (
             <Link 
               href={backUrl}
               className="flex items-center gap-2 text-white/70 hover:text-white transition-colors"
             >
               <ChevronLeft />
               Back
             </Link>
           ) : (
             <button 
               onClick={() => router.back()}
               className="flex items-center gap-2 text-white/70 hover:text-white transition-colors"
             >
               <ChevronLeft />
               Back
             </button>
           )}
        </div>

        {/* Bottom: Controls */}
        <div className="absolute bottom-0 inset-x-0 p-4 md:p-8 pb-[max(1rem,env(safe-area-inset-bottom,2rem))] space-y-4 pointer-events-auto">
           {/* Progress Bar */}
           <div className="group/progress relative h-1.5 bg-white/10 rounded-full cursor-pointer">
              <input 
                type="range"
                value={progress}
                onChange={handleSeek}
                className="absolute inset-x-0 -top-3 w-full h-8 opacity-0 z-10 cursor-pointer"
              />
              <div 
                className="absolute top-0 left-0 h-full bg-primary rounded-full" 
                style={{ width: `${isNaN(progress) ? 0 : progress}%` }}
              />
              <div 
                className="absolute top-1/2 -translate-y-1/2 w-4 h-4 bg-primary rounded-full opacity-0 group-hover/progress:opacity-100 transition-opacity shadow-[0_0_10px_rgba(255,87,34,0.8)]"
                style={{ left: `${isNaN(progress) ? 0 : progress}%`, marginLeft: "-8px" }}
              />
           </div>

           <div className="flex items-center justify-between">
              <div className="flex items-center gap-4 md:gap-6">
                <div className="flex items-center gap-2 md:gap-4">
                  <button onClick={() => handleSkip('backward')} className="p-2 hover:text-primary transition-colors tooltip-trigger" title={`Skip Backward ${skipDuration}s`}>
                    <RotateCcw className="w-5 h-5 md:w-6 md:h-6" />
                  </button>
                  <button onClick={togglePlay} className="p-2 hover:text-primary transition-colors w-10 h-10 flex items-center justify-center">
                      {isPlaying ? <Pause className="w-6 h-6 md:w-7 md:h-7" fill="currentColor" /> : <Play className="w-6 h-6 md:w-7 md:h-7" fill="currentColor" />}
                  </button>
                  <button onClick={() => handleSkip('forward')} className="p-2 hover:text-primary transition-colors tooltip-trigger" title={`Skip Forward ${skipDuration}s`}>
                    <RotateCw className="w-5 h-5 md:w-6 md:h-6" />
                  </button>
                </div>
                <div className="hidden md:flex items-center gap-2 group/volume">
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

              <div className="flex items-center gap-3 md:gap-6">
                <div className="relative">
                  <button onClick={() => setShowSettings(!showSettings)} className="p-2 hover:text-primary transition-colors">
                      <Settings className="w-5 h-5 md:w-6 md:h-6" />
                  </button>
                  {showSettings && (
                    <div className="absolute bottom-full right-0 mb-4 bg-black/95 border border-white/10 rounded-xl p-2 flex flex-col gap-1 min-w-[140px] shadow-2xl z-[70] backdrop-blur-xl">
                      <div className="text-[10px] text-white/40 px-3 py-1.5 uppercase tracking-widest font-bold">Skip Duration</div>
                      {[5, 10, 15, 30].map(val => (
                        <button 
                          key={val}
                          onClick={() => handleDurationChange(val)}
                          className={cn(
                            "text-left px-3 py-2.5 text-sm rounded-lg hover:bg-white/10 transition-colors",
                            skipDuration === val ? "text-primary bg-primary/10 font-bold" : "text-white/80"
                          )}
                        >
                          {val} seconds
                        </button>
                      ))}
                    </div>
                  )}
                </div>
                <button onClick={toggleFullscreen} className="p-2 hover:text-primary transition-colors">
                    <Maximize className="w-5 h-5 md:w-6 md:h-6" />
                </button>
              </div>
           </div>
        </div>
      </div>
    </div>
  );
};
