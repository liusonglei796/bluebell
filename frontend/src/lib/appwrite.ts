import { Client, Account, Databases, Storage, Functions, Query, ID } from 'appwrite';

const endpoint = import.meta.env.VITE_APPWRITE_ENDPOINT || 'https://cloud.appwrite.io/v1';
const projectId = import.meta.env.VITE_APPWRITE_PROJECT_ID || '6ab10b94003e7b324fdd';

export const DATABASE_ID = import.meta.env.VITE_APPWRITE_DATABASE_ID || 'bluebell_db';
export const STORAGE_BUCKET_ID = import.meta.env.VITE_APPWRITE_BUCKET_ID || 'bluebell_storage';

export const COLLECTIONS = {
  COMMUNITIES: 'communities',
  POSTS: 'posts',
  COMMENTS: 'comments',
  VOTES: 'votes',
  NOTIFICATIONS: 'notifications',
} as const;

export const client = new Client()
  .setEndpoint(endpoint)
  .setProject(projectId);

export const account = new Account(client);
export const databases = new Databases(client);
export const storage = new Storage(client);
export const functions = new Functions(client);

export { Query, ID };
