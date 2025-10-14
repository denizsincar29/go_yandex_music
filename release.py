# This script is used to create a new git tag based on the version specified in version.txt.
# It is needed to ensure you don't forget to change both the version in version.txt and the git tag.
# It uses the semantic_version library to compare versions.
#
# version.txt format:
# - First line: version number (e.g., 1.0.3)
# - Subsequent lines (optional): changelog description
#
# Usage: python release.py
# Or use the GitHub Actions workflow: gh workflow run release.yml

import git
import semantic_version

# initialize the repository
repo = git.Repo()
# check if the repository is clean (no uncommitted changes)
if repo.is_dirty(untracked_files=True):
    print("Repository is dirty. Please commit or stash your changes before tagging.")
    exit(1)

# find the latest tag
try:
    latest_tag = repo.git.describe(tags=True, always=True, long=True).split('-')[0]
    print(f"Latest tag: {latest_tag}")
    tag_version = semantic_version.Version(latest_tag[1:])  # It so much doesn't like the 'v' prefix
except Exception as e:
    print(f"No previous tags found: {e}")
    tag_version = semantic_version.Version('0.0.0')

# compare it with version.txt (first line is the version)
with open('version.txt', 'r') as f:
    lines = f.read().strip().split('\n')
    version = lines[0].strip()
    changelog = '\n'.join(lines[1:]).strip() if len(lines) > 1 else ''

print(f"Version in version.txt: {version}")
if changelog:
    print(f"Changelog: {changelog}")

file_version = semantic_version.Version(version)

# check if the file version is greater than the tag version
if file_version > tag_version:
    # if so, create a new tag
    new_tag = str(file_version)
    print(f"Creating new tag: v{new_tag}")
    
    # Use changelog from version.txt or ask for tag message
    if changelog:
        tag_message = f"Release v{new_tag}\n\n{changelog}"
        print(f"Using changelog from version.txt as tag message")
    else:
        tag_message = input("Enter tag message: ")
    
    repo.create_tag("v"+new_tag, message=tag_message)
    # push the new tag to the remote repository
    repo.git.push('origin', "v"+new_tag)  # GoReleaser, take it!
    print(f"Tag v{new_tag} created and pushed successfully!")
else:
    print("No new tag created. The version in version.txt is not greater than the latest tag.")
    print(f"Current version: {file_version}, Latest tag: {tag_version}")