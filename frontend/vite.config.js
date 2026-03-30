import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from "path";
var serverHost = process.env.AXIS_SERVER_HOST || "127.0.0.1";
var serverPort = Number(process.env.AXIS_SERVER_PORT || 8787);
export default defineConfig({
    plugins: [tailwindcss(), react()],
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        },
    },
    server: {
        host: process.env.AXIS_UI_HOST || "127.0.0.1",
        port: Number(process.env.AXIS_UI_PORT || 5173),
        proxy: {
            "/api": {
                target: "http://".concat(serverHost, ":").concat(serverPort),
                changeOrigin: true,
            },
        },
    },
});
