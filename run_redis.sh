if docker ps -a | grep -q redis-stack; then
    echo "Container 'redis-stack' already exists. Starting it..."
    docker start redis-stack
else
    echo "Container 'redis-stack' does not exist. Creating and starting a new one..."
    docker run --name redis-stack -p 6379:6379 -p 8001:8001 redis/redis-stack:latest
fi