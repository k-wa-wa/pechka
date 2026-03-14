"use client";

import React, { useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { Search, Bell, User, Menu, X } from "lucide-react";

export const Navbar = () => {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  return (
    <nav
      className="fixed top-0 w-full z-50 px-4 md:px-12 py-4 bg-gradient-to-b from-black/70 to-transparent"
    >
      <div className="max-w-7xl mx-auto flex items-center justify-between">
        {/* Left: Logo and Primary Links */}
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-2">
            <div className="relative w-8 h-8 rounded-lg overflow-hidden border border-primary/20">
              <Image
                src="/icon.png"
                alt="Pechka Logo"
                fill
                className="object-cover"
              />
            </div>
            <span className="text-xl font-bold tracking-tight text-gradient hidden sm:block">
              PECHKA
            </span>
          </Link>

          <div className="hidden md:flex items-center gap-6 text-sm font-medium text-foreground/70">
            <Link href="/videos" className="hover:text-primary transition-colors">Videos</Link>
            <Link href="/360" className="hover:text-primary transition-colors">360° VR</Link>
            <Link href="/gallery" className="hover:text-primary transition-colors">Gallery</Link>
          </div>
        </div>

        {/* Right: Search, Notifications, Profile */}
        <div className="flex items-center gap-4 md:gap-6">
          <button className="p-2 hover:bg-white/10 rounded-full transition-colors">
            <Search className="w-5 h-5" />
          </button>
          <button className="relative p-2 hover:bg-white/10 rounded-full transition-colors hidden sm:block">
            <Bell className="w-5 h-5" />
            <span className="absolute top-2 right-2 w-2 h-2 bg-primary rounded-full" />
          </button>
          <Link 
            href="/auth/login" 
            className="flex items-center justify-center w-10 h-10 rounded-full bg-surface border border-white/10 hover:border-primary/50 transition-all"
          >
            <User className="w-5 h-5" />
          </Link>
          <button 
            className="md:hidden p-2"
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
          >
            {isMobileMenuOpen ? <X /> : <Menu />}
          </button>
        </div>
      </div>

      {/* Mobile Menu */}
      {isMobileMenuOpen && (
        <div className="md:hidden absolute top-full left-0 w-full bg-black/90 backdrop-blur-md p-4 flex flex-col gap-4">
          <Link href="/videos" className="px-4 py-2 hover:bg-white/10 rounded">Videos</Link>
          <Link href="/360" className="px-4 py-2 hover:bg-white/10 rounded">360° VR</Link>
          <Link href="/gallery" className="px-4 py-2 hover:bg-white/10 rounded">Gallery</Link>
        </div>
      )}
    </nav>
  );
};
