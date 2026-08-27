# Bluebell

Bluebell 是一个支持多社区发帖、二级楼中楼讨论与双信息流（热榜/关注流）的内容互动社区平台。

## User & Identity

**User**:
系统中的注册用户，拥有全局唯一标识，可作为作者、关注者或互动发起者。
_Avoid_: Account, Member, Client

**Primary Author**:
帖子的创建发起者与首要所有者，唯一拥有删除帖子或管理联合作者名单的终极权限。
_Avoid_: Owner, Master, Creator (when distinguishing from Co-Author)

**Co-Author**:
帖子的协作者与联合作者，享有联合署名权与协作编辑权，但无权单方面删除整篇帖子，仅可主动退出署名。
_Avoid_: Collaborator, Secondary Author, Contributor

**Follower / Following**:
用户之间的单向订阅关注关系，关注者（Follower）可接收被关注者（Following）的内容动态。
_Avoid_: Fan, Friend, Subscriber

## Content & Taxonomy

**Community**:
按主题划分的独立社区板块，帖子必须归属于且仅归属于一个社区。
_Avoid_: Subreddit, Board, Section, Forum

**Post**:
社区内的核心内容发布单元，包含标题、正文、作者列表、标签和互动统计。
_Avoid_: Thread, Article, Topic, Tweet

**Tag**:
跨社区的内容分类标签，用于内容聚类与细粒度检索。
_Avoid_: Category, Label, Hashtag

**ContentHash**:
基于帖子标题与内容计算的指纹特征，用于社区内的内容查重与幂等防重。
_Avoid_: Fingerprint, Checksum, Digest

## Discussion

**Comment**:
针对帖子的讨论内容，全站统一采用二级楼中楼结构（根评论与针对性回复）。
_Avoid_: Remark, Reply (as standalone entity), Review

**Root Comment**:
直接针对帖子发表的一级顶层评论（RootID=0），作为整个讨论楼层的根节点。
_Avoid_: First-level comment, Parent comment, Thread root

**Sub Comment**:
在根评论楼层内发表的二级回复。其 RootID 始终指向所属一级根评论，ParentID 指向直接回复对象，ReplyToUID 指向被回复者，全站展示逻辑保持二级扁平化。
_Avoid_: Nested comment, Floor reply, Multi-level thread

## Interaction & Ranking

**Vote**:
对帖子进行的方向性投票操作（+1 赞同 / -1 反对 / 0 取消），作为计算 Gravity 排序分值的核心输入；仅赞同票（Upvote）可触发互动通知。
_Avoid_: Like (for Post), Dislike, Upvote/Downvote (as separate standalone terms)

**Like**:
对评论进行的单向正向点赞操作，不支持负向反对。
_Avoid_: Vote (for Comment), Upvote (for Comment), Thumb

**Bookmark & BookmarkFolder**:
用户对帖子的收藏动作及分类管理的收藏夹目录。
_Avoid_: Favorite, Star, Collection

**Gravity Score**:
基于时间衰减与净投票数动态计算的帖子热度分值。
_Avoid_: Hotness, Weight, Popularity rank

## Notification

**Notification**:
由系统或用户互动触发的站内消息提醒，聚合为回复（Reply）、赞同/点赞（Like/Upvote）、关注（Follow）和系统（System）四大类别。
_Avoid_: Message, Alert, Notice

## Feed

**Hot Feed**:
全站或特定社区内按 Gravity Score 降序排列的热门内容信息流。
_Avoid_: Trending, Top, Popular Feed

**Following Feed**:
基于用户关注订阅关系聚合的创作者最新帖子动态流。
_Avoid_: Timeline, Subscription Feed, Friend Feed
