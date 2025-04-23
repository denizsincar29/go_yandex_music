# This script is used to create a new git tag based on the version specified in version.txt.
# It is needet to ensure you don't forget to change both the version in version.txt and the git tag.
# It uses the semantic_version library to compare versions.

import git
import semantic_version

# initialize the repository
repo = git.Repo()
# check if the repository is clean (no uncommitted changes)
if repo.is_dirty(untracked_files=True):
    print("Repository is dirty. Please commit or stash your changes before tagging.")
    exit(1)
# find the latest tag
latest_tag = repo.git.describe(tags=True, always=True, long=True).split('-')[0]
print(f"Latest tag: {latest_tag}")
tag_version = semantic_version.Version(latest_tag[1:])  # It so much doesn't like the 'v' prefix
# compare it with version.txt
with open('version.txt', 'r') as f:
    version = f.read().strip()
print(f"Version in version.txt: {version}")
file_version = semantic_version.Version(version)
# check if the file version is greater than the tag version
if file_version > tag_version:
    # if so, create a new tag
    new_tag = str(file_version)
    print(f"Creating new tag: {new_tag}")
    tag_message = input("Enter tag message: ")
    repo.create_tag("v"+new_tag, message=tag_message)
    # push the new tag to the remote repository
    repo.git.push('origin', new_tag)  # GoReleaser, take it!
else:
    print("No new tag created. The version in version.txt is not greater than the latest tag.")