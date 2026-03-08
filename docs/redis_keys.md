# Redis Key Documentation

本文档列出了 Bluebell 项目中使用的所有 Redis Key 及其说明。

| Key 前缀 / 完整 Key | 类型 | 说明 | 示例 |
| :--- | :--- | :--- | :--- |
| `bluebell:active_access_token:{userID}` | String | 用户的 Access Token，用于 API 鉴权。 | `bluebell:active_access_token:1001` |
| `bluebell:active_refresh_token:{userID}` | String | 用户的 Refresh Token，用于刷新 Access Token。 | `bluebell:active_refresh_token:1001` |
| `bluebell:post:time` | ZSet | 全局帖子列表，按发布时间排序。<br>**Score**: 发布时间戳<br>**Member**: 帖子 ID | `bluebell:post:time` |
| `bluebell:post:score` | ZSet | 全局帖子列表，按分数（热度）排序。<br>**Score**: 帖子分数<br>**Member**: 帖子 ID | `bluebell:post:score` |
| `bluebell:post:voted:{postID}` | ZSet | 记录帖子的投票情况。<br>**Score**: 投票类型 (1:赞成, -1:反对, 0:取消)<br>**Member**: 用户 ID | `bluebell:post:voted:12345` |
| `bluebell:community:post:time:{communityID}` | ZSet | 指定社区下的帖子列表，按发布时间排序。<br>**Score**: 发布时间戳<br>**Member**: 帖子 ID | `bluebell:community:post:time:1` |
| `bluebell:community:post:score:{communityID}` | ZSet | 指定社区下的帖子列表，按分数（热度）排序。<br>**Score**: 帖子分数<br>**Member**: 帖子 ID | `bluebell:community:post:score:1` |

## 详细说明

### 1. 帖子排序 (Post Sorting)
系统维护了两个全局 ZSet 用于帖子排序：
- **`bluebell:post:time`**: 所有的帖子 ID 都会加入此集合，Score 为发布时的 Unix 时间戳，Member 为 **帖子 ID**。用于实现“最新发布”列表。
- **`bluebell:post:score`**: 初始 Score 为发布时间戳，Member 为 **帖子 ID**。其实时分数会随着用户投票而更新。用于实现“热门”列表。

### 2. 社区帖子排序 (Community Post Sorting)
为了支持按社区筛选帖子，每个社区都有独立的两个 ZSet，逻辑与全局排序一致：
- **`bluebell:community:post:time:{communityID}`**: 集合 Key 中的 ID 是 **社区 ID**，Member 为 **帖子 ID**。
- **`bluebell:community:post:score:{communityID}`**: 集合 Key 中的 ID 是 **社区 ID**，Member 为 **帖子 ID**。

### 3. 投票记录 (Vote Records)
为了防止用户重复投票以及计算分数，每个帖子都有一个对应的 ZSet：
- **`bluebell:post:voted:{postID}`**
- Member 为 `userID`，Score 为投票值。
- `Score = 1`: 赞成票
- `Score = -1`: 反对票
- `Score = 0`: 取消投票 (数据可能会被移除)

### 4. 用户认证 (User Authentication)
用于 JWT 鉴权机制：
- **Access Token**: 短期有效，用于访问 API。
- **Refresh Token**: 长期有效，用于在 Access Token 过期后获取新的 Token。
