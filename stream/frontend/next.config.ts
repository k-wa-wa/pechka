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
    return [
      {
        source: '/api/:path*',
        destination: 'http://nginx:80/api/:path*',
      },
      {
        source: '/assets/:path*',
        destination: 'http://nginx:80/assets/:path*',
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
