import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable static file serving for PWA
  trailingSlash: false,
  
  // Optimize for production
  compress: true,
  poweredByHeader: false,
  
  // PWA and performance optimizations
  experimental: {
    optimizeCss: true,
  },
  
  // Headers for PWA and security
  async headers() {
    return [
      {
        source: '/manifest.json',
        headers: [
          {
            key: 'Content-Type',
            value: 'application/manifest+json',
          },
          {
            key: 'Cache-Control',
            value: 'public, max-age=86400', // 24 hours
          },
        ],
      },
      {
        source: '/(.*)',
        headers: [
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'X-XSS-Protection',
            value: '1; mode=block',
          },
        ],
      },
    ]
  },
};

export default nextConfig;
