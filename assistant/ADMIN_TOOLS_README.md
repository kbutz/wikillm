# WikiLLM Assistant Admin Tools

## Overview

The WikiLLM Assistant now includes comprehensive admin tools for user management, memory inspection, conversation oversight, and system monitoring. These tools provide a complete administrative interface for managing the AI assistant system.

## Features

### 🔧 User Management
- **View All Users**: List all users with detailed statistics
- **Create Users**: Add new users with username, email, and full name
- **Delete Users**: Remove users and all associated data
- **User Details**: View comprehensive user information
- **User Impersonation**: Switch perspective to any user for debugging

### 🧠 Memory Management
- **Memory Inspection**: View and edit user memory data
- **Memory Search**: Search through memory entries
- **Memory Export**: Download user memory as JSON
- **Memory Editing**: In-place editing of memory sections
- **Memory Clearing**: Clear specific memory types or all memory

### 💬 Conversation Management
- **View Conversations**: List all user conversations with details
- **Conversation Messages**: View complete conversation history
- **Delete Conversations**: Remove conversations and all messages
- **Conversation Analytics**: Message counts, timestamps, and summaries

### 📊 System Monitoring
- **System Statistics**: Total users, conversations, messages, memory usage
- **Health Monitoring**: System status and error tracking
- **Data Export**: Bulk export of user data
- **Active User Tracking**: Monitor user activity patterns

## Architecture

### Backend Components

#### Admin API Routes (`admin_routes.py`)
- **User Management**: `/admin/users/*`
- **Memory Management**: `/admin/users/{user_id}/memory`
- **Conversation Management**: `/admin/conversations/*`
- **System Stats**: `/admin/system/stats`
- **Data Export**: `/admin/users/{user_id}/export`

#### API Integration (`main.py`)
- Admin routes registered with main FastAPI app
- Full integration with existing database models
- Proper error handling and logging

### Frontend Components

#### Admin Dashboard (`AdminDashboard.tsx`)
- Main admin interface with tabbed navigation
- User list with search and filtering
- Real-time statistics dashboard
- Modal dialogs for user creation

#### Memory Inspector (`MemoryInspector.tsx`)
- Advanced memory data viewer
- JSON editing with syntax validation
- Search functionality across memory data
- Export and import capabilities

#### Admin Service (`admin.ts`)
- TypeScript service for admin API calls
- Type-safe admin data models
- Error handling and loading states

## Usage

### Accessing Admin Tools

1. **From Main App**: Click the shield icon (🛡️) in the header
2. **Direct Access**: Navigate to admin dashboard through app routing

### User Management Workflow

1. **View Users**: Browse all users with statistics
2. **Select User**: Click on any user to view details
3. **Create User**: Use the "Add User" button
4. **Delete User**: Use the trash icon (confirms deletion)
5. **Export Data**: Use the download icon for user data

### Memory Management Workflow

1. **Select User**: Choose a user from the list
2. **Memory Tab**: Switch to the Memory tab
3. **Inspect Memory**: View organized memory sections
4. **Edit Memory**: Click edit icon to modify data
5. **Search Memory**: Use search bar to find specific entries
6. **Export Memory**: Download complete memory data

### Conversation Management Workflow

1. **Select User**: Choose a user from the list
2. **Conversations Tab**: Switch to Conversations tab
3. **View Details**: See conversation summaries and stats
4. **Delete Conversations**: Remove unwanted conversations
5. **Export Data**: Download conversation history

## Security Considerations

⚠️ **Important**: The admin tools currently have **NO AUTHENTICATION** for development purposes.

### For Production Use:

1. **Add Authentication**: Implement proper admin authentication
2. **Role-Based Access**: Different permission levels for different admins
3. **Audit Logging**: Track all admin actions
4. **Rate Limiting**: Prevent abuse of admin endpoints
5. **IP Whitelisting**: Restrict admin access to specific IPs

### Recommended Security Implementation:

```python
# Example admin middleware
def admin_middleware(request: Request):
    # Check admin token/session
    # Verify admin permissions
    # Log admin actions
    pass
```

## API Documentation

### Admin Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/users` | GET | List all users |
| `/admin/users` | POST | Create new user |
| `/admin/users/{user_id}` | GET | Get user details |
| `/admin/users/{user_id}` | DELETE | Delete user |
| `/admin/users/{user_id}/memory` | GET | Get user memory |
| `/admin/users/{user_id}/memory` | PUT | Update user memory |
| `/admin/users/{user_id}/memory` | DELETE | Clear user memory |
| `/admin/users/{user_id}/conversations` | GET | Get user conversations |
| `/admin/conversations/{conversation_id}` | DELETE | Delete conversation |
| `/admin/users/{user_id}/export` | GET | Export user data |
| `/admin/system/stats` | GET | Get system statistics |
| `/admin/users/{user_id}/impersonate` | POST | Impersonate user |

### Data Models

#### AdminUser
```typescript
interface AdminUser {
  id: number;
  username: string;
  email?: string;
  full_name?: string;
  created_at: string;
  updated_at: string;
  last_active?: string;
  conversation_count: number;
  memory_size: number;
  memory_entries: number;
}
```

#### AdminMemory
```typescript
interface AdminMemory {
  personal_info: Record<string, any>;
  conversation_history: any[];
  context_memory: Record<string, any>;
  preferences: Record<string, any>;
  size: number;
  last_updated: string;
}
```

## Development

### Setup

1. **Install Dependencies**:
   ```bash
   pip install -r requirements.txt
   cd frontend && npm install
   ```

2. **Run Tests**:
   ```bash
   chmod +x test_admin_setup.sh
   ./test_admin_setup.sh
   ```

3. **Start Services**:
   ```bash
   # Backend
   python main.py
   
   # Frontend
   cd frontend && npm start
   ```

### Database Schema

The admin tools use existing database models:
- `User`: User accounts and metadata
- `Conversation`: Chat conversations
- `Message`: Individual messages
- `UserMemory`: User memory entries
- `UserPreference`: User preferences
- `SystemLog`: System logs and errors

### Adding New Admin Features

1. **Backend**: Add new endpoints to `admin_routes.py`
2. **Frontend**: Add new components or extend existing ones
3. **Service**: Update `admin.ts` with new API calls
4. **Types**: Add TypeScript interfaces for new data models

## Troubleshooting

### Common Issues

1. **Admin Button Not Visible**: Check if Shield icon is imported and onAdminAccess prop is passed
2. **API Errors**: Verify backend is running and admin routes are registered
3. **Memory Inspector Not Opening**: Check if selectedUser is set and component is properly imported
4. **Database Errors**: Ensure database schema is up to date

### Debug Steps

1. **Check Browser Console**: Look for JavaScript errors
2. **Check Network Tab**: Verify API calls are being made
3. **Check Backend Logs**: Look for Python errors in console
4. **Test API Directly**: Use `/docs` endpoint to test admin APIs

### Performance Optimization

1. **Large User Lists**: Implement pagination for better performance
2. **Memory Data**: Use streaming for large memory exports
3. **Database Queries**: Add indexes for frequently queried fields
4. **Frontend Rendering**: Implement virtualization for large lists

## Future Enhancements

### Planned Features

1. **Real-time Updates**: WebSocket integration for live admin updates
2. **Bulk Operations**: Multi-select for batch user operations
3. **Data Visualization**: Charts and graphs for system metrics
4. **Advanced Search**: Full-text search across all user data
5. **Backup/Restore**: Complete system backup and restore functionality

### Configuration Options

1. **Admin Theme**: Dark/light mode toggle
2. **Display Options**: Customizable table columns and layouts
3. **Export Formats**: Support for CSV, XML, and other formats
4. **Notification System**: Alerts for system events and errors

## Contributing

### Code Style

- **Backend**: Follow PEP 8 Python style guide
- **Frontend**: Use TypeScript strict mode
- **Components**: Functional components with hooks
- **Styling**: Tailwind CSS utility classes

### Testing

- **Backend**: pytest for API endpoint testing
- **Frontend**: Jest/React Testing Library for component testing
- **Integration**: End-to-end testing with Cypress or Playwright

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Update documentation
5. Submit pull request with detailed description

## License

This admin tools implementation is part of the WikiLLM Assistant project and follows the same license terms as the main project.

## Support

For issues, questions, or contributions related to the admin tools:

1. **GitHub Issues**: Report bugs and request features
2. **Documentation**: Check this README and inline code comments
3. **Community**: Join discussions about admin tool enhancements

---

**Note**: This admin tools implementation provides a solid foundation for managing the WikiLLM Assistant system. Always implement proper security measures before deploying to production environments.
