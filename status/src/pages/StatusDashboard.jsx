import React, { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { cn } from "@/lib/utils"; // Assuming you have a utility for class names

// Helper function to format latency dynamically
const formatLatency = (seconds) => {
  if (seconds < 60) {
    return `${seconds.toFixed(1)} sec`;
  }
  const minutes = seconds / 60;
  if (minutes < 60) {
    return `${minutes.toFixed(1)} min`;
  }
  const hours = minutes / 60;
  if (hours < 24) {
    return `${hours.toFixed(1)} hours`;
  }
  const days = hours / 24;
  return `${days.toFixed(1)} days`;
};

// Helper function to capitalize type names
const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);

export default function StatusDashboard() {
  const [statusData, setStatusData] = useState({ timestamp: null, objects: {} });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchStatusData = async () => {
      setLoading(true); // Set loading true at the start of fetch
      setError(null);   // Clear previous errors
      try {
        const response = await fetch('/api/status-data');

        if (!response.ok) {
          const errorText = await response.text(); // Read response body as text
          throw new Error(`HTTP error! status: ${response.status}, body: ${errorText}`);
        }

        const contentType = response.headers.get('content-type');
        if (!contentType || !contentType.includes('application/json')) {
          throw new Error(`Expected application/json but received ${contentType}`);
        }

        const data = await response.json(); // Now safe to parse as JSON
        setStatusData(data);
      } catch (e) {
        console.error("Failed to fetch status data:", e.message); // Log the more informative error
        setError(`Failed to load data: ${e.message}`);
      } finally {
        setLoading(false);
      }
    };

    fetchStatusData();
    const interval = setInterval(fetchStatusData, 15000); // Fetch every 15 seconds
    return () => clearInterval(interval);
  }, []);

  // Define the desired order of object types
  const typeOrder = ['planets', 'dwarf_planets', 'moons', 'asteroids', 'spacecraft'];

  // Get sorted object types based on the defined order
  const sortedObjectTypes = Object.keys(statusData.objects || {})
    .filter(type => statusData.objects[type]?.length > 0) // Only include types with objects
    .sort((a, b) => {
      const indexA = typeOrder.indexOf(a);
      const indexB = typeOrder.indexOf(b);
      if (indexA === -1 && indexB === -1) return a.localeCompare(b); // Both not in order, sort alphabetically
      if (indexA === -1) return 1;  // a not in order, comes after
      if (indexB === -1) return -1; // b not in order, comes after
      return indexA - indexB;       // Both in order, sort by index
    });

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 p-8 text-white">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-4xl font-bold mb-4">Solar System Latency Status</h1>
        {statusData.timestamp && (
          <p className="text-sm text-gray-400 mb-8">
            Last updated: {new Date(statusData.timestamp).toLocaleString()}
          </p>
        )}

        {loading && <p>Loading celestial data...</p>}
        {error && <p className="text-red-500">{error}</p>}

        {!loading && !error && sortedObjectTypes.map(objectType => (
          <div key={objectType} className="mb-10">
            <h2 className="text-3xl font-semibold mb-6 border-b border-gray-600 pb-2">
              {capitalize(objectType.replace('_', ' '))}
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
              {(statusData.objects[objectType] || [])
                // Optional: Sort objects within a type, e.g., by distance
                .sort((a, b) => a.distance_km - b.distance_km)
                .map((obj) => (
                  <Card key={obj.name} className={cn(
                    "bg-white/10 text-white border",
                    obj.occluded ? "border-red-600/50" : "border-cyan-600/50"
                  )}>
                    <CardHeader>
                      <CardTitle className="flex justify-between items-center">
                        <span>{obj.name}</span>
                        <span className={cn(
                          "text-xs font-semibold px-2 py-0.5 rounded-full",
                          obj.occluded ? "bg-red-700" : "bg-green-700"
                        )}>
                          {obj.occluded ? 'Occluded' : 'Visible'}
                        </span>
                      </CardTitle>
                      {obj.parentName && (
                         <p className="text-xs text-gray-400 -mt-2">Orbiting {obj.parentName}</p>
                      )}
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        <div>
                          <p className="text-sm text-gray-400">Current Distance</p>
                          <p className="text-xl">{(obj.distance_km / 1e6).toFixed(1)}M km</p>
                        </div>
                        <div>
                          <p className="text-sm text-gray-400">One-Way Latency</p>
                          <p className="text-xl">{formatLatency(obj.latency_seconds)}</p>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
              ))}
            </div>
          </div>
        ))}
        {!loading && !error && sortedObjectTypes.length === 0 && (
           <p>No celestial object data available.</p>
        )}
      </div>
    </div>
  );
}
