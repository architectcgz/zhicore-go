# zhicore-search HTTP Schema

本目录记录 `zhicore-search` 的对外 HTTP contract。当前仅做计划化占位，字段级 endpoint schema 待后续按切片提取。

## Provider Owner

Search 拥有搜索索引、搜索建议、热门搜索词和搜索历史读模型。Search 不拥有文章、用户、评论或排行分数；结果中的文章详情以 Content contract 为事实源。

## 首批 endpoint 候选

| 方法 | 路径 | 用途 | 状态 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/search/posts` | 文章搜索 | API 族已识别 |
| `GET` | `/api/v1/search/suggest` | 搜索建议 | API 族已识别 |
| `GET` | `/api/v1/search/hot` | 热门搜索词 | API 族已识别 |
| `GET` | `/api/v1/search/history` | 当前用户搜索历史 | API 族已识别 |
| `DELETE` | `/api/v1/search/history` | 清理当前用户搜索历史 | API 族已识别 |

## 待提取 contract

- 搜索 query、分页、排序、高亮、降级和可见性滞后 SLA。
- 搜索结果中的 Content 摘要字段来源和是否回源补齐。
- 搜索历史的登录态、保留时间和清理语义。

## 禁止规则

- 不复制 Content / User DTO 为第二事实源。
- 不把搜索结果当作最终可见性判断；详情页仍以 Content 为准。
- 暂不创建前端 `src/api/search.ts`，直到至少一个 endpoint 达到 `Contract 草案`。
