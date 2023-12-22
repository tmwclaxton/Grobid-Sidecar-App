# Grobid Helper App

This is a simple go app built using [Gin](https://github.com/gin-gonic/gin). It exposes 2 endpoints on port `:591` / and /health.  It is used to limit the flow of requests to a grobid service

```bash
docker-compose -f docker-compose.yaml up
```

```bash
docker build -t grobids-friend:latest -f ./Dockerfile .
docker run -p 591:8080 --name grobids-friend --rm grobids-friend:latest  
```