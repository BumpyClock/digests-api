docker buildx build -t digests-api .
docker tag digests-api bumpyclock/digests-api:test
docker push bumpyclock/digests-api:test