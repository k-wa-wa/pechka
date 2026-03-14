import { ContentCard } from "@/components/content/ContentCard";

export default async function GalleryPage() {
  const apiUrl = process.env.INTERNAL_API_URL || "http://nginx:80";
  
  let galleries = [];
  try {
    const res = await fetch(`${apiUrl}/api/catalog/v1/catalog/home`, { cache: 'no-store' });
    const data = await res.json();
    
    const allSections = data.sections || [];
    const allItems = allSections.flatMap((s: any) => s.items || []);
    galleries = allItems.filter((item: any) => item.type === "image_gallery");
  } catch (error) {
    console.error("Failed to fetch gallery content:", error);
  }

  return (
    <main className="min-h-screen bg-black pt-24 px-4 md:px-12 pb-24">
      <header className="mb-12">
        <h1 className="text-4xl md:text-5xl font-black tracking-tight mb-2">Image Gallery</h1>
        <p className="text-foreground/50 border-l-2 border-primary pl-4">Sunning photography and digital art</p>
      </header>
      
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-6">
        {galleries.length > 0 ? (
          galleries.map((content: any) => (
            <ContentCard 
              key={content.id || content.short_id}
              id={content.short_id}
              title={content.title}
              thumbnailUrl={content.assets?.thumbnail}
              type={content.type}
              rating={content.rating}
            />
          ))
        ) : (
          <div className="col-span-full py-20 text-center text-foreground/20 italic">
            No gallery collections found.
          </div>
        )}
      </div>
    </main>
  );
}
