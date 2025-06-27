# AI Assistant - ChatGPT/Claude Clone

A production-ready AI assistant application that interfaces with LMStudio for local LLM inference, featuring conversation management, user memory (explicit and implicit), and context switching capabilities.

## Features

### Core Functionality
- **Local LLM Integration**: Seamless connection to LMStudio for private, local AI inference
- **Conversation Management**: Create, switch between, and manage multiple conversation contexts
- **Dual Memory System**: 
  - **Explicit Memory**: User-provided information and preferences
  - **Implicit Memory**: Automatically extracted patterns from conversations
- **Real-time Chat**: WebSocket support for streaming responses
- **User Personalization**: Adaptive responses based on learned user patterns

### Technical Features
- **FastAPI Backend**: Production-ready REST API with automatic documentation
- **SQLite Database**: Efficient local storage with SQLAlchemy ORM
- **React Frontend**: Modern, responsive TypeScript interface
- **Memory Management**: Intelligent consolidation and cleanup of stored memories
- **Background Processing**: Async memory extraction and conversation summarization

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   React Frontend│    │  FastAPI Backend│    │   LMStudio API  │
│                 │    │                 │    │                 │
│ • Chat UI       │◄──►│ • Conversation  │◄──►│ • Local LLM     │
│ • Memory View   │    │   Management    │    │ • Inference     │
│ • User Setup    │    │ • Memory System │    │                 │
└─────────────────┘    │ • User Auth     │    └─────────────────┘
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │ SQLite Database │
                       │                 │
                       │ • Users         │
                       │ • Conversations │
                       │ • Messages      │
                       │ • Memories      │
                       └─────────────────┘
```

## Installation & Setup

### Prerequisites
- Python 3.11+
- Node.js 18+
- LMStudio running locally on port 1234

### Backend Setup

1. **Clone and navigate to the assistant directory**:
   ```bash
   cd wikillm/assistant
   ```

2. **Create virtual environment**:
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

3. **Install dependencies**:
   ```bash
   pip install -r requirements.txt
   ```

4. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

5. **Initialize database**:
   ```bash
   python -c "from database import init_database; init_database()"
   ```

6. **Start the backend server**:
   ```bash
   python main.py
   ```

   The API will be available at `http://localhost:8000` with automatic documentation at `http://localhost:8000/docs`

### Frontend Setup

1. **Navigate to frontend directory**:
   ```bash
   cd frontend
   ```

2. **Install dependencies**:
   ```bash
   npm install
   ```

3. **Start development server**:
   ```bash
   npm start
   ```

   The frontend will be available at `http://localhost:3000`

### LMStudio Setup

1. **Download and install LMStudio** from [lmstudio.ai](https://lmstudio.ai)

2. **Load a compatible model** (recommended: Llama 2, Mistral, or Code Llama)

3. **Start the local server**:
   - Open LMStudio
   - Go to the "Local Server" tab
   - Load your chosen model
   - Start the server on port 1234 (default)

4. **Verify connection**:
   ```bash
   curl http://localhost:1234/v1/models
   ```

## API Endpoints

### Core Endpoints

#### User Management
- `POST /users/` - Create new user
- `GET /users/{user_id}` - Get user details
- `GET /users/` - List all users

#### Conversation Management
- `POST /conversations/` - Create new conversation
- `GET /conversations/{conversation_id}` - Get conversation details
- `GET /users/{user_id}/conversations` - Get user's conversations
- `DELETE /conversations/{conversation_id}` - Delete conversation

#### Chat
- `POST /chat` - Send message and get response
- `POST /chat/stream` - Stream chat response (Server-Sent Events)

#### Memory Management
- `GET /users/{user_id}/memory` - Get user memories
- `POST /users/{user_id}/memory` - Add explicit memory
- `DELETE /users/{user_id}/memory/{memory_id}` - Delete memory

#### System
- `GET /status` - System health and statistics
- `GET /health` - Simple health check

## Memory System

### Explicit Memory
User-provided information stored directly:
```python
{
    "memory_type": "explicit",
    "key": "name",
    "value": "John Doe",
    "confidence": 1.0
}
```

### Implicit Memory
Automatically extracted patterns:
```python
{
    "memory_type": "implicit",
    "key": "response_style",
    "value": "concise",
    "confidence": 0.8
}
```

### Memory Categories
- **Personal Information**: Name, profession, location
- **Preferences**: Response style, technical level, task breakdown
- **Communication Patterns**: Tone, formality, detail level
- **Domain Expertise**: Technical knowledge areas
- **Interaction History**: Frequently discussed topics

### Automatic Pattern Detection
The system automatically extracts user preferences from conversations:

```python
# Example: User says "Please keep it brief"
# System extracts:
{
  "key": "response_style",
  "value": "concise",
  "confidence": 0.8,
  "type": "implicit"
}
```

### Personal Information Extraction
Advanced regex patterns extract personal details:
- Names: "My name is John" → stores "name": "John"
- Professions: "I work as a developer" → stores "profession": "developer"
- Locations: "I live in Seattle" → stores "location": "Seattle"

### Memory Consolidation
- Automatically merges duplicate memories
- Maintains highest confidence values
- Prevents memory bloat
- Configurable cleanup policies

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `sqlite:///./assistant.db` | Database connection string |
| `LMSTUDIO_BASE_URL` | `http://localhost:1234` | LMStudio API base URL |
| `LMSTUDIO_MODEL` | `local-model` | Model identifier |
| `API_HOST` | `0.0.0.0` | API server host |
| `API_PORT` | `8000` | API server port |
| `MAX_CONVERSATION_HISTORY` | `50` | Max messages per context |
| `DEFAULT_TEMPERATURE` | `0.7` | LLM temperature setting |

### Database Configuration

The system uses SQLite by default for simplicity, but supports PostgreSQL for production:

```bash
# PostgreSQL example
DATABASE_URL=postgresql://user:password@localhost/assistant_db
```

## Usage Examples

### Creating a User
```python
import requests

response = requests.post('http://localhost:8000/users/', json={
    "username": "john_doe",
    "email": "john@example.com",
    "full_name": "John Doe"
})
user = response.json()
```

### Starting a Conversation
```python
# Send first message
response = requests.post('http://localhost:8000/chat', json={
    "message": "Hello, I'm new to Python programming",
    "user_id": user["id"]
})

# AI automatically detects beginner level and stores implicit memory
```

### Adding Explicit Memory
```python
requests.post(f'http://localhost:8000/users/{user_id}/memory', json={
    "memory_type": "explicit",
    "key": "programming_language",
    "value": "Python",
    "confidence": 1.0
})
```

## Development

### CLI Interface
```bash
python cli.py init          # Initialize database
python cli.py sample-data   # Create test data
python cli.py analyze       # Analyze memory patterns
python cli.py export 1      # Export user data
python cli.py serve         # Start server
```

### Running Tests
```bash
# Backend tests
pytest

# Frontend tests
cd frontend && npm test
```

### Code Quality
```bash
# Python linting
flake8 .
black .

# TypeScript checking
cd frontend && npm run type-check
```

### Database Migrations
```bash
# Generate migration
alembic revision --autogenerate -m "Description"

# Apply migrations
alembic upgrade head
```

## Performance Considerations

### Memory Management
- Automatic consolidation of duplicate memories
- Cleanup of low-confidence, old memories
- Configurable memory limits per user

### Database Optimization
- Indexed columns for fast queries
- Connection pooling for concurrent users
- WAL mode enabled for SQLite

### Caching Strategy
- In-memory conversation context caching
- User preference caching
- Model response caching for repeated queries

## Advanced Capabilities

### Smart Memory Management
- Confidence scoring for extracted information
- Temporal decay for outdated memories
- Context-aware memory retrieval
- Memory importance weighting

### Conversation Intelligence
- Auto-generated conversation titles
- Topic detection and categorization
- Conversation summarization
- Context window optimization

### User Experience
- Responsive, modern UI design  
- Real-time typing indicators
- Message timestamps and processing times
- Memory visualization panel
- Conversation export functionality

## Production Deployment

### Docker Deployment
```dockerfile
# Example Dockerfile structure
FROM python:3.11-slim
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

### Environment Configuration
- Use environment-specific `.env` files
- Configure proper CORS origins
- Set up database backups
- Monitor system resources

### Security Considerations
- Input validation and sanitization
- Rate limiting for API endpoints
- User authentication tokens
- Database connection security

## Troubleshooting

### Common Issues

1. **LMStudio Connection Failed**
   - Verify LMStudio is running on correct port
   - Check firewall settings
   - Ensure model is loaded

2. **Database Errors**
   - Check database file permissions
   - Verify SQLite installation
   - Run database initialization

3. **Memory Issues**
   - Monitor database size
   - Run memory cleanup tasks
   - Adjust memory limits

### Logs
- Application logs: `assistant.log`
- Database logs: Check SQLite error messages
- LMStudio logs: Check LMStudio console

## Future Enhancements

The architecture supports easy extension for:
- Multi-user authentication
- Real-time collaboration
- Plugin system for tools
- Advanced analytics
- Mobile applications
- Voice interface
- Multi-modal support

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run quality checks: `pytest && flake8`
5. Submit pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- OpenAI for ChatGPT inspiration
- Anthropic for Claude conversation patterns
- LMStudio for local LLM inference
- FastAPI and React communities
