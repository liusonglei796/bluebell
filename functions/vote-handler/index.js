import { Client, Databases, Query, ID } from 'node-appwrite';

/**
 * Appwrite Cloud Function: vote-handler
 * 
 * 作用：原子化处理帖子投票与 HackerNews/Reddit Gravity 热度动态更新
 */
export default async ({ req, res, log, error }) => {
  const client = new Client()
    .setEndpoint(process.env.APPWRITE_FUNCTION_API_ENDPOINT || 'https://cloud.appwrite.io/v1')
    .setProject(process.env.APPWRITE_FUNCTION_PROJECT_ID)
    .setKey(process.env.APPWRITE_API_KEY);

  const databases = new Databases(client);

  const DATABASE_ID = process.env.APPWRITE_DATABASE_ID || 'bluebell_db';
  const POSTS_COLLECTION_ID = 'posts';
  const VOTES_COLLECTION_ID = 'votes';

  try {
    // 1. 获取当前调用用户 ID
    const userId = req.headers['x-appwrite-user-id'];
    if (!userId) {
      return res.json({ code: 1002, message: '未授权：请先登录' }, 401);
    }

    // 2. 解析请求体
    let body = req.body;
    if (typeof body === 'string') {
      try {
        body = JSON.parse(body);
      } catch (e) {
        return res.json({ code: 1001, message: '无效的 JSON 请求体' }, 400);
      }
    }

    const { postId, direction } = body || {};
    if (!postId || ![1, -1, 0].includes(direction)) {
      return res.json({ code: 1001, message: '参数错误：postId 必填且 direction 必须为 1, -1 或 0' }, 400);
    }

    // 3. 查询帖子是否存在
    const post = await databases.getDocument(DATABASE_ID, POSTS_COLLECTION_ID, postId);
    if (!post) {
      return res.json({ code: 1004, message: '帖子不存在' }, 404);
    }

    // 4. 查询该用户对该帖子的历史投票记录
    const existingVotes = await databases.listDocuments(
      DATABASE_ID,
      VOTES_COLLECTION_ID,
      [
        Query.equal('post_id', postId),
        Query.equal('user_id', userId),
        Query.limit(1)
      ]
    );

    let oldDirection = 0;
    let voteDocId = null;

    if (existingVotes.total > 0) {
      oldDirection = existingVotes.documents[0].direction;
      voteDocId = existingVotes.documents[0].$id;
    }

    // 如果方向一致，无需重复操作
    if (oldDirection === direction) {
      return res.json({
        code: 1000,
        message: '操作成功（无需变更）',
        data: {
          postId,
          direction,
          score: post.score,
          gravityScore: post.gravity_score,
        },
      });
    }

    // 5. 更新或创建投票记录
    if (direction === 0) {
      // 取消投票
      if (voteDocId) {
        await databases.deleteDocument(DATABASE_ID, VOTES_COLLECTION_ID, voteDocId);
      }
    } else if (voteDocId) {
      // 更新现有投票
      await databases.updateDocument(DATABASE_ID, VOTES_COLLECTION_ID, voteDocId, {
        direction,
      });
    } else {
      // 新建投票
      await databases.createDocument(DATABASE_ID, VOTES_COLLECTION_ID, ID.unique(), {
        post_id: postId,
        user_id: userId,
        direction,
      });
    }

    // 6. 计算票数变化
    let newUpvotes = post.upvotes || 0;
    let newDownvotes = post.downvotes || 0;

    // 撤销旧投票
    if (oldDirection === 1) newUpvotes -= 1;
    if (oldDirection === -1) newDownvotes -= 1;

    // 应用新投票
    if (direction === 1) newUpvotes += 1;
    if (direction === -1) newDownvotes += 1;

    const newScore = newUpvotes - newDownvotes;

    // 7. 计算 HackerNews / Reddit Gravity 热度算法
    // G = (Score - 1) / (Hours + 2)^1.8
    const createdAt = new Date(post.$createdAt).getTime();
    const now = Date.now();
    const hours = Math.max(0, (now - createdAt) / (1000 * 60 * 60));
    const gravityScore = parseFloat(((newScore - 1) / Math.pow(hours + 2, 1.8)).toFixed(4));

    // 8. 原子写回帖子统计
    const updatedPost = await databases.updateDocument(
      DATABASE_ID,
      POSTS_COLLECTION_ID,
      postId,
      {
        upvotes: Math.max(0, newUpvotes),
        downvotes: Math.max(0, newDownvotes),
        score: newScore,
        gravity_score: gravityScore,
      }
    );

    log(`投票成功: Post=${postId}, User=${userId}, Direction: ${oldDirection} -> ${direction}, NewScore=${newScore}`);

    return res.json({
      code: 1000,
      message: '投票成功',
      data: {
        postId,
        direction,
        upvotes: updatedPost.upvotes,
        downvotes: updatedPost.downvotes,
        score: updatedPost.score,
        gravityScore: updatedPost.gravity_score,
      },
    });
  } catch (err) {
    error(`投票处理异常: ${err.message}`);
    return res.json({ code: 5000, message: err.message || '内部服务器错误' }, 500);
  }
};
