const staticExport = process.env.MEMOIR_STATIC_EXPORT === "1";

/** @type {import('next').NextConfig} */
const nextConfig = {
  ...(staticExport ? { output: "export" } : {}),
  experimental: {
    proxyClientMaxBodySize: "256mb",
  },
  ...(staticExport
    ? {}
    : {
        async rewrites() {
          return [
            {
              source: "/api/:path*",
              destination: "http://127.0.0.1:8090/api/:path*",
            },
            {
              source: "/media/:path*",
              destination: "http://127.0.0.1:8090/media/:path*",
            },
          ];
        },
      }),
};

export default nextConfig;
