# Start from golang base image
FROM golang:alpine as builder
RUN apk add build-base
# ENV GO111MODULE=on

# ADD maintainer info
LABEL maintainer="Steven Victor <chikodi543@gmail.com>"

# Install git.
# Git is rewquired for fetching the dependencies
RUN apk update && apk add --no-cache git

# Set the current working directory inside the container
WORKDIR /app

# COPY go.mod go.sum
COPY go.mod go.sum ./

# DOWNLOAD all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed
RUN go mod download

# Copy the source from current directory to the working Directory inside the container
COPY . .

# BUild the go app
RUN GGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

#Copy the pre-built binary file from the previuos stage. Observe we aloso copied the .env file
COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# EXPOSE port 8080 to the outside world
EXPOSE 8080

#COmmand to run the executable
CMD ["./main"]