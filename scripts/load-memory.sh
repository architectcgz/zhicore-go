#!/bin/bash
# 按优先级和关键词加载记忆

TOOL="${1:-claude}"
MEMORY_ROOT="$HOME/.agents/memory"
INDEX="$MEMORY_ROOT/MEMORY.md"

# 默认加载高优先级记忆
PRIORITY="${2:-high}"

echo "📚 Loading ${TOOL} memory (priority: ${PRIORITY})..."

# 1. 始终加载索引
if [ -f "$INDEX" ]; then
    echo "  ✓ Index loaded: $INDEX"
fi

# 2. 加载共享记忆
SHARED_COUNT=0
if [ -d "$MEMORY_ROOT/shared" ]; then
    for file in "$MEMORY_ROOT/shared"/*.md; do
        [ -f "$file" ] || continue

        # 检查优先级
        FILE_PRIORITY=$(grep -A 3 "^metadata:" "$file" | grep "priority:" | awk '{print $2}' || echo "medium")

        if [ "$PRIORITY" == "all" ] || [ "$PRIORITY" == "$FILE_PRIORITY" ]; then
            echo "  ✓ Shared: $(basename "$file") [${FILE_PRIORITY}]"
            ((SHARED_COUNT++))
        fi
    done
fi

# 3. 加载工具专属记忆
TOOL_COUNT=0
if [ -d "$MEMORY_ROOT/$TOOL" ]; then
    for file in "$MEMORY_ROOT/$TOOL"/*.md; do
        [ -f "$file" ] || continue

        FILE_PRIORITY=$(grep -A 3 "^metadata:" "$file" | grep "priority:" | awk '{print $2}' || echo "medium")

        if [ "$PRIORITY" == "all" ] || [ "$PRIORITY" == "$FILE_PRIORITY" ]; then
            echo "  ✓ ${TOOL}: $(basename "$file") [${FILE_PRIORITY}]"
            ((TOOL_COUNT++))
        fi
    done
fi

echo ""
echo "Summary: ${SHARED_COUNT} shared + ${TOOL_COUNT} tool-specific = $((SHARED_COUNT + TOOL_COUNT)) files loaded"
echo ""
echo "Usage:"
echo "  ./load-memory.sh claude        # Load high priority only"
echo "  ./load-memory.sh claude medium # Load medium priority"
echo "  ./load-memory.sh claude all    # Load all memories"
