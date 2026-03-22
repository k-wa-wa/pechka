"use client";

import type { Content } from "./types";

const TABS: { value: Content["content_type"]; label: string }[] = [
  { value: "video", label: "Videos" },
  { value: "vr360", label: "360° VR" },
  { value: "image_gallery", label: "Gallery" },
  { value: "ebook", label: "E-Book" },
];

interface Props {
  activeTab: Content["content_type"];
  onChange: (tab: Content["content_type"]) => void;
}

export function ContentTypeTabs({ activeTab, onChange }: Props) {
  return (
    <div className="flex p-1 bg-zinc-900 border border-white/10 rounded-xl mb-8 w-fit mx-auto">
      {TABS.map((tab) => (
        <button
          key={tab.value}
          onClick={() => onChange(tab.value)}
          className={`px-8 py-2 rounded-lg text-xs font-black uppercase tracking-widest transition-all ${
            activeTab === tab.value ? "bg-white text-black" : "text-white/40 hover:text-white"
          }`}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
