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
mkdir -p "$TARGET/playbooks"
cp "$TMPDIR/gofile/ansible/playbooks/gofile.yml" "$TARGET/playbooks/gofile.yml"
echo "    ✓ playbooks/gofile.yml"

# Create ansible.cfg if not exists (so ansible finds roles/ from playbooks/)
if [ ! -f "$TARGET/ansible.cfg" ]; then
    cat > "$TARGET/ansible.cfg" <<'EOF'
[defaults]
roles_path = ../roles
inventory = hosts.ini
EOF
    echo "    ✓ ansible.cfg"
fi

# Create hosts.ini if not exists
if [ ! -f "$TARGET/hosts.ini" ]; then
    cat > "$TARGET/hosts.ini" <<'EOF'
[servers]
# server1 ansible_host=192.168.1.100
# server2 ansible_host=192.168.1.101

[all:vars]
ansible_user=root
# ansible_ssh_private_key_file=~/.ssh/id_rsa
EOF
    echo "    ✓ hosts.ini"
fi

echo ""
echo "==> Done!"
echo ""
echo "Usage:"
echo "  # 1. Edit $TARGET/hosts.ini to add your servers"
echo "  # 2. Deploy (default: 2 instances on port 8080/8081)"
echo "  cd $TARGET && ansible-playbook playbooks/gofile.yml -e \"HOSTS=servers\""
echo ""
echo "  # Deploy with custom instances"
echo "  cd $TARGET && ansible-playbook playbooks/gofile.yml -e \"HOSTS=servers\" -e @extra_vars.yml"
