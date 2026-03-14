"use client";

import React from "react";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ChevronLeft } from "lucide-react";
import { motion } from "framer-motion";

export default function LoginPage() {
  return (
    <main className="relative min-h-screen flex items-center justify-center p-4 overflow-hidden">
      {/* Cinematic Background */}
      <div className="absolute inset-0">
        <Image
          src="/assets/posters/sintel01.jpg"
          alt="Background"
          fill
          className="object-cover brightness-[0.2] scale-105"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black via-transparent to-black/60" />
      </div>

      {/* Back to Home */}
      <Link 
        href="/" 
        className="absolute top-8 left-8 flex items-center gap-2 text-sm text-foreground/50 hover:text-primary transition-colors z-20"
      >
        <ChevronLeft className="w-4 h-4" />
        Back to Home
      </Link>

      {/* Login Card */}
      <motion.div
        initial={{ opacity: 0, y: 20, scale: 0.95 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
        className="relative z-10 w-full max-w-md glass p-8 md:p-12 rounded-3xl"
      >
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 border border-primary/20 mb-4">
             <Image src="/icon.png" alt="Logo" width={40} height={40} />
          </div>
          <h1 className="text-3xl font-bold tracking-tight">Welcome Back</h1>
          <p className="text-foreground/50 mt-2">Sign in to continue your adventure</p>
        </div>

        <form className="space-y-6" onSubmit={(e) => e.preventDefault()}>
          <Input 
            label="Email Address" 
            type="email" 
            placeholder="name@example.com" 
            required
          />
          <div className="space-y-1">
             <Input 
                label="Password" 
                type="password" 
                placeholder="••••••••" 
                required
              />
              <div className="text-right">
                <Link href="#" className="text-xs text-primary hover:underline">Forgot password?</Link>
              </div>
          </div>

          <Button isFullWidth size="lg">
            Sign In
          </Button>

          <div className="relative py-4">
             <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-white/5" />
             </div>
             <div className="relative flex justify-center text-xs">
                <span className="bg-transparent px-2 text-foreground/30 uppercase tracking-widest">or</span>
             </div>
          </div>

          <Button variant="outline" isFullWidth size="lg">
            Create an Account
          </Button>
        </form>

        <p className="mt-8 text-center text-xs text-foreground/30">
          By signing in, you agree to our <Link href="#" className="text-foreground/50 hover:underline">Terms of Service</Link> and <Link href="#" className="text-foreground/50 hover:underline">Privacy Policy</Link>.
        </p>
      </motion.div>
    </main>
  );
}
