import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    remotePatterns: [
      {
        protocol: "http",
        hostname: "localhost",
        port: "9000",
      },
      {
        protocol: "http",
        hostname: "minio",
        port: "9000",
      },
    ],
  },
  async rewrites() {
    const internalApiUrl = process.env.INTERNAL_API_URL || "http://nginx:80";
    return [
      {
        source: '/api/:path*',
        destination: `${internalApiUrl}/api/:path*`,
      },
      {
        source: '/assets/:path*',
        destination: `${internalApiUrl}/assets/:path*`,
      },
      {
        // MinIO direct access (keep as fallback if needed, but currently routing through nginx)
        source: "/minio-assets/:path*",
        destination: "http://minio:9000/assets/:path*",
      },
    ];
  },
};

export default nextConfig;
