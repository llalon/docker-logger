# docker-logger

Lightweight container to direct other docker container logs to file(s) with rotation and retention.

## Configuration

Configuration by environment variables:

| Variable            | Default                  | Description                         |
|---------------------|--------------------------|-------------------------------------|
| `LOG_DIR`           | `/app/logs`              | Directory for logs                  |
| `MAX_SIZE_MB`       | `10`                     | Max log file size in MB             |
| `MAX_BACKUPS`       | `5`                      | Max number of log files to keep     |
| `MAX_AGE_DAYS`      | `31`                     | Max age of log files                |
| `TARGET_CONTAINERS` | `all running containers` | Comma-separated container names/IDs |

## Running

To run for all containers:

```bash
docker run -d \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /path/on/host/logs:/app/logs \
  ghcr.io/llalon/docker-logger:latest
```

To run for specific containers only:

```bash
docker run -d \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /path/on/host/logs:/app/logs \
    -e TARGET_CONTAINERS="apache-container" \
    ghcr.io/llalon/docker-logger:latest
```
