from fastapi import APIRouter, HTTPException, Query
from fastapi.responses import JSONResponse
import openai
from cache import get_assistant, set_assistant
import os
from main import app
import json
from typing import Annotated
from pydantic import BaseModel
from typing import List
from asyncio import gather

router = APIRouter()

created_assistants = {}

def generate_prompt(type_name, count, properties, ansestry):
    return f'Please create {count} subtypes of {type_name}. Type {type_name} has the properties {properties} and an ansestry of {ansestry}'

model = "gpt-4o"

openai.api_key = os.getenv("OPENAI_API_KEY")

class TypeRequest(BaseModel):
    type: str
    count: Annotated[int, Query(gt=0, lt=11)] = 1 

class Subtype(BaseModel):
    parent: str | None = None 
    type: str
    properties: dict[str, str | List[str]] #should this be a list of tuples? 
    ancestry: List[str]
    children: List[str]
    book: str

class Book(BaseModel):
    name: str
    instructions: str
    fields: List[str]
    field_descriptions: List[str]
    assistant: str

@router.post("/type")
async def generate_type(request: TypeRequest) -> List[Subtype]:

    parent_raw = await app.subtypes_collection.find_one({ "type": request.type })
    if not parent_raw:
        raise HTTPException(status_code=404, detail="Parent not found")
    parent = Subtype(**parent_raw)

    book_raw = await app.book_collection.find_one({ "name": parent.book })
    book = Book(**book_raw)

    if book.name not in created_assistants:
        new_assistant = openai.beta.assistants.create(
            name=book.name,
            instructions=book.instructions,
            model=model
        )
        created_assistants[book.name] = new_assistant.id
    
    assistant_id = created_assistants[book.name]

    thread = openai.beta.threads.create()
    prompt = generate_prompt(request.type, request.count, parent.properties, parent.ancestry)

    openai.beta.threads.messages.create(
        thread_id=thread.id,
        role="user",
        content=prompt
    )
    run = openai.beta.threads.runs.create(
        thread_id=thread.id,
        assistant_id=assistant_id
    )

    while True:
        run_status = openai.beta.threads.runs.retrieve(thread_id=thread.id, run_id=run.id)
        # need to have fail safe or timeout status "completed" not returned
        if run_status.status == "completed":
            messages = openai.beta.threads.messages.list(thread_id=thread.id)
            response = next(msg for msg in messages.data if msg.role == "assistant").content[0].text.value

            # Save chat to Atlas
            app.types_log_collection.insert_one({
                "assistant_id": assistant_id,
                "thread_id": thread.id,
                "user_message": prompt,
                "assistant_response": response,
                "timestamp": run_status.completed_at 
            })

            #need better parser
            parsed_response = "\n".join(response.split("\n")[1:-1])

            subtypes = json.loads(parsed_response) # use a model here?

            #cap at 10 children (placeholder)
            remaining_children = 10 - len(parent.children)
            db_subtypes = []
            db_tasks = []
            for subtype in subtypes[:remaining_children]:
                #dynamicly generate pydandic model here?
                if "name" not in subtype:
                    print(f'invalid parse for {subtype}')
                    continue

                # Handle duplicate types
                # Can optimize to have a type book hash table in memory but would have distributed issues
                # This can be better optimized.
                # Perhaps the uniqueness should be ancestry-based instead of based on typename?
                # If this is the case, then it is as simple as checking the parent's children
                # Maybe this could be done in one bulk lookup? 
                # 10 children takes aboue 0.03 seconds of waiting
                empty_check = await app.subtypes_collection.count_documents({ "type": subtype["name"], "book": book.name })
                if empty_check != 0:
                    print('duplicate type')
                    continue

                query = { "type": request.type }
                update = {"$push": {"children": subtype["name"]}}
                app.subtypes_collection.update_one(query, update)

                new_ancestry = parent.ancestry
                new_ancestry.append(request.type)

                properties = {}
                for field in book.fields:
                    properties[field] = subtype[field]

                db_subtype = Subtype(parent=request.type, type=subtype["name"], properties=properties, ancestry=new_ancestry, children=[], book=book.name)
                db_task = app.subtypes_collection.insert_one(db_subtype.model_dump())
                db_subtypes.append(db_subtype)
                db_tasks.append(db_task)

            await gather(*db_tasks)
            break

    return db_subtypes
