import React, { useState } from 'react';
import AIAssistantApp from './components/AIAssistantApp';
import AdminDashboard from './components/AdminDashboard';
import PowerUserInterface from './components/PowerUserInterface';

function App() {
  const [currentView, setCurrentView] = useState<'chat' | 'admin' | 'power'>('chat');

  return (
    <div className="App">
      {currentView === 'admin' ? (
        <AdminDashboard onBack={() => setCurrentView('chat')} />
      ) : currentView === 'power' ? (
        <PowerUserInterface onBack={() => setCurrentView('chat')} />
      ) : (
        <AIAssistantApp 
          onAdminAccess={() => setCurrentView('admin')}
          onPowerUserAccess={() => setCurrentView('power')}
        />
      )}
    </div>
  );
}

export default App;
