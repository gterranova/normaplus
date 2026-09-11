# Specifies a parent image
FROM node:24.2.0 AS node

# Creates an app directory to hold your app’s source code
WORKDIR /app

# Copies everything from your root directory into /app
RUN git clone https://github.com/gterranova/normaplus.git

WORKDIR /app/normaplus/frontend

RUN npm install
RUN npm run build

# Specifies a parent image
FROM golang:1.24.5-bookworm AS builder

RUN export PATH=/bin:$PATH

# Install SQLite
RUN apt-get update && apt-get install -y sqlite3

# Creates an app directory to hold your app’s source code
WORKDIR /app
 
# Copies everything from your root directory into /app
COPY --from=node /app/normaplus/backend/. /app/normaplus/backend
COPY --from=node /app/normaplus/frontend/out/. /app/normaplus/backend/internal/assets/dist

WORKDIR /app/normaplus/backend

# Installs Go dependencies
RUN go mod download

# Builds your app with optional configuration
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server/main.go

# Deploy the application binary into a lean image
#FROM debian:12-slim AS build-release-stage
FROM pandoc/latex:latest-debian AS build-release-stage

# Update apt packages and install pandoc
#RUN apt update && apt upgrade -y && apt install pandoc -y

# Install texlive-full
#RUN apt-cache depends texlive-full \
#  | grep "Depends:" \
#  | grep -v "doc$" \
#  | cut -d ' ' -f 4 \
#  | xargs apt-get install --no-install-recommends -y

#RUN apt-get autoclean && apt-get autoremove
#RUN rm -rf /var/lib/apt/lists/* /var/cache/apt/*

WORKDIR /

# copy the ca-certificate.crt from the build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/normaplus/backend/server /app/server

# Tells Docker which network port your container listens on
EXPOSE 8080

WORKDIR /app

# Specifies the executable command that runs when the container starts
#CMD [ "/bin/sleep", "infinity" ]
ENTRYPOINT [ "/app/server" ]