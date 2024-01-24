docker buildx build -t digests-api .
docker tag digests-api bumpyclock/digests-api:latest
docker push bumpyclock/digests-api:latest