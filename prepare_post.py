import requests
import json

token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXBlIjoiYWNjZXNzIiwic3ViIjoiMTEwMDEyMTA1MDkyMDE0MDgiLCJleHAiOjE3NzgxNjkyOTAsImlhdCI6MTc3ODE2MjA5MH0.TMpu15KAALUfojdYdbS2TDfM1DPdHg9w3wKFL8OQ5j4"
headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

# 1. Get community list to find a community ID
r = requests.get("http://localhost:8081/api/v1/community", headers=headers)
print(f"Community: {r.text}")
communities = r.json()['data']
community_id = int(communities[0]['id'])

# 2. Create a post
post_data = {
    "title": "Benchmark Post",
    "content": "This is a post for go-wrk benchmark",
    "community_id": community_id
}
r = requests.post("http://localhost:8081/api/v1/post", headers=headers, json=post_data)
print(f"Create Post: {r.text}")

# 3. Get post list to find our post ID
r = requests.get("http://localhost:8081/api/v1/posts?page=1&size=10&order=time", headers=headers)
posts = r.json()['data']
post_id = int(posts[0]['id'])
print(f"POST_ID:{post_id}")
