import React from "react";
import { DetailView } from "@/components/content/DetailView";

export default async function DetailPage({ params }: { params: Promise<{ short_id: string }> }) {
  const { short_id } = await params;
  const apiUrl = process.env.INTERNAL_API_URL || "http://nginx:80";
  
  let content: any = null;
  try {
    const res = await fetch(`${apiUrl}/api/catalog/v1/catalog/contents/${short_id}`, { 
      next: { revalidate: 60 } 
    });
    
    if (res.ok) {
      content = await res.json();
    }
  } catch (error) {
    console.error("Failed to fetch content details:", error);
  }

  if (!content) {
    return (
      <div className="min-h-screen bg-black flex items-center justify-center text-foreground/20 italic">
        Content not found in this dimension.
      </div>
    );
  }

  return <DetailView content={content} />;
}
