# Stage 1: Builder
FROM rust:1.89.0-slim-trixie AS builder

WORKDIR /app

# Install system dependencies for building
RUN apt-get update && apt-get install -y \
    pkg-config \
    libbluetooth-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy Cargo.toml and Cargo.lock first to leverage Docker cache
COPY Cargo.toml Cargo.lock ./

# Build dummy project to cache dependencies
RUN mkdir src && echo "fn main() {println!(\"Hello\");}" > src/main.rs
RUN cargo build --release
RUN rm -rf src

# Copy the actual source code
COPY src ./src

# Build the application
RUN cargo build --release

# Stage 2: Runner
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    libbluetooth3 \
    && rm -rf /var/lib/apt/lists/*

# Copy the compiled binary from the builder stage
COPY --from=builder /app/target/release/ble-metrics ./ble-metrics

# Expose any necessary ports (if the application listens on one)
# For a BLE metrics application, it might not expose a port, but if it does, add it here.
# For now, assuming it's a CLI tool or pushes data, so no port exposed.

CMD ["./ble-metrics"]
