// status/src/pages/Landing.jsx
import React, { useState, useEffect } from 'react';

// Speed of light in km/s
const SPEED_OF_LIGHT = 299792.458;

export default function LandingPage() {
  const [celestialData, setCelestialData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState(null);
  
  // Function to calculate latency from distance in millions of km
  const calculateLatency = (distanceMillionKm) => {
    const distanceKm = distanceMillionKm * 1e6;
    const seconds = distanceKm / SPEED_OF_LIGHT;
    
    // Format based on magnitude
    if (seconds < 60) {
      return `${seconds.toFixed(2)} seconds`;
    } else if (seconds < 3600) {
      return `${(seconds / 60).toFixed(2)} minutes`;
    } else {
      return `${(seconds / 3600).toFixed(2)} hours`;
    }
  };
  
  // Function to fetch celestial body data
  const fetchCelestialData = async () => {
    try {
      setLoading(true);
      // Use the CORS-enabled proxy endpoint to get the data
      const response = await fetch('/api/debug/distances');
      if (!response.ok) {
        throw new Error('Failed to fetch data');
      }
      
      const text = await response.text();
      
      // Parse the plain text response
      const lines = text.split('\n');
      let timestamp = new Date().toISOString();
      
      // Extract the timestamp if available
      const timestampLine = lines.find(line => line.startsWith('Current Time:'));
      if (timestampLine) {
        timestamp = timestampLine.split('Current Time:')[1].trim();
      }
      
      setLastUpdated(timestamp);
      
      // Parse the distance data
      const data = [];
      const dataLines = lines.filter(line => line.includes('million km'));
      
      dataLines.forEach(line => {
        const parts = line.split(':');
        if (parts.length === 2) {
          const name = parts[0].trim();
          const fullText = parts[1].trim();
          const distanceMatch = fullText.match(/([0-9.]+) million km/);
          
          if (distanceMatch) {
            const distance = parseFloat(distanceMatch[1]);
            const latency = calculateLatency(distance);
            
            data.push({
              name,
              distance,
              latency,
              fullText
            });
          }
        }
      });
      
      // Sort data: planets first, then moons, then spacecraft
      data.sort((a, b) => {
        // Helper to determine type
        const getType = (name) => {
          if (name.includes('.')) return 2; // Moon
          if (['voyager1', 'voyager2', 'newhorizons', 'jwst', 'iss', 'perseverance'].includes(name)) return 3; // Spacecraft
          return 1; // Planet
        };
        
        const typeA = getType(a.name);
        const typeB = getType(b.name);
        
        if (typeA !== typeB) return typeA - typeB;
        
        // If same type, sort planets by distance, except Earth is always first
        if (typeA === 1) {
          if (a.name === 'earth') return -1;
          if (b.name === 'earth') return 1;
          return a.distance - b.distance;
        }
        
        // Alphabetical sort for moons and spacecraft
        return a.name.localeCompare(b.name);
      });
      
      setCelestialData(data);
    } catch (error) {
      console.error('Error fetching celestial data:', error);
      // Generate mock data if API fails
      generateMockData();
    } finally {
      setLoading(false);
    }
  };
  
  // Generate mock data in case the API is not available
  const generateMockData = () => {
    const mockData = [
      { name: 'earth', distance: 0, latency: '0 seconds' },
      { name: 'mars', distance: 225.0, latency: '12.5 minutes' },
      { name: 'jupiter', distance: 778.6, latency: '43.3 minutes' },
      { name: 'saturn', distance: 1433.5, latency: '79.7 minutes' },
      { name: 'voyager1', distance: 23000.0, latency: '21.3 hours' },
    ];
    
    setCelestialData(mockData);
    setLastUpdated(new Date().toISOString());
  };
  
  useEffect(() => {
    // Initial fetch
    fetchCelestialData();
    
    // Set up interval for periodic updates
    const interval = setInterval(fetchCelestialData, 60000); // Refresh every minute
    
    // Clean up interval on component unmount
    return () => clearInterval(interval);
  }, []);
  
  // Group celestial bodies by type
  const planets = celestialData.filter(item => !item.name.includes('.') && 
    !['voyager1', 'voyager2', 'newhorizons', 'jwst', 'iss', 'perseverance'].includes(item.name));
  
  const moons = celestialData.filter(item => item.name.includes('.'));
  
  const spacecraft = celestialData.filter(item => 
    ['voyager1', 'voyager2', 'newhorizons', 'jwst', 'iss', 'perseverance'].includes(item.name));
  
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
        
        {/* Current Distances Dashboard */}
        <div className="bg-white/10 p-6 rounded-lg mb-16">
          <div className="flex justify-between items-center mb-6">
            <h3 className="text-2xl font-bold text-white">Real-Time Solar System Distances</h3>
            <div className="text-gray-400 text-sm">
              {lastUpdated && (
                <p>Last updated: {new Date(lastUpdated).toLocaleString()}</p>
              )}
              <button 
                onClick={fetchCelestialData}
                className="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-xs ml-2"
              >
                Refresh
              </button>
            </div>
          </div>
          
          {loading ? (
            <div className="text-center py-8 text-gray-300">Loading celestial data...</div>
          ) : (
            <div className="space-y-8">
              {/* Planets */}
              <div>
                <h4 className="text-xl font-bold text-white mb-4">Planets</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {planets.map(planet => (
                    <div key={planet.name} className="bg-black/30 p-4 rounded-lg">
                      <h5 className="text-lg font-bold text-white capitalize">{planet.name}</h5>
                      <div className="mt-2 space-y-2">
                        <p className="text-gray-300">Distance: <span className="text-blue-400">{planet.distance.toFixed(2)} million km</span></p>
                        <p className="text-gray-300">Light latency: <span className="text-yellow-400">{planet.latency}</span></p>
                        <p className="text-gray-300 text-sm">Domain: <code className="bg-black/30 px-2 py-1 rounded">{planet.name}.latency.space</code></p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              
              {/* Moons, only show if we have moons data */}
              {moons.length > 0 && (
                <div>
                  <h4 className="text-xl font-bold text-white mb-4">Moons</h4>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {moons.map(moon => (
                      <div key={moon.name} className="bg-black/30 p-4 rounded-lg">
                        <h5 className="text-lg font-bold text-white">{moon.name}</h5>
                        <div className="mt-2 space-y-2">
                          <p className="text-gray-300">Distance: <span className="text-blue-400">{moon.distance.toFixed(3)} million km</span></p>
                          <p className="text-gray-300">Light latency: <span className="text-yellow-400">{moon.latency}</span></p>
                          <p className="text-gray-300 text-sm">Domain: <code className="bg-black/30 px-2 py-1 rounded">{moon.name}.latency.space</code></p>
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
                    {spacecraft.map(craft => (
                      <div key={craft.name} className="bg-black/30 p-4 rounded-lg">
                        <h5 className="text-lg font-bold text-white">{craft.name}</h5>
                        <div className="mt-2 space-y-2">
                          <p className="text-gray-300">Distance: <span className="text-blue-400">{craft.distance.toFixed(2)} million km</span></p>
                          <p className="text-gray-300">Light latency: <span className="text-yellow-400">{craft.latency}</span></p>
                          <p className="text-gray-300 text-sm">Domain: <code className="bg-black/30 px-2 py-1 rounded">{craft.name}.latency.space</code></p>
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
              <p className="text-gray-300 mb-4">Use the SOCKS5 proxy for any TCP/IP application.</p>
              
              <div className="space-y-4">
                <div>
                  <p className="text-gray-300 mb-2">1. Basic Usage:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    ssh -o ProxyCommand="nc -X 5 -x mars.latency.space:1080 %h %p" destination.server.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Connect to an SSH server through Mars latency.</p>
                </div>
                
                <div>
                  <p className="text-gray-300 mb-2">2. With Curl:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    curl --socks5 neptune.latency.space:1080 https://example.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Experience Neptune's ~4 hour round-trip latency when accessing a website.</p>
                </div>
                
                <div>
                  <p className="text-gray-300 mb-2">3. Browser Configuration:</p>
                  <p className="text-gray-400">Configure your browser's SOCKS proxy settings:</p>
                  <ul className="list-disc list-inside text-gray-400 ml-4">
                    <li>Host: jupiter.latency.space</li>
                    <li>Port: 1080</li>
                    <li>Type: SOCKS5</li>
                  </ul>
                </div>
                
                <div>
                  <p className="text-gray-300 mb-2">4. Target Domain Format:</p>
                  <code className="block bg-black/30 p-4 rounded">
                    ssh -o ProxyCommand="nc -X 5 -x example.com.mars.latency.space:1080 %h %p" destination.server.com
                  </code>
                  <p className="text-gray-400 text-sm mt-1">Combined domain format also works with SOCKS proxy.</p>
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
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mb-16">
          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-4">Advanced Usage</h3>
            <div className="space-y-4 text-gray-300">
              <div>
                <h4 className="font-bold">Multi-level Domain Routing</h4>
                <p className="text-sm">Access any website through multiple celestial bodies:</p>
                <code className="block bg-black/30 p-2 rounded mt-1">
                  http://www.example.com.titan.saturn.latency.space/
                </code>
                <p className="text-gray-400 text-xs mt-1">Routes to example.com with combined Saturn + Titan latency</p>
              </div>
              
              <div>
                <h4 className="font-bold">Custom Applications</h4>
                <p className="text-sm">Configure applications to use SOCKS5 proxy:</p>
                <ul className="list-disc list-inside text-sm">
                  <li>VoIP applications</li>
                  <li>Video conferencing</li>
                  <li>Messaging applications</li>
                  <li>Gaming (for the ultimate challenge)</li>
                </ul>
              </div>
              
              <div>
                <h4 className="font-bold">Spacecraft Tracking</h4>
                <p className="text-sm">Distances to spacecraft update in real-time based on actual trajectories</p>
                <p className="text-gray-400 text-xs mt-1">Try Voyager 1 for the ultimate latency experience (~40+ hours round trip!)</p>
              </div>
            </div>
          </div>

          <div className="bg-white/10 p-6 rounded-lg">
            <h3 className="text-xl font-bold text-white mb-4">System Features</h3>
            <ul className="list-disc list-inside space-y-3 text-gray-300">
              <li>Accurate astronomical calculations using Kepler's laws</li>
              <li>Real-time orbital positions updated hourly</li>
              <li>Support for all planets, major moons, and spacecraft</li>
              <li>HTTP and SOCKS5 proxy interfaces</li>
              <li>Realistic bandwidth limitations based on Deep Space Network capabilities</li>
              <li>SSL/TLS support for secure communications</li>
              <li>Multi-level subdomains for compound latency effects</li>
              <li>Documentation and debugging endpoints</li>
              <li>Rate limiting to prevent abuse</li>
              <li>Continuous integration and deployment</li>
            </ul>
          </div>
        </div>
        
        <div className="bg-white/10 p-6 rounded-lg mb-16">
          <h3 className="text-xl font-bold text-white mb-4">Use Cases</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-black/30 p-4 rounded-lg">
              <h4 className="font-bold text-white">Education</h4>
              <p className="text-gray-300 text-sm mt-2">Demonstrate the challenges of space communication to students and enthusiasts. Experience first-hand why Mars rovers can't be joysticked in real-time.</p>
            </div>
            
            <div className="bg-black/30 p-4 rounded-lg">
              <h4 className="font-bold text-white">Software Testing</h4>
              <p className="text-gray-300 text-sm mt-2">Test software behavior under extreme network conditions. Ensure applications can handle multi-minute or even hour-long latencies gracefully.</p>
            </div>
            
            <div className="bg-black/30 p-4 rounded-lg">
              <h4 className="font-bold text-white">Research</h4>
              <p className="text-gray-300 text-sm mt-2">Explore new protocols and communication strategies designed for high-latency environments. Test delay-tolerant networking concepts.</p>
            </div>
          </div>
        </div>
      </main>

      <footer className="bg-black/30 mt-16 py-8">
        <div className="max-w-7xl mx-auto px-4 text-center text-gray-400">
          <p>Built for space enthusiasts and network engineers</p>
          <p className="mt-2">All latencies based on real astronomical calculations</p>
          <p className="mt-4 text-sm">Data updates hourly based on actual orbital mechanics</p>
        </div>
      </footer>
    </div>
  );
}