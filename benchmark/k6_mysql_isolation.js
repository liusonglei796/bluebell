import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
// @ts-ignore
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

// ==========================================
// 1. 自定义业务监控指标
// ==========================================
const postsListTrend = new Trend('trend_posts_list_duration', true);
const postDetailTrend = new Trend('trend_post_detail_duration', true);
const createPostTrend = new Trend('trend_create_post_duration', true);
const voteTrend = new Trend('trend_vote_duration', true);

const successfulFlowRate = new Rate('rate_successful_flow');
const deadlockOrTimeoutCount = new Counter('counter_deadlock_or_lock_timeout');
const serverErrorCount = new Counter('counter_server_errors');

// ==========================================
// 2. 阶梯加压配置 (50 -> 150 -> 300 -> 500 VUs)
// ==========================================
export const options = {
  stages: [
    { duration: '10s', target: 50 },  // 预热并爬坡至 50 VUs
    { duration: '15s', target: 150 }, // 爬坡至 150 VUs
    { duration: '15s', target: 300 }, // 攀升至 300 VUs (锁竞争激化阶段)
    { duration: '15s', target: 500 }, // 冲刺至 500 VUs (极限压力阶段)
    { duration: '5s',  target: 0 },   // 降压收尾
  ],
  thresholds: {
    // 观察指标，不强制熔断退出以保证收集完整梯度数据
    http_req_duration: ['p(95)<3000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080/api/v1';

// ==========================================
// 3. Setup 阶段：批量初始化测试账号与初始帖子池
// ==========================================
export function setup() {
  console.log(`[k6 Setup] 正在初始化测试用户凭证与数据池，目标: ${BASE_URL}`);

  const userCount = 50;
  const tokens = [];

  for (let i = 1; i <= userCount; i++) {
    const username = `k6_iso_${Date.now()}_${i}_${Math.floor(Math.random() * 1000)}`;
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

    // 2. 登录
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
        const tok = body.data && (body.data.access_token || body.data.token);
        if (tok) {
          tokens.push(tok);
        }
      } catch (e) {}
    }
  }

  console.log(`[k6 Setup] 成功生成 ${tokens.length} 个独立用户 Token`);

  // 3. 创建若干初始帖子，保证详情点查和投票有靶标
  const initialPostIDs = [];
  const sampleToken = tokens.length > 0 ? tokens[0] : '';
  const initHeaders = {
    'Content-Type': 'application/json',
    ...(sampleToken ? { Authorization: `Bearer ${sampleToken}` } : {}),
  };

  for (let i = 1; i <= 20; i++) {
    const createRes = http.post(
      `${BASE_URL}/post`,
      JSON.stringify({
        community_id: 1,
        title: `Seed Post ${Date.now()}_${i}`,
        content: `Initial content for isolation benchmark testing ${i}`,
      }),
      { headers: initHeaders }
    );
    if (createRes.status === 200) {
      try {
        const b = JSON.parse(createRes.body);
        if (b.data && (b.data.id || b.data.post_id)) {
          initialPostIDs.push(b.data.id || b.data.post_id);
        }
      } catch (e) {}
    }
  }

  console.log(`[k6 Setup] 成功初始化 ${initialPostIDs.length} 条测试帖子`);

  return {
    tokens: tokens,
    postIDs: initialPostIDs,
  };
}

// ==========================================
// 4. VU 压测执行逻辑（70% 读 + 30% 写/更新 混合模型）
// ==========================================
export default function (data) {
  const tokens = data.tokens || [];
  const token = tokens.length > 0 ? tokens[__VU % tokens.length] : '';

  const authHeaders = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };

  const rand = Math.random();
  let opSuccess = true;

  if (rand < 0.40) {
    // ----------------------------------------------------
    // 场景 A (40% 读): 帖子列表分页 (MySQL 范围扫描 + 排序)
    // ----------------------------------------------------
    const page = Math.floor(Math.random() * 5) + 1;
    const start = Date.now();
    const res = http.get(`${BASE_URL}/posts?order=time&page=${page}&size=10`, {
      headers: authHeaders,
    });
    postsListTrend.add(Date.now() - start);

    const ok = check(res, {
      'list posts status is 200': (r) => r.status === 200,
    });
    if (!ok) {
      opSuccess = false;
      serverErrorCount.add(1);
    }
  } else if (rand < 0.70) {
    // ----------------------------------------------------
    // 场景 B (30% 读): 帖子详情点查 (MySQL 主键/唯一索引查找)
    // ----------------------------------------------------
    const postIDs = data.postIDs || [];
    if (postIDs.length > 0) {
      const targetID = postIDs[Math.floor(Math.random() * postIDs.length)];
      const start = Date.now();
      const res = http.get(`${BASE_URL}/post/${targetID}`, {
        headers: authHeaders,
      });
      postDetailTrend.add(Date.now() - start);

      const ok = check(res, {
        'post detail status is 200': (r) => r.status === 200,
      });
      if (!ok) {
        opSuccess = false;
        serverErrorCount.add(1);
      }
    }
  } else if (rand < 0.85) {
    // ----------------------------------------------------
    // 场景 C (15% 写): 并发发帖 (INSERT 间隙锁竞争触发源)
    // ----------------------------------------------------
    const start = Date.now();
    const communityID = (Math.floor(Math.random() * 3) + 1); // 集中在 1~3 社区增加冲突
    const res = http.post(
      `${BASE_URL}/post`,
      JSON.stringify({
        community_id: communityID,
        title: `Stress Post VU${__VU}_${Date.now()}_${Math.floor(Math.random() * 10000)}`,
        content: `Concurrent stress insert payload content ${Date.now()}`,
      }),
      { headers: authHeaders }
    );
    createPostTrend.add(Date.now() - start);

    if (res.status === 200) {
      try {
        const b = JSON.parse(res.body);
        if (b.data && (b.data.id || b.data.post_id) && data.postIDs.length < 500) {
          data.postIDs.push(b.data.id || b.data.post_id);
        }
      } catch (e) {}
    } else {
      opSuccess = false;
      if (res.body && (res.body.includes('Deadlock') || res.body.includes('Lock wait timeout'))) {
        deadlockOrTimeoutCount.add(1);
      } else {
        serverErrorCount.add(1);
      }
    }
  } else {
    // ----------------------------------------------------
    // 场景 D (15% 写): 并发投票 (SELECT ... FOR UPDATE + UPDATE 事务行锁争抢)
    // ----------------------------------------------------
    const postIDs = data.postIDs || [];
    if (postIDs.length > 0) {
      // 选取前 10 个热点帖子进行高集中度并发投票
      const hotIndex = Math.floor(Math.random() * Math.min(10, postIDs.length));
      const targetID = postIDs[hotIndex];
      const direction = Math.random() < 0.5 ? 1 : -1;

      const start = Date.now();
      const res = http.post(
        `${BASE_URL}/vote`,
        JSON.stringify({
          post_id: parseInt(targetID, 10),
          direction: direction,
        }),
        { headers: authHeaders }
      );
      voteTrend.add(Date.now() - start);

      if (res.status !== 200) {
        opSuccess = false;
        if (res.body && (res.body.includes('Deadlock') || res.body.includes('Lock wait timeout') || res.body.includes('1213') || res.body.includes('1205'))) {
          deadlockOrTimeoutCount.add(1);
        } else {
          serverErrorCount.add(1);
        }
      }
    }
  }

  successfulFlowRate.add(opSuccess ? 1 : 0);

  // 节流沉思时间：30ms ~ 80ms
  sleep(0.03 + Math.random() * 0.05);
}

// ==========================================
// 5. 结果报表输出
// ==========================================
export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
