#!/bin/bash
# Install gofile ansible roles and playbooks into current project
# Usage: curl -sL https://raw.githubusercontent.com/mars-base/gofile/main/ansible/install.sh | bash
#    or: bash ansible/install.sh [target_dir]

set -e

REPO="https://github.com/mars-base/gofile.git"
BRANCH="${GOFILE_VERSION:-main}"
TARGET="${1:-.}"
TMPDIR=$(mktemp -d)

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

echo "==> Cloning gofile ($BRANCH)..."
git clone --depth 1 --branch "$BRANCH" "$REPO" "$TMPDIR/gofile" 2>/dev/null

# Install roles
echo "==> Installing roles..."
for role in gofile; do
    if [ -d "$TARGET/roles/$role" ]; then
        echo "    roles/$role already exists, updating..."
        rm -rf "$TARGET/roles/$role"
    fi
    mkdir -p "$TARGET/roles"
    cp -r "$TMPDIR/gofile/ansible/roles/$role" "$TARGET/roles/$role"
    echo "    roles/$role installed"
done

# Install playbooks
echo "==> Installing playbooks..."
mkdir -p "$TARGET/playbooks"
for pb in gofile.yml; do
    cp "$TMPDIR/gofile/ansible/playbooks/$pb" "$TARGET/playbooks/$pb"
    echo "    playbooks/$pb installed"
done

echo ""
echo "Done. Usage:"
echo "  ansible-playbook -i hosts.ini playbooks/gofile.yml -e \"HOSTS=servers\""
