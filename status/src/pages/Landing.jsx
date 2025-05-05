// status/src/pages/Landing.jsx
import React, { useState, useEffect } from 'react';
import { formatFullDomain, formatMoonDomain } from '../lib/domainUtils';

// Helper function to format latency from seconds with more readable units
const formatLatency = (seconds) => {
  if (seconds < 0) return 'N/A'; // Handle potential negative values if any
  
  if (seconds < 60) {
    // Less than a minute: show seconds
    return `${seconds.toFixed(2)} seconds`;
  } else if (seconds < 3600) {
    // Less than an hour: show minutes and seconds
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.round(seconds % 60);
    return remainingSeconds > 0 
      ? `${minutes} min ${remainingSeconds} sec` 
      : `${minutes} min`;
  } else if (seconds < 86400) {
    // Less than a day: show hours and minutes
    const hours = Math.floor(seconds / 3600);
    const remainingMinutes = Math.round((seconds % 3600) / 60);
    return remainingMinutes > 0 
      ? `${hours} hr ${remainingMinutes} min` 
      : `${hours} hr`;
  } else {
    // Days and hours
    const days = Math.floor(seconds / 86400);
    const remainingHours = Math.round((seconds % 86400) / 3600);
    return remainingHours > 0 
      ? `${days} day${days !== 1 ? 's' : ''} ${remainingHours} hr` 
      : `${days} day${days !== 1 ? 's' : ''}`;
  }
};


export default function LandingPage() {
  const [celestialData, setCelestialData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState(null);

  // Function to fetch celestial body data from the new API endpoint
  const fetchCelestialData = async () => {
    try {
      setLoading(true);
      // Fetch from the new /api/status-data endpoint
      const response = await fetch('/api/status-data');
      if (!response.ok) {
        throw new Error(`Failed to fetch data: ${response.statusText}`);
      }

      const data = await response.json(); // Parse JSON response

      setLastUpdated(data.timestamp);

      // Process the structured JSON data
      const parsedData = [];
      for (const typeKey in data.objects) {
        const type = typeKey.replace(/s$/, ''); // Remove plural 's' (e.g., "planets" -> "planet")
        data.objects[typeKey].forEach(entry => {
          parsedData.push({
            name: entry.name, // API uses lowercase 'name'
            distance: entry.distance_km / 1e6, // Convert km to million km
            latencySeconds: entry.latency_seconds, // Keep raw seconds if needed elsewhere
            latency: formatLatency(entry.latency_seconds), // Format for display
            occluded: entry.occluded,
            type: type, // Store the object type
            parentName: entry.parentName || null // Store parent name if available
          });
        });
      }

      // Sort data: Earth first, then planets by distance, then moons alphabetically, then spacecraft alphabetically
      parsedData.sort((a, b) => {
        const typeOrder = { 'planet': 1, 'dwarf_planet': 1, 'moon': 2, 'asteroid': 3, 'spacecraft': 4 };
        const typeA = typeOrder[a.type] || 99;
        const typeB = typeOrder[b.type] || 99;

        if (a.name.toLowerCase() === 'earth') return -1;
        if (b.name.toLowerCase() === 'earth') return 1;

        if (typeA !== typeB) return typeA - typeB;

        // If same type, sort planets by distance
        if (a.type === 'planet' || a.type === 'dwarf_planet') {
          return a.distance - b.distance;
        }

        // Alphabetical sort for others of the same type
        return a.name.localeCompare(b.name);
      });


      setCelestialData(parsedData);
    } catch (error) {
      console.error('Error fetching celestial data:', error);
      // Consider setting an error state to display to the user
      setCelestialData([]); // Clear data on error
      // generateMockData(); // Removed mock data generation
    } finally {
      setLoading(false);
    }
  };

  // Removed generateMockData function

  useEffect(() => {
    // Initial fetch
    fetchCelestialData();

    // Set up interval for periodic updates
    const interval = setInterval(fetchCelestialData, 60000); // Refresh every minute

    // Clean up interval on component unmount
    return () => clearInterval(interval);
  }, []);

  // Group celestial bodies by type using the new 'type' field
  const planets = celestialData.filter(item => item.type === 'planet' || item.type === 'dwarf_planet');
  const moons = celestialData.filter(item => item.type === 'moon');
  const spacecraft = celestialData.filter(item => item.type === 'spacecraft');
  // Add other types like asteroids if needed
  const others = celestialData.filter(item => !['planet', 'dwarf_planet', 'moon', 'spacecraft'].includes(item.type));


  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800">
      <nav className="bg-black/30 p-4">
        <div className="max-w-7xl mx-auto flex justify-between items-center">
          <h1 className="text-2xl font-bold text-white">latency.space</h1>
          <div className="flex space-x-4">
            {/* Status link removed - now integrated with main site */}
            <a href="https://github.com/Bwooce/latency-space" className="text-white hover:text-blue-400">GitHub</a>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 py-16">
        <div className="text-center mb-16">
          <h2 className="text-5xl font-bold text-white mb-4">latency.space: Simulating Real-Time Space Communication Delays</h2>
          <p className="text-xl text-gray-300">Explore the vast distances of our solar system by experiencing the real communication delays to planets, moons, and spacecraft through HTTP and SOCKS5 proxies.</p>
        </div>

        {/* Current Distances Dashboard */}
        <div className="bg-white/10 p-6 rounded-lg mb-16">
          <div className="flex justify-between items-center mb-6">
            <h3 className="text-2xl font-bold text-white">Real-Time Solar System Status</h3>
            <div className="text-gray-400 text-sm">
              {lastUpdated && (
                <p>Last updated: {new Date(lastUpdated).toLocaleString()}</p>
              )}
              <button
                onClick={fetchCelestialData}
                className="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-xs ml-2 disabled:opacity-50"
                disabled={loading}
              >
                {loading ? 'Refreshing...' : 'Refresh'}
              </button>
            </div>
          </div>

          {loading && celestialData.length === 0 ? ( // Show loading only on initial load
            <div className="text-center py-8 text-gray-300">Loading celestial data...</div>
          ) : celestialData.length === 0 && !loading ? ( // Show error if fetch failed and not loading
             <div className="text-center py-8 text-red-400">Failed to load celestial data. Please try refreshing.</div>
          ) : (
            <div className="space-y-8">
              {/* Planets */}
              {planets.length > 0 && (
                <div>
                  <h4 className="text-xl font-bold text-white mb-4">Planets & Dwarf Planets</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {planets.map(item => ( // Changed variable name to item
                      <div key={item.name} className="bg-black/30 p-4 rounded-lg">
                        <h5 className="text-lg font-bold text-white capitalize">{item.name}</h5>
                        <div className="mt-2 space-y-1"> {/* Reduced spacing */}
                          <p className="text-gray-300 text-sm">Distance: <span className="text-blue-400">{item.distance.toFixed(2)} million km</span></p>
                          <p className="text-gray-300 text-sm">Latency: <span className="text-yellow-400">{item.latency}</span></p>
                           <p className={`text-sm ${item.occluded ? 'text-red-400' : 'text-green-400'}`}>
                             Status: {item.occluded ? 'Occluded' : 'Visible'}
                           </p>
                          <p className="text-gray-300 text-xs pt-1">Domain: <a href={`http://${formatFullDomain(item.name)}/`} target="_blank" rel="noopener noreferrer" className="text-cyan-400 hover:underline"><code className="bg-black/50 px-1 py-0.5 rounded text-xs">{formatFullDomain(item.name)}</code></a></p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Moons */}
              {moons.length > 0 && (
                <div>
                  <h4 className="text-xl font-bold text-white mb-4">Moons</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {moons.map(item => ( // Changed variable name to item
                      <div key={item.name} className="bg-black/30 p-4 rounded-lg">
                        {/* Display parent name if available */}
                        <h5 className="text-lg font-bold text-white capitalize">
                          {item.name}{item.parentName ? ` (${item.parentName})` : ''}
                        </h5>
                        <div className="mt-2 space-y-1"> {/* Reduced spacing */}
                          <p className="text-gray-300 text-sm">Distance: <span className="text-blue-400">{item.distance.toFixed(3)} million km</span></p>
                          <p className="text-gray-300 text-sm">Latency: <span className="text-yellow-400">{item.latency}</span></p>
                          <p className={`text-sm ${item.occluded ? 'text-red-400' : 'text-green-400'}`}>
                            Status: {item.occluded ? 'Occluded' : 'Visible'}
                          </p>
                           {/* Construct domain name based on parent */}
                          <p className="text-gray-300 text-xs pt-1">Domain: <a href={`http://${item.parentName ? formatMoonDomain(item.name, item.parentName) : formatFullDomain(item.name)}/`} target="_blank" rel="noopener noreferrer" className="text-cyan-400 hover:underline"><code className="bg-black/50 px-1 py-0.5 rounded text-xs">
                            {item.parentName ? formatMoonDomain(item.name, item.parentName) : formatFullDomain(item.name)}
                          </code></a></p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Spacecraft */}
              {spacecraft.length > 0 && (
                <div>
                  <h4 className="text-xl font-bold text-white mb-4">Spacecraft</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {spacecraft.map(item => ( // Changed variable name to item
                      <div key={item.name} className="bg-black/30 p-4 rounded-lg">
                        <h5 className="text-lg font-bold text-white capitalize">{item.name}</h5>
                        <div className="mt-2 space-y-1"> {/* Reduced spacing */}
                          <p className="text-gray-300 text-sm">Distance: <span className="text-blue-400">{item.distance.toFixed(2)} million km</span></p>
                          <p className="text-gray-300 text-sm">Latency: <span className="text-yellow-400">{item.latency}</span></p>
                           <p className={`text-sm ${item.occluded ? 'text-red-400' : 'text-green-400'}`}>
                             Status: {item.occluded ? 'Occluded' : 'Visible'}
                           </p>
                          <p className="text-gray-300 text-xs pt-1">Domain: <a href={`http://${formatFullDomain(item.name)}/`} target="_blank" rel="noopener noreferrer" className="text-cyan-400 hover:underline"><code className="bg-black/50 px-1 py-0.5 rounded text-xs">{formatFullDomain(item.name)}</code></a></p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

               {/* Others (e.g., Asteroids) */}
              {others.length > 0 && (
                <div>
                  <h4 className="text-xl font-bold text-white mb-4">Other Objects</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {others.map(item => (
                      <div key={item.name} className="bg-black/30 p-4 rounded-lg">
                        <h5 className="text-lg font-bold text-white capitalize">{item.name} ({item.type})</h5>
                        <div className="mt-2 space-y-1"> {/* Reduced spacing */}
                          <p className="text-gray-300 text-sm">Distance: <span className="text-blue-400">{item.distance.toFixed(2)} million km</span></p>
                          <p className="text-gray-300 text-sm">Latency: <span className="text-yellow-400">{item.latency}</span></p>
                           <p className={`text-sm ${item.occluded ? 'text-red-400' : 'text-green-400'}`}>
                             Status: {item.occluded ? 'Occluded' : 'Visible'}
                           </p>
                          <p className="text-gray-300 text-xs pt-1">Domain: <a href={`http://${formatFullDomain(item.name)}/`} target="_blank" rel="noopener noreferrer" className="text-cyan-400 hover:underline"><code className="bg-black/50 px-1 py-0.5 rounded text-xs">{formatFullDomain(item.name)}</code></a></p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Usage Guides */}
        <div className="bg-white/10 p-8 rounded-lg mb-16">
          <h3 className="text-2xl font-bold text-white mb-6">How to Use latency.space</h3>

          <div className="space-y-8">
            <div>
              <h4 className="text-xl font-bold text-white mb-3">HTTP Proxy</h4>
              <p className="text-gray-300 mb-4">Experience interplanetary latency when browsing the web or making HTTP requests.</p>

              <div className="space-y-4">
                <div>
                  <p className="text-gray-300 mb-2">1. Direct Domain Format:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    http://mars.latency.space/
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Adds Mars-to-Earth latency to any destination specified in your request.</p>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">2. Target Domain Format:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    http://example.com.mars.latency.space/
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Routes to example.com with Mars-to-Earth latency.</p>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">3. Query Parameter:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    http://mars.latency.space/?destination=https://example.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Specify destination in the query string parameter.</p>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">4. Curl Example:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    curl -x mars.latency.space:80 https://example.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Use as an HTTP proxy with curl.</p>
                </div>
              </div>
            </div>

            <div>
              <h4 className="text-xl font-bold text-white mb-3">SOCKS5 Proxy</h4>
              <p className="text-gray-300 mb-4">Use the SOCKS5 proxy (port 1080) for TCP connections. UDP traffic is also supported via the `UDP ASSOCIATE` command on the same host/port.</p>

              <div className="space-y-4">
                <div>
                  <p className="text-gray-300 mb-2">1. Basic TCP Usage (SSH):</p>
                  <code className="block bg-black/30 p-4 rounded">
                    ssh -o ProxyCommand="nc -X 5 -x mars.latency.space:1080 %h %p" destination.server.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Connect to an SSH server through Mars latency.</p>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">2. TCP With Curl:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    curl --socks5 neptune.latency.space:1080 https://example.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Experience Neptune's ~4 hour round-trip latency when accessing a website via TCP.</p>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">3. Browser Configuration (TCP):</p>
                  <p className="text-gray-400">Configure your browser's SOCKS proxy settings:</p>
                  <ul className="list-disc list-inside text-gray-400 ml-4">
                    <li>Host: jupiter.latency.space</li>
                    <li>Port: 1080</li>
                    <li>Type: SOCKS5</li>
                  </ul>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">4. Target Domain Format (TCP):</p>
                  <code className="block bg-black/30 p-4 rounded">
                    ssh -o ProxyCommand="nc -X 5 -x example.com.mars.latency.space:1080 %h %p" destination.server.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Combined domain format also works with SOCKS proxy for TCP.</p>
                </div>

                <div>
                  <p className="text-gray-300 mb-2">5. UDP Example (using netcat):</p>
                  <code className="block bg-black/30 p-4 rounded">
                    echo "hello" | nc -u -X 5 -x mars.latency.space:1080 1.1.1.1 53
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Send a UDP packet to 1.1.1.1:53 via Mars. Requires a netcat version supporting SOCKS5 UDP.</p>
                </div>
              </div>
            </div>

            <div>
              <h4 className="text-xl font-bold text-white mb-3">Debug and Information Endpoints</h4>
              <p className="text-gray-300 mb-4">Access detailed information about the system.</p>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="bg-black/30 p-4 rounded">
                  <p className="text-white font-bold">Distance Information</p>
                  <code className="block text-gray-400 mt-2">http://latency.space/_debug/distances</code>
                  <p className="text-gray-400 text-sm mt-1">Current distances from Earth to all bodies</p>
                </div>

                <div className="bg-black/30 p-4 rounded">
                  <p className="text-white font-bold">Detailed Body Information</p>
                  <code className="block text-gray-400 mt-2">http://latency.space/_debug/bodies</code>
                  <p className="text-gray-400 text-sm mt-1">Complete details of all celestial bodies</p>
                </div>

                <div className="bg-black/30 p-4 rounded">
                  <p className="text-white font-bold">Valid Domains</p>
                  <code className="block text-gray-400 mt-2">http://latency.space/_debug/domains</code>
                  <p className="text-gray-400 text-sm mt-1">List of valid domain formats</p>
                </div>

                <div className="bg-black/30 p-4 rounded">
                  <p className="text-white font-bold">Help Information</p>
                  <code className="block text-gray-400 mt-2">http://latency.space/_debug/help</code>
                  <p className="text-gray-400 text-sm mt-1">Usage instructions and examples</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Grid for Advanced Usage and System Features */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mb-16">
          {/* Column 1: Advanced Usage */}
          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-4">Advanced Usage</h3>
            {/* Content for Advanced Usage */}
            <div>
              <h4 className="font-bold">Custom Applications</h4>
              <p className="text-sm">Configure applications to use SOCKS5 proxy:</p>
              <ul className="list-disc list-inside text-sm">
                <li>VoIP applications</li>
                <li>Messaging applications</li>
                <li>Gaming (for the ultimate challenge)</li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold">Spacecraft Tracking</h4>
              <p className="text-sm">Distances to spacecraft update in real-time based on actual trajectories</p>
              <p className="text-gray-400 text-xs mt-1">Try Voyager 1 for the ultimate latency experience (~40+ hours round trip!)</p>
            </div>
          </div> {/* End Column 1 */}

          {/* Column 2: System Features */}
          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-4">System Features</h3>
            {/* Content for System Features */}
            <ul className="list-disc list-inside space-y-3 text-gray-300">
              <li>Accurate astronomical calculations using Kepler's laws</li>
              <li>Real-time orbital positions based on JPL Horizons data</li>
              <li>Support for all planets, major moons, and spacecraft</li>
              <li>HTTP and SOCKS5 proxy interfaces</li>
              <li>Documentation and debugging endpoints</li>
              <li>Continuous integration and deployment</li>
              <li>Open source project on GitHub</li>
            </ul>
          </div> {/* End Column 2 */}
        </div> {/* End Grid */}


        <div className="bg-white/10 p-6 rounded-lg mb-16">
          <h3 className="text-xl font-bold text-white mb-4">Use Cases</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-black/30 p-4 rounded-lg">
              <h4 className="font-bold text-white">Education</h4>
              <p className="text-gray-300 text-sm mt-2">Visually demonstrate space communication challenges. Show students why real-time control of Mars rovers isn't possible.</p>
            </div>

            <div className="bg-black/30 p-4 rounded-lg">
              <h4 className="font-bold text-white">Software Testing</h4>
              <p className="text-gray-300 text-sm mt-2">Test application robustness under high-latency network conditions. Identify issues caused by multi-minute or hour-long delays.</p>
            </div>

            <div className="bg-black/30 p-4 rounded-lg">
              <h4 className="font-bold text-white">Research</h4>
              <p className="text-gray-300 text-sm mt-2">Develop and evaluate new protocols and communication strategies for high-latency, delay-tolerant networks.</p>
            </div>
          </div>
        </div>
      </main>

      {/* Footer Section */}
      <footer className="bg-black/30 mt-16 py-8">
        <div className="max-w-7xl mx-auto px-4 text-center text-gray-400">
          <p>Built for space enthusiasts and network engineers</p>
          <p className="mt-2">All latencies based on real astronomical calculations</p>
          <p className="mt-4 text-sm">Data updates hourly based on actual orbital mechanics</p>
        </div>
      </footer>
      {/* End Footer Section */}
    </div>
  );
}
