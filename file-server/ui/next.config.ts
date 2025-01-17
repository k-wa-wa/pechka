import type { NextConfig } from "next"
import { withNextVideo } from "next-video/process"

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/resources/:path*",
        destination: `${process.env.API_URL}/resources/:path*`,
      },
      {
        source: "/api/:path*",
        destination: `${process.env.API_URL}/api/:path*`,
      },
    ]
  },
}

export default withNextVideo(nextConfig)
