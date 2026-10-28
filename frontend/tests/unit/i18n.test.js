import test from "node:test";
import assert from "node:assert/strict";
import { detectLanguage, saveLanguagePreference } from "../../src/i18n.js";

function withStorage(t, storage) {
  const descriptor = Object.getOwnPropertyDescriptor(
    globalThis,
    "localStorage",
  );
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: storage,
  });
  t.after(() => {
    if (descriptor)
      Object.defineProperty(globalThis, "localStorage", descriptor);
    else delete globalThis.localStorage;
  });
}

test("English is the default, including visitors with the old automatic Portuguese setting", (t) => {
  const entries = new Map([["lang", "pt"]]);
  withStorage(t, { getItem: (key) => entries.get(key) });
  assert.equal(detectLanguage(), "en");
});

test("only an explicit supported language preference overrides English", (t) => {
  const entries = new Map();
  withStorage(t, {
    getItem: (key) => entries.get(key),
    setItem: (key, value) => entries.set(key, value),
  });
  saveLanguagePreference("pt");
  assert.equal(detectLanguage(), "pt");
  saveLanguagePreference("en");
  assert.equal(detectLanguage(), "en");
  saveLanguagePreference("unknown");
  assert.equal(detectLanguage(), "en");
  entries.set("preferredLanguage", "unknown");
  assert.equal(detectLanguage(), "en");
});

test("unavailable storage does not prevent language selection or English fallback", (t) => {
  withStorage(t, {
    getItem() {
      throw new Error("blocked");
    },
    setItem() {
      throw new Error("blocked");
    },
  });
  assert.equal(detectLanguage(), "en");
  assert.doesNotThrow(() => saveLanguagePreference("pt"));
});
