# 统一评论体系与双 Feed 流领域架构

## Context
项目初期混杂了单层简易留言（`Remark`）与二级楼中楼评论（`Comment`）两套实现，且对于信息流（Feed）与互动方式（Vote vs Like）的概念边界不够明确，造成了命名混乱和架构冗余。

## Decision
1. **全面归一化为 Comment**：废弃所有 `Remark` 遗留实现与表结构，全站统一采用两级楼中楼结构 `Comment`（根评论与针对性回复），并通过纯 RabbitMQ 投递 `event.comment.created` 领域事件。
2. **双 Feed 体系解耦**：
   - **Hot Feed（热榜流）**：基于 HackerNews/Reddit 的 Gravity 算法热度评分（`Gravity Score`）与时间衰减，由 Redis ZSet 承载全站与社区的热门排序。
   - **Following Feed（关注流）**：基于用户之间的 `Follow` 订阅关系，采用推拉结合（Fanout）聚合关注创作者的最新帖子。
3. **交互方式语义划分**：
   - 对 **Post** 进行双向 `Vote`（赞同 +1 / 反对 -1 / 取消 0），直接作用于 Gravity 排序。
   - 对 **Comment** 仅进行单向 `Like`（点赞），不支持负向踩。
