'use client';

import { useEffect, useState } from "react";
import { Hero } from "@/components/home/Hero";
import { ContentCard } from "@/components/content/ContentCard";
import { apiClient } from "@/lib/api-client";
import { useAuth } from "@/components/providers/AuthProvider";
import { motion, AnimatePresence } from "framer-motion";

export default function Home() {
  const { isLoading: authLoading, token } = useAuth();
  const [sections, setSections] = useState<any[]>([]);
  const [isCatalogLoading, setIsCatalogLoading] = useState(true);

  const fetchCatalog = async () => {
    try {
      setIsCatalogLoading(true);
      const res = await apiClient.get('/api/catalog/v1/catalog/home');
      setSections(res.data.sections || []);
    } catch (error) {
      console.error("Failed to fetch catalog:", error);
    } finally {
      setIsCatalogLoading(false);
    }
  };

  useEffect(() => {
    // Only fetch catalog after auth mount is done (token might exist or be null)
    if (!authLoading) {
      fetchCatalog();
    }
  }, [authLoading, token]);

  const featuredContent = sections.length > 0 && sections[0].items && sections[0].items.length > 0 
    ? sections[0].items[0] 
    : null;

  return (
    <main className="relative min-h-screen">
      <Hero content={featuredContent} isLoading={isCatalogLoading} />
      
      {/* Content Rows Container */}
      <div className="relative z-10 -mt-16 md:-mt-32 px-4 md:px-12 pb-24">
        <AnimatePresence mode="wait">
          {isCatalogLoading ? (
            <motion.div 
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="py-40 text-center"
            >
              <div className="inline-block w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mb-4" />
              <p className="text-foreground/40 font-medium tracking-widest text-xs uppercase">Loading Experience...</p>
            </motion.div>
          ) : sections.length > 0 ? (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6 }}
            >
              {sections.map((section: any, idx: number) => (
                <section key={idx} className="mb-12">
                  <h2 className="text-2xl font-bold mb-6 tracking-tight flex items-center gap-2">
                    {section.title}
                    <span className="w-1.5 h-1.5 rounded-full bg-primary" />
                  </h2>
                  <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
                    {section.items && section.items.map((content: any) => (
                      <ContentCard 
                        key={content.id || content.short_id}
                        id={content.short_id}
                        title={content.title}
                        thumbnailUrl={content.assets?.thumbnail}
                        type={content.type}
                        rating={content.rating}
                      />
                    ))}
                  </div>
                </section>
              ))}
            </motion.div>
          ) : (
            <section className="mb-12 py-20 text-center">
              <h2 className="text-2xl font-bold mb-6 tracking-tight flex items-center justify-center gap-2">
                Suggested for You
                <span className="w-1.5 h-1.5 rounded-full bg-primary" />
              </h2>
              <div className="text-foreground/20 italic">
                No personalized content available yet.
              </div>
            </section>
          )}
        </AnimatePresence>
      </div>
      
      <footer className="py-12 px-4 border-t border-white/5 text-center text-foreground/30 text-sm">
        <p>© {new Date().getFullYear()} Pechka Streaming Service. Premium PWA Experience.</p>
      </footer>
    </main>
  );
}
