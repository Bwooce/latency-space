// status/src/pages/Landing.jsx
import React, { useState, useEffect, useRef } from 'react';
import RealtimeDashboard from '../components/RealtimeDashboard';

export default function LandingPage() {
  const [celestialData, setCelestialData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState(null);
  const [secondsAgo, setSecondsAgo] = useState(0);
  const lastFetchRef = useRef(Date.now());

  const fetchCelestialData = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/status-data');
      if (!response.ok) throw new Error(`Failed to fetch data: ${response.statusText}`);
      const data = await response.json();
      setLastUpdated(data.timestamp);
      lastFetchRef.current = Date.now();
      setSecondsAgo(0);

      const parsed = [];
      for (const typeKey in data.objects) {
        const type = typeKey.replace(/s$/, '');
        data.objects[typeKey].forEach((entry) => {
          parsed.push({
            name: entry.name,
            distance: entry.distance_km / 1e6, // million km
            latencySeconds: entry.latency_seconds,
            occluded: entry.occluded,
            type,
            parentName: entry.parentName || null,
          });
        });
      }

      parsed.sort((a, b) => {
        const order = { planet: 1, dwarf_planet: 1, moon: 2, asteroid: 3, spacecraft: 4 };
        const ta = order[a.type] || 99;
        const tb = order[b.type] || 99;
        if (a.name.toLowerCase() === 'earth') return -1;
        if (b.name.toLowerCase() === 'earth') return 1;
        if (ta !== tb) return ta - tb;
        if (a.type === 'planet' || a.type === 'dwarf_planet') return a.distance - b.distance;
        return a.name.localeCompare(b.name);
      });

      setCelestialData(parsed);
    } catch (error) {
      console.error('Error fetching celestial data:', error);
      setCelestialData([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCelestialData();
    const poll = setInterval(fetchCelestialData, 20000); // refresh every 20s
    const tick = setInterval(() => {
      setSecondsAgo(Math.round((Date.now() - lastFetchRef.current) / 1000));
    }, 1000);
    return () => {
      clearInterval(poll);
      clearInterval(tick);
    };
  }, []);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      {/* animated backdrop */}
      <div
        className="pointer-events-none fixed inset-0 -z-10 opacity-70"
        style={{
          background:
            'radial-gradient(1200px 600px at 70% -10%, rgba(56,189,248,0.10), transparent 60%),' +
            'radial-gradient(900px 500px at 10% 110%, rgba(168,85,247,0.10), transparent 60%),' +
            'linear-gradient(180deg, #020617 0%, #0b1220 100%)',
        }}
      />

      <nav className="border-b border-white/5 bg-black/30 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
          <h1 className="flex items-center gap-2 text-xl font-bold text-white">
            <span className="h-2.5 w-2.5 rounded-full bg-cyan-400 animate-live" />
            latency.space
          </h1>
          <a href="https://github.com/Bwooce/latency-space" className="text-sm text-slate-300 hover:text-cyan-400">GitHub</a>
        </div>
      </nav>

      <main className="mx-auto max-w-7xl px-4 py-14">
        <div className="mb-12 text-center">
          <h2 className="mx-auto max-w-4xl bg-gradient-to-b from-white to-slate-400 bg-clip-text text-4xl font-bold text-transparent md:text-5xl">
            Experience Real-Time Space Communication Delays
          </h2>
          <p className="mx-auto mt-4 max-w-2xl text-lg text-slate-400">
            Explore the solar system by feeling the actual light-travel delay to every planet, moon, and
            spacecraft — through HTTP and SOCKS5 proxies, computed from live orbital mechanics.
          </p>
        </div>

        <div className="mb-16">
          <RealtimeDashboard
            data={celestialData}
            lastUpdated={lastUpdated}
            loading={loading}
            secondsAgo={secondsAgo}
            onRefresh={fetchCelestialData}
          />
        </div>

        {/* ===== Usage guides ===== */}
        <div className="mb-16 rounded-2xl border border-white/10 bg-white/[0.03] p-8">
          <h3 className="mb-6 text-2xl font-bold text-white">How to Use latency.space</h3>

          <div className="space-y-8">
            <div>
              <h4 className="mb-3 text-xl font-bold text-white">HTTP Proxy</h4>
              <p className="mb-4 text-slate-400">Experience interplanetary latency when browsing the web or making HTTP requests.</p>
              <div className="space-y-4">
                <Guide label="Direct Domain Format" code="http://mars.latency.space/" note="Adds Mars-to-Earth latency to any destination in your request." />
                <Guide label="Target Domain Format" code="http://example.com.mars.latency.space/" note="Routes to example.com with Mars-to-Earth latency." />
                <Guide label="Curl Example" code="curl -x mars.latency.space:80 https://example.com" note="Use as an HTTP proxy with curl." />
              </div>
            </div>

            <div>
              <h4 className="mb-3 text-xl font-bold text-white">SOCKS5 Proxy</h4>
              <p className="mb-4 text-slate-400">Use the SOCKS5 proxy (port 1080) for TCP connections. UDP is supported via UDP ASSOCIATE.</p>
              <div className="space-y-4">
                <Guide label="With curl" code="curl --socks5-hostname neptune.latency.space:1080 https://example.com" note="Feel Neptune's multi-hour round trip when loading a site." />
                <Guide label="Browser (SOCKS5)" code="Host: jupiter.latency.space   Port: 1080   Type: SOCKS5" note="Configure your browser's SOCKS proxy settings." />
                <Guide label="UDP (netcat)" code={'echo "hi" | nc -u -X 5 -x mars.latency.space:1080 1.1.1.1 53'} note="Send a UDP packet to 1.1.1.1:53 via Mars." />
              </div>
            </div>

            <div>
              <h4 className="mb-3 text-xl font-bold text-white">Debug &amp; Info Endpoints</h4>
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <Endpoint title="Distances" url="/_debug/distances" note="Current distances from Earth to all bodies." />
                <Endpoint title="Allowed Hosts" url="/_debug/allowed-hosts" note="The destination allowlist (hosts and ports)." />
                <Endpoint title="Live Status Data" url="/api/status-data" note="The JSON feed powering this dashboard." />
                <Endpoint title="Help" url="/_debug/help" note="Usage instructions and examples." />
              </div>
            </div>
          </div>
        </div>

        <div className="mb-16 grid grid-cols-1 gap-6 md:grid-cols-3">
          <UseCase title="Education" body="Show why real-time control of Mars rovers isn't possible — students feel the delay firsthand." />
          <UseCase title="Software Testing" body="Test application robustness under extreme latency; surface bugs from multi-minute or hour-long delays." />
          <UseCase title="Research" body="Develop and evaluate protocols for high-latency, delay-tolerant networks." />
        </div>
      </main>

      <footer className="border-t border-white/5 bg-black/30 py-8">
        <div className="mx-auto max-w-7xl px-4 text-center text-sm text-slate-500">
          <p>Latencies computed from real astronomical calculations (Kepler's laws, J2000 elements).</p>
          <p className="mt-1">Dashboard refreshes every 20 seconds; orbital positions update continuously.</p>
        </div>
      </footer>
    </div>
  );
}

function Guide({ label, code, note }) {
  return (
    <div>
      <p className="mb-1.5 text-sm text-slate-300">{label}:</p>
      <code className="block overflow-x-auto rounded-lg border border-white/10 bg-black/40 p-3 text-sm text-cyan-200">{code}</code>
      <p className="mt-1 text-xs text-slate-500">{note}</p>
    </div>
  );
}

function Endpoint({ title, url, note }) {
  return (
    <a href={url} className="block rounded-lg border border-white/10 bg-black/30 p-4 transition-colors hover:border-cyan-400/40">
      <p className="font-semibold text-white">{title}</p>
      <code className="mt-1 block text-sm text-cyan-300">{url}</code>
      <p className="mt-1 text-xs text-slate-500">{note}</p>
    </a>
  );
}

function UseCase({ title, body }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.03] p-5">
      <h4 className="font-bold text-white">{title}</h4>
      <p className="mt-2 text-sm text-slate-400">{body}</p>
    </div>
  );
}
