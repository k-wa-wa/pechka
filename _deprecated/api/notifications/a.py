import json
import time
import hashlib
import hmac
import base64
import uuid
import requests

# Declare empty header dictionary
apiHeader = {}

token = "73f8a383f1a040360ce13b4d2b2cb821db5027b9fd2cfbc4d061aa7b90382d8c8e238aafa713a1cf177a18237dd4ed39"
secret = "a444c775a8fa6351fb5af8047fd75d5a"
nonce = uuid.uuid4()
t = int(round(time.time() * 1000))
string_to_sign = '{}{}{}'.format(token, t, nonce)

string_to_sign = bytes(string_to_sign, 'utf-8')
secret = bytes(secret, 'utf-8')

sign = base64.b64encode(hmac.new(secret, msg=string_to_sign, digestmod=hashlib.sha256).digest())
print('Authorization: {}'.format(token))
print('t: {}'.format(t))
print('sign: {}'.format(str(sign, 'utf-8')))
print('nonce: {}'.format(nonce))

# Build api header JSON
apiHeader['Authorization'] = token
apiHeader['Content-Type'] = 'application/json'
apiHeader['charset'] = 'utf8'
apiHeader['t'] = str(t)
apiHeader['sign'] = str(sign, 'utf-8')
apiHeader['nonce'] = str(nonce)


res = requests.get(
    "https://api.switch-bot.com/v1.1/devices",
    headers=apiHeader
)
print(res.text)
