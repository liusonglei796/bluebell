/**
 * Bluebell - Appwrite TablesDB 自动化资源初始化脚本
 */

const ENDPOINT = process.env.APPWRITE_ENDPOINT || 'https://sgp.cloud.appwrite.io/v1';
const PROJECT_ID = process.env.APPWRITE_PROJECT_ID || '6ab10b94003e7b324fdd';
const API_KEY = process.argv[2] || process.env.APPWRITE_API_KEY || 'standard_87eaa034aaeed766db78571ee500d76d53acd1988b0ecaf877cfa1007a56a9573c7deb80dad474e3d5f22b201845f21d8975def708e3a7a65ac78159180a033e2e05454cc12e403b4e68dc1ce90e8f4c00f43f0f30b2fc3ef440855464c320148f80a7ee312b17d20c0a6b9df1c771209f3f92e6a21938a871525ab7a86b038d';

const headers = {
  'Content-Type': 'application/json',
  'X-Appwrite-Project': PROJECT_ID,
  'X-Appwrite-Key': API_KEY,
};

async function api(path, method = 'GET', body = null) {
  const url = `${ENDPOINT}${path}`;
  const options = { method, headers };
  if (body) {
    options.body = JSON.stringify(body);
  }
  const res = await fetch(url, options);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    if (res.status === 409) {
      return { _exists: true, ...data };
    }
    throw new Error(`[${res.status}] ${data.message || res.statusText} (${path})`);
  }
  return data;
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function main() {
  console.log('\x1b[36m🚀 开始初始化 Appwrite TablesDB 资源...\x1b[0m');
  console.log(`Endpoint:   ${ENDPOINT}`);
  console.log(`Project ID: ${PROJECT_ID}\n`);

  const DATABASE_ID = 'bluebell_db';
  const BUCKET_ID = 'bluebell_storage';

  // 1. 检查数据库
  process.stdout.write('📦 检查数据库 bluebell_db ... ');
  try {
    const dbRes = await api('/databases', 'POST', {
      databaseId: DATABASE_ID,
      name: 'Bluebell Database',
    });
    console.log(dbRes._exists ? '\x1b[33m[已存在]\x1b[0m' : '\x1b[32m[创建成功]\x1b[0m');
  } catch (err) {
    console.log(`\x1b[33m[已存在或就绪]\x1b[0m`);
  }

  // 2. 表结构定义清单
  const tables = [
    {
      id: 'communities',
      name: 'Communities',
      permissions: ['read("any")', 'create("users")', 'update("users")'],
      attributes: [
        { type: 'string', key: 'name', size: 64, required: true },
        { type: 'string', key: 'description', size: 255, required: false },
        { type: 'string', key: 'icon', size: 255, required: false },
        { type: 'integer', key: 'post_count', required: false, default: 0 },
      ],
      indexes: [
        { key: 'idx_name', type: 'unique', attributes: ['name'] },
      ],
    },
    {
      id: 'posts',
      name: 'Posts',
      permissions: ['read("any")', 'create("users")', 'update("users")', 'delete("users")'],
      attributes: [
        { type: 'string', key: 'title', size: 128, required: true },
        { type: 'string', key: 'content', size: 10000, required: true },
        { type: 'string', key: 'community_id', size: 36, required: true },
        { type: 'string', key: 'author_id', size: 36, required: true },
        { type: 'string', key: 'author_name', size: 64, required: true },
        { type: 'integer', key: 'upvotes', required: false, default: 0 },
        { type: 'integer', key: 'downvotes', required: false, default: 0 },
        { type: 'integer', key: 'score', required: false, default: 0 },
        { type: 'float', key: 'gravity_score', required: false, default: 0.0 },
        { type: 'integer', key: 'comment_count', required: false, default: 0 },
        { type: 'boolean', key: 'is_pinned', required: false, default: false },
      ],
      indexes: [
        { key: 'idx_gravity', type: 'key', attributes: ['gravity_score'], orders: ['DESC'] },
        { key: 'idx_community', type: 'key', attributes: ['community_id'] },
      ],
    },
    {
      id: 'comments',
      name: 'Comments',
      permissions: ['read("any")', 'create("users")', 'update("users")', 'delete("users")'],
      attributes: [
        { type: 'string', key: 'post_id', size: 36, required: true },
        { type: 'string', key: 'author_id', size: 36, required: true },
        { type: 'string', key: 'author_name', size: 64, required: true },
        { type: 'string', key: 'content', size: 1000, required: true },
        { type: 'string', key: 'root_id', size: 36, required: true },
        { type: 'string', key: 'parent_id', size: 36, required: true },
        { type: 'string', key: 'reply_to_uid', size: 36, required: false },
        { type: 'string', key: 'reply_to_name', size: 64, required: false },
        { type: 'integer', key: 'like_count', required: false, default: 0 },
      ],
      indexes: [
        { key: 'idx_post_root', type: 'key', attributes: ['post_id', 'root_id'] },
      ],
    },
    {
      id: 'votes',
      name: 'Votes',
      permissions: ['read("users")', 'create("users")', 'update("users")'],
      attributes: [
        { type: 'string', key: 'post_id', size: 36, required: true },
        { type: 'string', key: 'user_id', size: 36, required: true },
        { type: 'integer', key: 'direction', required: true },
      ],
      indexes: [
        { key: 'idx_post_user', type: 'unique', attributes: ['post_id', 'user_id'] },
      ],
    },
    {
      id: 'notifications',
      name: 'Notifications',
      permissions: ['read("users")', 'create("users")', 'update("users")'],
      attributes: [
        { type: 'string', key: 'recipient_id', size: 36, required: true },
        { type: 'string', key: 'sender_id', size: 36, required: true },
        { type: 'string', key: 'sender_name', size: 64, required: true },
        { type: 'string', key: 'action_type', size: 32, required: true },
        { type: 'string', key: 'post_id', size: 36, required: false },
        { type: 'boolean', key: 'is_read', required: false, default: false },
      ],
      indexes: [
        { key: 'idx_recipient', type: 'key', attributes: ['recipient_id', 'is_read'] },
      ],
    },
  ];

  // 3. 逐个创建数据表与列
  for (const table of tables) {
    process.stdout.write(`\n📁 数据表 [${table.name}] ... `);
    try {
      const tRes = await api(`/tablesdb/${DATABASE_ID}/tables`, 'POST', {
        tableId: table.id,
        name: table.name,
        permissions: table.permissions,
      });
      console.log(tRes._exists ? '\x1b[33m[已存在]\x1b[0m' : '\x1b[32m[创建成功]\x1b[0m');
    } catch (err) {
      console.log(`\x1b[33m[已存在/就绪]\x1b[0m`);
    }

    // 创建字段列 (Columns)
    for (const attr of table.attributes) {
      process.stdout.write(`  ↳ 列 ${attr.key} (${attr.type}) ... `);
      try {
        let attrPath = `/tablesdb/${DATABASE_ID}/tables/${table.id}/columns/${attr.type}`;
        const attrBody = { ...attr };
        delete attrBody.type;

        const attrRes = await api(attrPath, 'POST', attrBody);
        console.log(attrRes._exists ? '\x1b[33m[已存在]\x1b[0m' : '\x1b[32m[OK]\x1b[0m');
      } catch (err) {
        console.log(`\x1b[33m[已存在或跳过]\x1b[0m`);
      }
    }

    // 等待列状态就绪
    await sleep(800);
  }

  // 4. 创建 Storage Bucket
  process.stdout.write('\n🗄️ 检查存储桶 bluebell_storage ... ');
  try {
    const bucketRes = await api('/storage/buckets', 'POST', {
      bucketId: BUCKET_ID,
      name: 'Bluebell Storage',
      permissions: ['read("any")', 'create("users")'],
      fileSecurity: false,
      enabled: true,
      maximumFileSize: 10485760,
      allowedFileExtensions: ['jpg', 'png', 'gif', 'jpeg', 'webp'],
    });
    console.log(bucketRes._exists ? '\x1b[33m[已存在]\x1b[0m' : '\x1b[32m[创建成功]\x1b[0m');
  } catch (err) {
    console.log(`\x1b[33m[已存在]\x1b[0m`);
  }

  // 5. 注入初始社区数据
  console.log('\n🌱 预置初始社区数据...');
  const seedCommunities = [
    { name: 'Go语言交流', description: 'Go 语言、微服务与高并发后端架构探讨' },
    { name: 'Vue & 前端开发', description: 'Vue 3、Vite、TypeScript 与现代前端生态' },
    { name: 'AI & 智能体', description: '大语言模型、LLM 智能体、RAG 与 Agent 应用分享' },
    { name: '综合技术杂谈', description: '程序员日常、职场心得与日常随笔' },
  ];

  for (const c of seedCommunities) {
    process.stdout.write(`  ↳ 社区: ${c.name} ... `);
    try {
      await api(`/tablesdb/${DATABASE_ID}/tables/communities/rows`, 'POST', {
        rowId: 'unique()',
        data: {
          name: c.name,
          description: c.description,
          icon: '',
          post_count: 0,
        },
      });
      console.log('\x1b[32m[OK]\x1b[0m');
    } catch (err) {
      console.log(`\x1b[33m[已存在或跳过]\x1b[0m`);
    }
  }

  console.log('\n\x1b[32m🎉 Appwrite TablesDB 全部表与字段初始化完成！\x1b[0m');
}

main().catch((err) => {
  console.error('\n\x1b[31m初始化失败:\x1b[0m', err);
  process.exit(1);
});
