#!/bin/bash

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
    echo "Permission denied"
    exit 1
fi

# Check if Helmify is already installed
if command -v helmify &>/dev/null; then
    echo "Helmify is already installed. Use 'helmify --help' for more information."
    exit 0
fi

# Fetch latest version
latest_version=$(curl -sSL "https://github.com/arttor/helmify/releases/latest" | grep -o 'tag/[v.0-9]*"' | head -n1 | cut -d'/' -f2 | tr -d '"')

if [ -z "$latest_version" ]; then
    echo "Failed to fetch the latest version. Please check the repository."
    exit 1
fi

echo "Latest version found: $latest_version"

# Information about the machine's architecture
architecture=$(uname -m)

#  Show the system name
system_name=$(uname -s)

# Download the tarball
wget "https://github.com/arttor/helmify/releases/download/${latest_version}/helmify_${system_name}_${architecture}.tar.gz"

# Download the tarball
tar -xvf "helmify_Linux_x86_64.tar.gz" -C /usr/local/bin/

# Clean up the downloaded tarball
rm "helmify_Linux_x86_64.tar.gz"

echo "Helmify installed successfully."