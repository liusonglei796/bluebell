import requests
import json

url = "http://localhost:8081/api/v1/login"
data = {
    "username": "testuser",
    "password": "testpassword"
}

# Try signup first
try:
    signup_url = "http://localhost:8081/api/v1/signup"
    signup_data = {
        "username": "testuser2",
        "password": "testpassword",
        "re_password": "testpassword",
        "email": "test2@example.com"
    }
    r = requests.post(signup_url, json=signup_data)
    print(f"Signup: {r.status_code} {r.text}")
except Exception as e:
    print(f"Signup error: {e}")

# Login
data = {
    "username": "testuser2",
    "password": "testpassword"
}
r = requests.post(url, json=data)
print(f"Login: {r.status_code} {r.text}")
if r.status_code == 200:
    res = r.json()
    if res['code'] == 1000:
        print(f"TOKEN:{res['data']['access_token']}")
