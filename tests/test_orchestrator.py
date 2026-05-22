"""Tests for HR Orchestrator"""

import pytest
import asyncio
from unittest.mock import AsyncMock, MagicMock, patch
from orchestrator.main import HREventInput, TaskRequest, TaskResponse


class TestHREventInput:
    def test_create_hr_event_input(self):
        input_data = HREventInput(
            candidate_name="John Doe",
            raw_resume="Experienced Go developer...",
            vacancy_title="Senior Developer"
        )
        
        assert input_data.candidate_name == "John Doe"
        assert "Go developer" in input_data.raw_resume
        assert input_data.vacancy_title == "Senior Developer"
    
    def test_hr_event_input_defaults(self):
        input_data = HREventInput(
            candidate_name="Jane Smith",
            raw_resume="Python developer with 5 years experience"
        )
        
        assert input_data.vacancy_title == "Go Backend Developer"
        assert input_data.preferred_interview_date is None


class TestTaskRequest:
    def test_create_task_request(self):
        task = TaskRequest(
            task_type="resume_parse",
            payload={"raw_text": "Test resume"},
            timeout=30
        )
        
        assert task.task_type == "resume_parse"
        assert task.payload["raw_text"] == "Test resume"
        assert task.timeout == 30
    
    def test_task_request_custom_timeout(self):
        task = TaskRequest(
            task_type="match",
            payload={},
            timeout=60
        )
        
        assert task.timeout == 60


class TestTaskResponse:
    def test_successful_response(self):
        response = TaskResponse(
            task_id="test-123",
            status="completed",
            result={"score": 85.5}
        )
        
        assert response.task_id == "test-123"
        assert response.status == "completed"
        assert response.result["score"] == 85.5
    
    def test_error_response(self):
        response = TaskResponse(
            task_id="test-456",
            status="timeout",
            error="Task exceeded 30 seconds"
        )
        
        assert response.status == "timeout"
        assert "timeout" in response.error


@pytest.mark.asyncio
async def test_async_task_processing():
    """Test async task processing simulation"""
    results = []
    
    async def process_task(task_id: str):
        await asyncio.sleep(0.1)
        return {"task_id": task_id, "processed": True}
    
    tasks = [process_task(f"task-{i}") for i in range(3)]
    results = await asyncio.gather(*tasks)
    
    assert len(results) == 3
    assert all(r["processed"] for r in results)


def test_stats_calculation():
    """Test statistics calculation"""
    processed = 10
    failed = 2
    
    total = processed + failed
    success_rate = (processed / total * 100) if total > 0 else 0
    
    assert success_rate == 83.33
    assert total == 12


def test_empty_stats():
    """Test statistics with no tasks"""
    processed = 0
    failed = 0
    
    total = processed + failed
    success_rate = (processed / total * 100) if total > 0 else 0
    
    assert success_rate == 0
    assert total == 0
