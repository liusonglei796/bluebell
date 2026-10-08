import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';

// 自定义监控指标
const postDetailDuration = new Trend('post_detail_duration', true);
const errorRate = new Rate('post_detail_error_rate');
const reqCounter = new Counter('post_detail_requests_total');

// 基础配置
const BASE_URL = __ENV.BASE_URL || 'http://127.0.0.1:8080/api/v1';

// 阶段并发配置（梯形爬坡，模拟逐渐升高的并发压力）
export const options = {
    scenarios: {
        preload_load: {
            executor: 'ramping-vus',
            startVUs: 10,
            stages: [
                { duration: '15s', target: 50 },   // 快速预热到 50 并发
                { duration: '30s', target: 100 },  // 攀升至 100 并发
                { duration: '45s', target: 200 },  // 冲击 200 并发（观察 MySQL 连接与响应拐点）
                { duration: '15s', target: 0 },    // 降压
            ],
            gracefulRampDown: '5s',
        },
    },
    thresholds: {
        http_req_failed: ['rate<0.05'], // 允许错误率低于 5%
        http_req_duration: ['p(95)<500'], // P95 目标低于 500ms
    },
};

// init 阶段读取预置数据（k6 规范要求 open 必须在全局作用域调用）
let postIDsFromTokens = [];
try {
    const rawContent = open('./tokens.json');
    const postMatch = rawContent.match(/"post_ids":\s*\[([\s\S]*?)\]/);
    if (postMatch && postMatch[1]) {
        const matchedIDs = postMatch[1].match(/\d+/g);
        if (matchedIDs && matchedIDs.length > 0) {
            postIDsFromTokens = matchedIDs;
        }
    }
} catch (e) {
    // ignore
}

// setup 阶段：读取或探测可用的帖子 ID
export function setup() {
    let postIDs = [];
    // 优先从 /posts 获取真实返回的 string ID，避免 JS 解析 64 位大整数丢失精度
    const res = http.get(`${BASE_URL}/posts?page=1&size=50&order=time`);
    if (res.status === 200) {
        try {
            const body = JSON.parse(res.body);
            if (body.data && Array.isArray(body.data) && body.data.length > 0) {
                postIDs = body.data.map(p => String(p.id));
                console.log(`[k6 Setup] 成功从 /posts 接口获取 ${postIDs.length} 个帖子 ID`);
            }
        } catch (err) {
            console.error('[k6 Setup] 解析 /posts 响应失败', err);
        }
    }

    if (postIDs.length === 0 && postIDsFromTokens.length > 0) {
        postIDs = postIDsFromTokens;
        console.log(`[k6 Setup] 成功从 tokens.json 提取 ${postIDs.length} 个大整数帖子 ID（保留精度）`);
    }

    if (postIDs.length === 0) {
        throw new Error('[k6 Setup] 无法获取任何有效的帖子 ID，请先运行 seed.go 预置数据！');
    }

    return { postIDs };
}

export default function (data) {
    const postIDs = data.postIDs;
    // 随机选择一个帖子 ID 请求详情
    const pid = postIDs[Math.floor(Math.random() * postIDs.length)];
    const url = `${BASE_URL}/post/${pid}`;

    const res = http.get(url, {
        headers: {
            'Content-Type': 'application/json',
        },
        tags: { endpoint: 'post_detail_preload' },
    });

    reqCounter.add(1);
    postDetailDuration.add(res.timings.duration);

    const isSuccess = check(res, {
        'status is 200': (r) => r.status === 200,
        'has author_name': (r) => {
            try {
                const b = JSON.parse(r.body);
                return b.code === 1000 && b.data && (b.data.author_name !== '' || (b.data.author_names && b.data.author_names.length > 0));
            } catch (_) {
                return false;
            }
        },
        'has community': (r) => {
            try {
                const b = JSON.parse(r.body);
                return b.code === 1000 && b.data && (b.data.community_name !== '' || b.data.community !== null);
            } catch (_) {
                return false;
            }
        },
    });

    errorRate.add(!isSuccess);

    // 适当控制请求节奏，模拟微观思索时间 (10ms ~ 30ms)
    sleep(0.02);
}

export function handleSummary(data) {
    return {
        'benchmark/preload_summary.json': JSON.stringify(data, null, 2),
    };
}
