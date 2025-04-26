import React from 'react';
import { Routes, Route } from 'react-router-dom';
// Import your page components here
import StatusDashboard from './pages/StatusDashboard';
import LandingPage from './pages/Landing'; // Corrected import name

// Import global styles
import './styles/index.css'; // Assuming you have a global CSS file

export default function App() {
  return (
    // Using a React Fragment to avoid adding an extra div to the DOM,
    // but you could use a <main> or <div> if you prefer.
    <>
      {/* The <Routes> component will handle the routing logic */}
      <Routes>
        {/* Define individual routes using the <Route> component */}
        <Route path="/" element={<LandingPage />} />
        <Route path="/status" element={<StatusDashboard />} />
        {/* Add more routes as needed */}
      </Routes>
    </>
  );
}
