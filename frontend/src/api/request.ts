import { appwriteService } from '../services/appwriteService';

/**
 * 兼容原有 Axios 调用方式的 Appwrite 请求适配器
 * 
 * 将 /login, /signup, /posts, /post, /community, /vote, /comments 等接口
 * 转换为直接调用 Appwrite Web SDK。
 */
const request = {
  async get(url: string, config?: any) {
    // 1. 获取社区列表 /community
    if (url === '/community') {
      return appwriteService.getCommunities();
    }

    // 2. 获取单个社区详情 /community/:id
    if (url.startsWith('/community/')) {
      const id = url.replace('/community/', '').split('?')[0] || '';
      return appwriteService.getCommunity(id);
    }

    // 3. 获取帖子列表 /posts?...
    if (url.startsWith('/posts')) {
      const searchParams = new URLSearchParams(url.includes('?') ? url.split('?')[1] : '');
      const page = parseInt(searchParams.get('page') || '1', 10);
      const size = parseInt(searchParams.get('size') || '20', 10);
      const order = searchParams.get('order') || 'score';
      const community_id = searchParams.get('community_id') || undefined;

      return appwriteService.getPosts({ page, size, order, community_id });
    }

    // 4. 获取帖子详情 /post/:id
    if (url.startsWith('/post/')) {
      const id = url.replace('/post/', '').split('?')[0] || '';
      return appwriteService.getPost(id);
    }

    // 5. 获取评论列表 /comments
    if (url.startsWith('/comments')) {
      let postId = '';
      if (config?.params?.post_id) {
        postId = String(config.params.post_id);
      } else if (url.includes('post_id=')) {
        const searchParams = new URLSearchParams(url.split('?')[1]);
        postId = searchParams.get('post_id') || '';
      }
      return appwriteService.getComments(postId);
    }

    return { code: 404, msg: `Not Found: ${url}` };
  },

  async post(url: string, data?: any) {
    // 1. 登录 /login
    if (url === '/login') {
      return appwriteService.login(data.username, data.password);
    }

    // 2. 注册 /signup
    if (url === '/signup') {
      return appwriteService.signup(data.username, data.password);
    }

    // 3. 发帖 /post
    if (url === '/post') {
      return appwriteService.createPost(data.title, data.content, String(data.community_id));
    }

    // 4. 创建社区 /community
    if (url === '/community') {
      return appwriteService.createCommunity(data.name, data.introduction || data.description || '');
    }

    // 5. 投票 /vote
    if (url === '/vote') {
      return appwriteService.vote(String(data.post_id), data.direction);
    }

    // 6. 发表评论 /comment
    if (url === '/comment') {
      return appwriteService.createComment(
        String(data.post_id),
        data.content,
        String(data.root_id || '0'),
        String(data.parent_id || '0')
      );
    }

    return { code: 404, msg: `Not Found: ${url}` };
  },
};

export default request;