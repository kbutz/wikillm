import React, { useState } from 'react';
import AIAssistantApp from './components/AIAssistantApp';
import AdminDashboard from './components/AdminDashboard';

function App() {
  const [showAdmin, setShowAdmin] = useState(false);

  return (
    <div className="App">
      {showAdmin ? (
        <AdminDashboard onBack={() => setShowAdmin(false)} />
      ) : (
        <AIAssistantApp onAdminAccess={() => setShowAdmin(true)} />
      )}
    </div>
  );
}

export default App;
