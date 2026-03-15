from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List
from scorer import VideoScorer
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Thumbnail Analyzer API")
scorer = VideoScorer()

class AnalyzeRequest(BaseModel):
    file_path: str
    points: List[float]

class AnalyzeResponse(BaseModel):
    best_timestamp: float

@app.post("/analyze", response_model=AnalyzeResponse)
async def analyze_video(request: AnalyzeRequest):
    logger.info(f"Received analysis request for: {request.file_path}")
    
    best_timestamp = 0.0
    best_score = -1.0
    
    if not request.points:
        raise HTTPException(status_code=400, detail="Points list cannot be empty")
    
    for ts in request.points:
        score = scorer.analyze_frame(request.file_path, ts)
        if score > best_score:
            best_score = score
            best_timestamp = ts
            
    return AnalyzeResponse(best_timestamp=best_timestamp)

@app.get("/health")
async def health_check():
    return {"status": "ok"}
