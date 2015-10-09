# Marco - Compose

Docker Compose backend for https://github.com/nickschuch/marco

## Usage

```bash
$ marco-compose --marco=http://localhost:81
```

NOTE: We assume the Marco daemon is already running.

## Docker

The following will setup Marco + Docker compose backend pushes.

```bash
$ docker run -d \
             --name=marco \
             -p 0.0.0.0:80:80 nickschuch/marco
$ docker run -d \
             --link marco:marco \
             -e "MARCO_COMPOSE_URL=http://marco:81" \
             -v /var/run/docker.sock:/var/run/docker.sock nickschuch/marco-compose
```

