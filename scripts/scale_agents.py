#!/usr/bin/env python3
"""
Dynamic agent scaling script for HR Multi-Agent System
Scales Go agents based on queue length or manual count
"""

import argparse
import asyncio
import json
import sys
import os
from typing import Optional

try:
    import httpx
except ImportError:
    print("httpx not installed. Install with: pip install httpx")
    sys.exit(1)


class AgentScaler:
    def __init__(self, base_url: str = "http://localhost:8000"):
        self.base_url = base_url
        self.nats_url = os.getenv("NATS_URL", "nats://localhost:4222")
    
    async def get_queue_stats(self) -> dict:
        """Get queue statistics from NATS monitoring"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{self.base_url}/stats")
                return response.json()
        except Exception as e:
            return {"error": str(e)}
    
    async def scale_agent(self, agent_type: str, replicas: int) -> dict:
        """
        Scale a specific agent type to specified replicas.
        Uses docker-compose to scale services.
        """
        import subprocess
        
        valid_agents = ["resume-parser", "vacancy-matcher", "interview-scheduler", "feedback-agent"]
        
        if agent_type not in valid_agents:
            print(f"Invalid agent type. Choose from: {', '.join(valid_agents)}")
            return {"error": "Invalid agent type"}
        
        service_name = f"hr-multi-agent-network-{agent_type.replace('-', '_')}"
        
        try:
            result = subprocess.run(
                ["docker-compose", "up", "-d", "--scale", f"{agent_type}={replicas}", agent_type],
                capture_output=True,
                text=True,
                cwd=os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
            )
            
            if result.returncode == 0:
                return {
                    "agent_type": agent_type,
                    "replicas": replicas,
                    "status": "scaled"
                }
            else:
                return {
                    "error": result.stderr,
                    "status": "failed"
                }
        except FileNotFoundError:
            return {"error": "docker-compose not found. Make sure Docker is installed."}
        except Exception as e:
            return {"error": str(e)}
    
    async def check_agent_health(self) -> dict:
        """Check health of all agents"""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{self.base_url}/health")
                return response.json()
        except Exception as e:
            return {"error": str(e)}


async def main():
    parser = argparse.ArgumentParser(description="Scale HR Multi-Agent System agents")
    parser.add_argument("--detectors", type=int, help="Number of vacancy matcher replicas")
    parser.add_argument("--parsers", type=int, help="Number of resume parser replicas")
    parser.add_argument("--schedulers", type=int, help="Number of interview scheduler replicas")
    parser.add_argument("--feedback", type=int, help="Number of feedback agent replicas")
    parser.add_argument("--status", action="store_true", help="Show current status")
    parser.add_argument("--url", default="http://localhost:8000", help="API base URL")
    
    args = parser.parse_args()
    
    scaler = AgentScaler(args.url)
    
    if args.status:
        print("Checking agent health...")
        health = await scaler.check_agent_health()
        print(json.dumps(health, indent=2))
        return
    
    scaled = []
    
    if args.detectors:
        result = await scaler.scale_agent("vacancy-matcher", args.detectors)
        scaled.append(result)
    
    if args.parsers:
        result = await scaler.scale_agent("resume-parser", args.parsers)
        scaled.append(result)
    
    if args.schedulers:
        result = await scaler.scale_agent("interview-scheduler", args.schedulers)
        scaled.append(result)
    
    if args.feedback:
        result = await scaler.scale_agent("feedback-agent", args.feedback)
        scaled.append(result)
    
    if scaled:
        print("Scaling results:")
        print(json.dumps(scaled, indent=2))
    else:
        parser.print_help()


if __name__ == "__main__":
    asyncio.run(main())
