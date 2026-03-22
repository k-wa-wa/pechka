"use client";

import React, { useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { Search, Bell, User, Menu, X } from "lucide-react";
import { useAuth } from "@/components/providers/AuthProvider";

export const Navbar = () => {
  const { user, logout, isLoading } = useAuth();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  return (
    <nav
      className="fixed top-0 w-full z-50 px-4 md:px-12 py-4 bg-gradient-to-b from-black/80 to-transparent backdrop-blur-sm"
    >
      <div className="max-w-7xl mx-auto flex items-center justify-between">
        {/* Left: Logo and Primary Links */}
        <div className="flex items-center gap-8">
          <Link href="/" className="flex items-center gap-2 group">
            <div className="relative w-8 h-8 rounded-lg overflow-hidden border border-primary/20 group-hover:border-primary/50 transition-all">
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

          <div className="hidden lg:flex items-center gap-6 text-sm font-medium text-foreground/60">
            <Link href="/videos" className="hover:text-primary transition-colors">Videos</Link>
            <Link href="/360" className="hover:text-primary transition-colors">360° VR</Link>
            <Link href="/gallery" className="hover:text-primary transition-colors">Gallery</Link>
          </div>
        </div>

        {/* Right: Search, Notifications, Profile */}
        <div className="flex items-center gap-3 md:gap-5">
          <button className="p-2 hover:bg-white/10 rounded-full transition-colors text-foreground/70">
            <Search className="w-5 h-5" />
          </button>
          
          {user && (
            <div className="flex items-center gap-4 animate-in fade-in slide-in-from-right-4 duration-500">
              <div className="hidden md:flex flex-col items-end">
                <span className="text-xs font-bold text-foreground/90 leading-tight tracking-tight">{user.displayName || user.email.split('@')[0]}</span>
                <span className="text-[9px] text-primary uppercase font-black tracking-[0.2em] opacity-80">{user.roles?.[0] || 'Member'}</span>
              </div>
              <div className="relative group">
                <button 
                  className="flex items-center justify-center w-10 h-10 rounded-full bg-surface border border-primary/20 hover:border-primary/60 transition-all overflow-hidden shadow-inner"
                >
                  <User className="w-5 h-5 text-primary" />
                </button>
                <div className="absolute top-full right-0 mt-3 w-52 py-2 bg-black/90 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 transform origin-top-right scale-95 group-hover:scale-100">
                  <div className="px-4 py-3 border-b border-white/5 mb-1">
                    <p className="text-[10px] text-foreground/30 uppercase font-bold tracking-widest mb-1">Account</p>
                    <p className="text-xs text-foreground/70 truncate font-medium">{user.email}</p>
                  </div>
                  {(user.roles.includes('admin') || user.permissions.includes('content:write')) && (
                    <Link href="/admin" className="block px-4 py-2.5 text-sm hover:bg-primary/10 hover:text-primary transition-colors flex items-center gap-2">
                       <span className="w-1.5 h-1.5 rounded-full bg-primary" />
                       Admin Dashboard
                    </Link>
                  )}
                  <button 
                    onClick={logout}
                    className="w-full text-left px-4 py-2.5 text-sm text-red-400/80 hover:bg-red-400/10 hover:text-red-400 transition-colors"
                  >
                    Sign Out
                  </button>
                </div>
              </div>
            </div>
          )}

          <button 
            className="lg:hidden p-2 text-foreground/70"
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
