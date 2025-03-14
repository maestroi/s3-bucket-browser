# S3 Bucket Browser

A modern web application for efficiently indexing and exploring the contents of an S3 bucket, with a focus on .tar.gz files and their associated metadata.

## Features

- **Modern UI**: Clean and responsive interface built with Vue 3 and Tailwind CSS
- **Real-time Updates**: WebSocket connection for live updates when new files are added to the bucket
- **Metadata Exploration**: View and search metadata for .tar.gz files
- **Advanced Filtering**: Filter by Solana version, feature set, status, and more
- **Caching**: Redis-based caching to optimize S3 API calls and reduce costs
- **Docker Support**: Easy deployment with Docker and Docker Compose
- **S3 Compatibility**: Works with AWS S3 and S3-compatible services like Wasabi, MinIO, etc.

## Architecture

### Backend (Golang)

- RESTful API for listing files and retrieving metadata
- WebSocket API for real-time updates
- Redis caching for optimized performance
- S3 integration for accessing bucket contents

### Frontend (Vue 3)

- Modern UI built with Tailwind CSS
- Real-time updates via WebSocket
- Advanced search and filtering capabilities
- Responsive design for all device sizes

## Setup

### Configuration

1. **Environment Variables**:
   - Copy the example environment file: `cp .env.example .env`
   - Edit the `.env` file with your actual S3 credentials and configuration

2. **Backend Configuration**:
   - Copy the example config file: `cp backend/config.example.json backend/config.json`
   - Edit `backend/config.json` with your actual configuration values

### Running with Docker Compose

```bash
# Start the application
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the application
docker-compose down
```

The application will be available at:
- Frontend: http://localhost:8081 (or the port specified in your .env file)
- Backend API: http://localhost:8080 (or the port specified in your .env file)

## Security Notes

- The `.env` file and `backend/config.json` contain sensitive information and are excluded from git in the `.gitignore` file.
- Never commit these files to the repository.
- Always use the example files as templates and create your own local copies with actual credentials.

## Development

### Backend (Go)

The backend is built with Go and provides API endpoints for:
- Listing files in the S3 bucket
- Getting file metadata
- Filtering and searching metadata
- WebSocket notifications for real-time updates

### Frontend (Vue.js)

The frontend is built with Vue 3 and provides a user interface for:
- Browsing files in the S3 bucket
- Viewing file metadata
- Filtering and searching metadata
- Real-time updates via WebSocket

## Troubleshooting

### Redis Connection Issues

If you see errors like:
```
Failed to create Redis cache: dial tcp [::1]:6379: connect: connection refused
```

Make sure:
1. The Redis container is running: `docker ps | grep s3browser-redis`
2. The Redis address in config.json is set to `s3browser-redis:6379`
3. The REDIS_HOST environment variable is set to `s3browser-redis`

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.
