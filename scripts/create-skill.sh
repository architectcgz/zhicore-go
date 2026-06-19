#!/bin/bash
# Create a new skill in ~/.agents/skills/ with automatic symlinks to Claude and Codex

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <skill-name>"
    echo "Example: $0 my-new-skill"
    exit 1
fi

SKILL_NAME="$1"
SKILL_DIR="$HOME/.agents/skills/$SKILL_NAME"

# Check if skill already exists
if [ -e "$SKILL_DIR" ]; then
    echo "Error: Skill '$SKILL_NAME' already exists at $SKILL_DIR"
    exit 1
fi

# Create skill directory
mkdir -p "$SKILL_DIR"
echo "✓ Created skill directory: $SKILL_DIR"

# Create SKILL.md template
cat > "$SKILL_DIR/SKILL.md" << 'EOF'
---
name: SKILL_NAME_PLACEHOLDER
description: Brief description of when to use this skill
---

# Skill Name

Brief description of what this skill does.

## When to Use

- Condition 1
- Condition 2

## Workflow

### Step 1: Description

Details...

### Step 2: Description

Details...

## Example

```bash
# Example commands
```

## Related Skills

- skill-name: description
EOF

# Replace placeholder
sed -i "s/SKILL_NAME_PLACEHOLDER/$SKILL_NAME/" "$SKILL_DIR/SKILL.md"
echo "✓ Created SKILL.md template"

# Create symlinks for Claude
if [ -d "$HOME/.claude/skills" ]; then
    if [ -e "$HOME/.claude/skills/$SKILL_NAME" ]; then
        rm -rf "$HOME/.claude/skills/$SKILL_NAME"
    fi
    ln -s "$SKILL_DIR" "$HOME/.claude/skills/"
    echo "✓ Created symlink for Claude"
else
    echo "⚠ Warning: ~/.claude/skills not found, skipping Claude symlink"
fi

# Create symlinks for Codex
if [ -d "$HOME/.codex/skills" ]; then
    if [ -e "$HOME/.codex/skills/$SKILL_NAME" ]; then
        rm -rf "$HOME/.codex/skills/$SKILL_NAME"
    fi
    ln -s "$SKILL_DIR" "$HOME/.codex/skills/"
    echo "✓ Created symlink for Codex"
else
    echo "⚠ Warning: ~/.codex/skills not found, skipping Codex symlink"
fi

echo ""
echo "✅ Skill '$SKILL_NAME' created successfully!"
echo ""
echo "Next steps:"
echo "1. Edit $SKILL_DIR/SKILL.md"
echo "2. cd ~/.agents && git add skills/$SKILL_NAME/"
echo "3. git commit -m 'Add: $SKILL_NAME skill'"
echo "4. git push"
