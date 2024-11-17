import React, { useState, useEffect } from 'react';
import { LineChart, XAxis, YAxis, Tooltip, Line, ResponsiveContainer } from 'recharts';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';

export default function StatusDashboard() {
  const [metrics, setMetrics] = useState({});

  useEffect(() => {
    const fetchMetrics = async () => {
      const response = await fetch('/api/metrics');
      const data = await response.json();
      setMetrics(data);
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 to-slate-800 p-8">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-4xl font-bold text-white mb-8">Solar System Latency Status</h1>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {Object.entries(metrics.planets || {}).map(([planet, data]) => (
            <Card key={planet} className="bg-white/10 text-white">
              <CardHeader>
                <CardTitle>{planet}.latency.space</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <p className="text-sm text-gray-300">Current Distance</p>
                    <p className="text-2xl">{(data.distance / 1e6).toFixed(1)}M km</p>
                  </div>
                  <div>
                    <p className="text-sm text-gray-300">Light Travel Time</p>
                    <p className="text-2xl">{(data.latency / 60).toFixed(1)} minutes</p>
                  </div>
                  <div className="h-32">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={data.history}>
                        <XAxis dataKey="time" stroke="#fff" />
                        <YAxis stroke="#fff" />
                        <Tooltip />
                        <Line type="monotone" dataKey="latency" stroke="#8884d8" />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    </div>
  );
}
