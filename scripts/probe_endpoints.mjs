const ENDPOINT = 'https://sgp.cloud.appwrite.io/v1';
const PROJECT_ID = '6ab10b94003e7b324fdd';
const API_KEY = 'standard_87eaa034aaeed766db78571ee500d76d53acd1988b0ecaf877cfa1007a56a9573c7deb80dad474e3d5f22b201845f21d8975def708e3a7a65ac78159180a033e2e05454cc12e403b4e68dc1ce90e8f4c00f43f0f30b2fc3ef440855464c320148f80a7ee312b17d20c0a6b9df1c771209f3f92e6a21938a871525ab7a86b038d';

const headers = {
  'Content-Type': 'application/json',
  'X-Appwrite-Project': PROJECT_ID,
  'X-Appwrite-Key': API_KEY,
};

async function check(path) {
  try {
    const res = await fetch(`${ENDPOINT}${path}`, { headers });
    const data = await res.json().catch(() => ({}));
    console.log(`[${res.status}] ${path} ->`, JSON.stringify(data).slice(0, 150));
  } catch (err) {
    console.log(`[ERR] ${path} ->`, err.message);
  }
}

async function main() {
  console.log('--- 探测 bluebell_db 资源结构 ---');
  await check('/databases/bluebell_db');
  await check('/databases/bluebell_db/collections');
  await check('/databases/bluebell_db/tables');
  await check('/tablesdb/bluebell_db/tables');
  await check('/databases/tablesdb/bluebell_db/tables');
  await check('/databases/bluebell_db/collections?queries[]=limit(1)');
}

main();
