from typing import Dict, List
from uuid import UUID
from langchain_core.messages import HumanMessage
from langgraph.prebuilt import create_react_agent
import os
from langchain_core.outputs import LLMResult
from langchain_google_genai import ChatGoogleGenerativeAI
from langchain_community.tools import DuckDuckGoSearchRun
from langchain_core.callbacks import BaseCallbackHandler
from typing import Any
from langgraph.checkpoint.memory import MemorySaver

search = DuckDuckGoSearchRun()


class CallbackHandler(BaseCallbackHandler):
    def on_llm_start(self, serialized: Dict[str, Any], prompts: List[str], *, run_id: UUID, parent_run_id: UUID | None = None, tags: List[str] | None = None, metadata: Dict[str, Any] | None = None, **kwargs: Any) -> Any:
        for p in prompts:
            print("----------prompt start----------")
            print(p)
            print("----------prompt end  ----------")
        return super().on_llm_start(serialized, prompts, run_id=run_id, parent_run_id=parent_run_id, tags=tags, metadata=metadata, **kwargs)

    def on_llm_end(self, response: LLMResult, *, run_id: UUID, parent_run_id: UUID | None = None, **kwargs: Any) -> Any:
        for ll in response.generations:
            for l in ll:
                print("----------llm result start----------")
                print(l.text)
                print("----------llm result end  ----------")
        return super().on_llm_end(response, run_id=run_id, parent_run_id=parent_run_id, **kwargs)
