### 0 - BUILD ARGUMENTS ###

# Set build and package variables
ARG BUILD_IMAGE="ghcr.io/cloudnative-pg/postgresql"
ARG BUILD_IMAGE_TAG="17-bookworm"

### 1 - DOWNLOADER STAGE ###

FROM alpine:3.22 AS downloader

# Copy install scripts
COPY tmp/install_scripts/ /tmp/install_scripts/

# Copy eget config file
COPY etc/eget/config.toml /etc/eget/config.toml

# Tell eget where its config file is
ENV EGET_CONFIG="/etc/eget/config.toml"

# List of needed packages
ENV APK_PACKAGES="curl perl-utils"

# Install the packages listed above, eget and the the packages defined in its config file
RUN apk add --no-cache ${APK_PACKAGES} && \
    cd /tmp/install_scripts/ && \
    ./install_eget.sh && \
    eget -D

### 2 - FINAL IMAGE STAGE ###

# Start from the build image as defined above with the ARGs
FROM ${BUILD_IMAGE}:${BUILD_IMAGE_TAG} AS final

# Switch to the root user temporarily
USER root

# Copy the packages downloaded by the downloader stage to /usr/local/bin
COPY --from=downloader /usr/local/bin/ /usr/local/bin/

# List of needed packages
ENV APT_PACKAGES="curl postgresql-17-pgvector vim wget"

# Install the packages listed above
RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y --no-install-recommends ${APT_PACKAGES} && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Switch back to the UID of the postgres user (26)
USER 26

# Set the work dir
WORKDIR /var/lib/postgresql/
