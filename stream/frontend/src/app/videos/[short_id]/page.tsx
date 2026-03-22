"use client";

import React, { useEffect, useState } from "react";
import { DetailView } from "@/components/content/DetailView";
import { apiClient } from "@/lib/api-client";
import { useParams } from "next/navigation";
import { Loader2 } from "lucide-react";

export default function DetailPage() {
  const { short_id } = useParams<{ short_id: string }>();
  const [content, setContent] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchContent = async () => {
      try {
        const res = await apiClient.get(`/api/catalog/v1/catalog/contents/${short_id}`);
        setContent(res.data);
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
      <div className="min-h-screen bg-black flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-primary animate-spin" />
      </div>
    );
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
