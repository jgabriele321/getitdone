import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Shift6 Task Manager",
  description: "AI-powered task management and productivity hub. Transform your to-do lists into organized, prioritized action plans.",
  keywords: ["task management", "productivity", "AI", "todo", "organization"],
  authors: [{ name: "Shift6 Studios" }],
  creator: "Shift6 Studios",
  publisher: "Shift6 Studios",
  formatDetection: {
    email: false,
    address: false,
    telephone: false,
  },
  metadataBase: new URL('https://shift6.dwings.app'),
  alternates: {
    canonical: '/',
  },
  openGraph: {
    title: "Shift6 Task Manager",
    description: "AI-powered task management and productivity hub",
    url: 'https://shift6.dwings.app',
    siteName: 'Shift6 Task Manager',
    images: [
      {
        url: '/logo-black.svg',
        width: 175,
        height: 78,
        alt: 'Shift6 Task Manager',
      },
    ],
    locale: 'en_US',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: "Shift6 Task Manager",
    description: "AI-powered task management and productivity hub",
    images: ['/logo-black.svg'],
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      'max-video-preview': -1,
      'max-image-preview': 'large',
      'max-snippet': -1,
    },
  },
  manifest: '/manifest.json',
  appleWebApp: {
    capable: true,
    statusBarStyle: 'default',
    title: 'Shift6 Tasks',
    startupImage: [
      '/logo-black.svg',
    ],
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <head>
        <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
        <meta name="theme-color" content="#000000" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="default" />
        <meta name="apple-mobile-web-app-title" content="Shift6 Tasks" />
        <meta name="application-name" content="Shift6 Tasks" />
        <meta name="msapplication-TileColor" content="#000000" />
        <meta name="msapplication-tap-highlight" content="no" />
        <link rel="icon" type="image/svg+xml" href="/logo-black.svg" />
        <link rel="apple-touch-icon" href="/logo-black.svg" />
        <link rel="mask-icon" href="/logo-black.svg" color="#000000" />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
