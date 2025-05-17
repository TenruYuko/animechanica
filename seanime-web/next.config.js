const isProd = process.env.NODE_ENV === 'production';
const isDesktop = process.env.NEXT_PUBLIC_PLATFORM === 'desktop';
const isMirror = process.env.NEXT_PUBLIC_PLATFORM === 'mirror';
const isDevBuild = process.env.NEXT_PUBLIC_DEVBUILD === 'true';
const internalHost = process.env.TAURI_DEV_HOST || '127.0.0.1';

// Allow cross-origin requests from IP addresses
const allowedOrigins = [
    '10.147.20.1',
    'localhost',
    '127.0.0.1',
    '0.0.0.0',
];

/** @type {import('next').NextConfig} */
const nextConfig = {
    output: "export",
    distDir: isDesktop ? (isDevBuild ? "../web-desktop" : "out-desktop") : undefined,
    cleanDistDir: true,
    reactStrictMode: false,
    images: {
        unoptimized: true,
    },
    transpilePackages: ["@uiw/react-textarea-code-editor", "@replit/codemirror-vscode-keymap"],
    // Configure assetPrefix or else the server won't properly resolve your assets.
    assetPrefix: isProd ? undefined : (isDesktop ? `http://${internalHost}:43210` : undefined),
    experimental: {
        reactCompiler: true,
        // Add allowed origins for cross-origin requests
        allowedDevOrigins: allowedOrigins.map(origin => `http://${origin}`),
    },
    devIndicators: false,
}

module.exports = nextConfig
