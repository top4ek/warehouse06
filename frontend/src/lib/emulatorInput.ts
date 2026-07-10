/** Host key code for vector06js (`keyboard.js` / rom.js message bridge). */
export function hostKeyCode(event: KeyboardEvent): number {
  if (event.keyCode > 0) return event.keyCode;
  const fromCode: Record<string, number> = {
    ArrowLeft: 37,
    ArrowUp: 38,
    ArrowRight: 39,
    ArrowDown: 40,
    Enter: 13,
    Backspace: 8,
    Tab: 9,
    Escape: 27,
    Space: 32,
    ShiftLeft: 16,
    ShiftRight: 16,
    ControlLeft: 17,
    ControlRight: 17,
    AltLeft: 18,
    AltRight: 18,
  };
  if (event.code in fromCode) return fromCode[event.code]!;
  if (event.code.startsWith("Key")) return event.code.charCodeAt(3);
  if (event.code.startsWith("Digit")) return event.code.charCodeAt(5);
  return 0;
}

export function postEmulatorKey(
  frame: HTMLIFrameElement | null | undefined,
  subcmd: "keydown" | "keyup",
  keycode: number,
) {
  if (!keycode || !frame?.contentWindow) return;
  frame.contentWindow.postMessage({ cmd: "input", subcmd, keycode }, window.location.origin);
}
