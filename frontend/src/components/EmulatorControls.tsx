import { useRef } from "react";
import type { ControlsConfig } from "../api/types";
import { resolveControls, type ControlButton } from "../lib/emulatorControls";

type KeyHandler = (subcmd: "keydown" | "keyup", keycode: number) => void;

function ControlKey({ button, onKey }: { button: ControlButton; onKey: KeyHandler }) {
  const pressedRef = useRef(false);

  function handleDown(event: React.PointerEvent<HTMLButtonElement>) {
    // Prevents focus grab and compatibility mouse events on touch.
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    onKey("keydown", button.keycode);
    pressedRef.current = !button.edge;
  }

  function handleUp() {
    if (!pressedRef.current) return;
    pressedRef.current = false;
    onKey("keyup", button.keycode);
  }

  return (
    <button
      type="button"
      className="emulator-controls__key"
      onPointerDown={handleDown}
      onPointerUp={handleUp}
      onPointerCancel={handleUp}
      onContextMenu={(event) => event.preventDefault()}
    >
      {button.label}
    </button>
  );
}

type Props = {
  controls: ControlsConfig | null | undefined;
  onKey: KeyHandler;
};

export default function EmulatorControls({ controls, onKey }: Props) {
  const rows = resolveControls(controls);
  return (
    <div className="emulator-controls">
      {rows.map((row, rowIndex) => (
        <div
          key={rowIndex}
          className="emulator-controls__row"
          style={{ gridTemplateColumns: `repeat(${row.length}, 1fr)` }}
        >
          {row.map((button, cellIndex) =>
            button ? (
              <ControlKey key={cellIndex} button={button} onKey={onKey} />
            ) : (
              <span key={cellIndex} aria-hidden="true" />
            ),
          )}
        </div>
      ))}
    </div>
  );
}
