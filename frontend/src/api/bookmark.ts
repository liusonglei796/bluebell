import request from './request';

export interface BookmarkStatus {
  bookmarked: boolean;
  count: number;
}

export interface Bookmark {
  post_id: string;
  post_title?: string;
  author_name?: string;
  community_id?: number;
  community_name?: string;
  created_at: string;
}

export interface BookmarkListResponse {
  bookmarks: Bookmark[];
  total: number;
  page: number;
  size: number;
}

export const bookmarkAPI = {
  createBookmark: (postId: string) => {
    return request.post('/bookmark', { post_id: postId });
  },

  deleteBookmark: (postId: string) => {
    return request.delete(`/bookmark/${postId}`);
  },

  getBookmarkStatus: (postId: string) => {
    return request.get<{ code: number; data: BookmarkStatus }>(`/bookmark/${postId}/status`);
  },

  getUserBookmarks: (userId: string, page = 1, size = 20) => {
    return request.get<{ code: number; data: BookmarkListResponse }>(`/user/${userId}/bookmarks`, {
      params: { page, size },
    });
  },
};