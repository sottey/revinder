"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const { handler, _test } = require("./index");

test("itemStatusValue maps unprocessed and pending to pending", () => {
  assert.equal(_test.itemStatusValue("unprocessed"), "pending");
  assert.equal(_test.itemStatusValue("pending"), "pending");
});

test("itemStatusValue maps failed to failed", () => {
  assert.equal(_test.itemStatusValue("failed"), "failed");
});

test("ItemStatusIntent reports unprocessed count", async () => {
  const originalFetch = global.fetch;
  const originalBaseURL = process.env.REVINDER_BRIDGE_BASE_URL;
  const originalToken = process.env.REVINDER_BRIDGE_TOKEN;

  process.env.REVINDER_BRIDGE_BASE_URL = "https://bridge.example";
  process.env.REVINDER_BRIDGE_TOKEN = "test-token";
  global.fetch = async function fetch(url, options) {
    assert.equal(url, "https://bridge.example/api/items?status=pending");
    assert.equal(options.method, "GET");
    assert.equal(options.headers.Authorization, "Bearer test-token");
    return {
      ok: true,
      json: async () => [{ id: 1 }, { id: 2 }]
    };
  };

  try {
    const response = await handler(intentEvent("ItemStatusIntent", {
      StatusType: "unprocessed"
    }));
    assert.equal(response.response.outputSpeech.text, "There are 2 unprocessed items.");
    assert.equal(response.response.shouldEndSession, true);
  } finally {
    global.fetch = originalFetch;
    restoreEnv("REVINDER_BRIDGE_BASE_URL", originalBaseURL);
    restoreEnv("REVINDER_BRIDGE_TOKEN", originalToken);
  }
});

test("ItemStatusIntent reports failed count", async () => {
  const originalFetch = global.fetch;
  const originalBaseURL = process.env.REVINDER_BRIDGE_BASE_URL;
  const originalToken = process.env.REVINDER_BRIDGE_TOKEN;

  process.env.REVINDER_BRIDGE_BASE_URL = "https://bridge.example";
  process.env.REVINDER_BRIDGE_TOKEN = "test-token";
  global.fetch = async function fetch(url) {
    assert.equal(url, "https://bridge.example/api/items?status=failed");
    return {
      ok: true,
      json: async () => [{ id: 1 }]
    };
  };

  try {
    const response = await handler(intentEvent("ItemStatusIntent", {
      StatusType: "failed"
    }));
    assert.equal(response.response.outputSpeech.text, "There is 1 failed item.");
  } finally {
    global.fetch = originalFetch;
    restoreEnv("REVINDER_BRIDGE_BASE_URL", originalBaseURL);
    restoreEnv("REVINDER_BRIDGE_TOKEN", originalToken);
  }
});

test("ItemStatusIntent reports no items", async () => {
  const originalFetch = global.fetch;
  const originalBaseURL = process.env.REVINDER_BRIDGE_BASE_URL;
  const originalToken = process.env.REVINDER_BRIDGE_TOKEN;

  process.env.REVINDER_BRIDGE_BASE_URL = "https://bridge.example";
  process.env.REVINDER_BRIDGE_TOKEN = "test-token";
  global.fetch = async function fetch() {
    return {
      ok: true,
      json: async () => []
    };
  };

  try {
    const response = await handler(intentEvent("ItemStatusIntent", {
      StatusType: "failed"
    }));
    assert.equal(response.response.outputSpeech.text, "There are no failed items.");
  } finally {
    global.fetch = originalFetch;
    restoreEnv("REVINDER_BRIDGE_BASE_URL", originalBaseURL);
    restoreEnv("REVINDER_BRIDGE_TOKEN", originalToken);
  }
});

function intentEvent(name, slots) {
  const alexaSlots = {};
  for (const [slotName, value] of Object.entries(slots)) {
    alexaSlots[slotName] = { value };
  }
  return {
    request: {
      type: "IntentRequest",
      requestId: "test-request",
      intent: {
        name,
        slots: alexaSlots
      }
    }
  };
}

function restoreEnv(name, value) {
  if (value === undefined) {
    delete process.env[name];
    return;
  }
  process.env[name] = value;
}
