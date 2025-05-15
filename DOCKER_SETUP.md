# Dockerized Seanime Setup

This directory contains a Docker Compose setup for running Seanime. The setup includes two different container options:

1. **Backend Service** - A minimal container running just the Seanime backend on port 43211 (mapped to host port 2500)
2. **Script Service** - A container that runs the start-app.sh script, which sets up both the backend and a proxy server

## Prerequisites

- Docker and Docker Compose installed on your system
- Git (to clone the repository)

## Quick Start

1. Make sure you're in the project directory:
   ```bash
   cd /aeternae/functional/dockers/animechanica
   ```

2. Choose which service to use:

   **Option 1: Backend Only (Recommended for External Access)**
   ```bash
   docker-compose up -d backend
   ```
   Access the application at: `http://localhost:2500`

   **Option 2: Script-based Setup**
   ```bash
   docker-compose up -d script
   ```
   Access the application at: `http://localhost:3000`

## Configuration

### Data Persistence

The application data is stored in a Docker volume and also mapped to the `./data` directory in your project folder. This ensures that your data persists across container restarts.

### Environment Variables

You can customize the setup by modifying the environment variables in the `docker-compose.yml` file:

- `SEANIME_DATA_DIR`: The directory where Seanime stores its data (default: `/app/data`)
- `DATA_DIR`: For the script service, the directory where data is stored (default: `/app/data`)

## Stopping the Application

To stop the containers:

```bash
docker-compose down
```

To stop and remove all containers, networks, and volumes:

```bash
docker-compose down -v
```

## Choosing Between Backend and Script Services

### Backend Service

- Simpler, more lightweight setup
- Runs only the Seanime backend
- Directly accessible from external networks
- Recommended for most users

### Script Service

- Runs both the backend and a proxy server
- Provides additional features for manga handling
- Better for development or testing
- Includes special handling for authentication and CORS

## Port Conflicts

If you see an error like `bind: address already in use`, it means the ports are already in use on your system. You can change the port mapping in the `docker-compose.yml` file:

```yaml
ports:
  - "8080:43211"  # Change 2500 to 8080 for the backend
```

## Logs

To view logs for the containers:

```bash
# View logs for all containers
docker-compose logs

# View logs for a specific container
docker-compose logs backend
docker-compose logs script

# Follow logs in real-time
docker-compose logs -f
```

## Updating

To update to a new version:

1. Pull the latest code:
   ```bash
   git pull
   ```

2. Rebuild and restart the containers:
   ```bash
   docker-compose down
   docker-compose up -d --build
   ```

## Advanced Configuration

For advanced configuration options, refer to the main Seanime documentation at https://seanime.rahim.app/docs.
