#!/bin/bash

# Start Go server (pastikan pakai 0.0.0.0)
go run main.go &

# Tunggu server siap
sleep 2

# Start ngrok
ngrok http --subdomain=workerbs 8080
