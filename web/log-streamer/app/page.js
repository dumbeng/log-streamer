// pages/index.js
import React from 'react';
import dynamic from 'next/dynamic';

// 动态导入LogViewer组件，禁用SSR
const LogViewerWithNoSSR = dynamic(
    () => import('@/components/LogViewer.client'),
    {ssr: false}
);

export default function Home() {
    return (
        <div className="flex flex-col items-center min-h-screen">
            <LogViewerWithNoSSR/>
        </div>
    );
}

