import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
// @ts-ignore
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
// @ts-ignore
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

// ==========================================
// 1. 自定义业务监控指标
// ==========================================
const postsListTrend = new Trend('trend_posts_list_duration', true);
const postDetailTrend = new Trend('trend_post_detail_duration', true);
const voteTrend = new Trend('trend_vote_duration', true);
const voteRateLimitedCount = new Counter('counter_vote_rate_limited');
const successfulFlowRate = new Rate('rate_successful_flow');

// ==========================================
// 2. 压测配置：阶梯加压模型 + 指标阈值
// ==========================================
export const options = {
  stages: [
    { duration: '30s', target: 50 },  // 预热并爬坡至 50 VUs
    { duration: '1m',  target: 50 },  // 50 VUs 稳态运行
    { duration: '30s', target: 200 }, // 爬坡至 200 VUs
    { duration: '2m',  target: 200 }, // 200 VUs 稳态运行 (峰值观测)
    { duration: '30s', target: 500 }, // 极限冲刺至 500 VUs
    { duration: '1m',  target: 500 }, // 500 VUs 极限观察 (限流/熔断检验)
    { duration: '30s', target: 0 },   // 降压收尾
  ],
  thresholds: {
    // 整体请求失败率低于 1% (排除预期的 429 限流)
    http_req_failed: ['rate<0.01'],
    // 核心延迟指标要求
    http_req_duration: ['p(95)<300', 'p(99)<800'],
    trend_posts_list_duration: ['p(95)<150'],
    trend_post_detail_duration: ['p(95)<100'],
    trend_vote_duration: ['p(95)<200'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080/api/v1';

// ==========================================
// 3. Setup 阶段：准备测试用户 Token 与数据
// ==========================================
export function setup() {
  console.log(`[k6 Setup] 正在初始化压测环境，目标服务: ${BASE_URL}`);

  let tokens = [];

  // 尝试自动批量注册/登录 50 个专用压测账号 (如果之前没有预先生成)
  const userCount = 50;
  for (let i = 1; i <= userCount; i++) {
    const username = `k6_user_${Date.now()}_${i}`;
    const password = 'password123';

    // 1. 注册
    http.post(
      `${BASE_URL}/signup`,
      JSON.stringify({
        username: username,
        password: password,
        re_password: password,
      }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    // 2. 登录获取 Token
    const loginRes = http.post(
      `${BASE_URL}/login`,
      JSON.stringify({
        username: username,
        password: password,
      }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    if (loginRes.status === 200) {
      try {
        const body = JSON.parse(loginRes.body);
        if (body.data && body.data.token) {
          tokens.push(body.data.token);
        }
      } catch (e) {}
    }
  }

  console.log(`[k6 Setup] 成功获取 ${tokens.length} 个独立测试用户凭证！`);

  // 预拉取一次帖子列表以备测试详情和投票使用
  let initialPostIDs = [];
  const postsRes = http.get(`${BASE_URL}/posts?order=score&page=1&size=20`);
  if (postsRes.status === 200) {
    try {
      const body = JSON.parse(postsRes.body);
      if (body.data && Array.isArray(body.data)) {
        initialPostIDs = body.data.map((p) => p.id);
      }
    } catch (e) {}
  }

  return {
    tokens: tokens,
    postIDs: initialPostIDs,
  };
}

// ==========================================
// 4. VU 压测执行逻辑（全链路混合业务模拟）
// ==========================================
export default function (data) {
  const tokens = data.tokens || [];
  // 每个 VU 轮询绑定独立用户 Token，实现真实租户隔离
  const token = tokens.length > 0 ? tokens[__VU % tokens.length] : '';

  const authHeaders = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };

  let flowSuccess = true;

  // ----------------------------------------------------
  // Step 1: 100% 概率浏览热榜 (GET /posts?order=score)
  // ----------------------------------------------------
  const listStart = Date.now();
  const listRes = http.get(`${BASE_URL}/posts?order=score&page=1&size=10`, {
    headers: authHeaders,
  });
  postsListTrend.add(Date.now() - listStart);

  const listOk = check(listRes, {
    'posts list status is 200': (r) => r.status === 200,
  });
  if (!listOk) flowSuccess = false;

  let postIDs = data.postIDs || [];
  try {
    const listBody = JSON.parse(listRes.body);
    if (listBody.data && Array.isArray(listBody.data) && listBody.data.length > 0) {
      postIDs = listBody.data.map((p) => p.id);
    }
  } catch (e) {}

  // 模拟用户读完列表后的思考停顿 (100ms ~ 300ms)
  sleep(0.1 + Math.random() * 0.2);

  // ----------------------------------------------------
  // Step 2: 80% 概率点击查看帖子详情 (GET /post/:id)
  // ----------------------------------------------------
  let targetPostID = null;
  if (postIDs.length > 0 && Math.random() < 0.8) {
    targetPostID = postIDs[Math.floor(Math.random() * postIDs.length)];

    const detailStart = Date.now();
    const detailRes = http.get(`${BASE_URL}/post/${targetPostID}`, {
      headers: authHeaders,
    });
    postDetailTrend.add(Date.now() - detailStart);

    const detailOk = check(detailRes, {
      'post detail status is 200': (r) => r.status === 200,
    });
    if (!detailOk) flowSuccess = false;

    // 模拟阅读帖子正文耗时 (200ms ~ 500ms)
    sleep(0.2 + Math.random() * 0.3);
  }

  // ----------------------------------------------------
  // Step 3: 20% 概率对帖子执行投票 (POST /vote)
  // ----------------------------------------------------
  if (targetPostID && token && Math.random() < 0.25) {
    const voteStart = Date.now();
    // 随机投赞成票(1)或取消投票(0)
    const direction = Math.random() < 0.8 ? 1 : 0;

    const voteRes = http.post(
      `${BASE_URL}/vote`,
      JSON.stringify({
        post_id: parseInt(targetPostID, 10),
        direction: direction,
      }),
      { headers: authHeaders }
    );
    voteTrend.add(Date.now() - voteStart);

    if (voteRes.status === 429) {
      // 记录单用户令牌桶限流触发次数 (属于正常防御拦截)
      voteRateLimitedCount.add(1);
    } else {
      const voteOk = check(voteRes, {
        'vote status is 200': (r) => r.status === 200,
      });
      if (!voteOk) flowSuccess = false;
    }
  }

  successfulFlowRate.add(flowSuccess ? 1 : 0);

  // 每次迭代间歇冷却 (100ms ~ 400ms)
  sleep(0.1 + Math.random() * 0.3);
}

// ==========================================
// 5. 结果报表输出：HTML 报表 + 控制台摘要
// ==========================================
export function handleSummary(data) {
  return {
    'summary.html': htmlReport(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
