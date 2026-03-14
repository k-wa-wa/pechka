import { Hero } from "@/components/home/Hero";
import { ContentCard } from "@/components/content/ContentCard";

export default async function Home() {
  const apiUrl = process.env.INTERNAL_API_URL || "http://nginx:80";
  
  let sections = [];
  try {
    const fetchUrl = `${apiUrl}/api/catalog/v1/catalog/home`;
    const res = await fetch(fetchUrl, { 
      cache: 'no-store',
      // Add a small timeout/retry logic internally if needed, but here we just ensure the URL is correct
    });
    
    if (!res.ok) {
      console.error(`Failed to fetch catalog: ${res.status} ${res.statusText}`);
    } else {
      const data = await res.json();
      sections = data.sections || [];
    }
  } catch (error) {
    console.error("Failed to fetch catalog from", apiUrl, ":", error);
  }

  const featuredContent = sections.length > 0 && sections[0].items && sections[0].items.length > 0 
    ? sections[0].items[0] 
    : null;

  return (
    <main className="relative min-h-screen">
      <Hero content={featuredContent} />
      
      {/* Content Rows Container */}
      <div className="relative z-10 -mt-16 md:-mt-32 px-4 md:px-12 pb-24">
        {sections.length > 0 ? (
          sections.map((section: any, idx: number) => (
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
          ))
        ) : (
          <section className="mb-12">
            <h2 className="text-2xl font-bold mb-6 tracking-tight flex items-center gap-2">
              Suggested for You
              <span className="w-1.5 h-1.5 rounded-full bg-primary" />
            </h2>
            <div className="col-span-full py-20 text-center text-foreground/20 italic">
              No content available in the multiverse yet.
            </div>
          </section>
        )}
      </div>
      
      {/* Decorative footer element */}
      <footer className="py-12 px-4 border-t border-white/5 text-center text-foreground/30 text-sm">
        <p>© {new Date().getFullYear()} Pechka Streaming Service. Premium PWA Experience.</p>
      </footer>
    </main>
  );
}
