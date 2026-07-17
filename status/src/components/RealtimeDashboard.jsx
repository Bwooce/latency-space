// status/src/components/RealtimeDashboard.jsx
//
// Animated, realtime view of the solar-system latency data. Values tween
// smoothly between polls, and each body shows a "signal in transit" animation
// whose speed is proportional to its light-travel latency — so the delay you
// are simulating is something you can watch, not just read.
import React, { useState, useEffect, useRef } from 'react';
import { formatFullDomain, formatMoonDomain } from '../lib/domainUtils';

// ---- helpers ---------------------------------------------------------------

export const formatLatency = (seconds) => {
  if (seconds == null || seconds < 0) return 'N/A';
  if (seconds < 60) return `${seconds.toFixed(2)} s`;
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60);
    const s = Math.round(seconds % 60);
    return s > 0 ? `${m}m ${s}s` : `${m}m`;
  }
  if (seconds < 86400) {
    const h = Math.floor(seconds / 3600);
    const m = Math.round((seconds % 3600) / 60);
    return m > 0 ? `${h}h ${m}m` : `${h}h`;
  }
  const d = Math.floor(seconds / 86400);
  const h = Math.round((seconds % 86400) / 3600);
  return h > 0 ? `${d}d ${h}h` : `${d}d`;
};

// Where the body sits on its track, as a fraction [0..1] of the full width,
// log-scaled from 1s to 24h of one-way latency. Near bodies sit close to
// Earth; distant ones sit far to the right. This is what makes the photon
// "representative": the signal covers a proportionally longer path for a
// more-distant body.
const trackFraction = (latencySeconds) => {
  const min = Math.log10(1);
  const max = Math.log10(86400); // 24h
  const v = Math.log10(Math.max(latencySeconds, 1));
  const f = (v - min) / (max - min);
  return Math.min(Math.max(f, 0.12), 0.96); // keep both endpoints on-screen
};

// Seconds for a photon to cross the FULL track width. The photon always moves
// at this same visual speed (light is constant); a body's travel time follows
// only from how far away it sits (trackFraction), so the delay is honest —
// distant bodies genuinely take longer because the signal has farther to go.
const CROSS_SECONDS = 7;

// Colour by latency magnitude: near = green, far = red.
const latencyColor = (seconds) => {
  if (seconds < 10) return { text: 'text-emerald-300', dot: 'bg-emerald-400', bar: 'from-emerald-500 to-emerald-300' };
  if (seconds < 120) return { text: 'text-yellow-300', dot: 'bg-yellow-400', bar: 'from-yellow-500 to-yellow-300' };
  if (seconds < 1800) return { text: 'text-orange-300', dot: 'bg-orange-400', bar: 'from-orange-500 to-orange-300' };
  return { text: 'text-rose-300', dot: 'bg-rose-400', bar: 'from-rose-500 to-rose-300' };
};

// Fraction [0..1] of a log scale from 1s to 24h, for the latency bar width.
const latencyFraction = (seconds) => {
  const min = Math.log10(1);
  const max = Math.log10(86400);
  const v = Math.log10(Math.max(seconds, 1));
  return Math.min(Math.max((v - min) / (max - min), 0.04), 1);
};

// ---- animated number -------------------------------------------------------

function useTween(target, duration = 800) {
  const [value, setValue] = useState(target);
  const fromRef = useRef(target);
  const startRef = useRef(null);
  const rafRef = useRef(null);

  useEffect(() => {
    if (target === value) return;
    fromRef.current = value;
    startRef.current = null;
    const from = fromRef.current;
    const delta = target - from;

    const step = (ts) => {
      if (startRef.current === null) startRef.current = ts;
      const t = Math.min((ts - startRef.current) / duration, 1);
      const eased = 1 - Math.pow(1 - t, 3); // easeOutCubic
      setValue(from + delta * eased);
      if (t < 1) rafRef.current = requestAnimationFrame(step);
    };
    rafRef.current = requestAnimationFrame(step);
    return () => cancelAnimationFrame(rafRef.current);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target, duration]);

  return value;
}

function AnimatedNumber({ value, decimals = 2, suffix = '' }) {
  const tweened = useTween(value);
  return <span>{tweened.toFixed(decimals)}{suffix}</span>;
}

// ---- signal-in-transit track ----------------------------------------------

function SignalTrack({ latencySeconds, occluded }) {
  const f = trackFraction(latencySeconds);
  // Constant photon speed: crossing time is proportional to the distance (f).
  const dur = Math.max(f * CROSS_SECONDS, 0.4);
  const pct = `${(f * 100).toFixed(1)}%`;
  return (
    <div className="relative h-6 mt-3 mb-1">
      {/* faint full-width guide */}
      <div className="absolute top-1/2 left-0 right-0 h-px bg-white/5" />
      {/* active beam: Earth -> body (length = distance) */}
      <div className="absolute top-1/2 left-0 h-px bg-gradient-to-r from-cyan-500/50 to-slate-500/20" style={{ width: pct }} />
      {/* Earth endpoint */}
      <div className="absolute top-1/2 -translate-y-1/2 left-0 w-2.5 h-2.5 rounded-full bg-sky-400 shadow-[0_0_8px_2px_rgba(56,189,248,0.6)]" title="Earth" />
      {/* body endpoint at its distance */}
      <div
        className={`absolute top-1/2 -translate-x-1/2 -translate-y-1/2 w-2.5 h-2.5 rounded-full ${occluded ? 'bg-rose-500 animate-occ' : 'bg-amber-300'} shadow-[0_0_8px_2px_rgba(252,211,77,0.5)]`}
        style={{ left: pct }}
        title="Body"
      />
      {/* travelling signal — moves within the beam (width f) at constant speed;
          hidden when occluded (no line of sight) */}
      {!occluded && (
        <div className="absolute top-1/2 -translate-y-1/2 left-0" style={{ width: pct }}>
          <div
            className="animate-signal absolute top-0 -translate-y-1/2 w-1.5 h-1.5 rounded-full bg-white shadow-[0_0_10px_3px_rgba(255,255,255,0.85)]"
            style={{ animationDuration: `${dur}s` }}
          />
        </div>
      )}
    </div>
  );
}

// ---- body card -------------------------------------------------------------

function BodyCard({ item, index }) {
  const c = latencyColor(item.latencySeconds);
  // Only moons use the parent-qualified domain (e.g. phobos.mars.latency.space).
  // Planets/dwarf planets/spacecraft carry parentName "Sun" but address as
  // <name>.latency.space, so don't treat their parent as a domain level.
  const isMoon = item.type === 'moon';
  const domain = isMoon ? formatMoonDomain(item.name, item.parentName) : formatFullDomain(item.name);
  const barPct = (latencyFraction(item.latencySeconds) * 100).toFixed(1);

  return (
    <div
      className="animate-card-in group relative rounded-xl border border-white/10 bg-slate-900/60 p-4 backdrop-blur transition-all hover:border-cyan-400/40 hover:bg-slate-900/80 hover:shadow-[0_0_24px_-6px_rgba(34,211,238,0.35)]"
      style={{ animationDelay: `${Math.min(index * 30, 400)}ms` }}
    >
      <div className="flex items-baseline justify-between">
        <h5 className="text-lg font-semibold text-white capitalize">
          {item.name}
          {isMoon && item.parentName && <span className="ml-1 text-xs font-normal text-slate-400">/ {item.parentName}</span>}
        </h5>
        <span className={`inline-flex items-center gap-1 text-[11px] ${item.occluded ? 'text-rose-300' : 'text-emerald-300'}`}>
          <span className={`h-1.5 w-1.5 rounded-full ${item.occluded ? 'bg-rose-400 animate-occ' : 'bg-emerald-400'}`} />
          {item.occluded ? 'occluded' : 'visible'}
        </span>
      </div>

      <SignalTrack latencySeconds={item.latencySeconds} occluded={item.occluded} />

      <div className="mt-2 flex items-end justify-between">
        <div>
          <div className="text-[11px] uppercase tracking-wide text-slate-400">one-way latency</div>
          <div className={`font-mono text-xl font-bold ${c.text}`}>{formatLatency(item.latencySeconds)}</div>
        </div>
        <div className="text-right">
          <div className="text-[11px] uppercase tracking-wide text-slate-400">distance</div>
          <div className="font-mono text-sm text-sky-300">
            <AnimatedNumber value={item.distance} decimals={item.distance < 10 ? 3 : 2} /> M km
          </div>
        </div>
      </div>

      {/* latency magnitude bar */}
      <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-white/5">
        <div className={`h-full rounded-full bg-gradient-to-r ${c.bar} transition-all duration-700`} style={{ width: `${barPct}%` }} />
      </div>

      <a
        href={`http://${domain}/`}
        target="_blank"
        rel="noopener noreferrer"
        className="mt-3 block truncate text-xs text-slate-500 transition-colors group-hover:text-cyan-400"
        title={domain}
      >
        <code>{domain}</code>
      </a>
    </div>
  );
}

// ---- section + dashboard ---------------------------------------------------

function Section({ title, items }) {
  if (!items.length) return null;
  return (
    <div>
      <h4 className="mb-3 flex items-center gap-2 text-lg font-semibold text-white">
        {title}
        <span className="rounded-full bg-white/10 px-2 py-0.5 text-xs font-normal text-slate-300">{items.length}</span>
      </h4>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {items.map((item, i) => <BodyCard key={item.name} item={item} index={i} />)}
      </div>
    </div>
  );
}

export default function RealtimeDashboard({ data, lastUpdated, loading, onRefresh, secondsAgo, stale }) {
  const planets = data.filter((d) => d.type === 'planet' || d.type === 'dwarf_planet');
  const moons = data.filter((d) => d.type === 'moon');
  const spacecraft = data.filter((d) => d.type === 'spacecraft');
  const others = data.filter((d) => !['planet', 'dwarf_planet', 'moon', 'spacecraft'].includes(d.type));

  const haveData = data.length > 0;
  // Reconnecting = a poll failed but we still have the last good snapshot to show.
  const reconnecting = stale && haveData;

  return (
    <div className="rounded-2xl border border-white/10 bg-gradient-to-b from-slate-900/70 to-slate-950/70 p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h3 className="flex items-center gap-3 text-2xl font-bold text-white">
          Real-Time Solar System Status
          {reconnecting ? (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-amber-500/15 px-2.5 py-1 text-xs font-medium text-amber-300">
              <span className="h-2 w-2 rounded-full bg-amber-400 animate-live" />
              RECONNECTING
            </span>
          ) : (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-300">
              <span className="h-2 w-2 rounded-full bg-emerald-400 animate-live" />
              LIVE
            </span>
          )}
        </h3>
        <div className="flex items-center gap-3 text-sm text-slate-400">
          <span>{lastUpdated ? `updated ${secondsAgo}s ago` : 'connecting…'}</span>
          <button
            onClick={onRefresh}
            disabled={loading}
            className="rounded-md bg-cyan-600/80 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-cyan-500 disabled:opacity-50"
          >
            {loading ? 'refreshing…' : 'refresh'}
          </button>
        </div>
      </div>

      {loading && !haveData ? (
        <div className="py-10 text-center text-slate-300">Acquiring signal…</div>
      ) : !haveData ? (
        <div className="py-10 text-center text-rose-400">Failed to load celestial data. Try refreshing.</div>
      ) : (
        // Once we have a snapshot we keep showing it, even if a later poll fails
        // (e.g. during a deploy) — the RECONNECTING badge signals the staleness
        // instead of blanking the whole dashboard.
        <div className={`space-y-8 transition-opacity ${reconnecting ? 'opacity-60' : 'opacity-100'}`}>
          <Section title="Planets &amp; Dwarf Planets" items={planets} />
          <Section title="Moons" items={moons} />
          <Section title="Spacecraft" items={spacecraft} />
          <Section title="Asteroids &amp; Other" items={others} />
        </div>
      )}
    </div>
  );
}
