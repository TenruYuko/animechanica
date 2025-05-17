# Seanime Docker Setup with Gluetun Mullvad VPN

This Docker setup provides a leak-free environment for running Seanime with mpv and qBittorrent, all behind a Mullvad VPN using Gluetun.

## Features

- Seanime media server with full web interface
- MPV media player included in the container
- qBittorrent for downloading torrents
- Gluetun VPN client configured for Mullvad OpenVPN
- Built-in firewall rules to prevent any traffic leakage outside the VPN
- Health monitoring for VPN connection status

## Prerequisites

- Docker and Docker Compose installed on your system
- A valid Mullvad VPN subscription
- A valid Mullvad VPN subscription

## Setup Instructions

### 1. Prepare the directory structure

Create the necessary directories for the Docker setup:

```bash
mkdir -p ./data ./qbittorrent/config ./gluetun
```

These directories will store the configuration and data for Seanime, qBittorrent, and Gluetun.

### 2. Configure the docker-compose.yml file

Edit the `docker-compose.yml` file to update the following:

- Replace `mullvad_username` with your Mullvad account number
- Update the `SERVER_COUNTRIES` value to your preferred server location
- Update the `/path/to/anime` to point to your anime collection
- Update the `/path/to/downloads` to specify where you want downloads to be stored

Alternatively, use the provided setup script which will prompt you for all these values.

## Running the Stack

Build and start all containers:

```bash
docker-compose up -d
```

Check if the containers are running properly:

```bash
docker-compose ps
```

## Accessing Services

- **Seanime**: Access the web interface at http://localhost:3000
- **qBittorrent**: Access the web UI at http://localhost:8080 (default credentials: admin/adminadmin)

## Configuration

### Seanime Configuration

Seanime stores its configuration in the `./data` directory. The first time you access the web interface, you'll need to set up Seanime as per the official documentation.

### qBittorrent Configuration

Log in to qBittorrent web UI and configure it as needed. The qBittorrent configuration is stored in the `./qbittorrent/config` directory.

## Troubleshooting

### VPN Connection Issues

Check the logs of the VPN container:

```bash
docker-compose logs vpn
```

If the VPN isn't connecting, verify your Mullvad credentials or try a different server country.

### Checking VPN Connection

To verify that traffic is going through the VPN:

```bash
docker exec -it gluetun-vpn curl https://am.i.mullvad.net/json
```

This should return a JSON response indicating you're connected to Mullvad.

You can also check the Gluetun health server:

```bash
curl http://localhost:9999/healthcheck
```

This endpoint will return a 200 OK status if the VPN is connected properly.

## Notes

- The Seanime and qBittorrent containers share the VPN network stack, ensuring all traffic is routed through the VPN.
- Gluetun's built-in firewall is configured to block all traffic if the VPN connection drops, preventing leaks.
- Specific ports (3000 for Seanime and 8080 for qBittorrent) are opened to allow web access while blocking other traffic.
- To use MPV for playback, Seanime will call the MPV instance inside the container.

## Additional Gluetun Features

- Gluetun provides a health check endpoint at http://localhost:9999/healthcheck
- You can customize VPN servers by changing the `SERVER_COUNTRIES` environment variable
- For more advanced configuration, see the [Gluetun documentation](https://github.com/qdm12/gluetun-wiki)

## Stopping the Stack

```bash
docker-compose down
```

## Rebuilding After Changes

If you make changes to the Dockerfile or need to rebuild the Seanime container:

```bash
docker-compose build seanime
docker-compose up -d
```
