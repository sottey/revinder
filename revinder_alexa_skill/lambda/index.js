"use strict";

const DEFAULT_LIST_NAME = "Home";
const DEFAULT_TIME_ZONE = process.env.DEFAULT_TIME_ZONE || "America/Los_Angeles";

exports.handler = async function handler(event) {
  try {
    const request = event.request || {};

    if (request.type === "LaunchRequest") {
      return alexaResponse("Tell me the task to add or memory to save.", false, "What should I add?");
    }

    if (request.type !== "IntentRequest") {
      return alexaResponse("I could not handle that request.", true);
    }

    const intentName = request.intent && request.intent.name;

    if (intentName === "AMAZON.HelpIntent") {
      return alexaResponse(
        "Say, add a task, then the task. Or say, that, then the memory to save.",
        false,
        "What should I add?"
      );
    }

    if (intentName === "AMAZON.CancelIntent" || intentName === "AMAZON.StopIntent") {
      return alexaResponse("Canceled.", true);
    }

    if (intentName === "AddTaskIntent") {
      return handleAddTask(request);
    }

    if (intentName === "RememberIntent") {
      return handleRemember(request);
    }

    return alexaResponse("I could not handle that request.", true);
  } catch (error) {
    console.error(error);
    return alexaResponse("I could not add that.", true);
  }
};

async function handleAddTask(request) {
    const slots = request.intent.slots || {};
    const taskTitle = slotValue(slots.TaskText);

    if (!taskTitle) {
      return alexaResponse("I did not hear the task.", false, "What task should I add?");
    }

    const dueDate = slotValue(slots.DueDate);
    const dueTime = slotValue(slots.DueTime);
    const dueAt = buildDueAt(dueDate, dueTime);
    const tags = parseTags(slotValue(slots.Tags));

    await createItem({
      revinder_id: request.requestId,
      source: "alexa",
      type: "task",
      text: taskTitle,
      title: taskTitle,
      list_name: DEFAULT_LIST_NAME,
      due_at: dueAt,
      notes: null,
      tags,
      metadata: {
        due_date: dueDate || null,
        due_time: dueTime || null,
        all_day: Boolean(dueDate && !dueTime)
      }
    });

    return alexaResponse("Added.", true);
}

async function handleRemember(request) {
  const slots = request.intent.slots || {};
  const memoryText = slotValue(slots.MemoryText);

  if (!memoryText) {
    return alexaResponse("I did not hear the memory.", false, "What should I remember?");
  }

  await createItem({
    revinder_id: request.requestId,
    source: "alexa",
    type: "memory",
    text: memoryText,
    title: memoryText,
    list_name: "",
    due_at: null,
    notes: null,
    tags: [],
    metadata: {}
  });

  return alexaResponse("Remembered.", true);
}

function alexaResponse(text, shouldEndSession, repromptText) {
  const response = {
    outputSpeech: {
      type: "PlainText",
      text
    },
    shouldEndSession
  };

  if (repromptText) {
    response.reprompt = {
      outputSpeech: {
        type: "PlainText",
        text: repromptText
      }
    };
  }

  return {
    version: "1.0",
    response
  };
}

function slotValue(slot) {
  if (!slot || !slot.value) {
    return "";
  }
  return slot.value.trim();
}

function parseTags(value) {
  if (!value) {
    return [];
  }

  return value
    .split(/\s*(?:,| and )\s*/i)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function buildDueAt(dateValue, timeValue) {
  if (!dateValue) {
    return null;
  }
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
    return null;
  }
  if (!timeValue) {
    return `${dateValue}T00:00:00${timeZoneOffset(dateValue, "00:00")}`;
  }
  if (!/^\d{2}:\d{2}$/.test(timeValue)) {
    return null;
  }

  return `${dateValue}T${timeValue}:00${timeZoneOffset(dateValue, timeValue)}`;
}

function timeZoneOffset(dateValue, timeValue) {
  const utcDate = new Date(`${dateValue}T${timeValue}:00.000Z`);
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: DEFAULT_TIME_ZONE,
    timeZoneName: "shortOffset",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  }).formatToParts(utcDate);
  const zoneName = parts.find((part) => part.type === "timeZoneName");
  const match = zoneName && zoneName.value.match(/^GMT([+-])(\d{1,2})(?::(\d{2}))?$/);

  if (!match) {
    return "Z";
  }

  const sign = match[1];
  const hours = match[2].padStart(2, "0");
  const minutes = (match[3] || "00").padStart(2, "0");
  return `${sign}${hours}:${minutes}`;
}

async function createItem(payload) {
  const baseUrl = requiredEnv("REVINDER_BRIDGE_BASE_URL").replace(/\/+$/, "");
  const token = requiredEnv("REVINDER_BRIDGE_TOKEN");
  const response = await fetch(`${baseUrl}/api/items`, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${token}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`revinder_bridge returned ${response.status}: ${body}`);
  }

  return response.json();
}

function requiredEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}
