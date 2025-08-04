#!/usr/bin/env python3
"""
Sample Python file for testing document indexing.
This file demonstrates various Python constructs and patterns.
"""

import asyncio
import logging
from typing import Dict, List, Optional
from dataclasses import dataclass
from abc import ABC, abstractmethod


@dataclass
class Task:
    """Represents a task in the system."""
    id: str
    title: str
    description: str
    completed: bool = False
    priority: int = 1


class TaskRepository(ABC):
    """Abstract repository for task persistence."""
    
    @abstractmethod
    async def save(self, task: Task) -> bool:
        """Save a task to storage."""
        pass
    
    @abstractmethod
    async def get(self, task_id: str) -> Optional[Task]:
        """Retrieve a task by ID."""
        pass
    
    @abstractmethod
    async def list_all(self) -> List[Task]:
        """List all tasks."""
        pass


class InMemoryTaskRepository(TaskRepository):
    """In-memory implementation of task repository."""
    
    def __init__(self):
        self._tasks: Dict[str, Task] = {}
        self._logger = logging.getLogger(__name__)
    
    async def save(self, task: Task) -> bool:
        """Save a task to memory."""
        try:
            self._tasks[task.id] = task
            self._logger.info(f"Saved task: {task.id}")
            return True
        except Exception as e:
            self._logger.error(f"Failed to save task {task.id}: {e}")
            return False
    
    async def get(self, task_id: str) -> Optional[Task]:
        """Retrieve a task by ID."""
        return self._tasks.get(task_id)
    
    async def list_all(self) -> List[Task]:
        """List all tasks."""
        return list(self._tasks.values())


class TaskService:
    """Service for managing tasks."""
    
    def __init__(self, repository: TaskRepository):
        self.repository = repository
        self._logger = logging.getLogger(__name__)
    
    async def create_task(self, title: str, description: str, priority: int = 1) -> Task:
        """Create a new task."""
        if not title.strip():
            raise ValueError("Task title cannot be empty")
        
        task = Task(
            id=f"task_{len(await self.repository.list_all()) + 1}",
            title=title.strip(),
            description=description.strip(),
            priority=max(1, min(5, priority))  # Clamp priority between 1-5
        )
        
        success = await self.repository.save(task)
        if not success:
            raise RuntimeError(f"Failed to save task: {task.id}")
        
        self._logger.info(f"Created task: {task.title}")
        return task
    
    async def complete_task(self, task_id: str) -> bool:
        """Mark a task as completed."""
        task = await self.repository.get(task_id)
        if not task:
            return False
        
        task.completed = True
        return await self.repository.save(task)
    
    async def get_pending_tasks(self) -> List[Task]:
        """Get all pending (not completed) tasks."""
        all_tasks = await self.repository.list_all()
        return [task for task in all_tasks if not task.completed]


async def main():
    """Main function demonstrating the task system."""
    logging.basicConfig(level=logging.INFO)
    
    # Initialize repository and service
    repository = InMemoryTaskRepository()
    service = TaskService(repository)
    
    # Create some tasks
    task1 = await service.create_task("Learn Python", "Study Python fundamentals", 3)
    task2 = await service.create_task("Build web app", "Create a Flask application", 4)
    task3 = await service.create_task("Write tests", "Add unit tests for all modules", 5)
    
    # List pending tasks
    pending = await service.get_pending_tasks()
    print(f"Pending tasks: {len(pending)}")
    
    # Complete a task
    await service.complete_task(task1.id)
    
    # List pending tasks again
    pending = await service.get_pending_tasks()
    print(f"Pending tasks after completion: {len(pending)}")


if __name__ == "__main__":
    asyncio.run(main())