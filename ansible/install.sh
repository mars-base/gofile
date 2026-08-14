#!/bin/bash
# Install gofile ansible roles and playbooks into your project
# Usage: curl -sL https://raw.githubusercontent.com/mars-base/gofile/main/ansible/install.sh | bash
#    or: curl -sL https://raw.githubusercontent.com/mars-base/gofile/main/ansible/install.sh | bash -s -- /path/to/target

set -e

REPO="https://github.com/mars-base/gofile.git"
BRANCH="${GOFILE_VERSION:-main}"
TMPDIR=$(mktemp -d)

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

echo "==> Cloning gofile ($BRANCH)..."
git clone --depth 1 --branch "$BRANCH" "$REPO" "$TMPDIR/gofile" 2>/dev/null

# Determine target directory
TARGET="${1:-.}"

mkdir -p "$TARGET"
echo "==> Target directory: $TARGET"

# Install role
echo "==> Installing role..."
if [ -d "$TARGET/roles/gofile" ]; then
    echo "    roles/gofile already exists, updating..."
    rm -rf "$TARGET/roles/gofile"
fi
mkdir -p "$TARGET/roles"
cp -r "$TMPDIR/gofile/ansible/roles/gofile" "$TARGET/roles/gofile"
echo "    ✓ roles/gofile"

# Install playbook
echo "==> Installing playbook..."
mkdir -p "$TARGET/playbooks/tools"
cp "$TMPDIR/gofile/ansible/playbooks/gofile.yml" "$TARGET/playbooks/tools/gofile.yml"
echo "    ✓ playbooks/tools/gofile.yml"

echo ""
echo "==> Done!"
echo ""
echo "Usage:"
echo "  ansible-playbook -i hosts.ini $TARGET/playbooks/tools/gofile.yml -e \"HOSTS=servers\""
echo ""
echo "  # Deploy with custom instances"
echo "  ansible-playbook -i hosts.ini $TARGET/playbooks/tools/gofile.yml -e \"HOSTS=servers\" -e @extra_vars.yml"
