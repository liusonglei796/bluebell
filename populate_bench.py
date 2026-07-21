import requests
import json

def get_token():
    url = 'http://localhost:80/api/v1/login'
    signup_url = 'http://localhost:80/api/v1/signup'
    user_data = {
        'username': 'bench_user_final',
        'password': 'password123',
        're_password': 'password123',
        'email': 'bench_final@example.com'
    }
    
    try:
        requests.post(signup_url, json=user_data)
    except: pass
    
    r = requests.post(url, json={'username': 'bench_user_final', 'password': 'password123'})
    return r.json()['data']['access_token']

def populate():
    token = get_token()
    headers = {'Authorization': f'Bearer {token}', 'Content-Type': 'application/json'}
    
    # Get community
    r = requests.get('http://localhost:80/api/v1/community', headers=headers)
    communities = r.json()['data']
    c_id = communities[0]['id']
    
    # Create posts
    print("Starting data population...")
    for i in range(100):
        post_data = {
            'title': f'Benchmark Post {i}',
            'content': 'Data for go-wrk benchmark test. ' * 10,
            'community_id': int(c_id)
        }
        requests.post('http://localhost:80/api/v1/post', headers=headers, json=post_data)
        if i % 20 == 0:
            print(f'Created {i} posts...')
    print("Population complete.")

if __name__ == '__main__':
    populate()
