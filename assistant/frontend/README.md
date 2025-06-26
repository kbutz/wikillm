# AI Assistant Frontend

This is the React frontend for the AI Assistant application.

## Features

- Modern, responsive chat interface
- Real-time conversation management  
- User memory visualization
- Conversation switching and management
- User setup and profile management
- TypeScript for type safety
- Tailwind CSS for styling

## Development

```bash
# Install dependencies
npm install

# Start development server
npm start

# Build for production
npm run build

# Run tests
npm test
```

## Environment Configuration

Create a `.env` file in the frontend directory:

```
REACT_APP_API_URL=http://localhost:8000
```

## Components

- **AIAssistantApp**: Main application component
- **MessageBubble**: Individual chat message display
- **LoadingMessage**: Loading indicator for AI responses
- **UserSetupModal**: Initial user registration
- **MemoryPanel**: User memory visualization

## API Integration

The frontend uses a service class (`ApiService`) to communicate with the FastAPI backend, providing:

- User management
- Conversation operations
- Message sending/receiving
- Memory management
- System status monitoring

## TypeScript Types

All API interfaces and data models are defined in `/src/types/index.ts` for full type safety.
