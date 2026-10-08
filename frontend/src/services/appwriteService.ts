import { account, ID } from '../lib/appwrite';

const ENDPOINT = import.meta.env.VITE_APPWRITE_ENDPOINT || 'https://sgp.cloud.appwrite.io/v1';
const PROJECT_ID = import.meta.env.VITE_APPWRITE_PROJECT_ID || '6ab10b94003e7b324fdd';
const DATABASE_ID = import.meta.env.VITE_APPWRITE_DATABASE_ID || 'bluebell_db';

/**
 * 封装 TablesDB 的 REST API 调用，支持自动携带客户端 Session Cookie
 */
async function tableApi(path: string, method = 'GET', body: any = null) {
  const url = `${ENDPOINT}${path}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Appwrite-Project': PROJECT_ID,
  };

  const options: RequestInit = {
    method,
    headers,
    credentials: 'include', // 携带 Appwrite 客户端 Cookie Session
  };

  if (body) {
    options.body = JSON.stringify(body);
  }

  const res = await fetch(url, options);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.message || `Request failed with status ${res.status}`);
  }
  return data;
}

function toEmail(input: string): string {
  if (input.includes('@')) return input;
  return `${input.trim().toLowerCase()}@bluebell.com`;
}

export const appwriteService = {
  // ==================== 1. 认证鉴权 (Auth) ====================
  async login(usernameOrEmail: string, password: string) {
    const email = toEmail(usernameOrEmail);
    try {
      try {
        await account.createEmailPasswordSession(email, password);
      } catch (sessionErr: any) {
        // 如果当前已有活动 Session，忽略并直接复用
        if (!sessionErr.message?.includes('session is active')) {
          throw sessionErr;
        }
      }
      const user = await account.get();
      return {
        code: 1000,
        msg: 'success',
        data: {
          access_token: user.$id,
          refresh_token: user.$id,
          user_id: user.$id,
          username: user.name || usernameOrEmail,
          role: 1,
        },
      };
    } catch (err: any) {
      return {
        code: 1002,
        msg: err.message || '登录失败，请检查账号密码',
      };
    }
  },

  async signup(usernameOrEmail: string, password: string) {
    const email = toEmail(usernameOrEmail);
    const name = usernameOrEmail.split('@')[0];
    try {
      await account.create(ID.unique(), email, password, name);
      // 注册后自动登录建立 Session
      await account.createEmailPasswordSession(email, password);
      return {
        code: 1000,
        msg: 'success',
        data: null,
      };
    } catch (err: any) {
      return {
        code: 1001,
        msg: err.message || '注册失败',
      };
    }
  },

  async logout() {
    try {
      await account.deleteSession('current');
    } catch (e) {
      // ignore
    }
  },

  async getCurrentUser() {
    try {
      const user = await account.get();
      return {
        user_id: user.$id,
        username: user.name || user.email.split('@')[0] || 'User',
        role: 1,
      };
    } catch {
      return null;
    }
  },

  // ==================== 2. 社区板块 (Communities) ====================
  async getCommunities() {
    try {
      const res = await tableApi(`/tablesdb/${DATABASE_ID}/tables/communities/rows?limit=100`);
      const list = (res.rows || []).map((row: any) => ({
        id: row.$id,
        name: row.name,
        description: row.description || '',
        icon: row.icon || '',
        post_count: row.post_count || 0,
      }));
      return { code: 1000, msg: 'success', data: list };
    } catch (err: any) {
      return { code: 5000, msg: err.message, data: [] };
    }
  },

  async getCommunity(id: string) {
    try {
      const row = await tableApi(`/tablesdb/${DATABASE_ID}/tables/communities/rows/${id}`);
      return {
        code: 1000,
        msg: 'success',
        data: {
          id: row.$id,
          name: row.name,
          description: row.description || '',
          icon: row.icon || '',
          post_count: row.post_count || 0,
        },
      };
    } catch (err: any) {
      return { code: 4004, msg: err.message, data: null };
    }
  },

  async createCommunity(name: string, description: string) {
    try {
      const row = await tableApi(`/tablesdb/${DATABASE_ID}/tables/communities/rows`, 'POST', {
        rowId: 'unique()',
        data: {
          name,
          description,
          icon: '',
          post_count: 0,
        },
      });
      return { code: 1000, msg: 'success', data: { id: row.$id, name } };
    } catch (err: any) {
      return { code: 5000, msg: err.message };
    }
  },

  // ==================== 3. 帖子模块 (Posts) ====================
  async getPosts(params: { page?: number; size?: number; order?: string; community_id?: string }) {
    try {
      const page = params.page || 1;
      const size = params.size || 20;
      const offset = (page - 1) * size;
      
      let queryParams = `limit=${size}&offset=${offset}`;
      if (params.order === 'score') {
        queryParams += `&orderDesc=gravity_score`;
      } else {
        queryParams += `&orderDesc=$createdAt`;
      }
      if (params.community_id) {
        queryParams += `&queries[]=equal("community_id",["${params.community_id}"])`;
      }

      const res = await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows?${queryParams}`);
      const list = (res.rows || []).map((row: any) => ({
        id: row.$id,
        title: row.title,
        content: row.content,
        community_id: row.community_id,
        author_id: row.author_id,
        author_name: row.author_name,
        score: row.score || 0,
        upvotes: row.upvotes || 0,
        downvotes: row.downvotes || 0,
        comment_count: row.comment_count || 0,
        is_pinned: row.is_pinned || false,
        created_at: row.$createdAt,
      }));

      return { code: 1000, msg: 'success', data: list, total: res.total || list.length };
    } catch (err: any) {
      return { code: 5000, msg: err.message, data: [] };
    }
  },

  async getPost(id: string) {
    try {
      const row = await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows/${id}`);
      return {
        code: 1000,
        msg: 'success',
        data: {
          id: row.$id,
          title: row.title,
          content: row.content,
          community_id: row.community_id,
          author_id: row.author_id,
          author_name: row.author_name,
          score: row.score || 0,
          upvotes: row.upvotes || 0,
          downvotes: row.downvotes || 0,
          comment_count: row.comment_count || 0,
          created_at: row.$createdAt,
        },
      };
    } catch (err: any) {
      return { code: 4004, msg: err.message, data: null };
    }
  },

  async createPost(title: string, content: string, communityId: string) {
    try {
      const user = await account.get();
      const row = await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows`, 'POST', {
        rowId: 'unique()',
        data: {
          title,
          content,
          community_id: communityId,
          author_id: user.$id,
          author_name: user.name || user.email.split('@')[0],
          upvotes: 0,
          downvotes: 0,
          score: 0,
          gravity_score: 0.0,
          comment_count: 0,
          is_pinned: false,
        },
      });
      return { code: 1000, msg: 'success', data: { id: row.$id } };
    } catch (err: any) {
      return { code: 5000, msg: err.message };
    }
  },

  // ==================== 4. 投票计算 (Votes) ====================
  async vote(postId: string, direction: number) {
    try {
      const user = await account.get();
      const post = await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows/${postId}`);

      // 查询用户旧票
      const existing = await tableApi(
        `/tablesdb/${DATABASE_ID}/tables/votes/rows?queries[]=equal("post_id",["${postId}"])&queries[]=equal("user_id",["${user.$id}"])&limit=1`
      );

      let oldDirection = 0;
      let voteRowId = null;
      if (existing.total > 0 && existing.rows[0]) {
        oldDirection = existing.rows[0].direction;
        voteRowId = existing.rows[0].$id;
      }

      if (oldDirection === direction) {
        return { code: 1000, msg: 'success' };
      }

      // 更新或新建投票记录
      if (direction === 0) {
        if (voteRowId) await tableApi(`/tablesdb/${DATABASE_ID}/tables/votes/rows/${voteRowId}`, 'DELETE');
      } else if (voteRowId) {
        await tableApi(`/tablesdb/${DATABASE_ID}/tables/votes/rows/${voteRowId}`, 'PATCH', {
          data: { direction },
        });
      } else {
        await tableApi(`/tablesdb/${DATABASE_ID}/tables/votes/rows`, 'POST', {
          rowId: 'unique()',
          data: {
            post_id: postId,
            user_id: user.$id,
            direction,
          },
        });
      }

      // 计算并更新 post 统计
      let newUpvotes = post.upvotes || 0;
      let newDownvotes = post.downvotes || 0;
      if (oldDirection === 1) newUpvotes -= 1;
      if (oldDirection === -1) newDownvotes -= 1;
      if (direction === 1) newUpvotes += 1;
      if (direction === -1) newDownvotes += 1;
      const newScore = newUpvotes - newDownvotes;

      // Gravity 热度算法
      const createdAt = new Date(post.$createdAt).getTime();
      const hours = Math.max(0, (Date.now() - createdAt) / (1000 * 60 * 60));
      const gravityScore = parseFloat(((newScore - 1) / Math.pow(hours + 2, 1.8)).toFixed(4));

      await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows/${postId}`, 'PATCH', {
        data: {
          upvotes: Math.max(0, newUpvotes),
          downvotes: Math.max(0, newDownvotes),
          score: newScore,
          gravity_score: gravityScore,
        },
      });

      return { code: 1000, msg: 'success', data: { score: newScore } };
    } catch (err: any) {
      return { code: 5000, msg: err.message };
    }
  },

  // ==================== 5. 评论模块 (Comments) ====================
  async getComments(postId: string) {
    try {
      const res = await tableApi(
        `/tablesdb/${DATABASE_ID}/tables/comments/rows?queries[]=equal("post_id",["${postId}"])&orderDesc=$createdAt&limit=100`
      );

      const rootComments: any[] = [];
      const subRepliesMap = new Map<string, any[]>();

      for (const row of res.rows || []) {
        const item = {
          id: row.$id,
          post_id: row.post_id,
          author_id: row.author_id,
          author_name: row.author_name,
          content: row.content,
          root_id: row.root_id,
          parent_id: row.parent_id,
          created_at: row.$createdAt,
          sub_replies: [] as any[],
        };

        if (row.root_id === '0' || !row.root_id) {
          rootComments.push(item);
        } else {
          const rootKey = String(row.root_id);
          if (!subRepliesMap.has(rootKey)) {
            subRepliesMap.set(rootKey, []);
          }
          subRepliesMap.get(rootKey)!.push(item);
        }
      }

      for (const root of rootComments) {
        root.sub_replies = subRepliesMap.get(String(root.id)) || [];
      }

      return {
        code: 1000,
        msg: 'success',
        data: {
          comments: rootComments,
          total: res.total || rootComments.length,
        },
      };
    } catch (err: any) {
      return { code: 5000, msg: err.message, data: { comments: [], total: 0 } };
    }
  },

  async createComment(postId: string, content: string, rootId = '0', parentId = '0') {
    try {
      const user = await account.get();
      const row = await tableApi(`/tablesdb/${DATABASE_ID}/tables/comments/rows`, 'POST', {
        rowId: 'unique()',
        data: {
          post_id: postId,
          author_id: user.$id,
          author_name: user.name || user.email.split('@')[0],
          content,
          root_id: String(rootId || '0'),
          parent_id: String(parentId || '0'),
          like_count: 0,
        },
      });

      // 递增评论数
      try {
        const post = await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows/${postId}`);
        await tableApi(`/tablesdb/${DATABASE_ID}/tables/posts/rows/${postId}`, 'PATCH', {
          data: {
            comment_count: (post.comment_count || 0) + 1,
          },
        });
      } catch {}

      return { code: 1000, msg: 'success', data: { id: row.$id } };
    } catch (err: any) {
      return { code: 5000, msg: err.message };
    }
  },
};
