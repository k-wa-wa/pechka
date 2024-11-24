import asyncio
from pydantic import BaseModel
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from langchain_core.messages import HumanMessage
from langgraph.prebuilt import create_react_agent
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_community.tools import DuckDuckGoSearchRun
from langgraph.checkpoint.memory import MemorySaver


memory = MemorySaver()
model = ChatGoogleGenerativeAI(model="gemini-1.5-flash", temperature=0)
tools = [DuckDuckGoSearchRun()]
agent_executor = create_react_agent(model, tools, checkpointer=memory)


async def chat(message: str):
    async for event in agent_executor.astream_events(
        {"messages": [HumanMessage(content=message)]},
        {"configurable": {"thread_id": "abc123"}},
        version="v1"
    ):
        kind = event["event"]
        if kind == "on_chain_start":
            if (
                event["name"] == "Agent"
            ):  # Was assigned when creating the agent with `.with_config({"run_name": "Agent"})`
                print(
                    f"Starting agent: {event['name']} with input: {event['data'].get('input')}"
                )
        elif kind == "on_chain_end":
            if (
                event["name"] == "Agent"
            ):  # Was assigned when creating the agent with `.with_config({"run_name": "Agent"})`
                print()
                print("--")
                print(
                    f"Done agent: {event['name']} with output: {event['data'].get('output')['output']}"
                )
        if kind == "on_chat_model_stream":
            content = event["data"]["chunk"].content
            if content:
                # Empty content in the context of OpenAI means
                # that the model is asking for a tool to be invoked.
                # So we only print non-empty content
                print(content, end="|")
                yield content
        elif kind == "on_tool_start":
            print("--")
            print(
                f"Starting tool: {event['name']} with inputs: {event['data'].get('input')}"
            )
        elif kind == "on_tool_end":
            print(f"Done tool: {event['name']}")
            print(f"Tool output was: {event['data'].get('output')}")
            print("--")


app = FastAPI()
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "http://localhost:5173",
    ],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class Message(BaseModel):
    role: str
    content: str


class PostChatReq(BaseModel):
    model: str
    messages: list[Message]


class PostChatRes(BaseModel):
    model: str
    created_at: str
    message: Message
    done: bool


@app.post("/api/chat")
async def post_chat(req: Request):
    reqBody = PostChatReq.model_validate_json(await req.body())

    async def generate_chat():
        message = "\n".join([m.content for m in reqBody.messages])
        async for m in chat(message):
            res = PostChatRes(model="", created_at="", message=Message(role="assistant", content=m), done=False)
            yield f"{res.model_dump_json()}\n\n"
        res = PostChatRes(model="", created_at="", message=Message(role="model", content=""), done=True)
        yield f"{res.model_dump_json()}\n\n"

    return StreamingResponse(content=generate_chat(), media_type="application/x-ndjson")


async def mock_chat(message):
    for s in message:
        await asyncio.sleep(0.01)
        yield s


@app.post("/mock/api/chat")
async def mock_post_chat(req: Request):
    reqBody = PostChatReq.model_validate_json(await req.body())

    async def generate_chat():
        message = "\n".join([m.content for m in reqBody.messages])
        async for m in mock_chat(message):
            res = PostChatRes(model="", created_at="", message=Message(role="assistant", content=m), done=False)
            yield f"{res.model_dump_json()}\n\n"
        res = PostChatRes(model="", created_at="", message=Message(role="model", content=""), done=True)
        yield f"{res.model_dump_json()}\n\n"

    return StreamingResponse(content=generate_chat(), media_type="application/x-ndjson")
