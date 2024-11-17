// status/src/pages/Landing.jsx
import React from 'react';

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800">
      <nav className="bg-black/30 p-4">
        <div className="max-w-7xl mx-auto flex justify-between items-center">
          <h1 className="text-2xl font-bold text-white">latency.space</h1>
          <div className="flex space-x-4">
            <a href="https://status.latency.space" className="text-white hover:text-blue-400">Status</a>
            <a href="https://docs.latency.space" className="text-white hover:text-blue-400">Docs</a>
            <a href="https://github.com/yourusername/latency-space" className="text-white hover:text-blue-400">GitHub</a>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 py-16">
        <div className="text-center mb-16">
          <h2 className="text-5xl font-bold text-white mb-4">Experience Interplanetary Latency</h2>
          <p className="text-xl text-gray-300">Simulate real-time communication delays across the solar system</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8 mb-16">
          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-2">Planets</h3>
            <p className="text-gray-300 mb-4">Access any planet via their subdomain</p>
            <div className="space-y-2 text-gray-400">
              <code className="block">mars.latency.space</code>
              <code className="block">jupiter.latency.space</code>
              <code className="block">saturn.latency.space</code>
            </div>
          </div>

          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-2">Moons</h3>
            <p className="text-gray-300 mb-4">Include additional delay for major moons</p>
            <div className="space-y-2 text-gray-400">
              <code className="block">phobos.mars.latency.space</code>
              <code className="block">europa.jupiter.latency.space</code>
              <code className="block">titan.saturn.latency.space</code>
            </div>
          </div>

          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-2">Deep Space</h3>
            <p className="text-gray-300 mb-4">Experience spacecraft communication delays</p>
            <div className="space-y-2 text-gray-400">
              <code className="block">voyager1.latency.space</code>
              <code className="block">jwst.latency.space</code>
            </div>
          </div>
        </div>

        <div className="bg-white/10 p-8 rounded-lg mb-16">
          <h3 className="text-2xl font-bold text-white mb-4">Quick Start</h3>
          <div className="space-y-4">
            <div>
              <p className="text-gray-300 mb-2">HTTP Request:</p>
              <code className="block bg-black/30 p-4 rounded">
                curl http://mars.latency.space/
              </code>
            </div>
            <div>
              <p className="text-gray-300 mb-2">UDP Connection:</p>
              <code className="block bg-black/30 p-4 rounded">
                nc -u mars.latency.space 53
              </code>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-4">Current Latencies</h3>
            <div className="space-y-2 text-gray-300">
              <p>Mars: ~3-22 minutes</p>
              <p>Jupiter: ~35-52 minutes</p>
              <p>Saturn: ~68-84 minutes</p>
              <p>Voyager 1: ~21.5 hours</p>
            </div>
          </div>

          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-4">Features</h3>
            <ul className="list-disc list-inside space-y-2 text-gray-300">
              <li>Real-time orbital calculations</li>
              <li>TCP and UDP support</li>
              <li>Speed-of-light delay simulation</li>
              <li>Live metrics and monitoring</li>
              <li>Deep space network constraints</li>
            </ul>
          </div>
        </div>
      </main>

      <footer className="bg-black/30 mt-16 py-8">
        <div className="max-w-7xl mx-auto px-4 text-center text-gray-400">
          <p>Built for space enthusiasts and network engineers</p>
          <p className="mt-2">All latencies based on real astronomical calculations</p>
        </div>
      </footer>
    </div>
  );
}

