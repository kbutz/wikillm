FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better Docker layer caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create directory for database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8000

# Set environment variables
ENV DATABASE_URL=sqlite:///./data/assistant.db
ENV API_HOST=0.0.0.0
ENV API_PORT=8000

# Initialize database and start server
CMD ["python", "-c", "from database import init_database; init_database(); import uvicorn; from main import app; uvicorn.run(app, host='0.0.0.0', port=8000)"]
