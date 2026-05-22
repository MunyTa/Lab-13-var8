"""
HR Multi-Agent System Orchestrator
Automates resume parsing, vacancy matching, interview scheduling, and feedback collection
Includes auction-based task distribution
"""

import asyncio
import json
import uuid
import logging
import httpx
from datetime import datetime
from typing import Optional, Dict, Any, List
from contextlib import asynccontextmanager

import nats
from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel
import redis.asyncio as redis

from orchestrator.llm_agent import LLMFeedbackAgent

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

results_store: Dict[str, asyncio.Future] = {}
processed_count = 0
failed_count = 0

SUBJECTS = {
    "resume_parser": "hr.resume.parse",
    "matcher": "hr.vacancy.match",
    "scheduler": "hr.interview.schedule",
    "feedback": "hr.feedback.collect",
    "completed": "hr.tasks.completed",
    "auction_request": "hr.auction.bid_request",
    "auction_result": "hr.auction.bid_result",
}


class HREventInput(BaseModel):
    candidate_name: str
    raw_resume: str
    vacancy_title: Optional[str] = "Go Backend Developer"
    preferred_interview_date: Optional[str] = None


class TaskRequest(BaseModel):
    task_type: str
    payload: Dict[str, Any]
    timeout: int = 30


class TaskResponse(BaseModel):
    task_id: str
    status: str
    result: Optional[Dict[str, Any]] = None
    error: Optional[str] = None


class BidResponse(BaseModel):
    agent_id: str
    task_id: str
    cost: float
    skill: float
    availability: float


nc: Optional[nats.NATS] = None
redis_client: Optional[redis.Redis] = None
llm_agent: Optional[LLMFeedbackAgent] = None
auction_bids: Dict[str, List[BidResponse]] = {}


async def connect_nats():
    global nc
    try:
        nc = await nats.connect("nats://localhost:4222")
        logger.info("Connected to NATS")
    except Exception as e:
        logger.warning(f"NATS not available: {e}")
        nc = None


async def connect_redis():
    global redis_client
    try:
        redis_client = redis.from_url("redis://localhost:6379")
        await redis_client.ping()
        logger.info("Connected to Redis")
    except Exception as e:
        logger.warning(f"Redis not available: {e}")
        redis_client = None


async def on_result(msg):
    global processed_count, failed_count
    
    try:
        data = json.loads(msg.data.decode())
        task_id = data.get("task_id")
        
        if task_id in results_store:
            results_store[task_id].set_result(data)
            processed_count += 1
            logger.info(f"Task {task_id} completed")
            
            if redis_client:
                await redis_client.incr("hr:processed_tasks")
        else:
            logger.warning(f"Unknown task result: {task_id}")
    except Exception as e:
        logger.error(f"Error processing result: {e}")
        failed_count += 1


async def on_auction_result(msg):
    """Handle auction bid responses from agents"""
    global auction_bids
    
    try:
        data = json.loads(msg.data.decode())
        task_id = data.get("task_id")
        
        if task_id not in auction_bids:
            auction_bids[task_id] = []
        
        bid = BidResponse(
            agent_id=data.get("agent_id"),
            task_id=task_id,
            cost=data.get("cost", 999.0),
            skill=data.get("skill", 0.0),
            availability=data.get("availability", 0.0)
        )
        auction_bids[task_id].append(bid)
        logger.info(f"Received bid from {bid.agent_id} for task {task_id}: cost={bid.cost}")
        
    except Exception as e:
        logger.error(f"Error processing auction bid: {e}")


async def start_listener():
    global nc
    if nc is None:
        return
        
    try:
        await nc.subscribe(SUBJECTS["completed"], cb=on_result)
        logger.info(f"Subscribed to {SUBJECTS['completed']}")
        
        await nc.subscribe(SUBJECTS["auction_result"], cb=on_auction_result)
        logger.info(f"Subscribed to {SUBJECTS['auction_result']}")
    except Exception as e:
        logger.error(f"Failed to subscribe: {e}")


async def run_auction(task_id: str, task_type: str, timeout: float = 2.0) -> Optional[str]:
    """
    Run auction for task assignment.
    Publishes bid request, collects bids, selects best agent.
    """
    global auction_bids, nc
    
    if nc is None:
        return None
    
    auction_bids[task_id] = []
    
    task = {
        "id": task_id,
        "type": task_type,
        "payload": json.dumps({"auction": True})
    }
    
    await nc.publish(SUBJECTS["auction_request"], json.dumps(task).encode())
    logger.info(f"Published auction request for task {task_id}")
    
    await asyncio.sleep(timeout)
    
    bids = auction_bids.get(task_id, [])
    if not bids:
        logger.warning(f"No bids received for task {task_id}")
        return None
    
    # Select best agent (lowest cost, highest skill, highest availability)
    best_bid = min(bids, key=lambda b: (b.cost, -b.skill, -b.availability))
    logger.info(f"Selected agent {best_bid.agent_id} for task {task_id}")
    
    del auction_bids[task_id]
    return best_bid.agent_id


async def check_queue_length() -> int:
    """Check NATS queue length for dynamic scaling"""
    try:
        async with httpx.AsyncClient() as client:
            response = await client.get("http://nats:8222/varz", timeout=5.0)
            if response.status_code == 200:
                data = response.json()
                return data.get("subscriptions", 0)
    except:
        pass
    return 0


async def auto_scale_agents():
    """Automatically scale agents based on queue length"""
    global nc
    
    QUEUE_THRESHOLD = 10
    MAX_REPLICAS = 5
    
    while True:
        try:
            queue_len = await check_queue_length()
            
            if queue_len > QUEUE_THRESHOLD:
                logger.info(f"Queue length {queue_len} exceeds threshold {QUEUE_THRESHOLD}")
                
                if redis_client:
                    current = await redis_client.get("hr:agent_replicas")
                    replicas = int(current or 1)
                    
                    if replicas < MAX_REPLICAS:
                        await redis_client.incr("hr:agent_replicas")
                        logger.info(f"Scaled up to {replicas + 1} replicas")
            
            await asyncio.sleep(5)
        except Exception as e:
            logger.error(f"Auto-scaling error: {e}")
            await asyncio.sleep(5)


@asynccontextmanager
async def lifespan(app: FastAPI):
    global llm_agent
    
    await connect_nats()
    await connect_redis()
    await start_listener()
    
    llm_agent = LLMFeedbackAgent()
    
    # Start auto-scaler
    if nc:
        asyncio.create_task(auto_scale_agents())
    
    yield
    
    if nc:
        await nc.close()
    if redis_client:
        await redis_client.close()


app = FastAPI(
    title="HR Multi-Agent System",
    description="Automated HR pipeline with auction-based task distribution",
    version="1.0.0",
    lifespan=lifespan
)

templates = Jinja2Templates(directory="web/templates")


@app.get("/", response_class=HTMLResponse)
async def dashboard():
    return templates.TemplateResponse("index.html", {
        "request": {},
        "title": "HR Multi-Agent Dashboard"
    })


@app.get("/health")
async def health_check():
    return {
        "status": "healthy",
        "nats_connected": nc is not None,
        "redis_connected": redis_client is not None,
        "processed_tasks": processed_count,
        "failed_tasks": failed_count
    }


@app.post("/tasks/hr", response_model=TaskResponse)
async def create_hr_task(task: TaskRequest):
    global results_store, failed_count
    
    if nc is None:
        raise HTTPException(status_code=503, detail="NATS not connected")
    
    task_id = str(uuid.uuid4())
    
    # Run auction to select best agent
    selected_agent = await run_auction(task_id, task.task_type)
    
    future = asyncio.Future()
    results_store[task_id] = future
    
    payload = {
        "id": task_id,
        "type": task.task_type,
        "payload": json.dumps(task.payload),
        "selected_agent": selected_agent
    }
    
    subject = SUBJECTS.get(task.task_type, SUBJECTS["resume_parser"])
    
    try:
        await nc.publish(subject, json.dumps(payload).encode())
        logger.info(f"Published task {task_id} to {subject} (selected: {selected_agent})")
        
        result = await asyncio.wait_for(future, timeout=task.timeout)
        
        return TaskResponse(
            task_id=task_id,
            status="completed",
            result=result
        )
        
    except asyncio.TimeoutError:
        failed_count += 1
        if task_id in results_store:
            del results_store[task_id]
        return TaskResponse(
            task_id=task_id,
            status="timeout",
            error=f"Task exceeded {task.timeout} seconds"
        )
    except Exception as e:
        failed_count += 1
        if task_id in results_store:
            del results_store[task_id]
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/pipeline/hr")
async def run_hr_pipeline(input: HREventInput, background_tasks: BackgroundTasks):
    if nc is None:
        raise HTTPException(status_code=503, detail="NATS not connected")
    
    task_id = str(uuid.uuid4())
    pipeline_id = f"pipeline-{task_id[:8]}"
    
    future = asyncio.Future()
    results_store[task_id] = future
    
    payload = {
        "pipeline_id": pipeline_id,
        "candidate_name": input.candidate_name,
        "vacancy_title": input.vacancy_title,
        "raw_text": input.raw_resume,
        "preferred_date": input.preferred_interview_date
    }
    
    try:
        await nc.publish(SUBJECTS["resume_parser"], json.dumps({
            "id": task_id,
            "type": "resume_parse",
            "payload": json.dumps(payload)
        }).encode())
        
        logger.info(f"Started HR pipeline {pipeline_id}")
        
        result = await asyncio.wait_for(future, timeout=60)
        
        return {
            "pipeline_id": pipeline_id,
            "status": "completed",
            "task_id": task_id,
            "result": result
        }
        
    except asyncio.TimeoutError:
        return {
            "pipeline_id": pipeline_id,
            "status": "timeout",
            "task_id": task_id,
            "error": "Pipeline exceeded 60 seconds"
        }


@app.get("/stats")
async def get_stats():
    stats = {
        "processed_tasks": processed_count,
        "failed_tasks": failed_count,
        "success_rate": (processed_count / (processed_count + failed_count) * 100) if (processed_count + failed_count) > 0 else 0
    }
    
    if redis_client:
        try:
            total = await redis_client.get("hr:processed_tasks")
            candidates = await redis_client.get("hr:total_candidates_processed")
            replicas = await redis_client.get("hr:agent_replicas")
            stats["redis_total_processed"] = int(total or 0)
            stats["redis_candidates"] = int(candidates or 0)
            stats["current_replicas"] = int(replicas or 1)
        except:
            pass
    
    return stats


@app.post("/llm/explain")
async def explain_with_llm(task_id: str):
    if llm_agent is None:
        raise HTTPException(status_code=503, detail="LLM agent not initialized")
    
    try:
        explanation = await llm_agent.explain_result(task_id)
        return {"task_id": task_id, "explanation": explanation}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/metrics")
async def metrics():
    return {
        "processed": processed_count,
        "failed": failed_count,
        "active_tasks": len(results_store)
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
