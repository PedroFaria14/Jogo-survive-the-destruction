import { useEffect } from "react";

function typing(event) {
  const el = event.target;
  return (
    el?.isContentEditable ||
    ["INPUT", "TEXTAREA", "SELECT", "BUTTON"].includes(el?.tagName)
  );
}

const keys = {
  a: "left",
  arrowleft: "left",
  d: "right",
  arrowright: "right",
  w: "jump",
  " ": "jump",
  shift: "dash",
};

export function useGameControls({ active, keysRef, send, onJump, onDash }) {
  useEffect(() => {
    if (!active) return;
    const releaseAll = () => {
      keysRef.current = { left: false, right: false, jump: false, dash: false };
      send();
    };
    const down = (event) => {
      const key = keys[event.key.toLowerCase()];
      if (!key || typing(event)) return;
      event.preventDefault();
      if (event.repeat) return;
      keysRef.current[key] = true;
      if (key === "jump") onJump();
      if (key === "dash") onDash();
      send();
    };
    const up = (event) => {
      const key = keys[event.key.toLowerCase()];
      if (!key) return;
      keysRef.current[key] = false;
      send();
    };
    const visibility = () => {
      if (document.hidden) releaseAll();
    };
    window.addEventListener("keydown", down);
    window.addEventListener("keyup", up);
    window.addEventListener("blur", releaseAll);
    document.addEventListener("visibilitychange", visibility);
    return () => {
      window.removeEventListener("keydown", down);
      window.removeEventListener("keyup", up);
      window.removeEventListener("blur", releaseAll);
      document.removeEventListener("visibilitychange", visibility);
      releaseAll();
    };
  }, [active, keysRef, send, onJump, onDash]);
}
