# Git 与工作区

- 涉及项目仓库内的实现类改动、修复、补测试、重构或提交时，默认优先使用隔离工作区；如果当前主工作区干净、当前任务是唯一活动任务、且不需要并行隔离，则可以直接在主工作区完成，不强制新建 worktree。
- 如果主工作区已有未提交改动、已经承载其他任务、需要并行推进多个任务，或用户明确要求隔离开发，则必须先通过 `git worktree add` 创建共享独立工作区，再在该工作区内完成开发和提交。
- 一个任务只允许使用一个共享 worktree；禁止在 worktree 内再次创建 worktree。
- 当任务直接在干净主工作区执行时，仍然要保持“一次只做一个任务边界”的提交纪律；不要因为省去 worktree 就混入其他问题的改动。
- 对带前端依赖的仓库，如果为任务创建了 worktree，默认优先复用主工作区已经安装好的 `node_modules`；不要再为单个项目额外发明 `bootstrap-frontend-deps.sh` 之类的专项脚本，除非现有共享/项目内依赖发现机制已经证明不够用。
- 对全局 agent 配置、个人工具配置、仓库外文件、纯说明性文档的小范围编辑，不默认触发 worktree；除非用户明确要求，或这些文件本身属于当前仓库并需要按正常开发流程提交。
- commit 约定（提交时机、最小可审阅与按问题边界拆分、message 格式、默认禁止 `Co-Authored-By`（除非用户主动声明或项目显式允许）、按仓库 commit policy / hook 叠加）统一以全局 `committing-changes` skill 为事实源；提交前先按该 skill 的 ✓Check 自查。
- 当用户说“合并回去”“合并分支”“merge 回主线/主分支”时，默认理解为执行真实的 Git 合并语义（`git merge`、`git cherry-pick`、`git rebase` 后 fast-forward，或用户明确指定的等价历史操作）；不得擅自降级成手动拷贝文件、直接补丁回填或仅把改动带到工作区。若当前工作区不适合直接合并，应先说明阻塞点并切换到合适的干净集成上下文，再继续完成真正的合并。
- 默认推送顺序：先尝试通过 `gh` 的 credential helper 走 HTTPS 推送；只有在 `gh` 不可用、未登录，或仓库/环境明确只能走 SSH 时，才回退到 SSH 推送。
- 推荐推送形式：
  `git -c credential.helper='!gh auth git-credential' push https://github.com/<owner>/<repo>.git <src-ref>:<dst-ref>`
- 若 `gh auth status` 不可用或未登录，再使用常规 `git push origin ...` 或 SSH 路径。
- 新建项目、初始化项目级 `AGENTS.md`，或补修 agent 入口时，必须同时创建项目根 `CLAUDE.md -> AGENTS.md`，并运行 `bash ~/workspace/projects/scripts/check-agent-entrypoints.sh <project-root>` 验证；需要核查全局入口时运行 `bash ~/workspace/projects/scripts/check-agent-entrypoints.sh --global`。
- 当修改 `~/.agents/harness/workflows/<workflow-name>/` 下的 shared workflow package 后，结束本轮前必须对目标仓库显式执行一次 `bash ~/.agents/harness/workflow-sync.sh <repo-root> <workflow-name>`；不要假设项目会自动同步。
