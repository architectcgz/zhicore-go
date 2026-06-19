#!/usr/bin/env bash
# 初始化脚本：为 Codex 和 Claude 创建指向 ~/.agents/skills/ 的软链接
# 用途：克隆 ~/.agents/ 后运行一次，批量创建所有跨目录软链接
#
# 组织结构：
# 1. 扁平 skill: ~/.agents/skills/<skill>/SKILL.md
# 2. 容器 skill: ~/.agents/skills/<container>/<skill>/SKILL.md
#
# 链接策略：
# - 扁平 skill 直接链接到入口层（如 adapt -> ~/.agents/skills/adapt）
# - 容器 skill 只链接容器本身（如 superpowers -> ~/.agents/skills/superpowers）
#   不展开子 skill，保持容器语义

set -euo pipefail

AGENTS_SKILLS_DIR="$HOME/.agents/skills"
CODEX_SKILLS_DIR="$HOME/.codex/skills"
CLAUDE_SKILLS_DIR="$HOME/.claude/skills"

# 检查 ~/.agents/skills 是否存在
if [ ! -d "$AGENTS_SKILLS_DIR" ]; then
  echo "错误: $AGENTS_SKILLS_DIR 不存在"
  exit 1
fi

# 获取一级目录（扁平 skills 和容器目录）
readarray -t TOP_LEVEL_ITEMS < <(ls -1 "$AGENTS_SKILLS_DIR" | grep -v "^README")

echo "找到 ${#TOP_LEVEL_ITEMS[@]} 个一级项目（扁平 skills + 容器）"

# 辅助函数：创建软链接
# $1: 目标目录（如 ~/.codex/skills）
# $2: skill/容器名称
# $3: 源路径
create_link() {
  local target_dir=$1
  local item_name=$2
  local source_path=$3
  local target="$target_dir/$item_name"

  if [ -L "$target" ]; then
    # 已存在软链接，检查是否指向正确
    current_target=$(readlink "$target")
    if [ "$current_target" = "$source_path" ]; then
      echo "  ✓ $item_name (已存在且正确)"
    else
      echo "  ! $item_name (已存在但指向 $current_target，跳过)"
    fi
  elif [ -e "$target" ]; then
    # 存在同名文件或目录
    echo "  ! $item_name (已存在实体文件/目录，跳过)"
  else
    # 创建软链接
    ln -s "$source_path" "$target"
    echo "  + $item_name (已创建)"
  fi
}

# 为 Codex 创建软链接
if [ -d "$CODEX_SKILLS_DIR" ]; then
  echo ""
  echo "=== 为 Codex 创建软链接 ==="
  for item in "${TOP_LEVEL_ITEMS[@]}"; do
    source_path="$AGENTS_SKILLS_DIR/$item"
    create_link "$CODEX_SKILLS_DIR" "$item" "$source_path"
  done
else
  echo ""
  echo "警告: $CODEX_SKILLS_DIR 不存在，跳过 Codex 软链接"
fi

# 为 Claude 创建软链接
if [ -d "$CLAUDE_SKILLS_DIR" ]; then
  echo ""
  echo "=== 为 Claude 创建软链接 ==="
  for item in "${TOP_LEVEL_ITEMS[@]}"; do
    source_path="$AGENTS_SKILLS_DIR/$item"
    create_link "$CLAUDE_SKILLS_DIR" "$item" "$source_path"
  done
else
  echo ""
  echo "警告: $CLAUDE_SKILLS_DIR 不存在，跳过 Claude 软链接"
fi

echo ""
echo "完成！"
