

curl http://localhost:8002/pechka/mock/api/chat \
-H "Content-Type: application/json" \
-d '{
  "model": "llama3",
  "messages": [
    {
      "role": "user",
      "content": "why is the sky blue?"
    }
  ]
}'

curl http://localhost:8002/api/chat \
-H "Content-Type: application/json" \
-d '{
  "model": "llama3"
}'



docker run -p 3000:8080 -e OLLAMA_BASE_URL=https://cec5-160-237-86-106.ngrok-free.app --restart always ghcr.io/open-webui/open-webui:main

curl http://localhost:11434/api/chat \
-H "Content-Type: application/json" \
-d '{
  "model": "llama2:2b",
  "messages": [
    {
      "role": "user",
      "content": "why is the sky blue?"
    }
  ]
}'

docker run -p 8000:8000 --env-file .env -it b



curl http://localhost:5173/api/chat \
-H "Content-Type: application/json" \
-d '{
  "model": "llama2:2b",
  "messages": [
    {
      "role": "user",
      "content": "why is the sky blue?"
    }
  ]
}'