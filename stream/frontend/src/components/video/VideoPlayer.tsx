"use client";

import React, { useEffect, useRef, useState, useCallback } from "react";
import Hls from "hls.js";
import { Play, Pause, Volume2, VolumeX, Maximize, ChevronLeft, Settings, ChevronsLeft, ChevronsRight, RotateCcw, RotateCw } from "lucide-react";
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
      setProgress((video.currentTime / video.duration) * 100);
    };

    video.addEventListener("timeupdate", updateProgress);
    return () => video.removeEventListener("timeupdate", updateProgress);
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
    if (videoRef.current) {
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
    if (containerRef.current?.requestFullscreen) {
      containerRef.current.requestFullscreen();
    }
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignore if user is typing in an input (not directly relevant here but good practice)
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
          toggleFullscreen();
          break;
        case 'm':
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

  const handleVideoAreaClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const currentTime = new Date().getTime();
    const timeDiff = currentTime - lastTapTimeRef.current;
    
    if (timeDiff < 300) {
      // Double tap
      if (tapTimeoutRef.current) clearTimeout(tapTimeoutRef.current);
      const rect = e.currentTarget.getBoundingClientRect();
      const clickX = e.clientX - rect.left;
      if (clickX < rect.width / 2) {
        handleSkip('backward');
      } else {
        handleSkip('forward');
      }
      lastTapTimeRef.current = 0;
    } else {
      // Single tap
      lastTapTimeRef.current = currentTime;
      tapTimeoutRef.current = setTimeout(() => {
        togglePlay();
        lastTapTimeRef.current = 0;
      }, 300);
    }
  };

  // Original toggleMute removed because it's replaced by useCallback

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
      className="relative w-full h-full bg-black flex items-center justify-center group"
      onMouseMove={handleMouseMove}
      onClick={handleVideoAreaClick}
    >
      <video
        ref={videoRef}
        className="w-full max-h-screen"
        playsInline
      />

      {/* Skip Indicator */}
      {skipIndicator?.show && (
        <div className={cn(
          "absolute top-1/2 -translate-y-1/2 flex flex-col items-center justify-center p-6 md:p-8 rounded-full bg-black/60 text-white pointer-events-none animate-in fade-in zoom-in duration-200 z-50",
          skipIndicator.type === 'backward' ? "left-1/4 -translate-x-1/2" : "right-1/4 translate-x-1/2"
        )}>
          {skipIndicator.type === 'backward' ? <ChevronsLeft className="w-8 h-8 md:w-12 md:h-12" /> : <ChevronsRight className="w-8 h-8 md:w-12 md:h-12" />}
          <span className="text-sm md:text-base font-bold mt-2">{skipDuration}s</span>
        </div>
      )}

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
                <div className="flex items-center gap-4">
                  <button onClick={() => handleSkip('backward')} className="hover:text-primary transition-colors tooltip-trigger" title={`Skip Backward ${skipDuration}s`}>
                    <RotateCcw className="w-5 h-5" />
                  </button>
                  <button onClick={togglePlay} className="hover:text-primary transition-colors w-8 h-8 flex items-center justify-center">
                      {isPlaying ? <Pause fill="currentColor" /> : <Play fill="currentColor" />}
                  </button>
                  <button onClick={() => handleSkip('forward')} className="hover:text-primary transition-colors tooltip-trigger" title={`Skip Forward ${skipDuration}s`}>
                    <RotateCw className="w-5 h-5" />
                  </button>
                </div>
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
                <div className="relative">
                  <button onClick={() => setShowSettings(!showSettings)} className="hover:text-primary transition-colors pr-2">
                      <Settings className="w-5 h-5" />
                  </button>
                  {showSettings && (
                    <div className="absolute bottom-full right-0 mb-4 bg-black/90 border border-white/10 rounded-lg p-2 flex flex-col gap-1 min-w-[120px] shadow-xl z-50">
                      <div className="text-xs text-white/50 px-2 py-1 uppercase tracking-wider font-semibold">Skip Duration</div>
                      {[5, 10, 15, 30].map(val => (
                        <button 
                          key={val}
                          onClick={() => handleDurationChange(val)}
                          className={cn(
                            "text-left px-3 py-2 text-sm rounded hover:bg-white/10 transition-colors",
                            skipDuration === val ? "text-primary font-bold" : "text-white"
                          )}
                        >
                          {val} seconds
                        </button>
                      ))}
                    </div>
                  )}
                </div>
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
