# GitHub Actions Release Workflow Fix

## Problem
The GoReleaser job was being skipped when a git tag already existed. This was observed in workflow run #18490192374.

## Root Cause
When a tag already existed:
1. The `check_tag` step would detect the tag exists and skip the subsequent steps
2. The `create_tag` step would be skipped (because the tag exists)
3. The `create_tag.outputs.tag` was never set (because the step was skipped)
4. The `goreleaser` job depended on `needs.auto-tag.outputs.tag` which referenced `steps.create_tag.outputs.tag`
5. Since this output was empty, the goreleaser job was skipped

## Solution
Modified the workflow to always output the tag value from the `check_tag` step, regardless of whether the tag already exists or not:

1. **Changed `check_tag` step**: Now outputs the tag in both branches (tag exists or doesn't exist)
2. **Changed job outputs**: Changed from `steps.create_tag.outputs.tag` to `steps.check_tag.outputs.tag`
3. **Removed duplicate output**: Removed the `tag` output from `create_tag` step since it's now set in `check_tag`

## Behavior After Fix
- **When tag doesn't exist**: Creates tag, reads changelog from version.txt, runs GoReleaser with custom changelog
- **When tag already exists**: Skips tag creation, uses GoReleaser automatic changelog, still creates the release

This allows re-running the workflow to regenerate a release even if the tag already exists.
