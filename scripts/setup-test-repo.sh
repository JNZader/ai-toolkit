#!/bin/bash
# scripts/setup-test-repo.sh

set -e

REPO_NAME="goreview-test-repo"
GITHUB_USER=$(gh api user -q .login)

echo "Setting up test repository..."

# Create repo if not exists
if ! gh repo view "$GITHUB_USER/$REPO_NAME" > /dev/null 2>&1; then
    gh repo create "$REPO_NAME" --private --description "Test repo for GoReview"
    echo "Created repository: $GITHUB_USER/$REPO_NAME"
fi

# Clone locally
TEMP_DIR=$(mktemp -d)
gh repo clone "$GITHUB_USER/$REPO_NAME" "$TEMP_DIR"
cd "$TEMP_DIR"

# Create test files with various issues
cat > main.py << 'EOF'
"""Main module with intentional issues for testing."""
import pickle
import yaml
import subprocess

# TODO: refactor this function
def process_user_input(user_input):
    """Process user input unsafely."""
    # SQL injection vulnerability
    query = f"SELECT * FROM users WHERE name = '{user_input}'"

    # Command injection
    subprocess.call(f"echo {user_input}", shell=True)

    # Unsafe deserialization
    data = pickle.loads(user_input)

    # Unsafe YAML
    config = yaml.load(user_input)

    # Hardcoded password
    password = "admin123"

    return data

def main():
    print("Hello")  # Debug print
    # debugger  # Debugger statement

if __name__ == "__main__":
    main()
EOF

# Commit initial files
git add .
git commit -m "Add test files with intentional issues"
git push origin main

# Create a PR
git checkout -b test-pr
echo "// More issues" >> main.py
git add .
git commit -m "Add more issues for testing"
git push -u origin test-pr

gh pr create --title "Test PR for GoReview" --body "This PR contains intentional issues for testing GoReview"

echo ""
echo "Test repository setup complete!"
echo "Repository: https://github.com/$GITHUB_USER/$REPO_NAME"
echo ""

cd -
rm -rf "$TEMP_DIR"
