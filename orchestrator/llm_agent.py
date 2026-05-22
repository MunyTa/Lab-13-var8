"""
LLM Agent for generating explanations and insights
Uses Ollama for local LLM inference (optional)
"""

import os
import logging
from typing import Optional, Dict, Any

logger = logging.getLogger(__name__)


class LLMFeedbackAgent:
    """
    LLM-powered agent that generates explanations for HR decisions.
    Uses Ollama for local inference when OLLAMA_URL is configured.
    """
    
    def __init__(self):
        self.ollama_url = os.getenv("OLLAMA_URL", "http://localhost:11434")
        self.model = os.getenv("OLLAMA_MODEL", "llama2")
        self.enabled = os.getenv("OLLAMA_URL") is not None
        
        if self.enabled:
            logger.info(f"LLM Agent initialized with Ollama at {self.ollama_url}")
        else:
            logger.info("LLM Agent running in simulation mode (Ollama not configured)")
    
    async def explain_result(self, task_id: str) -> str:
        """
        Generate an explanation for a candidate evaluation result.
        """
        if not self.enabled:
            return self._generate_mock_explanation(task_id)
        
        try:
            import httpx
            
            prompt = f"""Analyze the following HR evaluation and provide insights:

Task ID: {task_id}

Consider:
1. Strengths of the candidate
2. Areas for improvement
3. Hiring recommendation with justification

Provide a concise, professional response.
"""
            
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.post(
                    f"{self.ollama_url}/api/generate",
                    json={
                        "model": self.model,
                        "prompt": prompt,
                        "stream": False
                    }
                )
                
                if response.status_code == 200:
                    result = response.json()
                    return result.get("response", "No explanation generated")
                else:
                    logger.warning(f"Ollama returned status {response.status_code}")
                    return self._generate_mock_explanation(task_id)
                    
        except Exception as e:
            logger.error(f"LLM request failed: {e}")
            return self._generate_mock_explanation(task_id)
    
    async def generate_interview_questions(self, candidate_profile: Dict[str, Any]) -> list:
        """
        Generate tailored interview questions based on candidate profile.
        """
        if not self.enabled:
            return self._generate_mock_questions(candidate_profile)
        
        try:
            import httpx
            
            prompt = f"""Based on this candidate profile, generate 5 interview questions:

{str(candidate_profile)}

Focus on:
- Technical skills assessment
- Behavioral questions
- Situational questions

Return questions as a JSON array.
"""
            
            async with httpx.AsyncClient(timeout=30.0) as client:
                response = await client.post(
                    f"{self.ollama_url}/api/generate",
                    json={
                        "model": self.model,
                        "prompt": prompt,
                        "stream": False
                    }
                )
                
                if response.status_code == 200:
                    result = response.json()
                    return self._parse_questions(result.get("response", ""))
                else:
                    return self._generate_mock_questions(candidate_profile)
                    
        except Exception as e:
            logger.error(f"LLM request failed: {e}")
            return self._generate_mock_questions(candidate_profile)
    
    def _generate_mock_explanation(self, task_id: str) -> str:
        return f"""HR Pipeline Analysis for Task {task_id}:

Summary:
- Resume was successfully parsed and skills extracted
- Candidate matched against vacancy requirements
- Interview was scheduled with relevant interviewers
- Feedback was collected and analyzed

Recommendation:
Based on the multi-agent pipeline analysis, this candidate demonstrates strong alignment with the position requirements. The interview feedback indicates positive reception from the hiring team.

Next Steps:
1. Review detailed feedback in the dashboard
2. Coordinate with HR for offer preparation
3. Schedule follow-up interviews if needed
"""
    
    def _generate_mock_questions(self, profile: Dict[str, Any]) -> list:
        return [
            "Can you describe a complex Go project you worked on?",
            "How do you handle debugging in distributed systems?",
            "Describe your experience with microservices architecture.",
            "How do you approach code reviews?",
            "Tell me about a time you had to make a critical technical decision."
        ]
    
    def _parse_questions(self, response: str) -> list:
        questions = []
        for line in response.split('\n'):
            line = line.strip()
            if line and (line.startswith('-') or line.startswith('1.') or line[0].isdigit()):
                questions.append(line.lstrip('- 0123456789.').strip())
        return questions if questions else self._generate_mock_questions({})
