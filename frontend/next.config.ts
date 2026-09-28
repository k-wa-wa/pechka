import type { NextConfig } from 'next'

const nextConfig: NextConfig = {
  output: 'standalone',
  allowedDevOrigins: ['192.168.1.151', '192.168.*', '10.*', '172.*', 'localhost:*'],
  images: {
    remotePatterns: [
      { protocol: 'http', hostname: '*' },
      { protocol: 'https', hostname: '*' },
    ],
  },
  // Local mock API mode (`npm run dev:mock`): proxy same-origin /api/v1/*
  // browser requests to the mock server, mirroring the nginx reverse proxy
  // used in preview/prod so lib/api.ts doesn't need CORS-aware branching.
  async rewrites() {
    if (!process.env.MOCK_API_URL) return []
    return [
      {
        source: '/api/v1/:path*',
        destination: `${process.env.MOCK_API_URL}/api/v1/:path*`,
      },
    ]
  },
}

export default nextConfig
