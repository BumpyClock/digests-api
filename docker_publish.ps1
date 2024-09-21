# Build the Docker image
docker buildx build -t digests-api .

# Tag the Docker image
docker tag digests-api bumpyclock/digests-api:latest

# Push the Docker image to the repository
docker push bumpyclock/digests-api:latest