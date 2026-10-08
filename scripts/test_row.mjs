const ENDPOINT = 'https://sgp.cloud.appwrite.io/v1';
const PROJECT_ID = '6ab10b94003e7b324fdd';
const API_KEY = 'standard_87eaa034aaeed766db78571ee500d76d53acd1988b0ecaf877cfa1007a56a9573c7deb80dad474e3d5f22b201845f21d8975def708e3a7a65ac78159180a033e2e05454cc12e403b4e68dc1ce90e8f4c00f43f0f30b2fc3ef440855464c320148f80a7ee312b17d20c0a6b9df1c771209f3f92e6a21938a871525ab7a86b038d';

const headers = {
  'Content-Type': 'application/json',
  'X-Appwrite-Project': PROJECT_ID,
  'X-Appwrite-Key': API_KEY,
};

async function testRow() {
  const res = await fetch(`${ENDPOINT}/tablesdb/bluebell_db/tables/communities/rows`, {
    method: 'POST',
    headers,
    body: JSON.stringify({
      rowId: 'unique()',
      data: {
        name: 'Go语言交流',
      },
    }),
  });
  const data = await res.json();
  console.log('Create Row Status:', res.status, data);
}

testRow();
