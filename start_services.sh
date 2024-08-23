#!/bin/bash

docker-compose build

echo "Starting redis..."
docker-compose up -d redis
echo "redis started."

echo "Starting postgres..."
docker-compose up -d postgres
echo "postgres started."

echo "Starting rabbitmq..."
docker-compose up -d rabbitmq
echo "rabbitmq started."

echo "Waiting for 30 seconds..."
sleep 30s

echo "Starting chat service..."
docker-compose up -d chat
echo "chat service started."

echo "Waiting for 10 seconds..."
sleep 10s

echo "Starting places service..."
docker-compose up -d places
echo "places service started."

echo "Waiting for 10 seconds..."
sleep 10s

echo "Starting charity service..."
docker-compose up -d charity
echo "charity service started."

echo "Waiting for 10 seconds..."
sleep 10s

echo "Starting purchases service..."
docker-compose up -d purchases
echo "purchases service started."

echo "Waiting for 10 seconds..."
sleep 10s

echo "Starting votes service..."
docker-compose up -d votes
echo "votes service started."

echo "Waiting for 10 seconds..."
sleep 10s

echo "Starting notifications service..."
docker-compose up -d notifications
echo "notifications service started."

echo "Waiting for 10 seconds..."
sleep 10s

echo "Starting gateway service..."
docker-compose up -d gateway
echo "gateway service started."

echo "All services have been started."
